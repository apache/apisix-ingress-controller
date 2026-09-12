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
	"strings"
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
	pkgmetrics "github.com/apache/apisix-ingress-controller/pkg/metrics"
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

	// syncLocks serializes, per cacheKey, reading that GatewayProxy's current resource
	// snapshot together with pushing it
	syncLocks *keyedMutex

	// standaloneSyncer owns apisix-standalone's ADC diff-baseline recovery: the
	// BypassCache decision, the one retry for a stale conf_version, and the per-term
	// record of which cacheKeys it has rebuilt. Unused for every other backend type.
	standaloneSyncer *adcclient.StandaloneSyncer

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
		client:           cli,
		store:            store,
		configManager:    configManager,
		debugProvider:    common.NewADCDebugProvider(store, configManager),
		syncLocks:        newKeyedMutex(),
		standaloneSyncer: adcclient.NewStandaloneSyncer(cli, logger),
		Options:          o,
		translator:       translator.NewTranslator(log, o.ListenerPortMatchMode),
		updater:          updater,
		readier:          readier,
		syncCh:           make(chan struct{}, 1),
		log:              logger,
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
	if gp, ok := obj.(*v1alpha1.GatewayProxy); ok {
		return d.deleteGatewayProxyConfig(ctx, utils.GatewayProxyKey(gp.Namespace, gp.Name))
	}
	nnk := utils.NamespacedNameKind(obj)

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

	removed, err := d.removeResourceState(nnk, resourceTypes, labels)
	if err != nil {
		return err
	}
	// Syncing pushes the whole store to every data plane. Objects this controller never
	// configured delete nothing, and reconciles for them are frequent, so notify only
	// when the store actually changed.
	if len(removed) > 0 {
		d.syncNotify()
	}
	return nil
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

	evicted := d.configManager.Update(rk, configs)
	if err := d.evictFromStore(evicted, resourceTypes, labels); err != nil {
		return err
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

	evicted := d.configManager.Get(rk)
	d.configManager.Delete(rk)
	if err := d.evictFromStore(evicted, resourceTypes, labels); err != nil {
		return nil, err
	}
	return evicted, nil
}

// evictFromStore deletes a resource's contribution from each of the given configs' cached
// snapshots. Callers must already hold d.Lock.
func (d *apisixProvider) evictFromStore(
	configs map[types.NamespacedNameKind]adctypes.Config,
	resourceTypes []string,
	labels map[string]string,
) error {
	for _, cfg := range configs {
		if err := d.store.Delete(cfg.Name, resourceTypes, labels); err != nil {
			return fmt.Errorf("store delete failed for config %s: %w", cfg.Name, err)
		}
	}
	return nil
}

// syncConfigNow reads name's current data (via build, called only once this cacheKey's
// lock is actually held) and pushes it -- one atomic read-then-push step per cacheKey, so
// whichever caller is granted the lock decides what to push only once it holds it: nothing
// it sends can already be stale relative to whatever the other caller committed to the
// store before losing the race for the same key. See keyedMutex.
func (d *apisixProvider) syncConfigNow(
	ctx context.Context,
	name string,
	build func() (adcclient.SyncInput, error),
) (types.ADCExecutionErrors, error) {
	unlock := d.syncLocks.Lock(name)
	defer unlock()

	input, err := build()
	if err != nil {
		return types.ADCExecutionErrors{}, err
	}
	execErrs := d.pushConfig(ctx, input)
	if len(execErrs.Errors) > 0 {
		return execErrs, execErrs
	}
	return execErrs, nil
}

// pushConfig sends input to its data plane and shapes whatever failed into the form
// status reporting consumes. apisix-standalone goes through standaloneSyncer, which may
// rebuild ADC's diff baseline and retry once; every other backend type is a single
// one-shot push through the adc client, which never retries.
//
// Metrics are recorded here, once per call, around whichever of those two logical syncs
// ran: the adc client itself records nothing, since a caller that retries may drive it
// more than once for what is, from the outside, one sync attempt, and only this layer
// knows when that attempt is actually over.
func (d *apisixProvider) pushConfig(ctx context.Context, input adcclient.SyncInput) types.ADCExecutionErrors {
	backend := input.Config.BackendType
	if backend == "" {
		backend = d.DefaultBackendMode
	}

	startTime := time.Now()
	resourceType := strings.Join(input.ResourceTypes, ",")
	if resourceType == "" {
		resourceType = "all"
	}

	var errs []error
	if backend == adcclient.BackendAPISIXStandalone {
		errs = d.standaloneSyncer.Sync(ctx, input)
	} else if err := d.client.Sync(ctx, input); err != nil {
		errs = []error{err}
	}

	status := adctypes.StatusSuccess
	if len(errs) > 0 {
		status = "failure"
		errorType := "unknown"
		var addrErr types.ADCExecutionServerAddrError
		if errors.As(errs[len(errs)-1], &addrErr) {
			errorType = "sync_failed"
		}
		pkgmetrics.RecordExecutionError(input.Name, errorType)
	}
	pkgmetrics.RecordSyncDuration(input.Name, resourceType, status, time.Since(startTime).Seconds())

	var execErrs types.ADCExecutionErrors
	for _, err := range errs {
		execErrs.Errors = append(execErrs.Errors, toADCExecutionError(input.Name, err))
	}
	return execErrs
}

// toADCExecutionError shapes one sync error into the per-config form status reporting
// consumes. A parsed per-server error travels through with its structured detail intact;
// anything else becomes a bare message.
func toADCExecutionError(name string, err error) types.ADCExecutionError {
	var addrErr types.ADCExecutionServerAddrError
	if errors.As(err, &addrErr) {
		return types.ADCExecutionError{Name: name, FailedErrors: []types.ADCExecutionServerAddrError{addrErr}}
	}
	return types.ADCExecutionError{Name: name, FailedErrors: []types.ADCExecutionServerAddrError{{Err: err.Error()}}}
}

// syncEvictedConfigsNow pushes an empty resource set for each of the given configs
// immediately, instead of waiting for the next scheduled sync round -- through the same
// per-cacheKey lock the periodic sync uses, so it can never race a periodic round for the
// same GatewayProxy. Used only when the deleted resource is a Gateway or IngressClass --
// resourceTypes is empty for those, so the preceding removeResourceState call already
// reset each config's whole cached snapshot via Store.Delete, and that reset should reach
// the data plane promptly. Failures are logged, not surfaced as a status update -- this
// mirrors the deferred path, which only reports through the next scheduled sync round.
func (d *apisixProvider) syncEvictedConfigsNow(
	ctx context.Context,
	configs map[types.NamespacedNameKind]adctypes.Config,
	resourceTypes []string,
	labels map[string]string,
) {
	for _, cfg := range configs {
		_, err := d.syncConfigNow(ctx, cfg.Name, func() (adcclient.SyncInput, error) {
			return adcclient.SyncInput{
				Name:          cfg.Name,
				Config:        cfg,
				Resources:     &adctypes.Resources{},
				ResourceTypes: resourceTypes,
				Labels:        labels,
			}, nil
		})
		if err != nil {
			d.log.Error(err, "failed to sync deleted config", "config", cfg)
		}
	}
}

// deleteGatewayProxyConfig removes a GatewayProxy's local state and pushes an empty
// resource snapshot to its data plane before forgetting the connection configuration.
// The per-key lock keeps this cleanup ordered with periodic and event-driven syncs.
func (d *apisixProvider) deleteGatewayProxyConfig(ctx context.Context, key types.NamespacedNameKind) error {
	unlock := d.syncLocks.Lock(key.String())
	defer unlock()

	d.Lock()
	config, ok := d.configManager.GetConfig(key)
	configName := key.String()
	if ok {
		configName = config.Name
	}
	if err := d.store.Delete(configName, nil, nil); err != nil {
		d.Unlock()
		return fmt.Errorf("store delete failed for config %s: %w", configName, err)
	}
	d.Unlock()

	if !ok {
		// Repeated deletion is idempotent and also removes stale associations left by
		// an earlier cleanup.
		d.Lock()
		d.configManager.DeleteConfig(key)
		d.Unlock()
		return nil
	}

	execErrs := d.pushConfig(ctx, adcclient.SyncInput{
		Name:      config.Name,
		Config:    config,
		Resources: &adctypes.Resources{},
	})
	if len(execErrs.Errors) > 0 {
		// Keep the old connection config so a later reconciliation can retry cleanup.
		return execErrs
	}

	d.Lock()
	d.configManager.DeleteConfig(key)
	d.Unlock()
	return nil
}

func (d *apisixProvider) buildConfig(tctx *provider.TranslateContext, nnk types.NamespacedNameKind) (map[types.NamespacedNameKind]adctypes.Config, error) {
	configs := make(map[types.NamespacedNameKind]adctypes.Config, len(tctx.ResourceParentRefs[nnk]))
	for _, gp := range tctx.GatewayProxies {
		config, err := d.translator.TranslateGatewayProxyToConfig(tctx, &gp, d.DefaultResolveEndpoints)
		if err != nil {
			return nil, err
		}
		if config == nil {
			continue
		}
		configs[utils.GatewayProxyKey(gp.Namespace, gp.Name)] = *config
	}
	return configs, nil
}

func (d *apisixProvider) Start(ctx context.Context) error {
	// Start only runs once this pod has won the election, and a leadership change is the
	// one thing that leaves the ADC sidecar holding a baseline from an earlier term: it
	// survives the manager container, the configuration it was derived from does not.
	// Rebuild every baseline from the data plane before syncing from it.
	d.standaloneSyncer.InvalidateBaselines()

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

// sync pushes every GatewayProxy AIC currently knows about, config by config -- each
// one's current resource snapshot is only read once syncConfigNow actually holds that
// cacheKey's lock, so a slow round can never push a snapshot that was already stale by the
// time its turn came up. All of this round's results are still collected into one
// statusesMap and handed to handleADCExecutionErrors together, exactly as a single batched
// sync would: that logic diffs against last round's full picture, not per-config.
func (d *apisixProvider) sync(ctx context.Context) error {
	configs := d.configManager.List()

	statusesMap := map[string]types.ADCExecutionErrors{}
	var errs []error
	for _, config := range configs {
		execErrs, err := d.syncConfigNow(ctx, config.Name, func() (adcclient.SyncInput, error) {
			resources, err := d.store.GetResources(config.Name)
			if err != nil {
				return adcclient.SyncInput{}, fmt.Errorf("failed to get resources from store: %w", err)
			}
			return adcclient.SyncInput{Name: config.Name, Config: config, Resources: resources}, nil
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("config %s: %w", config.Name, err))
		}
		if len(execErrs.Errors) > 0 {
			statusesMap[config.Name] = execErrs
		}
	}

	d.handleADCExecutionErrors(statusesMap)
	return errors.Join(errs...)
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

	nnk := utils.GatewayProxyKey(gp.Namespace, gp.Name)
	if config == nil {
		return d.deleteGatewayProxyConfig(tctx, nnk)
	}

	referrers := tctx.GatewayProxyReferrers[utils.NamespacedName(gp)]
	d.Lock()
	d.configManager.SetConfigRefs(nnk, referrers)
	d.configManager.UpdateConfig(nnk, *config)
	d.Unlock()
	d.syncNotify()
	return nil
}
