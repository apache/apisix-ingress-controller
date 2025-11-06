// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package apisix

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/go-logr/logr"
	networkingv1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	adcclient "github.com/apache/apisix-ingress-controller/internal/adc/client"
	"github.com/apache/apisix-ingress-controller/internal/adc/translator"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	"github.com/apache/apisix-ingress-controller/internal/manager/readiness"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/internal/provider/common"
	"github.com/apache/apisix-ingress-controller/internal/types"
	"github.com/apache/apisix-ingress-controller/internal/utils"
)

const (
	ProviderTypeAPISIX = "apisix"

	RetryBaseDelay = 1 * time.Second
	RetryMaxDelay  = 1000 * time.Second

	MinSyncPeriod = 1 * time.Second
)

type apisixProvider struct {
	provider.Options
	sync.Mutex

	translator *translator.Translator

	updater         status.Updater
	statusUpdateMap map[types.NamespacedNameKind][]string

	readier readiness.ReadinessManager

	syncCh chan struct{}

	client *adcclient.Client
	log    logr.Logger
}

func New(log logr.Logger, updater status.Updater, readier readiness.ReadinessManager, opts ...provider.Option) (provider.Provider, error) {
	o := provider.Options{}
	o.ApplyOptions(opts)
	if o.DefaultBackendMode == "" {
		o.DefaultBackendMode = ProviderTypeAPISIX
	}

	cli, err := adcclient.New(log, o.SyncTimeout)
	if err != nil {
		return nil, err
	}

	return &apisixProvider{
		client:     cli,
		Options:    o,
		translator: translator.NewTranslator(log),
		updater:    updater,
		readier:    readier,
		syncCh:     make(chan struct{}, 1),
		log:        log.WithName("provider"),
	}, nil
}

func (d *apisixProvider) Register(pathPrefix string, mux *http.ServeMux) {
	d.client.ADCDebugProvider.SetupHandler(pathPrefix, mux)
}

func (d *apisixProvider) Update(ctx context.Context, tctx *provider.TranslateContext, obj client.Object) error {
	d.log.V(1).Info("updating object", "object", obj)
	var (
		result        *translator.TranslateResult
		resourceTypes []string
		err           error
	)

	rk := utils.NamespacedNameKind(obj)

	switch t := obj.(type) {
	case *gatewayv1.HTTPRoute:
		result, err = d.translator.TranslateHTTPRoute(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, adctypes.TypeService)
	case *gatewayv1alpha2.TCPRoute:
		result, err = d.translator.TranslateTCPRoute(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, adctypes.TypeService)
	case *gatewayv1alpha2.UDPRoute:
		result, err = d.translator.TranslateUDPRoute(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, adctypes.TypeService)
	case *gatewayv1alpha2.TLSRoute:
		result, err = d.translator.TranslateTLSRoute(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, adctypes.TypeService)
	case *gatewayv1.GRPCRoute:
		result, err = d.translator.TranslateGRPCRoute(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, adctypes.TypeService)
	case *gatewayv1.Gateway:
		result, err = d.translator.TranslateGateway(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, adctypes.TypeGlobalRule, adctypes.TypeSSL, adctypes.TypePluginMetadata)
	case *networkingv1.Ingress:
		result, err = d.translator.TranslateIngress(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, adctypes.TypeService, adctypes.TypeSSL)
	case *v1alpha1.Consumer:
		result, err = d.translator.TranslateConsumerV1alpha1(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, adctypes.TypeConsumer)
	case *networkingv1.IngressClass:
		result, err = d.translator.TranslateIngressClass(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, adctypes.TypeGlobalRule, adctypes.TypePluginMetadata)
	case *apiv2.ApisixRoute:
		result, err = d.translator.TranslateApisixRoute(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, adctypes.TypeService)
	case *apiv2.ApisixGlobalRule:
		result, err = d.translator.TranslateApisixGlobalRule(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, adctypes.TypeGlobalRule)
	case *apiv2.ApisixTls:
		result, err = d.translator.TranslateApisixTls(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, adctypes.TypeSSL)
	case *apiv2.ApisixConsumer:
		result, err = d.translator.TranslateApisixConsumer(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, adctypes.TypeConsumer)
	case *v1alpha1.GatewayProxy:
		return d.updateConfigForGatewayProxy(tctx, t)
	}
	if err != nil {
		return err
	}
	if result == nil {
		return nil
	}

	configs, err := d.buildConfig(tctx, rk)
	if err != nil {
		return err
	}

	if len(configs) == 0 {
		return nil
	}

	defer d.syncNotify()

	task := adcclient.Task{
		Key:           rk,
		Name:          rk.String(),
		Labels:        label.GenLabel(obj),
		Configs:       configs,
		ResourceTypes: resourceTypes,
		Resources: &adctypes.Resources{
			GlobalRules:    result.GlobalRules,
			PluginMetadata: result.PluginMetadata,
			Services:       result.Services,
			SSLs:           result.SSL,
			Consumers:      result.Consumers,
		},
	}
	d.log.V(1).Info("updating config", "task", task)

	return d.client.UpdateConfig(ctx, task)
}

func (d *apisixProvider) Delete(ctx context.Context, obj client.Object) error {
	d.log.V(1).Info("deleting object", "object", obj)

	var resourceTypes []string
	var labels map[string]string
	switch obj.(type) {
	case *gatewayv1.HTTPRoute, *apiv2.ApisixRoute, *gatewayv1.GRPCRoute, *gatewayv1alpha2.TCPRoute, *gatewayv1alpha2.UDPRoute, *gatewayv1alpha2.TLSRoute:
		resourceTypes = append(resourceTypes, adctypes.TypeService)
		labels = label.GenLabel(obj)
	case *gatewayv1.Gateway:
		// delete all resources
	case *networkingv1.Ingress:
		resourceTypes = append(resourceTypes, adctypes.TypeService, adctypes.TypeSSL)
		labels = label.GenLabel(obj)
	case *v1alpha1.Consumer:
		resourceTypes = append(resourceTypes, adctypes.TypeConsumer)
		labels = label.GenLabel(obj)
	case *networkingv1.IngressClass:
		// delete all resources
	case *apiv2.ApisixGlobalRule:
		resourceTypes = append(resourceTypes, adctypes.TypeGlobalRule)
		labels = label.GenLabel(obj)
	case *apiv2.ApisixTls:
		resourceTypes = append(resourceTypes, adctypes.TypeSSL)
		labels = label.GenLabel(obj)
	case *apiv2.ApisixConsumer:
		resourceTypes = append(resourceTypes, adctypes.TypeConsumer)
		labels = label.GenLabel(obj)
	}
	nnk := utils.NamespacedNameKind(obj)

	// Full synchronization is performed on a gateway by gateway basis
	// and it is not possible to perform scheduled synchronization
	// on deleted gateway level resources
	if len(resourceTypes) == 0 {
		return d.client.Delete(ctx, adcclient.Task{
			Key:    nnk,
			Name:   nnk.String(),
			Labels: labels,
		})
	}
	defer d.syncNotify()
	return d.client.DeleteConfig(ctx, adcclient.Task{
		Key:           nnk,
		Name:          nnk.String(),
		Labels:        labels,
		ResourceTypes: resourceTypes,
	})
}

func (d *apisixProvider) buildConfig(tctx *provider.TranslateContext, nnk types.NamespacedNameKind) (map[types.NamespacedNameKind]adctypes.Config, error) {
	configs := make(map[types.NamespacedNameKind]adctypes.Config, len(tctx.ResourceParentRefs[nnk]))
	for _, gp := range tctx.GatewayProxies {
		config, err := d.translator.TranslateGatewayProxyToConfig(tctx, &gp, d.DefaultResolveEndpoints)
		if err != nil {
			return nil, err
		}
		configs[utils.NamespacedNameKind(&gp)] = *config
	}
	return configs, nil
}

func (d *apisixProvider) Start(ctx context.Context) error {
	d.readier.WaitReady(ctx, 5*time.Minute)
	initalSyncDelay := d.InitSyncDelay
	if initalSyncDelay > 0 {
		time.AfterFunc(initalSyncDelay, d.syncNotify)
	}

	syncPeriod := d.SyncPeriod
	if syncPeriod < MinSyncPeriod {
		syncPeriod = MinSyncPeriod
	}
	ticker := time.NewTicker(syncPeriod)
	defer ticker.Stop()

	retrier := common.NewRetrier(common.NewExponentialBackoff(RetryBaseDelay, RetryMaxDelay))

	for {
		select {
		case <-d.syncCh:
		case <-ticker.C:
		case <-retrier.C():
		case <-ctx.Done():
			retrier.Reset()
			return nil
		}
		if err := d.sync(ctx); err != nil {
			d.log.Error(err, "failed to sync")
			retrier.Next()
		} else {
			retrier.Reset()
		}
	}
}

func (d *apisixProvider) sync(ctx context.Context) error {
	statusesMap, err := d.client.Sync(ctx)
	d.handleADCExecutionErrors(statusesMap)
	return err
}

func (d *apisixProvider) syncNotify() {
	select {
	case d.syncCh <- struct{}{}:
	default:
	}
}

func (d *apisixProvider) handleADCExecutionErrors(statusesMap map[string]types.ADCExecutionErrors) {
	statusUpdateMap := d.resolveADCExecutionErrors(statusesMap)
	d.handleStatusUpdate(statusUpdateMap)
	d.log.V(1).Info("handled ADC execution errors", "status_record", statusesMap, "status_update", statusUpdateMap)
}

func (d *apisixProvider) NeedLeaderElection() bool {
	return true
}

// updateConfigForGatewayProxy update config for all referrers of the GatewayProxy
func (d *apisixProvider) updateConfigForGatewayProxy(tctx *provider.TranslateContext, gp *v1alpha1.GatewayProxy) error {
	config, err := d.translator.TranslateGatewayProxyToConfig(tctx, gp, d.DefaultResolveEndpoints)
	if err != nil {
		return err
	}

	nnk := utils.NamespacedNameKind(gp)
	if config == nil {
		d.client.ConfigManager.DeleteConfig(nnk)
		return nil
	}
	referrers := tctx.GatewayProxyReferrers[utils.NamespacedName(gp)]
	d.client.ConfigManager.SetConfigRefs(nnk, referrers)
	d.client.ConfigManager.UpdateConfig(nnk, *config)
	d.syncNotify()
	return nil
}
