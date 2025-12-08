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

package readiness

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	types "github.com/apache/apisix-ingress-controller/internal/types"
)

// Filter defines an interface to match unstructured Kubernetes objects.
type Filter interface {
	Match(obj *unstructured.Unstructured) bool
}

// GVKFilter is a functional implementation of Filter using a function type.
type GVKFilter func(obj *unstructured.Unstructured) bool

func (f GVKFilter) Match(obj *unstructured.Unstructured) bool {
	return f(obj)
}

// GVKConfig defines a set of GVKs and an optional filter to match the objects.
type GVKConfig struct {
	GVKs   []schema.GroupVersionKind
	Filter Filter
}

// readinessManager prevents premature full sync to the data plane on controller startup.
//
// Background:
// On startup, the controller watches CRDs and periodically performs full sync to the data plane.
// If a sync occurs before all resources have been reconciled, it may push incomplete data,
// causing traffic disruption.
//
// This manager tracks whether all relevant resources have been processed at least once.
// It is used to delay full sync until initial reconciliation is complete.
type ReadinessManager interface {
	RegisterGVK(configs ...GVKConfig)
	Start(ctx context.Context) error
	IsReady() bool
	WaitReady(ctx context.Context, timeout time.Duration) bool
	Done(obj client.Object, namespacedName k8stypes.NamespacedName)
}

type readinessManager struct {
	client    client.Client
	configs   []GVKConfig
	state     map[schema.GroupVersionKind]map[k8stypes.NamespacedName]struct{}
	mu        sync.RWMutex
	startOnce sync.Once
	started   chan struct{}
	done      chan struct{}

	isReady atomic.Bool

	log logr.Logger
}

// ReadinessManager tracks readiness of specific resources across the cluster.
func NewReadinessManager(client client.Client, log logr.Logger) ReadinessManager {
	return &readinessManager{
		client:  client,
		state:   make(map[schema.GroupVersionKind]map[k8stypes.NamespacedName]struct{}),
		started: make(chan struct{}),
		done:    make(chan struct{}),
		isReady: atomic.Bool{},
		log:     log.WithName("readiness"),
	}
}

// RegisterGVK registers one or more GVKConfig objects for readiness tracking.
func (r *readinessManager) RegisterGVK(configs ...GVKConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.configs = append(r.configs, configs...)
}

// Start initializes the readiness state from the Kubernetes API.
// Should be called only after informer cache has synced.
func (r *readinessManager) Start(ctx context.Context) error {
	var err error
	r.startOnce.Do(func() {
		for _, cfg := range r.configs {
			for _, gvk := range cfg.GVKs {
				uList := &unstructured.UnstructuredList{}
				uList.SetGroupVersionKind(gvk)
				if listErr := r.client.List(ctx, uList); listErr != nil {
					err = fmt.Errorf("list %s failed: %w", gvk.String(), listErr)
					return
				}
				var expected []k8stypes.NamespacedName
				for _, item := range uList.Items {
					if cfg.Filter != nil && !cfg.Filter.Match(&item) {
						continue
					}
					expected = append(expected, k8stypes.NamespacedName{
						Namespace: item.GetNamespace(),
						Name:      item.GetName(),
					})
				}
				if len(expected) > 0 {
					r.log.Info("registering readiness state", "gvk", gvk, "registered_count", len(expected))
					r.log.V(1).Info("registered resources for readiness", "gvk", gvk, "resources", expected)
					r.registerState(gvk, expected)
				}
			}
		}
		close(r.started)
		if len(r.state) == 0 && !r.isReady.Load() {
			r.isReady.Store(true)
			close(r.done)
		}
		r.log.Info("readiness manager started")
	})
	return err
}

func (r *readinessManager) registerState(gvk schema.GroupVersionKind, list []k8stypes.NamespacedName) {
	if _, ok := r.state[gvk]; !ok {
		r.state[gvk] = make(map[k8stypes.NamespacedName]struct{})
	}
	for _, name := range list {
		r.state[gvk][name] = struct{}{}
	}
}

// Done marks the resource as ready by removing it from the pending state.
func (r *readinessManager) Done(obj client.Object, nn k8stypes.NamespacedName) {
	if r.IsReady() {
		return
	}
	<-r.started

	r.mu.Lock()
	defer r.mu.Unlock()
	gvk := types.GvkOf(obj)
	r.log.Info("marking resource as done", "gvk", gvk, "name", nn, "state_count", len(r.state[gvk]))
	if _, ok := r.state[gvk]; !ok {
		return
	}
	delete(r.state[gvk], nn)
	if len(r.state[gvk]) == 0 {
		delete(r.state, gvk)
	}
	if len(r.state) == 0 && !r.isReady.Load() {
		r.isReady.Store(true)
		close(r.done)
	}
}

func (r *readinessManager) IsReady() bool {
	return r.isReady.Load()
}

// WaitReady blocks until readiness is achieved, a timeout occurs, or context is cancelled.
func (r *readinessManager) WaitReady(ctx context.Context, timeout time.Duration) bool {
	if r.IsReady() {
		return true
	}

	select {
	case <-r.started:
	case <-ctx.Done():
		return false
	}

	select {
	case <-ctx.Done():
		return false
	case <-time.After(timeout):
		return false
	case <-r.done:
		return true
	}
}
