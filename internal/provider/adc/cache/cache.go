// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cache

import (
	types "github.com/apache/apisix-ingress-controller/api/adc"
)

type Cache interface {
	Insert(obj any) error
	Delete(obj any) error

	// InsertSSL adds or updates ssl to cache.
	InsertSSL(*types.SSL) error
	// InsertUpstream adds or updates upstream to cache.
	InsertService(*types.Service) error
	// InsertConsumer adds or updates consumer to cache.
	InsertConsumer(*types.Consumer) error

	// GetSSL finds the ssl from cache according to the primary index (id).
	GetSSL(string) (*types.SSL, error)
	// GetUpstream finds the upstream from cache according to the primary index (id).
	GetService(string) (*types.Service, error)
	// GetConsumer finds the consumer from cache according to the primary index (username).
	GetConsumer(string) (*types.Consumer, error)

	// DeleteSSL deletes the specified ssl in cache.
	DeleteSSL(*types.SSL) error
	// DeleteUpstream deletes the specified upstream in cache.
	DeleteService(*types.Service) error
	// DeleteGlobalRule deletes the specified stream_route in cache.
	DeleteConsumer(*types.Consumer) error

	// ListSSL lists all ssl objects in cache.
	ListSSL(...ListOption) ([]*types.SSL, error)
	// ListUpstreams lists all upstreams in cache.
	ListServices(...ListOption) ([]*types.Service, error)
	// ListConsumers lists all consumer objects in cache.
	ListConsumers(...ListOption) ([]*types.Consumer, error)
}

type ListOption interface {
	ApplyToList(*ListOptions)
}

type ListOptions struct {
	KindLabelSelector *KindLabelSelector
}

func (o *ListOptions) ApplyToList(lo *ListOptions) {
	if o.KindLabelSelector != nil {
		lo.KindLabelSelector = o.KindLabelSelector
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
