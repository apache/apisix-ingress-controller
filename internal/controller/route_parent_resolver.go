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
	"sync"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/internal/controller/config"
)

type resolvedParentClass struct {
	gatewayUID types.UID
	class      *gatewayv1.GatewayClass
}

type routeParentOwnership struct {
	uid     types.UID
	parents map[client.ObjectKey]resolvedParentClass
}

// routeParentResolver remembers ownership only for parents previously resolved
// for this route. A never-resolved sibling must not prevent deletion cleanup.
// Like the provider's desired configuration, this state is rebuilt on startup.
type routeParentResolver struct {
	mu      sync.Mutex
	parents map[client.ObjectKey]routeParentOwnership
}

func (r *routeParentResolver) Forget(key client.ObjectKey) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.parents, key)
}

func (r *routeParentResolver) Resolve(ctx context.Context, cli client.Client, log logr.Logger, route client.Object, refs []gatewayv1.ParentReference) ([]RouteParentRefContext, error) {
	// controller-runtime never reconciles one route concurrently, so the lock only
	// guards the map, not the parent lookups between the read and the write.
	routeKey := client.ObjectKeyFromObject(route)
	r.mu.Lock()
	previous := r.parents[routeKey].parents
	if r.parents[routeKey].uid != route.GetUID() {
		previous = nil
	}
	r.mu.Unlock()
	next := make(map[client.ObjectKey]resolvedParentClass)
	gateways, err := parseRouteParentRefs(ctx, cli, log, route, refs, func(gw *gatewayv1.Gateway) (*gatewayv1.GatewayClass, error) {
		key := client.ObjectKeyFromObject(gw)
		gc := &gatewayv1.GatewayClass{}
		if err := cli.Get(ctx, client.ObjectKey{Name: string(gw.Spec.GatewayClassName)}, gc); err != nil {
			if client.IgnoreNotFound(err) != nil {
				return nil, err
			}
			known, ok := previous[key]
			if !ok || known.gatewayUID != gw.UID || known.class.Name != string(gw.Spec.GatewayClassName) {
				return nil, nil
			}
			// Keep resolving the surviving Gateway's current listeners without
			// retaining configuration belonging to a different, deleted parent.
			gc = known.class
		}
		if string(gc.Spec.ControllerName) == config.ControllerConfig.ControllerName {
			next[key] = resolvedParentClass{gatewayUID: gw.UID, class: gc}
		}
		return gc, nil
	})
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(next) == 0 {
		delete(r.parents, routeKey)
	} else {
		if r.parents == nil {
			r.parents = make(map[client.ObjectKey]routeParentOwnership)
		}
		r.parents[routeKey] = routeParentOwnership{uid: route.GetUID(), parents: next}
	}
	return gateways, nil
}
