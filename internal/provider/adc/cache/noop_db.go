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
	types "github.com/apache/apisix-ingress-controller/api/adc"
)

type noopCache struct {
}

// NewMemDBCache creates a Cache object backs with a memory DB.
func NewNoopDBCache() (Cache, error) {
	return &noopCache{}, nil
}

func (c *noopCache) Insert(obj any) error {
	return nil
}

func (c *noopCache) Delete(obj any) error {
	return nil
}

func (c *noopCache) InsertSSL(ssl *types.SSL) error {
	return nil
}

func (c *noopCache) InsertService(u *types.Service) error {
	return nil
}

func (c *noopCache) InsertGlobalRule(gr *types.GlobalRuleItem) error {
	return nil
}

func (c *noopCache) InsertConsumer(consumer *types.Consumer) error {
	return nil
}

func (c *noopCache) GetSSL(id string) (*types.SSL, error) {
	return nil, nil
}

func (c *noopCache) GetService(id string) (*types.Service, error) {
	return nil, nil
}

func (c *noopCache) GetGlobalRule(id string) (*types.GlobalRuleItem, error) {
	return nil, nil
}

func (c *noopCache) GetConsumer(username string) (*types.Consumer, error) {
	return nil, nil
}

func (c *noopCache) ListSSL(...ListOption) ([]*types.SSL, error) {
	return nil, nil
}

func (c *noopCache) ListServices(...ListOption) ([]*types.Service, error) {
	return nil, nil
}

func (c *noopCache) ListStreamRoutes(...ListOption) ([]*types.StreamRoute, error) {
	return nil, nil
}

func (c *noopCache) ListGlobalRules(...ListOption) ([]*types.GlobalRuleItem, error) {
	return nil, nil
}

func (c *noopCache) ListConsumers(...ListOption) ([]*types.Consumer, error) {
	return nil, nil
}

func (c *noopCache) DeleteSSL(ssl *types.SSL) error {
	return nil
}

func (c *noopCache) DeleteService(u *types.Service) error {
	return nil
}

func (c *noopCache) DeleteGlobalRule(gr *types.GlobalRuleItem) error {
	return nil
}

func (c *noopCache) DeleteConsumer(consumer *types.Consumer) error {
	return nil
}
