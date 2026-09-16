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

package cache

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	"github.com/apache/apisix-ingress-controller/internal/types"
)

const configName = "GatewayProxy/ns/gp"

var gatewayProxy = types.NamespacedNameKind{Kind: types.KindGatewayProxy, Namespace: "ns", Name: "gp"}

func ownerNamed(kind, name string) types.NamespacedNameKind {
	return types.NamespacedNameKind{Kind: kind, Namespace: "ns", Name: name}
}

func labelsOf(owner types.NamespacedNameKind) map[string]string {
	return map[string]string{label.LabelKind: owner.Kind, label.LabelNamespace: owner.Namespace, label.LabelName: owner.Name}
}

func service(id string, owner types.NamespacedNameKind) *adctypes.Service {
	return &adctypes.Service{Metadata: adctypes.Metadata{ID: id, Name: "name-" + id, Labels: labelsOf(owner)}}
}

func TestLookupFindsTheOwnerOfEveryTopLevelType(t *testing.T) {
	route := ownerNamed(types.KindApisixRoute, "route")
	tls := ownerNamed(types.KindApisixTls, "tls")
	consumer := ownerNamed(types.KindConsumer, "consumer")
	globalRule := ownerNamed(types.KindApisixGlobalRule, "global")

	s := NewStore(logr.Discard())
	require.NoError(t, s.Insert(configName, []string{adctypes.TypeService}, &adctypes.Resources{Services: []*adctypes.Service{service("svc", route)}}, labelsOf(route)))
	require.NoError(t, s.Insert(configName, []string{adctypes.TypeSSL}, &adctypes.Resources{SSLs: []*adctypes.SSL{{Metadata: adctypes.Metadata{ID: "ssl"}}}}, labelsOf(tls)))
	require.NoError(t, s.Insert(configName, []string{adctypes.TypeConsumer}, &adctypes.Resources{Consumers: []*adctypes.Consumer{{Username: "alice"}}}, labelsOf(consumer)))
	require.NoError(t, s.SetGlobalRules(configName, globalRule, adctypes.GlobalRule{"prometheus": map[string]any{}}))
	require.NoError(t, s.SetPluginMetadata(configName, adctypes.PluginMetadata{"http-logger": map[string]any{}}))

	cases := []struct {
		resourceType, id, name string
		owner                  types.NamespacedNameKind
	}{
		{adctypes.TypeService, "svc", "name-svc", route},
		{adctypes.TypeSSL, "ssl", "ssl", tls},
		{adctypes.TypeConsumer, "alice", "alice", consumer},
		{adctypes.TypeGlobalRule, "prometheus", "prometheus", globalRule},
		{adctypes.TypePluginMetadata, "http-logger", "http-logger", gatewayProxy},
	}
	for _, tc := range cases {
		t.Run(tc.resourceType, func(t *testing.T) {
			entity, ok := s.Lookup(configName, tc.resourceType, tc.id)
			require.True(t, ok)
			assert.Equal(t, tc.owner, entity.Owner)
			assert.Equal(t, tc.name, entity.Name)
		})
	}

	_, ok := s.Lookup(configName, adctypes.TypeService, "missing")
	assert.False(t, ok)
	_, ok = s.Lookup(configName, adctypes.TypePluginMetadata, "missing")
	assert.False(t, ok)
	_, ok = s.Lookup("GatewayProxy/ns/other", adctypes.TypeService, "svc")
	assert.False(t, ok)
}

func TestInsertForgetsTheOwnerOfReplacedResources(t *testing.T) {
	route := ownerNamed(types.KindApisixRoute, "route")
	s := NewStore(logr.Discard())
	require.NoError(t, s.Insert(configName, []string{adctypes.TypeService}, &adctypes.Resources{Services: []*adctypes.Service{service("old", route)}}, labelsOf(route)))
	require.NoError(t, s.Insert(configName, []string{adctypes.TypeService}, &adctypes.Resources{Services: []*adctypes.Service{service("new", route)}}, labelsOf(route)))

	_, ok := s.Lookup(configName, adctypes.TypeService, "old")
	assert.False(t, ok)
	_, ok = s.Lookup(configName, adctypes.TypeService, "new")
	assert.True(t, ok)

	require.NoError(t, s.Delete(configName, []string{adctypes.TypeService}, labelsOf(route)))
	_, ok = s.Lookup(configName, adctypes.TypeService, "new")
	assert.False(t, ok)
}

func TestDeleteWithoutResourceTypesDeletesNothing(t *testing.T) {
	route := ownerNamed(types.KindApisixRoute, "route")
	s := NewStore(logr.Discard())
	require.NoError(t, s.Insert(configName, []string{adctypes.TypeService}, &adctypes.Resources{Services: []*adctypes.Service{service("svc", route)}}, labelsOf(route)))

	require.NoError(t, s.Delete(configName, nil, nil))
	_, ok := s.Lookup(configName, adctypes.TypeService, "svc")
	assert.True(t, ok)

	s.DeleteAll(configName)
	_, ok = s.Lookup(configName, adctypes.TypeService, "svc")
	assert.False(t, ok)
}

func TestSetGlobalRulesReplacesOnlyThatOwnersPlugins(t *testing.T) {
	globalRule := ownerNamed(types.KindApisixGlobalRule, "global")
	s := NewStore(logr.Discard())
	require.NoError(t, s.SetGlobalRules(configName, gatewayProxy, adctypes.GlobalRule{"cors": map[string]any{"a": "b"}}))
	require.NoError(t, s.SetGlobalRules(configName, globalRule, adctypes.GlobalRule{"prometheus": map[string]any{}, "old": map[string]any{}}))
	require.NoError(t, s.SetGlobalRules(configName, globalRule, adctypes.GlobalRule{"prometheus": map[string]any{}}))

	resources, _, err := s.GetResources(configName)
	require.NoError(t, err)
	assert.Equal(t, adctypes.GlobalRule{"cors": map[string]any{"a": "b"}, "prometheus": map[string]any{}}, resources.GlobalRules)

	require.NoError(t, s.SetGlobalRules(configName, globalRule, nil))
	resources, _, err = s.GetResources(configName)
	require.NoError(t, err)
	assert.Equal(t, adctypes.GlobalRule{"cors": map[string]any{"a": "b"}}, resources.GlobalRules)
}

func TestSetGlobalRulesOfTheSameNameOverwritesAndAttributesToTheLastWriter(t *testing.T) {
	globalRule := ownerNamed(types.KindApisixGlobalRule, "global")
	s := NewStore(logr.Discard())
	require.NoError(t, s.SetGlobalRules(configName, gatewayProxy, adctypes.GlobalRule{"prometheus": map[string]any{"from": "gp"}}))
	require.NoError(t, s.SetGlobalRules(configName, globalRule, adctypes.GlobalRule{"prometheus": map[string]any{"from": "agr"}}))

	resources, _, err := s.GetResources(configName)
	require.NoError(t, err)
	entity, ok := s.Lookup(configName, adctypes.TypeGlobalRule, "prometheus")
	require.True(t, ok)
	assert.Equal(t, map[string]any{"from": "agr"}, resources.GlobalRules["prometheus"])
	assert.Equal(t, globalRule, entity.Owner, "what is pushed and who it is attributed to always agree")
}

func TestSetPluginMetadataReplacesEverything(t *testing.T) {
	s := NewStore(logr.Discard())
	require.NoError(t, s.SetPluginMetadata(configName, adctypes.PluginMetadata{"old": map[string]any{}}))
	require.NoError(t, s.SetPluginMetadata(configName, adctypes.PluginMetadata{"new": map[string]any{}}))

	resources, _, err := s.GetResources(configName)
	require.NoError(t, err)
	assert.Equal(t, adctypes.PluginMetadata{"new": map[string]any{}}, resources.PluginMetadata)
}

func TestOwnedEntities(t *testing.T) {
	route := ownerNamed(types.KindApisixRoute, "route")
	other := ownerNamed(types.KindApisixRoute, "other")
	s := NewStore(logr.Discard())
	require.NoError(t, s.Insert(configName, []string{adctypes.TypeService}, &adctypes.Resources{Services: []*adctypes.Service{service("a", route), service("b", route)}}, labelsOf(route)))
	require.NoError(t, s.Insert(configName, []string{adctypes.TypeService}, &adctypes.Resources{Services: []*adctypes.Service{service("c", other)}}, labelsOf(other)))
	require.NoError(t, s.SetGlobalRules(configName, gatewayProxy, adctypes.GlobalRule{"cors": map[string]any{}}))
	require.NoError(t, s.SetPluginMetadata(configName, adctypes.PluginMetadata{"http-logger": map[string]any{}}))

	ids := func(entities []Entity) []string {
		var out []string
		for _, e := range entities {
			out = append(out, e.Type+"/"+e.ID)
		}
		return out
	}
	assert.ElementsMatch(t, []string{"service/a", "service/b"}, ids(s.OwnedEntities(configName, route)))
	assert.ElementsMatch(t, []string{"global_rule/cors", "plugin_metadata/http-logger"}, ids(s.OwnedEntities(configName, gatewayProxy)))
}

func TestChangedSinceIgnoresRewritesOfIdenticalContent(t *testing.T) {
	route := ownerNamed(types.KindApisixRoute, "route")
	s := NewStore(logr.Discard())
	write := func(plugins adctypes.Plugins) {
		svc := service("svc", route)
		svc.Plugins = plugins
		require.NoError(t, s.Insert(configName, []string{adctypes.TypeService}, &adctypes.Resources{Services: []*adctypes.Service{svc}}, labelsOf(route)))
	}
	write(adctypes.Plugins{"cors": map[string]any{"allow_origins": "*"}})
	revision := s.Revision()

	write(adctypes.Plugins{"cors": map[string]any{"allow_origins": "*"}})
	require.NoError(t, s.SetGlobalRules(configName, gatewayProxy, nil))
	require.NoError(t, s.SetPluginMetadata(configName, nil))
	assert.False(t, s.ChangedSince(configName, route, revision))
	assert.False(t, s.OwnerChangedSince(route, revision))
	assert.False(t, s.ChangedSince(configName, gatewayProxy, revision))

	write(adctypes.Plugins{"cors": map[string]any{"allow_origins": "example.com"}})
	assert.True(t, s.ChangedSince(configName, route, revision))
	assert.True(t, s.OwnerChangedSince(route, revision))
	assert.False(t, s.ChangedSince("GatewayProxy/ns/other", route, revision))

	revision = s.Revision()
	require.NoError(t, s.SetPluginMetadata(configName, adctypes.PluginMetadata{"http-logger": map[string]any{}}))
	assert.True(t, s.ChangedSince(configName, gatewayProxy, revision))

	revision = s.Revision()
	s.DeleteAll(configName)
	assert.True(t, s.ChangedSince(configName, route, revision), "wiping a cacheKey changes everything it held")
}

// typedPluginConfig declares its fields out of alphabetical order, like a translator's
// typed plugin config can.
type typedPluginConfig struct {
	Zone string `json:"zone"`
	Area string `json:"area"`
}

// TestChangedSinceComparesWhatWouldBePushed covers a stored object, whose plugin configs
// have been through a JSON round trip, compared against a freshly translated one holding
// a typed config.
func TestChangedSinceComparesWhatWouldBePushed(t *testing.T) {
	route := ownerNamed(types.KindApisixRoute, "route")
	s := NewStore(logr.Discard())
	write := func() {
		svc := service("svc", route)
		svc.Plugins = adctypes.Plugins{"typed": &typedPluginConfig{Zone: "z", Area: "a"}}
		require.NoError(t, s.Insert(configName, []string{adctypes.TypeService}, &adctypes.Resources{Services: []*adctypes.Service{svc}}, labelsOf(route)))
		require.NoError(t, s.SetGlobalRules(configName, route, adctypes.GlobalRule{"typed": &typedPluginConfig{Zone: "z", Area: "a"}}))
	}
	write()
	revision := s.Revision()

	write()
	require.NoError(t, s.Insert(configName, []string{adctypes.TypeSSL}, &adctypes.Resources{}, labelsOf(route)))
	assert.False(t, s.ChangedSince(configName, route, revision))
}
