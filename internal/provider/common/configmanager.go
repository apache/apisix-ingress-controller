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

package common

import (
	"sync"
)

type ConfigManager[K comparable, T any] struct {
	mu                 sync.Mutex
	resourceConfigKeys map[K][]K
	configs            map[K]T
	configRefs         map[K][]K
}

func NewConfigManager[K comparable, T any]() *ConfigManager[K, T] {
	return &ConfigManager[K, T]{
		resourceConfigKeys: make(map[K][]K),
		configs:            make(map[K]T),
		configRefs:         make(map[K][]K),
	}
}

func (s *ConfigManager[K, T]) GetConfigRefs(key K) []K {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.configRefs[key]
}

func (s *ConfigManager[K, T]) GetConfigRefsByResourceKey(key K) []K {
	s.mu.Lock()
	defer s.mu.Unlock()
	configKeys, ok := s.resourceConfigKeys[key]
	if !ok {
		return nil
	}
	refs := make([]K, 0, len(configKeys))
	for _, k := range configKeys {
		if ref, ok := s.configRefs[k]; ok {
			refs = append(refs, ref...)
		}
	}
	return refs
}

func (s *ConfigManager[K, T]) SetConfigRefs(key K, refs []K) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.configRefs[key] = refs
}

func (s *ConfigManager[K, T]) Get(key K) map[K]T {
	s.mu.Lock()
	defer s.mu.Unlock()

	resourceConfigKeys := s.resourceConfigKeys[key]
	configs := make(map[K]T, len(resourceConfigKeys))
	for _, parent := range resourceConfigKeys {
		if cfg, ok := s.configs[parent]; ok {
			configs[parent] = cfg
		}
	}
	return configs
}

func (s *ConfigManager[K, T]) List() map[K]T {
	s.mu.Lock()
	defer s.mu.Unlock()

	configs := make(map[K]T, len(s.configs))
	for k, v := range s.configs {
		configs[k] = v
	}
	return configs
}

func (s *ConfigManager[K, T]) UpdateConfig(key K, cfg T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.configs[key] = cfg
}

func (s *ConfigManager[K, T]) Update(
	key K,
	mapRefs map[K]T,
) (discard map[K]T) {
	s.mu.Lock()
	defer s.mu.Unlock()

	parentRefSet := make(map[K]struct{})
	oldParentRefs := s.resourceConfigKeys[key]
	newRefs := make([]K, 0, len(mapRefs))

	for k, v := range mapRefs {
		newRefs = append(newRefs, k)
		s.configs[k] = v
		parentRefSet[k] = struct{}{}
	}
	s.resourceConfigKeys[key] = newRefs
	discard = make(map[K]T)
	for _, old := range oldParentRefs {
		if _, stillUsed := parentRefSet[old]; !stillUsed {
			if cfg, ok := s.configs[old]; ok {
				discard[old] = cfg
			}
		}
	}

	return discard
}

func (s *ConfigManager[K, T]) Set(key K, cfg T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.configs[key] = cfg
}

func (s *ConfigManager[K, T]) Delete(key K) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.resourceConfigKeys, key)
	delete(s.configs, key)
	delete(s.configRefs, key)
}

func (s *ConfigManager[K, T]) DeleteConfig(key K) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.configs, key)
	delete(s.configRefs, key)
}
