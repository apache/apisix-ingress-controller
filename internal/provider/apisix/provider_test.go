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
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	adcclient "github.com/apache/apisix-ingress-controller/internal/adc/client"
	"github.com/apache/apisix-ingress-controller/internal/adc/translator"
	controllerconfig "github.com/apache/apisix-ingress-controller/internal/controller/config"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/internal/types"
	"github.com/apache/apisix-ingress-controller/internal/utils"
)

func TestDeleteGatewayProxyUsesCanonicalKeyWithoutTypeMeta(t *testing.T) {
	requests := make(chan adcclient.ADCServerRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var request adcclient.ADCServerRequest
		require.NoError(t, json.Unmarshal(body, &request))
		requests <- request
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write([]byte(`{}`))
		require.NoError(t, err)
	}))
	defer server.Close()
	t.Setenv("ADC_SERVER_URL", server.URL)

	cli, err := adcclient.New(logr.Discard(), ProviderTypeAPISIX, time.Second)
	require.NoError(t, err)
	key := utils.GatewayProxyKey("default", "gp-a")
	config := adctypes.Config{
		Name:        key.String(),
		BackendType: ProviderTypeAPISIX,
		ServerAddrs: []string{server.URL},
		Token:       "old-token",
	}
	cli.ConfigManager.Update(key, map[types.NamespacedNameKind]adctypes.Config{key: config})
	require.NoError(t, cli.Store.Insert(config.Name, []string{adctypes.TypeService}, &adctypes.Resources{
		Services: []*adctypes.Service{{Metadata: adctypes.Metadata{ID: "route-a"}}},
	}, nil))

	d := &apisixProvider{client: cli, log: logr.Discard()}
	deleted := &v1alpha1.GatewayProxy{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "gp-a"},
	}

	require.NoError(t, d.Delete(context.Background(), deleted))
	select {
	case request := <-requests:
		assert.Equal(t, key.String(), request.Task.Opts.CacheKey)
		assert.Empty(t, request.Task.Config.Services)
	case <-time.After(time.Second):
		t.Fatal("expected an empty configuration sync for the deleted GatewayProxy")
	}
	_, ok := cli.ConfigManager.GetConfig(key)
	assert.False(t, ok)
	resources, err := cli.Store.GetResources(config.Name)
	require.NoError(t, err)
	assert.Empty(t, resources.Services)
}

// TestDeleteNotifiesSyncOnlyWhenConfigWasRemoved covers the cost side of route
// ownership: a sync pushes the whole store to every data plane, and reconciles
// for routes this controller never configured are frequent (any EndpointSlice
// event on a shared backend enqueues them), so those must not notify.
func TestDeleteNotifiesSyncOnlyWhenConfigWasRemoved(t *testing.T) {
	cli, err := adcclient.New(logr.Discard(), ProviderTypeAPISIX, time.Second)
	require.NoError(t, err)

	d := &apisixProvider{
		client: cli,
		syncCh: make(chan struct{}, 1),
		log:    logr.Discard(),
	}

	route := &gatewayv1.HTTPRoute{
		TypeMeta: metav1.TypeMeta{
			Kind:       "HTTPRoute",
			APIVersion: gatewayv1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "route"},
	}

	require.NoError(t, d.Delete(context.Background(), route))
	require.Empty(t, d.syncCh, "a route this controller never configured must not trigger a sync")

	cli.ConfigManager.Update(utils.NamespacedNameKind(route), map[types.NamespacedNameKind]adctypes.Config{
		{Namespace: "default", Name: "proxy", Kind: "GatewayProxy"}: {Name: "proxy"},
	})

	require.NoError(t, d.Delete(context.Background(), route))
	require.Len(t, d.syncCh, 1, "removing configuration this controller pushed must trigger a sync")
}

func TestBuildConfigSkipsInactiveGatewayProxy(t *testing.T) {
	d := &apisixProvider{
		translator: translator.NewTranslator(logr.Discard(), controllerconfig.ListenerPortMatchModeOff),
	}
	tctx := provider.NewDefaultTranslateContext(context.Background())
	resourceKey := types.NamespacedNameKind{Namespace: "app", Name: "route-a", Kind: "HTTPRoute"}
	tctx.GatewayProxies[resourceKey] = v1alpha1.GatewayProxy{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "gp-a"},
	}

	configs, err := d.buildConfig(tctx, resourceKey)

	require.NoError(t, err)
	require.Empty(t, configs)
}
