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

package client

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/types"
)

type gatewayProxyDeleteExecutor struct {
	errs      []error
	configs   []adctypes.Config
	resources []*adctypes.Resources
	readErr   error
}

func (f *gatewayProxyDeleteExecutor) Execute(_ context.Context, config adctypes.Config, args []string) error {
	f.configs = append(f.configs, config)
	executor := &HTTPADCExecutor{}
	_, _, filePath, err := executor.parseArgs(args)
	if err == nil {
		var resources *adctypes.Resources
		resources, err = executor.loadResourcesFromFile(filePath)
		if err == nil {
			f.resources = append(f.resources, resources)
		}
	}
	if err != nil {
		f.readErr = err
		return err
	}
	if len(f.errs) == 0 {
		return nil
	}
	err = f.errs[0]
	f.errs = f.errs[1:]
	return err
}

func (f *gatewayProxyDeleteExecutor) Validate(context.Context, adctypes.Config, []string) error {
	return nil
}

func newGatewayProxyDeleteClient(t *testing.T, executor ADCExecutor) *Client {
	t.Helper()
	client, err := New(logr.Discard(), "apisix", time.Second)
	require.NoError(t, err)
	client.executor = executor
	return client
}

func seedGatewayProxyStore(t *testing.T, client *Client, key, resourceKey types.NamespacedNameKind) adctypes.Config {
	t.Helper()
	config := adctypes.Config{
		Name:        key.String(),
		BackendType: "apisix",
		ServerAddrs: []string{"http://apisix:9180"},
		Token:       "old-token",
	}
	client.ConfigManager.Update(resourceKey, map[types.NamespacedNameKind]adctypes.Config{key: config})
	require.NoError(t, client.Store.Insert(config.Name, []string{adctypes.TypeService},
		&adctypes.Resources{
			Services: []*adctypes.Service{{Metadata: adctypes.Metadata{ID: resourceKey.String()}}},
		}, nil))
	return config
}

func TestDeleteGatewayProxyConfigSyncsEmptyConfigAndIsIsolated(t *testing.T) {
	executor := &gatewayProxyDeleteExecutor{}
	client := newGatewayProxyDeleteClient(t, executor)
	gpA := types.NamespacedNameKind{Namespace: "default", Name: "gp-a", Kind: "GatewayProxy"}
	gpB := types.NamespacedNameKind{Namespace: "default", Name: "gp-b", Kind: "GatewayProxy"}
	routeA := types.NamespacedNameKind{Namespace: "app", Name: "route-a", Kind: "HTTPRoute"}
	routeB := types.NamespacedNameKind{Namespace: "app", Name: "route-b", Kind: "HTTPRoute"}
	configA := seedGatewayProxyStore(t, client, gpA, routeA)
	seedGatewayProxyStore(t, client, gpB, routeB)

	require.NoError(t, client.DeleteGatewayProxyConfig(context.Background(), gpA))
	require.NoError(t, executor.readErr)
	require.Len(t, executor.configs, 1)
	assert.Equal(t, configA, executor.configs[0])
	require.Len(t, executor.resources, 1)
	assert.Equal(t, &adctypes.Resources{}, executor.resources[0])

	_, ok := client.ConfigManager.GetConfig(gpA)
	assert.False(t, ok)
	assert.Empty(t, client.ConfigManager.Get(routeA))
	_, err := client.Store.ListGlobalRules(configA.Name)
	require.Error(t, err)

	_, ok = client.ConfigManager.GetConfig(gpB)
	assert.True(t, ok, "cleaning gp-a must not remove gp-b")
	assert.NotEmpty(t, client.ConfigManager.Get(routeB))
	_, err = client.Store.ListGlobalRules(gpB.String())
	require.NoError(t, err)
}

func TestDeleteGatewayProxyConfigRetainsOldConfigForRetry(t *testing.T) {
	executor := &gatewayProxyDeleteExecutor{errs: []error{rejection("connection refused")}}
	client := newGatewayProxyDeleteClient(t, executor)
	gp := types.NamespacedNameKind{Namespace: "default", Name: "gp-a", Kind: "GatewayProxy"}
	route := types.NamespacedNameKind{Namespace: "app", Name: "route-a", Kind: "HTTPRoute"}
	oldConfig := seedGatewayProxyStore(t, client, gp, route)

	require.Error(t, client.DeleteGatewayProxyConfig(context.Background(), gp))
	config, ok := client.ConfigManager.GetConfig(gp)
	require.True(t, ok, "the old endpoint must remain available for a retry")
	assert.Equal(t, oldConfig, config)
	_, err := client.Store.ListGlobalRules(oldConfig.Name)
	require.Error(t, err, "the periodic sync must observe an empty Store after failure")

	require.NoError(t, client.DeleteGatewayProxyConfig(context.Background(), gp))
	_, ok = client.ConfigManager.GetConfig(gp)
	assert.False(t, ok)
	require.Len(t, executor.resources, 2)
	assert.Equal(t, &adctypes.Resources{}, executor.resources[0])
	assert.Equal(t, &adctypes.Resources{}, executor.resources[1])
}

func TestDeleteGatewayProxyConfigIsIdempotentWithoutStoredConfig(t *testing.T) {
	executor := &gatewayProxyDeleteExecutor{}
	client := newGatewayProxyDeleteClient(t, executor)
	gp := types.NamespacedNameKind{Namespace: "default", Name: "gp-a", Kind: "GatewayProxy"}

	require.NoError(t, client.DeleteGatewayProxyConfig(context.Background(), gp))
	require.NoError(t, client.DeleteGatewayProxyConfig(context.Background(), gp))
	assert.Empty(t, executor.configs)
}
