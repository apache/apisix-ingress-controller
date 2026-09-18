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
	"testing"

	"github.com/stretchr/testify/assert"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/types"
)

func apisixRoute(name string) types.NamespacedNameKind {
	return types.NamespacedNameKind{Kind: types.KindApisixRoute, Namespace: "ns", Name: name}
}

func TestSkipTableMarkFailingKeepsEntriesNotMentionedAgain(t *testing.T) {
	tbl := newSkipTable()
	first := wireKey{resourceType: adctypes.TypeService, id: "svc1"}
	second := wireKey{resourceType: adctypes.TypeService, id: "svc2"}

	tbl.MarkFailing("gp1", map[wireKey]exclusion{first: {owner: apisixRoute("a"), reason: "first"}})
	tbl.MarkFailing("gp1", map[wireKey]exclusion{second: {owner: apisixRoute("b"), reason: "second"}})

	got := tbl.Excluded("gp1")
	assert.Equal(t, "first", got[first].reason, "an excluded resource is never reported again, which doesn't mean it was fixed")
	assert.Equal(t, "second", got[second].reason)
	assert.Empty(t, tbl.Excluded("gp2"))
}

func TestSkipTableExcludedAndSnapshotAreCopies(t *testing.T) {
	tbl := newSkipTable()
	key := wireKey{resourceType: adctypes.TypeService, id: "svc1"}
	tbl.MarkFailing("gp1", map[wireKey]exclusion{key: {owner: apisixRoute("a"), reason: "reason"}})

	tbl.Excluded("gp1")[key] = exclusion{reason: "mutated"}
	tbl.Snapshot()["gp1"][key] = exclusion{reason: "mutated"}

	assert.Equal(t, "reason", tbl.Excluded("gp1")[key].reason)
}

func TestSkipTableClearOwnerRemovesOnlyThatOwnerEverywhere(t *testing.T) {
	tbl := newSkipTable()
	tbl.MarkFailing("gp1", map[wireKey]exclusion{
		{resourceType: adctypes.TypeService, id: "svc1"}:               {owner: apisixRoute("a")},
		{resourceType: adctypes.TypeRoute, parentID: "svc2", id: "r1"}: {owner: apisixRoute("b")},
	})
	tbl.MarkFailing("gp2", map[wireKey]exclusion{{resourceType: adctypes.TypeService, id: "svc1"}: {owner: apisixRoute("a")}})

	tbl.ClearOwner(apisixRoute("a"))

	assert.Equal(t, map[wireKey]exclusion{{resourceType: adctypes.TypeRoute, parentID: "svc2", id: "r1"}: {owner: apisixRoute("b")}}, tbl.Excluded("gp1"))
	assert.NotContains(t, tbl.Snapshot(), "gp2")
}

func TestSkipTableClearCacheKey(t *testing.T) {
	tbl := newSkipTable()
	key := wireKey{resourceType: adctypes.TypeService, id: "svc1"}
	tbl.MarkFailing("gp1", map[wireKey]exclusion{key: {owner: apisixRoute("a")}})
	tbl.MarkFailing("gp2", map[wireKey]exclusion{key: {owner: apisixRoute("a")}})

	tbl.ClearCacheKey("gp1")

	assert.Empty(t, tbl.Excluded("gp1"))
	assert.NotEmpty(t, tbl.Excluded("gp2"))
}

func TestExcludeDropsTopLevelResources(t *testing.T) {
	keepSvc := &adctypes.Service{Metadata: adctypes.Metadata{ID: "keep"}}
	dropSvc := &adctypes.Service{Metadata: adctypes.Metadata{ID: "drop"}}
	keepConsumer := &adctypes.Consumer{Username: "keep"}
	keepSSL := &adctypes.SSL{Metadata: adctypes.Metadata{ID: "keep"}}
	resources := &adctypes.Resources{
		Services:       []*adctypes.Service{keepSvc, dropSvc},
		Consumers:      []*adctypes.Consumer{keepConsumer, {Username: "drop"}},
		SSLs:           []*adctypes.SSL{keepSSL, {Metadata: adctypes.Metadata{ID: "drop"}}},
		GlobalRules:    adctypes.GlobalRule{"keep": map[string]any{}, "drop": map[string]any{}},
		PluginMetadata: adctypes.PluginMetadata{"keep": map[string]any{}, "drop": map[string]any{}},
	}

	got := exclude(resources, map[wireKey]exclusion{
		{resourceType: adctypes.TypeService, id: "drop"}:        {},
		{resourceType: adctypes.TypeConsumer, id: "drop"}:       {},
		{resourceType: adctypes.TypeSSL, id: "drop"}:            {},
		{resourceType: adctypes.TypeGlobalRule, id: "drop"}:     {},
		{resourceType: adctypes.TypePluginMetadata, id: "drop"}: {},
	})

	assert.Equal(t, []*adctypes.Service{keepSvc}, got.Services)
	assert.Equal(t, []*adctypes.Consumer{keepConsumer}, got.Consumers)
	assert.Equal(t, []*adctypes.SSL{keepSSL}, got.SSLs)
	assert.Equal(t, adctypes.GlobalRule{"keep": map[string]any{}}, got.GlobalRules)
	assert.Equal(t, adctypes.PluginMetadata{"keep": map[string]any{}}, got.PluginMetadata)

	assert.Len(t, resources.Services, 2, "the input is not modified")
	assert.Len(t, resources.GlobalRules, 2, "the input is not modified")
	assert.Len(t, resources.PluginMetadata, 2, "the input is not modified")
}

func TestExcludeDropsNestedResourcesOnlyUnderTheirParent(t *testing.T) {
	route := &adctypes.Route{Metadata: adctypes.Metadata{ID: "r"}}
	streamRoute := &adctypes.StreamRoute{Metadata: adctypes.Metadata{ID: "sr"}}
	affected := &adctypes.Service{
		Metadata:     adctypes.Metadata{ID: "affected"},
		Routes:       []*adctypes.Route{route, {Metadata: adctypes.Metadata{ID: "keep"}}},
		StreamRoutes: []*adctypes.StreamRoute{streamRoute},
	}
	// Same child ids under another service, which must be left alone.
	other := &adctypes.Service{
		Metadata:     adctypes.Metadata{ID: "other"},
		Routes:       []*adctypes.Route{route},
		StreamRoutes: []*adctypes.StreamRoute{streamRoute},
	}
	cred := adctypes.Credential{Metadata: adctypes.Metadata{ID: "cred"}}
	keepCred := adctypes.Credential{Metadata: adctypes.Metadata{ID: "keep"}}
	consumer := &adctypes.Consumer{Username: "alice", Credentials: []adctypes.Credential{cred, keepCred}}
	resources := &adctypes.Resources{Services: []*adctypes.Service{affected, other}, Consumers: []*adctypes.Consumer{consumer}}

	got := exclude(resources, map[wireKey]exclusion{
		{resourceType: adctypes.TypeRoute, parentID: "affected", id: "r"}:              {},
		{resourceType: adctypes.TypeStreamRoute, parentID: "affected", id: "sr"}:       {},
		{resourceType: adctypes.TypeConsumerCredential, parentID: "alice", id: "cred"}: {},
	})

	assert.Equal(t, []string{"keep"}, []string{got.Services[0].Routes[0].ID})
	assert.Len(t, got.Services[0].Routes, 1)
	assert.Empty(t, got.Services[0].StreamRoutes)
	assert.Same(t, other, got.Services[1], "a service with nothing excluded is kept as is")
	assert.Equal(t, []adctypes.Credential{keepCred}, got.Consumers[0].Credentials)
	assert.Len(t, affected.Routes, 2, "the input is not modified")
	assert.Len(t, consumer.Credentials, 2, "the input is not modified")
}

func TestExcludeWithNothingExcluded(t *testing.T) {
	resources := &adctypes.Resources{Services: []*adctypes.Service{{Metadata: adctypes.Metadata{ID: "svc"}}}}
	assert.Same(t, resources, exclude(resources, nil))
}
