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
	internaltypes "github.com/apache/apisix-ingress-controller/internal/types"
)

// GlobalRuleRow is one global_rules plugin, keyed by the plugin name. Owner is the
// Kubernetes resource that declared it: a GatewayProxy or an ApisixGlobalRule.
type GlobalRuleRow struct {
	ID     string
	Owner  internaltypes.NamespacedNameKind
	Config any
}

func (r *GlobalRuleRow) DeepCopy() *GlobalRuleRow {
	out := *r
	out.Config = copyPluginConfig(r.ID, r.Config)
	return &out
}

// PluginMetadataRow is one plugin_metadata entry, keyed by the plugin name. It needs no
// owner: its only source is the GatewayProxy the cacheKey itself names.
type PluginMetadataRow struct {
	ID     string
	Config any
}

func (r *PluginMetadataRow) DeepCopy() *PluginMetadataRow {
	out := *r
	out.Config = copyPluginConfig(r.ID, r.Config)
	return &out
}

func copyPluginConfig(name string, config any) any {
	return types.Plugins{name: config}.DeepCopy()[name]
}

type Cache interface {
	Insert(obj any) error
	Delete(obj any) error

	// InsertSSL adds or updates ssl to cache.
	InsertSSL(*types.SSL) error
	// InsertUpstream adds or updates upstream to cache.
	InsertService(*types.Service) error
	// InsertConsumer adds or updates consumer to cache.
	InsertConsumer(*types.Consumer) error
	// InsertGlobalRule adds or updates global rule to cache.
	InsertGlobalRule(*GlobalRuleRow) error
	// InsertPluginMetadata adds or updates plugin metadata to cache.
	InsertPluginMetadata(*PluginMetadataRow) error

	// GetSSL finds the ssl from cache according to the primary index (id).
	GetSSL(string) (*types.SSL, error)
	// GetUpstream finds the upstream from cache according to the primary index (id).
	GetService(string) (*types.Service, error)
	// GetConsumer finds the consumer from cache according to the primary index (username).
	GetConsumer(string) (*types.Consumer, error)
	// GetGlobalRule finds the global rule from cache according to the primary index (id).
	GetGlobalRule(string) (*GlobalRuleRow, error)
	// GetPluginMetadata finds the plugin metadata from cache according to the primary index (id).
	GetPluginMetadata(string) (*PluginMetadataRow, error)

	// DeleteSSL deletes the specified ssl in cache.
	DeleteSSL(*types.SSL) error
	// DeleteUpstream deletes the specified upstream in cache.
	DeleteService(*types.Service) error
	// DeleteConsumer deletes the specified consumer in cache.
	DeleteConsumer(*types.Consumer) error
	// DeleteGlobalRule deletes the specified global rule in cache.
	DeleteGlobalRule(*GlobalRuleRow) error
	// DeletePluginMetadata deletes the specified plugin metadata in cache.
	DeletePluginMetadata(*PluginMetadataRow) error

	// ListSSL lists all ssl objects in cache.
	ListSSL(...ListOption) ([]*types.SSL, error)
	// ListUpstreams lists all upstreams in cache.
	ListServices(...ListOption) ([]*types.Service, error)
	// ListConsumers lists all consumer objects in cache.
	ListConsumers(...ListOption) ([]*types.Consumer, error)
	// ListGlobalRules lists all global rule objects in cache.
	ListGlobalRules(...ListOption) ([]*GlobalRuleRow, error)
	// ListPluginMetadata lists all plugin metadata objects in cache.
	ListPluginMetadata(...ListOption) ([]*PluginMetadataRow, error)
}

type ListOption interface {
	ApplyToList(*ListOptions)
}

type ListOptions struct {
	KindLabelSelector *KindLabelSelector
	OwnerSelector     *OwnerSelector
}

func (o *ListOptions) ApplyToList(lo *ListOptions) {
	if o.KindLabelSelector != nil {
		lo.KindLabelSelector = o.KindLabelSelector
	}
	if o.OwnerSelector != nil {
		lo.OwnerSelector = o.OwnerSelector
	}
}

func (o *ListOptions) ApplyOptions(opts []ListOption) *ListOptions {
	for _, opt := range opts {
		opt.ApplyToList(o)
	}
	return o
}

type KindLabelSelector struct {
	Kind      string
	Name      string
	Namespace string
}

func (o *KindLabelSelector) ApplyToList(opts *ListOptions) {
	opts.KindLabelSelector = o
}

// OwnerSelector lists only the global rules declared by Owner.
type OwnerSelector struct {
	Owner internaltypes.NamespacedNameKind
}

func (o *OwnerSelector) ApplyToList(opts *ListOptions) {
	opts.OwnerSelector = o
}
