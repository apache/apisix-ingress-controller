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

// neverStale is the stale callback for tests that aren't exercising MarkFailing's own
// staleness check.
func neverStale(types.NamespacedNameKind) bool { return false }

func TestSkipTableMarkFailingKeepsEntriesNotMentionedAgain(t *testing.T) {
	tbl := newSkipTable()
	first := wireKey{adctypes.TypeService, "svc1"}
	second := wireKey{adctypes.TypeService, "svc2"}

	assert.Equal(t, 1, tbl.MarkFailing("gp1", map[wireKey]exclusion{first: {owner: apisixRoute("a"), reason: "first"}}, neverStale))
	assert.Equal(t, 1, tbl.MarkFailing("gp1", map[wireKey]exclusion{second: {owner: apisixRoute("b"), reason: "second"}}, neverStale))
	assert.Equal(t, 0, tbl.MarkFailing("gp1", map[wireKey]exclusion{first: {owner: apisixRoute("a"), reason: "again"}}, neverStale), "an entry already there is not new")

	got := tbl.Excluded("gp1")
	assert.Equal(t, "again", got[first].reason)
	assert.Equal(t, "second", got[second].reason, "an entry ADC stopped mentioning stays: it was excluded, not fixed")
	assert.Empty(t, tbl.Excluded("gp2"))
}

func TestSkipTableExcludedAndSnapshotAreCopies(t *testing.T) {
	tbl := newSkipTable()
	key := wireKey{adctypes.TypeService, "svc1"}
	tbl.MarkFailing("gp1", map[wireKey]exclusion{key: {owner: apisixRoute("a"), reason: "reason"}}, neverStale)

	tbl.Excluded("gp1")[key] = exclusion{reason: "mutated"}
	tbl.Snapshot()["gp1"][key] = exclusion{reason: "mutated"}

	assert.Equal(t, "reason", tbl.Excluded("gp1")[key].reason)
}

func TestSkipTableClearOwnerRemovesOnlyThatOwnerEverywhere(t *testing.T) {
	tbl := newSkipTable()
	tbl.MarkFailing("gp1", map[wireKey]exclusion{
		{adctypes.TypeService, "svc1"}: {owner: apisixRoute("a")},
		{adctypes.TypeService, "svc2"}: {owner: apisixRoute("b")},
	}, neverStale)
	tbl.MarkFailing("gp2", map[wireKey]exclusion{{adctypes.TypeService, "svc1"}: {owner: apisixRoute("a")}}, neverStale)

	tbl.ClearOwner(apisixRoute("a"))

	assert.Equal(t, map[wireKey]exclusion{{adctypes.TypeService, "svc2"}: {owner: apisixRoute("b")}}, tbl.Excluded("gp1"))
	assert.NotContains(t, tbl.Snapshot(), "gp2")
}

func TestSkipTableMarkFailingSkipsEntriesTheStaleCallbackRejects(t *testing.T) {
	tbl := newSkipTable()
	first := wireKey{adctypes.TypeService, "svc1"}
	second := wireKey{adctypes.TypeService, "svc2"}
	stale := func(owner types.NamespacedNameKind) bool { return owner == apisixRoute("a") }

	added := tbl.MarkFailing("gp1", map[wireKey]exclusion{
		first:  {owner: apisixRoute("a"), reason: "stale"},
		second: {owner: apisixRoute("b"), reason: "fresh"},
	}, stale)

	assert.Equal(t, 1, added, "the stale entry is not counted")
	got := tbl.Excluded("gp1")
	assert.NotContains(t, got, first, "an owner the caller reports as stale is never recorded")
	assert.Equal(t, "fresh", got[second].reason)
}

func TestSkipTableClearCacheKey(t *testing.T) {
	tbl := newSkipTable()
	key := wireKey{adctypes.TypeService, "svc1"}
	tbl.MarkFailing("gp1", map[wireKey]exclusion{key: {owner: apisixRoute("a")}}, neverStale)
	tbl.MarkFailing("gp2", map[wireKey]exclusion{key: {owner: apisixRoute("a")}}, neverStale)

	tbl.ClearCacheKey("gp1")

	assert.Empty(t, tbl.Excluded("gp1"))
	assert.NotEmpty(t, tbl.Excluded("gp2"))
}

func TestExcludeDropsExcludedServices(t *testing.T) {
	keep := &adctypes.Service{Metadata: adctypes.Metadata{ID: "keep"}}
	resources := &adctypes.Resources{
		Services:  []*adctypes.Service{keep, {Metadata: adctypes.Metadata{ID: "drop"}}},
		Consumers: []*adctypes.Consumer{{Username: "drop"}},
	}

	got := exclude(resources, map[wireKey]exclusion{{adctypes.TypeService, "drop"}: {}})

	assert.Equal(t, []*adctypes.Service{keep}, got.Services)
	assert.Len(t, got.Consumers, 1, "only services are excluded so far")
	assert.Len(t, resources.Services, 2, "the input is not modified")
}

func TestExcludeWithNothingExcluded(t *testing.T) {
	resources := &adctypes.Resources{Services: []*adctypes.Service{{Metadata: adctypes.Metadata{ID: "svc"}}}}
	assert.Same(t, resources, exclude(resources, nil))
}
