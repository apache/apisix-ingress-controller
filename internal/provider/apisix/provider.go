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
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-logr/logr"
	networkingv1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/adc/cache"
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

// apisixProvider owns AIC's own view of what should be live: which Kubernetes resource
// targets which GatewayProxy config (configManager) and the merged, translated resource
// snapshot per config (store). It builds the input the adc client package needs and hands
// it over on every call; the client package holds none of this state itself.
type apisixProvider struct {
	provider.Options
	sync.Mutex

	translator *translator.Translator

	store         *cache.Store
	configManager *common.ConfigManager[types.NamespacedNameKind, adctypes.Config]
	debugProvider *common.ADCDebugProvider

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

	logger := log.WithName("provider")

	cli, err := adcclient.New(logger, o.DefaultBackendMode, o.SyncTimeout)
	if err != nil {
		return nil, err
	}

	store := cache.NewStore(logger)
	configManager := common.NewConfigManager[types.NamespacedNameKind, adctypes.Config]()

	return &apisixProvider{
		client:        cli,
		store:         store,
		configManager: configManager,
		debugProvider: common.NewADCDebugProvider(store, configManager),
		Options:       o,
		translator:    translator.NewTranslator(log, o.ListenerPortMatchMode),
		updater:       updater,
		readier:       readier,
		syncCh:        make(chan struct{}, 1),
		log:           logger,
	}, nil
}

func (d *apisixProvider) Register(pathPrefix string, mux *http.ServeMux) {
	d.debugProvider.SetupHandler(pathPrefix, mux)
}

func (d *apisixProvider) Update(ctx context.Context, tctx *provider.TranslateContext, obj client.Object) error {
	d.log.V(1).Info("updating object", "object", utils.NamespacedNameKind(obj))
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
	case *gatewayv1.TCPRoute:
		result, err = d.translator.TranslateTCPRoute(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, adctypes.TypeService)
	case *gatewayv1.UDPRoute:
		result, err = d.translator.TranslateUDPRoute(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, adctypes.TypeService)
	case *gatewayv1.TLSRoute:
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

	resources := &adctypes.Resources{
		GlobalRules:    result.GlobalRules,
		PluginMetadata: result.PluginMetadata,
		Services:       result.Services,
		SSLs:           result.SSL,
		Consumers:      result.Consumers,
	}
	labels := label.GenLabel(obj)
	d.log.V(1).Info("updating config", "resourceKey", rk, "configs", configs, "resourceTypes", resourceTypes)

	return d.applyResourceState(rk, configs, resourceTypes, resources, labels)
}

func (d *apisixProvider) Delete(ctx context.Context, obj client.Object) error {
	d.log.V(1).Info("deleting object", "object", obj)

	var resourceTypes []string
	var labels map[string]string
	switch obj.(type) {
	case *gatewayv1.HTTPRoute, *apiv2.ApisixRoute, *gatewayv1.GRPCRoute, *gatewayv1.TCPRoute, *gatewayv1.UDPRoute, *gatewayv1.TLSRoute:
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
		removed, err := d.removeResourceState(nnk, resourceTypes, labels)
		if err != nil {
			return err
		}
		d.syncEvictedConfigsNow(ctx, removed, resourceTypes, labels)
		return nil
	}

	defer d.syncNotify()
	_, err := d.removeResourceState(nnk, resourceTypes, labels)
	return err
}

// applyResourceState upserts a resource's config associations and its contribution to each
// target config's cached resource snapshot -- the AIC-side bookkeeping the adc client
// package no longer holds itself.
func (d *apisixProvider) applyResourceState(
	rk types.NamespacedNameKind,
	configs map[types.NamespacedNameKind]adctypes.Config,
	resourceTypes []string,
	resources *adctypes.Resources,
	labels map[string]string,
) error {
	d.Lock()
	defer d.Unlock()

	discarded := d.configManager.Update(rk, configs)

	for _, cfg := range discarded {
		if err := d.store.Delete(cfg.Name, resourceTypes, labels); err != nil {
			return fmt.Errorf("store delete failed for config %s: %w", cfg.Name, err)
		}
	}
	for _, cfg := range configs {
		if err := d.store.Insert(cfg.Name, resourceTypes, resources, labels); err != nil {
			return fmt.Errorf("store insert failed for config %s: %w", cfg.Name, err)
		}
	}
	return nil
}

// removeResourceState forgets a resource's config associations and evicts its contribution
// from each config it used to reference, returning those configs so an immediate-push
// caller (see syncEvictedConfigsNow) knows what to push right away.
func (d *apisixProvider) removeResourceState(
	rk types.NamespacedNameKind,
	resourceTypes []string,
	labels map[string]string,
) (map[types.NamespacedNameKind]adctypes.Config, error) {
	d.Lock()
	defer d.Unlock()

	removed := d.configManager.Get(rk)
	d.configManager.Delete(rk)
	for _, cfg := range removed {
		if err := d.store.Delete(cfg.Name, resourceTypes, labels); err != nil {
			return nil, fmt.Errorf("store delete failed for config %s: %w", cfg.Name, err)
		}
	}
	return removed, nil
}

// syncEvictedConfigsNow pushes an empty resource set for each of the given configs
// immediately, instead of waiting for the next scheduled sync round. Used only when the
// deleted resource is a Gateway or IngressClass -- resourceTypes is empty for those, so the
// preceding removeResourceState call already reset each config's whole cached snapshot via
// Store.Delete, and that reset should reach the data plane promptly. Failures are logged,
// not surfaced as a status update -- this mirrors the deferred path, which only reports
// through the next scheduled sync round.
func (d *apisixProvider) syncEvictedConfigsNow(
	ctx context.Context,
	configs map[types.NamespacedNameKind]adctypes.Config,
	resourceTypes []string,
	labels map[string]string,
) {
	if len(configs) == 0 {
		return
	}
	inputs := make([]adcclient.SyncInput, 0, len(configs))
	for _, cfg := range configs {
		inputs = append(inputs, adcclient.SyncInput{
			Name:          cfg.Name,
			Config:        cfg,
			Resources:     &adctypes.Resources{},
			ResourceTypes: resourceTypes,
			Labels:        labels,
		})
	}
	if _, err := d.client.Sync(ctx, inputs); err != nil {
		d.log.Error(err, "failed to sync deleted configs", "configs", configs)
	}
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
	// Start only runs once this pod has won the election, and a leadership change is the
	// one thing that leaves the ADC sidecar holding a baseline from an earlier term: it
	// survives the manager container, the configuration it was derived from does not.
	// Rebuild every baseline from the data plane before syncing from it.
	d.client.InvalidateADCCache()

	d.log.Info("starting provider, waiting for readiness")
	d.readier.WaitReady(ctx, 5*time.Minute)
	d.log.Info("Ready detected, starting sync loop")

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
	inputs, resourceErr := d.buildSyncInputs()
	statusesMap, syncErr := d.client.Sync(ctx, inputs)
	d.handleADCExecutionErrors(statusesMap)
	return errors.Join(resourceErr, syncErr)
}

// buildSyncInputs organizes this round's full config set -- every GatewayProxy AIC
// currently knows about, each with its merged translated resource snapshot -- into the
// input the adc client package needs. The client package never gathers this itself.
func (d *apisixProvider) buildSyncInputs() ([]adcclient.SyncInput, error) {
	configs := d.configManager.List()
	if len(configs) == 0 {
		return nil, nil
	}

	inputs := make([]adcclient.SyncInput, 0, len(configs))
	var errs []error
	for _, config := range configs {
		resources, err := d.store.GetResources(config.Name)
		if err != nil {
			d.log.Error(err, "failed to get resources from store", "name", config.Name)
			errs = append(errs, fmt.Errorf("config %s: %w", config.Name, err))
			continue
		}
		inputs = append(inputs, adcclient.SyncInput{
			Name:      config.Name,
			Config:    config,
			Resources: resources,
		})
	}
	return inputs, errors.Join(errs...)
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
		d.Lock()
		d.configManager.DeleteConfig(nnk)
		d.Unlock()
		return nil
	}

	referrers := tctx.GatewayProxyReferrers[utils.NamespacedName(gp)]
	d.Lock()
	d.configManager.SetConfigRefs(nnk, referrers)
	d.configManager.UpdateConfig(nnk, *config)
	d.Unlock()
	d.syncNotify()
	return nil
}
