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
		rebuiltBaselines: make(map[string]struct{}),
		syncCh:           make(chan struct{}, 1),
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
