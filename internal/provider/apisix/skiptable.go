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

// wireKey identifies one ADC resource as an ADC event names it. parentID is set only for
// the nested types: the service a route or stream route lives in, the consumer a
// credential lives in.
type wireKey struct {
	resourceType string
	parentID     string
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

// ClearCacheKey removes every entry of cacheKey.
func (t *skipTable) ClearCacheKey(cacheKey string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.entries, cacheKey)
}

// exclude returns resources without the excluded wire resources. Neither resources nor
// the objects it points to are modified; an object with nothing excluded is kept as is.
func exclude(resources *adctypes.Resources, excluded map[wireKey]exclusion) *adctypes.Resources {
	if resources == nil || len(excluded) == 0 {
		return resources
	}
	has := func(resourceType, parentID, id string) bool {
		_, ok := excluded[wireKey{resourceType, parentID, id}]
		return ok
	}

	result := *resources
	if resources.Services != nil {
		result.Services = make([]*adctypes.Service, 0, len(resources.Services))
		for _, svc := range resources.Services {
			if has(adctypes.TypeService, "", svc.ID) {
				continue
			}
			result.Services = append(result.Services, excludeFromService(svc, has))
		}
	}
	if resources.Consumers != nil {
		result.Consumers = make([]*adctypes.Consumer, 0, len(resources.Consumers))
		for _, consumer := range resources.Consumers {
			if has(adctypes.TypeConsumer, "", consumer.Username) {
				continue
			}
			result.Consumers = append(result.Consumers, excludeFromConsumer(consumer, has))
		}
	}
	if resources.SSLs != nil {
		result.SSLs = make([]*adctypes.SSL, 0, len(resources.SSLs))
		for _, ssl := range resources.SSLs {
			if !has(adctypes.TypeSSL, "", ssl.ID) {
				result.SSLs = append(result.SSLs, ssl)
			}
		}
	}
	result.GlobalRules = adctypes.GlobalRule(excludePlugins(adctypes.Plugins(resources.GlobalRules), adctypes.TypeGlobalRule, has))
	result.PluginMetadata = adctypes.PluginMetadata(excludePlugins(adctypes.Plugins(resources.PluginMetadata), adctypes.TypePluginMetadata, has))
	return &result
}

func excludeFromService(svc *adctypes.Service, has func(resourceType, parentID, id string) bool) *adctypes.Service {
	changed := false
	routes := make([]*adctypes.Route, 0, len(svc.Routes))
	for _, route := range svc.Routes {
		if has(adctypes.TypeRoute, svc.ID, route.ID) {
			changed = true
			continue
		}
		routes = append(routes, route)
	}
	streamRoutes := make([]*adctypes.StreamRoute, 0, len(svc.StreamRoutes))
	for _, sr := range svc.StreamRoutes {
		if has(adctypes.TypeStreamRoute, svc.ID, sr.ID) {
			changed = true
			continue
		}
		streamRoutes = append(streamRoutes, sr)
	}
	if !changed {
		return svc
	}
	filtered := *svc
	filtered.Routes = routes
	filtered.StreamRoutes = streamRoutes
	return &filtered
}

func excludeFromConsumer(consumer *adctypes.Consumer, has func(resourceType, parentID, id string) bool) *adctypes.Consumer {
	changed := false
	credentials := make([]adctypes.Credential, 0, len(consumer.Credentials))
	for _, cred := range consumer.Credentials {
		if has(adctypes.TypeConsumerCredential, consumer.Username, cred.ID) {
			changed = true
			continue
		}
		credentials = append(credentials, cred)
	}
	if !changed {
		return consumer
	}
	filtered := *consumer
	filtered.Credentials = credentials
	return &filtered
}

func excludePlugins(plugins adctypes.Plugins, resourceType string, has func(resourceType, parentID, id string) bool) adctypes.Plugins {
	var result adctypes.Plugins
	for name := range plugins {
		if has(resourceType, "", name) {
			if result == nil {
				result = make(adctypes.Plugins, len(plugins))
				for k, v := range plugins {
					result[k] = v
				}
			}
			delete(result, name)
		}
	}
	if result == nil {
		return plugins
	}
	return result
}
