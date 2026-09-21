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
	"sync"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/types"
)

// wireKey identifies one ADC resource as an ADC event names it.
type wireKey struct {
	resourceType string
	id           string
}

// exclusion is why a wire resource is left out of its cacheKey's push.
type exclusion struct {
	// owner is the Kubernetes resource the dropped resource belongs to.
	owner types.NamespacedNameKind
	// name identifies the dropped resource in a status message.
	name   string
	reason string
}

// skipTable holds, per cacheKey, the wire resources ADC rejected. sync leaves them out of
// the next push, so one bad resource stops blocking every other resource under the same
// GatewayProxy.
//
// An entry is cleared only when its owner's content is written to the store again: an
// excluded resource never reaches ADC, so ADC no longer reporting it says nothing about
// whether it was fixed.
type skipTable struct {
	mu      sync.Mutex
	entries map[string]map[wireKey]exclusion
}

func newSkipTable() *skipTable {
	return &skipTable{entries: make(map[string]map[wireKey]exclusion)}
}

// MarkFailing records failing for cacheKey, keeping entries it doesn't mention, and
// returns how many of them were not already excluded.
func (t *skipTable) MarkFailing(cacheKey string, failing map[wireKey]exclusion) int {
	if len(failing) == 0 {
		return 0
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.entries[cacheKey] == nil {
		t.entries[cacheKey] = make(map[wireKey]exclusion, len(failing))
	}
	added := 0
	for key, ex := range failing {
		if _, ok := t.entries[cacheKey][key]; !ok {
			added++
		}
		t.entries[cacheKey][key] = ex
	}
	return added
}

// Excluded returns a copy of cacheKey's entries.
func (t *skipTable) Excluded(cacheKey string) map[wireKey]exclusion {
	t.mu.Lock()
	defer t.mu.Unlock()
	snapshot := make(map[wireKey]exclusion, len(t.entries[cacheKey]))
	for key, ex := range t.entries[cacheKey] {
		snapshot[key] = ex
	}
	return snapshot
}

// Snapshot returns a copy of every cacheKey's entries.
func (t *skipTable) Snapshot() map[string]map[wireKey]exclusion {
	t.mu.Lock()
	defer t.mu.Unlock()
	snapshot := make(map[string]map[wireKey]exclusion, len(t.entries))
	for cacheKey, byKey := range t.entries {
		copied := make(map[wireKey]exclusion, len(byKey))
		for key, ex := range byKey {
			copied[key] = ex
		}
		snapshot[cacheKey] = copied
	}
	return snapshot
}

// ClearOwner removes owner's entries in every cacheKey.
func (t *skipTable) ClearOwner(owner types.NamespacedNameKind) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for cacheKey, byKey := range t.entries {
		for key, ex := range byKey {
			if ex.owner == owner {
				delete(byKey, key)
			}
		}
		if len(byKey) == 0 {
			delete(t.entries, cacheKey)
		}
	}
}

// ClearCacheKey removes every entry of cacheKey.
func (t *skipTable) ClearCacheKey(cacheKey string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.entries, cacheKey)
}

// exclude returns resources without the excluded wire resources. Neither resources nor
// the objects it points to are modified.
func exclude(resources *adctypes.Resources, excluded map[wireKey]exclusion) *adctypes.Resources {
	if resources == nil || len(excluded) == 0 {
		return resources
	}
	result := *resources
	if resources.Services != nil {
		result.Services = make([]*adctypes.Service, 0, len(resources.Services))
		for _, svc := range resources.Services {
			if _, ok := excluded[wireKey{adctypes.TypeService, svc.ID}]; !ok {
				result.Services = append(result.Services, svc)
			}
		}
	}
	return &result
}
