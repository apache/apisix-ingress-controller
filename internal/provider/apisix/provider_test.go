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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/adc/cache"
	adcclient "github.com/apache/apisix-ingress-controller/internal/adc/client"
	"github.com/apache/apisix-ingress-controller/internal/provider/common"
	"github.com/apache/apisix-ingress-controller/internal/types"
	"github.com/apache/apisix-ingress-controller/internal/utils"
)

// withMockADCServer starts an ADC server stub and points ADC_SERVER_URL at it for the
// duration of the test. The handler itself is how a test inspects what it received.
func withMockADCServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Setenv("ADC_SERVER_URL", server.URL)
	t.Cleanup(server.Close)
}

// newTestProvider builds a minimally-wired apisixProvider against the given mock ADC
// server -- every field Client/Delete/sync touch, none of the manager/controller ones.
func newTestProvider(t *testing.T) *apisixProvider {
	t.Helper()
	cli, err := adcclient.New(logr.Discard(), ProviderTypeAPISIX, time.Second)
	require.NoError(t, err)
	return &apisixProvider{
		client:           cli,
		store:            cache.NewStore(logr.Discard()),
		configManager:    common.NewConfigManager[types.NamespacedNameKind, adctypes.Config](),
		syncLocks:        newKeyedMutex(),
		standaloneSyncer: adcclient.NewStandaloneSyncer(cli, logr.Discard()),
		syncCh:           make(chan struct{}, 1),
		skipped:          newSkipTable(),
		gatewayEvents:    make(chan event.GenericEvent, gatewayEventsBuffer),
		log:              logr.Discard(),
	}
}

// TestDeleteNotifiesSyncOnlyWhenConfigWasRemoved covers the cost side of route
// ownership: a sync pushes the whole store to every data plane, and reconciles
// for routes this controller never configured are frequent (any EndpointSlice
// event on a shared backend enqueues them), so those must not notify.
func TestDeleteNotifiesSyncOnlyWhenConfigWasRemoved(t *testing.T) {
	d := newTestProvider(t)

	route := &gatewayv1.HTTPRoute{
		TypeMeta: metav1.TypeMeta{
			Kind:       "HTTPRoute",
			APIVersion: gatewayv1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "route"},
	}

	require.NoError(t, d.Delete(context.Background(), route))
	require.Empty(t, d.syncCh, "a route this controller never configured must not trigger a sync")

	d.configManager.Update(utils.NamespacedNameKind(route), map[types.NamespacedNameKind]adctypes.Config{
		{Namespace: "default", Name: "proxy", Kind: "GatewayProxy"}: {Name: "proxy"},
	})

	require.NoError(t, d.Delete(context.Background(), route))
	require.Len(t, d.syncCh, 1, "removing configuration this controller pushed must trigger a sync")
}

// TestDeleteTriggersImmediateSyncForEvictedConfigs covers the immediate-push branch of
// Delete: a Gateway going away must reach the data plane right away -- an empty resource
// set for the config it referenced -- not wait for the next scheduled sync round.
func TestDeleteTriggersImmediateSyncForEvictedConfigs(t *testing.T) {
	var mu sync.Mutex
	var received []adcclient.ADCServerRequest

	withMockADCServer(t, func(w http.ResponseWriter, r *http.Request) {
		var req adcclient.ADCServerRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		mu.Lock()
		received = append(received, req)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(adctypes.SyncResult{Status: adctypes.StatusSuccess})
	})

	d := newTestProvider(t)

	gw := &gatewayv1.Gateway{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Gateway",
			APIVersion: gatewayv1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "gw"},
	}
	d.configManager.Update(utils.NamespacedNameKind(gw), map[types.NamespacedNameKind]adctypes.Config{
		{Namespace: "default", Name: "proxy", Kind: "GatewayProxy"}: {
			Name:        "proxy",
			BackendType: "apisix",
			ServerAddrs: []string{"http://apisix:9080"},
		},
	})

	require.NoError(t, d.Delete(context.Background(), gw))

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, received, 1, "deleting a Gateway must push immediately, not wait for the next scheduled round")
	assert.Equal(t, "proxy", received[0].Task.Opts.CacheKey)
	assert.Empty(t, received[0].Task.Config.Services, "the evicted config's push must carry an empty resource set")
}

// TestSyncStillPushesHealthyConfigsWhenAnotherFails covers sync's error aggregation: one
// GatewayProxy's push failing must not stop the others in the same round from being
// attempted, and the failure must still be reported.
func TestSyncStillPushesHealthyConfigsWhenAnotherFails(t *testing.T) {
	var mu sync.Mutex
	seen := map[string]bool{}

	withMockADCServer(t, func(w http.ResponseWriter, r *http.Request) {
		var req adcclient.ADCServerRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		mu.Lock()
		seen[req.Task.Opts.CacheKey] = true
		mu.Unlock()
		if req.Task.Opts.CacheKey == "bad" {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message": "boom"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(adctypes.SyncResult{Status: adctypes.StatusSuccess})
	})

	d := newTestProvider(t)
	for _, name := range []string{"bad", "good"} {
		key := types.NamespacedNameKind{Namespace: "default", Name: name, Kind: "GatewayProxy"}
		d.configManager.UpdateConfig(key, adctypes.Config{
			Name:        name,
			BackendType: "apisix",
			ServerAddrs: []string{"http://apisix:9080"},
		})
	}

	err := d.sync(context.Background())
	require.Error(t, err, "one config failing must still be reported")
	assert.Contains(t, err.Error(), "bad")

	mu.Lock()
	defer mu.Unlock()
	assert.True(t, seen["bad"], "the failing config must still have been attempted")
	assert.True(t, seen["good"], "a config failing must not stop the others from being pushed")
}

// TestApplyResourceStateAttributesGatewayProxyPluginsToTheGatewayProxy covers the one
// reconcile that writes content of two owners: a Gateway's own listener certificates, and
// the plugins its GatewayProxy declares.
func TestApplyResourceStateAttributesGatewayProxyPluginsToTheGatewayProxy(t *testing.T) {
	d := newTestProvider(t)
	gw1 := types.NamespacedNameKind{Kind: types.KindGateway, Namespace: "ns", Name: "gw1"}
	gw2 := types.NamespacedNameKind{Kind: types.KindGateway, Namespace: "ns", Name: "gw2"}
	oldProxy := types.NamespacedNameKind{Kind: types.KindGatewayProxy, Namespace: "ns", Name: "old"}
	newProxy := types.NamespacedNameKind{Kind: types.KindGatewayProxy, Namespace: "ns", Name: "new"}
	configFor := func(proxy types.NamespacedNameKind) map[types.NamespacedNameKind]adctypes.Config {
		return map[types.NamespacedNameKind]adctypes.Config{proxy: {Name: proxy.String()}}
	}
	resourcesOf := func(sslID string) *adctypes.Resources {
		return &adctypes.Resources{
			SSLs:           []*adctypes.SSL{{Metadata: adctypes.Metadata{ID: sslID, Labels: labelsOf(gw1)}}},
			GlobalRules:    adctypes.GlobalRule{"cors": map[string]any{}},
			PluginMetadata: adctypes.PluginMetadata{"http-logger": map[string]any{}},
		}
	}

	require.NoError(t, d.applyResourceState(gw1, configFor(oldProxy), []string{adctypes.TypeSSL}, resourcesOf("ssl1"), labelsOf(gw1), pluginsFromGatewayProxy))
	gw2Resources := resourcesOf("ssl2")
	gw2Resources.SSLs[0].Labels = labelsOf(gw2)
	require.NoError(t, d.applyResourceState(gw2, configFor(oldProxy), []string{adctypes.TypeSSL}, gw2Resources, labelsOf(gw2), pluginsFromGatewayProxy))

	ssl, ok := d.store.Lookup(oldProxy.String(), adctypes.TypeSSL, "ssl1")
	require.True(t, ok)
	assert.Equal(t, gw1, ssl.Owner)
	for _, resourceType := range []string{adctypes.TypeGlobalRule, adctypes.TypePluginMetadata} {
		id := map[string]string{adctypes.TypeGlobalRule: "cors", adctypes.TypePluginMetadata: "http-logger"}[resourceType]
		entity, ok := d.store.Lookup(oldProxy.String(), resourceType, id)
		require.True(t, ok, resourceType)
		assert.Equal(t, oldProxy, entity.Owner, "%s comes from the GatewayProxy, not the Gateway that was reconciled", resourceType)
	}
	assert.Len(t, d.store.OwnedEntities(oldProxy.String(), oldProxy), 2, "Gateways sharing a GatewayProxy write its plugins once")

	require.NoError(t, d.applyResourceState(gw1, configFor(newProxy), []string{adctypes.TypeSSL}, resourcesOf("ssl1"), labelsOf(gw1), pluginsFromGatewayProxy))
	_, ok = d.store.Lookup(oldProxy.String(), adctypes.TypeSSL, "ssl1")
	assert.False(t, ok, "the Gateway's own certificate leaves the config it no longer references")
	_, ok = d.store.Lookup(oldProxy.String(), adctypes.TypeGlobalRule, "cors")
	assert.True(t, ok, "the old GatewayProxy's own plugins stay in its own config")
}

func TestApplyResourceStateRetriesSkippedResourcesOnlyWhenTheirContentChanged(t *testing.T) {
	d := newTestProvider(t)
	route := types.NamespacedNameKind{Kind: types.KindApisixRoute, Namespace: "ns", Name: "route"}
	proxy := types.NamespacedNameKind{Kind: types.KindGatewayProxy, Namespace: "ns", Name: "gp"}
	configs := map[types.NamespacedNameKind]adctypes.Config{proxy: {Name: proxy.String()}}
	apply := func(plugins adctypes.Plugins) {
		require.NoError(t, d.applyResourceState(route, configs, []string{adctypes.TypeService}, &adctypes.Resources{
			Services: []*adctypes.Service{{Metadata: adctypes.Metadata{ID: "svc", Labels: labelsOf(route)}, Plugins: plugins}},
		}, labelsOf(route), pluginsNone))
	}
	key := wireKey{resourceType: adctypes.TypeService, id: "svc"}

	apply(adctypes.Plugins{"bad": map[string]any{}})
	d.skipped.MarkFailing(proxy.String(), map[wireKey]exclusion{key: {owner: route}})

	apply(adctypes.Plugins{"bad": map[string]any{}})
	assert.Contains(t, d.skipped.Excluded(proxy.String()), key, "a reconcile that rewrites the same content must not retry it")

	apply(adctypes.Plugins{"fixed": map[string]any{}})
	assert.NotContains(t, d.skipped.Excluded(proxy.String()), key, "changed content gets another try")
}

func TestRemoveResourceStateRemovesAnApisixGlobalRulesPlugins(t *testing.T) {
	d := newTestProvider(t)
	globalRule := types.NamespacedNameKind{Kind: types.KindApisixGlobalRule, Namespace: "ns", Name: "global"}
	proxy := types.NamespacedNameKind{Kind: types.KindGatewayProxy, Namespace: "ns", Name: "gp"}
	configs := map[types.NamespacedNameKind]adctypes.Config{proxy: {Name: proxy.String()}}
	require.NoError(t, d.store.SetGlobalRules(proxy.String(), proxy, adctypes.GlobalRule{"cors": map[string]any{}}))
	require.NoError(t, d.applyResourceState(globalRule, configs, nil, &adctypes.Resources{
		GlobalRules: adctypes.GlobalRule{"prometheus": map[string]any{}},
	}, labelsOf(globalRule), pluginsFromResource))

	resources, _, err := d.store.GetResources(proxy.String())
	require.NoError(t, err)
	assert.Len(t, resources.GlobalRules, 2)

	_, err = d.removeResourceState(globalRule, nil, labelsOf(globalRule), pluginsFromResource, false)
	require.NoError(t, err)
	resources, _, err = d.store.GetResources(proxy.String())
	require.NoError(t, err)
	assert.Equal(t, adctypes.GlobalRule{"cors": map[string]any{}}, resources.GlobalRules, "only the deleted resource's own plugins go away")
}

func TestSyncLeavesSkippedResourcesOutOfThePush(t *testing.T) {
	var mu sync.Mutex
	var received []adcclient.ADCServerRequest
	withMockADCServer(t, func(w http.ResponseWriter, r *http.Request) {
		var req adcclient.ADCServerRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		mu.Lock()
		received = append(received, req)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(adctypes.SyncResult{Status: adctypes.StatusSuccess})
	})

	d := newTestProvider(t)
	d.updater = &fakeUpdater{}
	route := types.NamespacedNameKind{Kind: types.KindApisixRoute, Namespace: "ns", Name: "route"}
	proxy := types.NamespacedNameKind{Kind: types.KindGatewayProxy, Namespace: "ns", Name: "gp"}
	config := adctypes.Config{Name: proxy.String(), BackendType: "apisix", ServerAddrs: []string{"http://apisix:9080"}}
	require.NoError(t, d.applyResourceState(route, map[types.NamespacedNameKind]adctypes.Config{proxy: config}, []string{adctypes.TypeService}, &adctypes.Resources{
		Services: []*adctypes.Service{
			{Metadata: adctypes.Metadata{ID: "good", Labels: labelsOf(route)}},
			{Metadata: adctypes.Metadata{ID: "bad", Labels: labelsOf(route)}},
		},
	}, labelsOf(route), pluginsNone))
	d.skipped.MarkFailing(proxy.String(), map[wireKey]exclusion{{resourceType: adctypes.TypeService, id: "bad"}: {owner: route}})

	require.NoError(t, d.sync(context.Background()))

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, received, 1)
	require.Len(t, received[0].Task.Config.Services, 1)
	assert.Equal(t, "good", received[0].Task.Config.Services[0].ID)
}

// TestSyncRetriesImmediatelyOnceTheRejectedResourceIsExcluded covers what happens after a
// push is rejected: the next push leaves the rejected resource out and carries every
// other resource's pending changes, so it must not wait for the retry backoff.
func TestSyncRetriesImmediatelyOnceTheRejectedResourceIsExcluded(t *testing.T) {
	route := types.NamespacedNameKind{Kind: types.KindApisixRoute, Namespace: "ns", Name: "route"}
	proxy := types.NamespacedNameKind{Kind: types.KindGatewayProxy, Namespace: "ns", Name: "gp"}
	rejectService := adcResp{
		status: http.StatusUnprocessableEntity,
		body: adctypes.SyncResult{
			Status: "all_failed",
			Failed: []adctypes.SyncStatus{{
				Reason: "unknown plugin [nope]",
				Event:  adctypes.StatusEvent{ResourceType: adctypes.TypeService, ResourceID: "bad"},
			}},
		},
	}
	requests := scriptedADC(t, rejectService, respOK())

	d := newTestProvider(t)
	d.updater = &fakeUpdater{}
	config := adctypes.Config{Name: proxy.String(), BackendType: "apisix", ServerAddrs: []string{"http://apisix:9180"}}
	require.NoError(t, d.applyResourceState(route, map[types.NamespacedNameKind]adctypes.Config{proxy: config}, []string{adctypes.TypeService}, &adctypes.Resources{
		Services: []*adctypes.Service{
			{Metadata: adctypes.Metadata{ID: "good", Labels: labelsOf(route)}},
			{Metadata: adctypes.Metadata{ID: "bad", Labels: labelsOf(route)}},
		},
	}, labelsOf(route), pluginsNone))

	require.Error(t, d.sync(context.Background()))
	require.Len(t, d.syncCh, 1, "the round that excluded the rejected resource must push again right away")
	<-d.syncCh

	require.NoError(t, d.sync(context.Background()))
	sent := requests()
	require.Len(t, sent, 2)
	require.Len(t, sent[1].Task.Config.Services, 1)
	assert.Equal(t, "good", sent[1].Task.Config.Services[0].ID)
	assert.Empty(t, d.syncCh, "a round that excluded nothing new must not ask for another push")
}
