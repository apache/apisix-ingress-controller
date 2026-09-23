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

package controller

import (
	"context"
	"errors"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/apache/apisix-ingress-controller/internal/controller/config"
	"github.com/apache/apisix-ingress-controller/internal/utils"
)

// ErrNamespaceNotWatched is returned for an object whose namespace does not
// match the configured namespace selector.
var ErrNamespaceNotWatched = errors.New("namespace is not watched by the namespace selector")

var namespaceSelector labels.Selector

// SetNamespaceSelector limits the IngressClass scoped resources (Ingress and
// apisix.apache.org/v2 resources) handled by the controller to the namespaces
// matching the namespace_selector entries, see config.ParseNamespaceSelector.
// Without an entry every namespace is watched.
func SetNamespaceSelector(entries []string) error {
	selector, err := config.ParseNamespaceSelector(entries)
	if err != nil {
		return err
	}
	namespaceSelector = selector
	return nil
}

func namespaceSelectorEnabled() bool {
	return namespaceSelector != nil
}

func namespaceLabelsMatch(nsLabels map[string]string) bool {
	return !namespaceSelectorEnabled() || namespaceSelector.Matches(labels.Set(nsLabels))
}

// IsWatchedNamespace reports whether objects in the namespace are handled by
// the controller under the configured namespace selector.
func IsWatchedNamespace(ctx context.Context, c client.Client, namespace string) (bool, error) {
	if !namespaceSelectorEnabled() || namespace == "" {
		return true, nil
	}
	var ns corev1.Namespace
	if err := c.Get(ctx, client.ObjectKey{Name: namespace}, &ns); err != nil {
		if k8serrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return namespaceLabelsMatch(ns.Labels), nil
}

func checkWatchedNamespace(ctx context.Context, c client.Client, obj client.Object) error {
	watched, err := IsWatchedNamespace(ctx, c, obj.GetNamespace())
	if err != nil {
		return err
	}
	if !watched {
		return ErrNamespaceNotWatched
	}
	return nil
}

// namespaceSelectorChangedPredicate passes a Namespace update only when the
// namespace moves into or out of the watched set. A new namespace holds no
// objects yet, and the objects of a deleted namespace are deleted one by one.
func namespaceSelectorChangedPredicate() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc:  func(event.CreateEvent) bool { return false },
		DeleteFunc:  func(event.DeleteEvent) bool { return false },
		GenericFunc: func(event.GenericEvent) bool { return false },
		UpdateFunc: func(e event.UpdateEvent) bool {
			return namespaceLabelsMatch(e.ObjectOld.GetLabels()) != namespaceLabelsMatch(e.ObjectNew.GetLabels())
		},
	}
}

// watchNamespaceSelector requeues every object listed by newList in a namespace
// whose labels start or stop matching the namespace selector, so that the
// objects are synced or retracted accordingly. The event filter of the
// controller must let Namespace events through.
func watchNamespaceSelector(bdr *builder.Builder, c client.Client, log logr.Logger, newList func() client.ObjectList) *builder.Builder {
	if !namespaceSelectorEnabled() {
		return bdr
	}
	return bdr.Watches(&corev1.Namespace{},
		handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
			list := newList()
			if err := c.List(ctx, list, client.InNamespace(obj.GetName())); err != nil {
				log.Error(err, "failed to list objects for namespace", "namespace", obj.GetName())
				return nil
			}
			var requests []reconcile.Request
			if err := meta.EachListItem(list, func(item runtime.Object) error {
				if o, ok := item.(client.Object); ok {
					requests = append(requests, reconcile.Request{NamespacedName: utils.NamespacedName(o)})
				}
				return nil
			}); err != nil {
				log.Error(err, "failed to iterate objects for namespace", "namespace", obj.GetName())
			}
			return requests
		}),
		builder.WithPredicates(namespaceSelectorChangedPredicate()),
	)
}
