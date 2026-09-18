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

func ssl(id string, owner types.NamespacedNameKind) *adctypes.SSL {
	return &adctypes.SSL{Metadata: adctypes.Metadata{ID: id, Labels: labelsOf(owner)}}
}

func consumer(username string, owner types.NamespacedNameKind) *adctypes.Consumer {
	return &adctypes.Consumer{Username: username, Metadata: adctypes.Metadata{Labels: labelsOf(owner)}}
}

func TestLookupFindsTheOwnerOfEveryTopLevelType(t *testing.T) {
	route := ownerNamed(types.KindApisixRoute, "route")
	tls := ownerNamed(types.KindApisixTls, "tls")
	consumerOwner := ownerNamed(types.KindConsumer, "consumer")
	globalRule := ownerNamed(types.KindApisixGlobalRule, "global")

	s := NewStore(logr.Discard())
	require.NoError(t, s.Insert(configName, []string{adctypes.TypeService}, &adctypes.Resources{Services: []*adctypes.Service{service("svc", route)}}, labelsOf(route)))
	require.NoError(t, s.Insert(configName, []string{adctypes.TypeSSL}, &adctypes.Resources{SSLs: []*adctypes.SSL{ssl("ssl", tls)}}, labelsOf(tls)))
	require.NoError(t, s.Insert(configName, []string{adctypes.TypeConsumer}, &adctypes.Resources{Consumers: []*adctypes.Consumer{consumer("alice", consumerOwner)}}, labelsOf(consumerOwner)))
	require.NoError(t, s.SetGlobalRules(configName, globalRule, adctypes.GlobalRule{"prometheus": map[string]any{}}))
	require.NoError(t, s.SetPluginMetadata(configName, adctypes.PluginMetadata{"http-logger": map[string]any{}}))

	cases := []struct {
		resourceType, id, name string
		owner                  types.NamespacedNameKind
	}{
		{adctypes.TypeService, "svc", "name-svc", route},
		{adctypes.TypeSSL, "ssl", "ssl", tls},
		{adctypes.TypeConsumer, "alice", "alice", consumerOwner},
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

// TestLookupFindsARouteByItsOwnLabels covers the one nested type Lookup already
// answers for: a route's own owner, which the service holding it doesn't always share
// (e.g. a traffic-split service combining rules from several ApisixRoutes).
func TestLookupFindsARouteByItsOwnLabels(t *testing.T) {
	svcOwner := ownerNamed(types.KindApisixRoute, "service-writer")
	routeOwner := ownerNamed(types.KindApisixRoute, "route-writer")
	s := NewStore(logr.Discard())
	withRoute := service("svc", svcOwner)
	withRoute.Routes = []*adctypes.Route{{Metadata: adctypes.Metadata{ID: "r1", Labels: labelsOf(routeOwner)}}}
	require.NoError(t, s.Insert(configName, []string{adctypes.TypeService}, &adctypes.Resources{Services: []*adctypes.Service{withRoute}}, labelsOf(svcOwner)))

	entity, ok := s.Lookup(configName, adctypes.TypeRoute, "r1")
	require.True(t, ok)
	assert.Equal(t, routeOwner, entity.Owner, "the route's own owner, not the service's")

	_, ok = s.Lookup(configName, adctypes.TypeRoute, "missing")
	assert.False(t, ok)
}

func TestSetGlobalRulesReplacesOnlyThatOwnersPlugins(t *testing.T) {
	globalRule := ownerNamed(types.KindApisixGlobalRule, "global")
	s := NewStore(logr.Discard())
	require.NoError(t, s.SetGlobalRules(configName, gatewayProxy, adctypes.GlobalRule{"cors": map[string]any{"a": "b"}}))
	require.NoError(t, s.SetGlobalRules(configName, globalRule, adctypes.GlobalRule{"prometheus": map[string]any{}, "old": map[string]any{}}))
	require.NoError(t, s.SetGlobalRules(configName, globalRule, adctypes.GlobalRule{"prometheus": map[string]any{}}))

	resources, err := s.GetResources(configName)
	require.NoError(t, err)
	assert.Equal(t, adctypes.GlobalRule{"cors": map[string]any{"a": "b"}, "prometheus": map[string]any{}}, resources.GlobalRules)

	require.NoError(t, s.SetGlobalRules(configName, globalRule, nil))
	resources, err = s.GetResources(configName)
	require.NoError(t, err)
	assert.Equal(t, adctypes.GlobalRule{"cors": map[string]any{"a": "b"}}, resources.GlobalRules)
}

func TestSetGlobalRulesOfTheSameNameOverwritesAndAttributesToTheLastWriter(t *testing.T) {
	globalRule := ownerNamed(types.KindApisixGlobalRule, "global")
	s := NewStore(logr.Discard())
	require.NoError(t, s.SetGlobalRules(configName, gatewayProxy, adctypes.GlobalRule{"prometheus": map[string]any{"from": "gp"}}))
	require.NoError(t, s.SetGlobalRules(configName, globalRule, adctypes.GlobalRule{"prometheus": map[string]any{"from": "agr"}}))

	resources, err := s.GetResources(configName)
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

	resources, err := s.GetResources(configName)
	require.NoError(t, err)
	assert.Equal(t, adctypes.PluginMetadata{"new": map[string]any{}}, resources.PluginMetadata)
}

// TestLookupReadsTheEntitysOwnLabelsNotInsertsArgument covers why Lookup can't source
// the owner from anywhere but the entity's own stored labels: those are also what
// KindLabelSelector matches Delete and a future Insert against, so this is the only
// choice that can never disagree with which owner a selector-based lookup would find.
// Insert's Labels argument is deliberately wrong here to prove it plays no part.
func TestLookupReadsTheEntitysOwnLabelsNotInsertsArgument(t *testing.T) {
	owner := ownerNamed(types.KindApisixRoute, "owner")
	wrong := ownerNamed(types.KindApisixRoute, "wrong")
	s := NewStore(logr.Discard())

	require.NoError(t, s.Insert(configName, []string{adctypes.TypeSSL}, &adctypes.Resources{SSLs: []*adctypes.SSL{ssl("ssl", owner)}}, labelsOf(wrong)))

	entity, ok := s.Lookup(configName, adctypes.TypeSSL, "ssl")
	require.True(t, ok)
	assert.Equal(t, owner, entity.Owner, "the ssl's own labels, not whatever Insert was called with")
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
	assert.True(t, ok, "an empty resourceTypes deletes nothing; see DeleteAll for wiping a whole cacheKey")

	s.DeleteAll(configName)
	_, ok = s.Lookup(configName, adctypes.TypeService, "svc")
	assert.False(t, ok)
}

func TestOwnedEntities(t *testing.T) {
	route := ownerNamed(types.KindApisixRoute, "route")
	other := ownerNamed(types.KindApisixRoute, "other")
	s := NewStore(logr.Discard())

	withChildren := service("svc", route)
	withChildren.Routes = []*adctypes.Route{{Metadata: adctypes.Metadata{ID: "r1", Name: "r1"}}}
	withChildren.StreamRoutes = []*adctypes.StreamRoute{{Metadata: adctypes.Metadata{ID: "sr1"}}}
	require.NoError(t, s.Insert(configName, []string{adctypes.TypeService}, &adctypes.Resources{Services: []*adctypes.Service{withChildren}}, labelsOf(route)))
	require.NoError(t, s.Insert(configName, []string{adctypes.TypeSSL}, &adctypes.Resources{SSLs: []*adctypes.SSL{ssl("ssl", route)}}, labelsOf(route)))
	require.NoError(t, s.Insert(configName, []string{adctypes.TypeConsumer}, &adctypes.Resources{Consumers: []*adctypes.Consumer{consumer("alice", other)}}, labelsOf(other)))

	got := s.OwnedEntities(configName, route)
	assert.Len(t, got, 2, "only what route itself owns, not other's consumer")

	var svcEntity Entity
	for _, e := range got {
		if e.Type == adctypes.TypeService {
			svcEntity = e
		}
	}
	require.Equal(t, "svc", svcEntity.ID)
	assert.Len(t, svcEntity.Children, 2, "the service's route and stream route")

	assert.Empty(t, s.OwnedEntities(configName, ownerNamed(types.KindApisixRoute, "nobody")))
}

func TestOwnedEntitiesIncludesGlobalRulesAndPluginMetadataOfTheGatewayProxy(t *testing.T) {
	globalRule := ownerNamed(types.KindApisixGlobalRule, "global")
	s := NewStore(logr.Discard())
	require.NoError(t, s.SetGlobalRules(configName, gatewayProxy, adctypes.GlobalRule{"cors": map[string]any{}}))
	require.NoError(t, s.SetGlobalRules(configName, globalRule, adctypes.GlobalRule{"prometheus": map[string]any{}}))
	require.NoError(t, s.SetPluginMetadata(configName, adctypes.PluginMetadata{"http-logger": map[string]any{}}))

	gpEntities := s.OwnedEntities(configName, gatewayProxy)
	assert.Len(t, gpEntities, 2, "the GatewayProxy's own global rule and the cacheKey's plugin metadata")

	agrEntities := s.OwnedEntities(configName, globalRule)
	require.Len(t, agrEntities, 1)
	assert.Equal(t, adctypes.TypeGlobalRule, agrEntities[0].Type)
	assert.Equal(t, "prometheus", agrEntities[0].ID)
}
