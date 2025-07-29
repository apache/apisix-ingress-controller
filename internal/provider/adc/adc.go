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

package adc

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/api7/gopkg/pkg/log"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	networkingv1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	"github.com/apache/apisix-ingress-controller/internal/manager/readiness"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/internal/provider/adc/translator"
	"github.com/apache/apisix-ingress-controller/internal/types"
	"github.com/apache/apisix-ingress-controller/internal/utils"
	pkgmetrics "github.com/apache/apisix-ingress-controller/pkg/metrics"
)

type adcConfig struct {
	Name        string
	ServerAddrs []string
	Token       string
	TlsVerify   bool
}

// MarshalJSON implements custom JSON marshaling for adcConfig
// It excludes the Token field for security reasons
func (c adcConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Name        string   `json:"name"`
		ServerAddrs []string `json:"serverAddrs"`
		TlsVerify   bool     `json:"tlsVerify"`
	}{
		Name:        c.Name,
		ServerAddrs: c.ServerAddrs,
		TlsVerify:   c.TlsVerify,
	})
}

type BackendMode string

const (
	BackendModeAPISIXStandalone string = "apisix-standalone"
	BackendModeAPISIX           string = "apisix"
)

type adcClient struct {
	sync.Mutex

	syncLock sync.Mutex

	translator *translator.Translator
	// gateway/ingressclass -> adcConfig
	configs map[types.NamespacedNameKind]adcConfig
	// httproute/consumer/ingress/gateway -> gateway/ingressclass
	parentRefs map[types.NamespacedNameKind][]types.NamespacedNameKind

	store *Store

	executor ADCExecutor

	Options

	updater         status.Updater
	statusUpdateMap map[types.NamespacedNameKind][]string

	readier readiness.ReadinessManager

	syncCh chan struct{}
}

type Task struct {
	Name          string
	Resources     adctypes.Resources
	Labels        map[string]string
	ResourceTypes []string
	configs       []adcConfig
}

func New(updater status.Updater, readier readiness.ReadinessManager, opts ...Option) (provider.Provider, error) {
	o := Options{}
	o.ApplyOptions(opts)

	return &adcClient{
		Options:    o,
		translator: &translator.Translator{},
		configs:    make(map[types.NamespacedNameKind]adcConfig),
		parentRefs: make(map[types.NamespacedNameKind][]types.NamespacedNameKind),
		store:      NewStore(),
		executor:   &DefaultADCExecutor{},
		updater:    updater,
		readier:    readier,
		syncCh:     make(chan struct{}, 1),
	}, nil
}

func (d *adcClient) Update(ctx context.Context, tctx *provider.TranslateContext, obj client.Object) error {
	log.Debugw("updating object", zap.Any("object", obj))
	var (
		result        *translator.TranslateResult
		resourceTypes []string
		err           error
	)

	rk := utils.NamespacedNameKind(obj)

	switch t := obj.(type) {
	case *gatewayv1.HTTPRoute:
		result, err = d.translator.TranslateHTTPRoute(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, "service")
	case *gatewayv1.Gateway:
		result, err = d.translator.TranslateGateway(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, "global_rule", "ssl", "plugin_metadata")
	case *networkingv1.Ingress:
		result, err = d.translator.TranslateIngress(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, "service", "ssl")
	case *v1alpha1.Consumer:
		result, err = d.translator.TranslateConsumerV1alpha1(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, "consumer")
	case *networkingv1.IngressClass:
		result, err = d.translator.TranslateIngressClass(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, "global_rule", "plugin_metadata")
	case *apiv2.ApisixRoute:
		result, err = d.translator.TranslateApisixRoute(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, "service")
	case *apiv2.ApisixGlobalRule:
		result, err = d.translator.TranslateApisixGlobalRule(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, "global_rule")
	case *apiv2.ApisixTls:
		result, err = d.translator.TranslateApisixTls(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, "ssl")
	case *apiv2.ApisixConsumer:
		result, err = d.translator.TranslateApisixConsumer(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, "consumer")
	case *v1alpha1.GatewayProxy:
		return d.updateConfigForGatewayProxy(tctx, t)
	}
	if err != nil {
		return err
	}
	if result == nil {
		return nil
	}

	oldParentRefs := d.getParentRefs(rk)
	if err := d.updateConfigs(rk, tctx); err != nil {
		return err
	}
	newParentRefs := d.getParentRefs(rk)
	deleteConfigs := d.findConfigsToDelete(oldParentRefs, newParentRefs)
	configs := d.getConfigs(rk)

	// sync delete
	if len(deleteConfigs) > 0 {
		err = d.sync(ctx, Task{
			Name:          obj.GetName(),
			Labels:        label.GenLabel(obj),
			ResourceTypes: resourceTypes,
			configs:       deleteConfigs,
		})
		if err != nil {
			return err
		}
		for _, config := range deleteConfigs {
			if err := d.store.Delete(config.Name, resourceTypes, label.GenLabel(obj)); err != nil {
				log.Errorw("failed to delete resources from store",
					zap.String("name", config.Name),
					zap.Error(err),
				)
				return err
			}
		}
	}

	resources := adctypes.Resources{
		GlobalRules:    result.GlobalRules,
		PluginMetadata: result.PluginMetadata,
		Services:       result.Services,
		SSLs:           result.SSL,
		Consumers:      result.Consumers,
	}
	log.Debugw("update resources", zap.Any("resources", resources))

	for _, config := range configs {
		if err := d.store.Insert(config.Name, resourceTypes, resources, label.GenLabel(obj)); err != nil {
			log.Errorw("failed to insert resources into store",
				zap.String("name", config.Name),
				zap.Error(err),
			)
			return err
		}
	}

	// This mode is full synchronization,
	// which only needs to be saved in cache
	// and triggered by a timer for synchronization
	if d.BackendMode == BackendModeAPISIXStandalone || d.BackendMode == BackendModeAPISIX {
		d.syncNotify()
		return nil
	}

	return d.sync(ctx, Task{
		Name:          obj.GetName(),
		Labels:        label.GenLabel(obj),
		Resources:     resources,
		ResourceTypes: resourceTypes,
		configs:       configs,
	})
}

func (d *adcClient) Delete(ctx context.Context, obj client.Object) error {
	log.Debugw("deleting object", zap.Any("object", obj))

	var resourceTypes []string
	var labels map[string]string
	switch obj.(type) {
	case *gatewayv1.HTTPRoute, *apiv2.ApisixRoute:
		resourceTypes = append(resourceTypes, "service")
		labels = label.GenLabel(obj)
	case *gatewayv1.Gateway:
		// delete all resources
	case *networkingv1.Ingress:
		resourceTypes = append(resourceTypes, "service", "ssl")
		labels = label.GenLabel(obj)
	case *v1alpha1.Consumer:
		resourceTypes = append(resourceTypes, "consumer")
		labels = label.GenLabel(obj)
	case *networkingv1.IngressClass:
		// delete all resources
	case *apiv2.ApisixGlobalRule:
		resourceTypes = append(resourceTypes, "global_rule")
		labels = label.GenLabel(obj)
	case *apiv2.ApisixTls:
		resourceTypes = append(resourceTypes, "ssl")
		labels = label.GenLabel(obj)
	case *apiv2.ApisixConsumer:
		resourceTypes = append(resourceTypes, "consumer")
		labels = label.GenLabel(obj)
	}

	rk := utils.NamespacedNameKind(obj)

	configs := d.getConfigs(rk)
	defer d.deleteConfigs(rk)

	for _, config := range configs {
		if err := d.store.Delete(config.Name, resourceTypes, labels); err != nil {
			log.Errorw("failed to delete resources from store",
				zap.String("name", config.Name),
				zap.Error(err),
			)
			return err
		}
	}

	log.Debugw("successfully deleted resources from store", zap.Any("object", obj))

	switch d.BackendMode {
	case BackendModeAPISIXStandalone, BackendModeAPISIX:
		// Full synchronization is performed on a gateway by gateway basis
		// and it is not possible to perform scheduled synchronization
		// on deleted gateway level resources
		if len(resourceTypes) == 0 {
			return d.sync(ctx, Task{
				Name:    obj.GetName(),
				configs: configs,
			})
		} else {
			d.syncNotify()
		}
		return nil
	default:
		log.Errorw("unknown backend mode", zap.String("mode", d.BackendMode))
		return errors.New("unknown backend mode: " + d.BackendMode)
	}
}

func (d *adcClient) Start(ctx context.Context) error {
	d.readier.WaitReady(ctx, 5*time.Minute)

	initalSyncDelay := d.InitSyncDelay
	if initalSyncDelay > 0 {
		time.AfterFunc(initalSyncDelay, func() {
			if err := d.Sync(ctx); err != nil {
				log.Error(err)
				return
			}
		})
	}

	if d.SyncPeriod < 1 {
		return nil
	}
	ticker := time.NewTicker(d.SyncPeriod)
	defer ticker.Stop()
	for {
		synced := false
		select {
		case <-d.syncCh:
			synced = true
		case <-ticker.C:
			synced = true
		case <-ctx.Done():
			return nil
		}
		if synced {
			if err := d.Sync(ctx); err != nil {
				log.Error(err)
			}
		}
	}
}

func (d *adcClient) Sync(ctx context.Context) error {
	d.syncLock.Lock()
	defer d.syncLock.Unlock()

	log.Debug("syncing all resources")

	if len(d.configs) == 0 {
		return nil
	}

	cfg := map[string]adcConfig{}
	for _, config := range d.configs {
		cfg[config.Name] = config
	}

	log.Debugw("syncing resources with multiple configs", zap.Any("configs", cfg))

	failedMap := map[string]types.ADCExecutionErrors{}
	var failedConfigs []string
	for name, config := range cfg {
		resources, err := d.store.GetResources(name)
		if err != nil {
			log.Errorw("failed to get resources from store", zap.String("name", name), zap.Error(err))
			failedConfigs = append(failedConfigs, name)
			continue
		}
		if resources == nil {
			continue
		}

		if err := d.sync(ctx, Task{
			Name:      name + "-sync",
			configs:   []adcConfig{config},
			Resources: *resources,
		}); err != nil {
			log.Errorw("failed to sync resources", zap.String("name", name), zap.Error(err))
			failedConfigs = append(failedConfigs, name)
			var execErrs types.ADCExecutionErrors
			if errors.As(err, &execErrs) {
				failedMap[name] = execErrs
			}
		}
	}
	d.handleADCExecutionErrors(failedMap)
	if len(failedConfigs) > 0 {
		return fmt.Errorf("failed to sync %d configs: %s",
			len(failedConfigs),
			strings.Join(failedConfigs, ", "))
	}
	return nil
}

func (d *adcClient) sync(ctx context.Context, task Task) error {
	log.Debugw("syncing resources", zap.Any("task", task))

	if len(task.configs) == 0 {
		log.Warnw("no adc configs provided", zap.Any("task", task))
		return nil
	}

	// Record file I/O duration
	fileIOStart := time.Now()
	syncFilePath, cleanup, err := prepareSyncFile(task.Resources)
	if err != nil {
		pkgmetrics.RecordFileIODuration("prepare_sync_file", "failure", time.Since(fileIOStart).Seconds())
		return err
	}
	pkgmetrics.RecordFileIODuration("prepare_sync_file", "success", time.Since(fileIOStart).Seconds())
	defer cleanup()

	args := BuildADCExecuteArgs(syncFilePath, task.Labels, task.ResourceTypes)

	var errs types.ADCExecutionErrors
	for _, config := range task.configs {
		// Record sync duration for each config
		startTime := time.Now()
		resourceType := strings.Join(task.ResourceTypes, ",")
		if resourceType == "" {
			resourceType = "all"
		}

		err := d.executor.Execute(ctx, d.BackendMode, config, args)
		duration := time.Since(startTime).Seconds()

		status := "success"
		if err != nil {
			status = "failure"
			log.Errorw("failed to execute adc command", zap.Error(err), zap.Any("config", config))

			var execErr types.ADCExecutionError
			if errors.As(err, &execErr) {
				errs.Errors = append(errs.Errors, execErr)
				pkgmetrics.RecordExecutionError(config.Name, execErr.Name)
			} else {
				pkgmetrics.RecordExecutionError(config.Name, "unknown")
			}
		}

		// Record metrics
		pkgmetrics.RecordSyncDuration(config.Name, resourceType, status, duration)
	}

	if len(errs.Errors) > 0 {
		return errs
	}
	return nil
}

func (d *adcClient) syncNotify() {
	select {
	case d.syncCh <- struct{}{}:
	default:
	}
}

func prepareSyncFile(resources any) (string, func(), error) {
	data, err := json.Marshal(resources)
	if err != nil {
		return "", nil, err
	}

	tmpFile, err := os.CreateTemp("", "adc-task-*.json")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
	}
	if _, err := tmpFile.Write(data); err != nil {
		cleanup()
		return "", nil, err
	}

	log.Debugw("generated adc file", zap.String("filename", tmpFile.Name()), zap.String("json", string(data)))

	return tmpFile.Name(), cleanup, nil
}

func (d *adcClient) handleADCExecutionErrors(statusesMap map[string]types.ADCExecutionErrors) {
	statusUpdateMap := d.resolveADCExecutionErrors(statusesMap)
	d.handleStatusUpdate(statusUpdateMap)
}

func (d *adcClient) NeedLeaderElection() bool {
	return true
}
