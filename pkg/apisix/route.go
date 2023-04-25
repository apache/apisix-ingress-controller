// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package apisix

import (
	"context"
	"encoding/json"

	"go.uber.org/zap"

	"github.com/apache/apisix-ingress-controller/pkg/apisix/cache"
	"github.com/apache/apisix-ingress-controller/pkg/id"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	v1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

type routeClient struct {
	url     string
	cluster *cluster
}

func newRouteClient(c *cluster) Route {
	return &routeClient{
		url:     c.baseURL + "/routes",
		cluster: c,
	}
}

// Get returns the Route.
// FIXME, currently if caller pass a non-existent resource, the Get always passes
// through cache.
func (r *routeClient) Get(ctx context.Context, name string) (*v1.Route, error) {
	log.Debugw("try to look up route",
		zap.String("name", name),
		zap.String("url", r.url),
		zap.String("cluster", r.cluster.name),
	)
	rid := id.GenID(name)
	route, err := r.cluster.cache.GetRoute(rid)
	if err == nil {
		return route, nil
	}
	if err != cache.ErrNotFound {
		log.Errorw("failed to find route in cache, will try to lookup from APISIX",
			zap.String("name", name),
			zap.Error(err),
		)
	} else {
		log.Debugw("failed to find route in cache, will try to lookup from APISIX",
			zap.String("name", name),
			zap.Error(err),
		)
	}

	// TODO Add mutex here to avoid dog-pile effect
	route, err = r.cluster.GetRoute(ctx, r.url, rid)
	if err != nil {
		return nil, err
	}

	if err := r.cluster.cache.InsertRoute(route); err != nil {
		log.Errorf("failed to reflect route create to cache: %s", err)
		return nil, err
	}
	return route, nil
}

// List is only used in cache warming up. So here just pass through
// to APISIX.
func (r *routeClient) List(ctx context.Context) ([]*v1.Route, error) {
	log.Debugw("try to list routes in APISIX",
		zap.String("cluster", r.cluster.name),
		zap.String("url", r.url),
	)
	routeItems, err := r.cluster.listResource(ctx, r.url, "route")
	if err != nil {
		log.Errorf("failed to list routes: %s", err)
		return nil, err
	}

	var items []*v1.Route
	for i, item := range routeItems {
		route, err := item.route()
		if err != nil {
			log.Errorw("failed to convert route item",
				zap.String("url", r.url),
				zap.String("route_key", item.Key),
				zap.String("route_value", string(item.Value)),
				zap.Error(err),
			)
			return nil, err
		}

		items = append(items, route)
		log.Debugf("list route #%d, body: %s", i, string(item.Value))
	}

	return items, nil
}

func (r *routeClient) Create(ctx context.Context, obj *v1.Route, shouldCompare bool) (*v1.Route, error) {
	if v, skip := skipRequest(r.cluster, shouldCompare, r.url, obj.ID, obj); skip {
		return v, nil
	}

	log.Debugw("try to create route",
		zap.Strings("hosts", obj.Hosts),
		zap.String("name", obj.Name),
		zap.String("cluster", r.cluster.name),
		zap.String("url", r.url),
	)

	if err := r.cluster.HasSynced(ctx); err != nil {
		return nil, err
	}
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	url := r.url + "/" + obj.ID
	log.Debugw("creating route", zap.ByteString("body", data), zap.String("url", url))
	resp, err := r.cluster.createResource(ctx, url, "route", data)
	if err != nil {
		log.Errorf("failed to create route: %s", err)
		return nil, err
	}

	route, err := resp.route()
	if err != nil {
		return nil, err
	}
	if err := r.cluster.cache.InsertRoute(route); err != nil {
		log.Errorf("failed to reflect route create to cache: %s", err)
		return nil, err
	}
	if err := r.cluster.generatedObjCache.InsertRoute(obj); err != nil {
		log.Errorf("failed to cache generated route object: %s", err)
		return nil, err
	}
	return route, nil
}

func (r *routeClient) Delete(ctx context.Context, obj *v1.Route) error {
	if r.cluster.adapter != nil {
		data, err := json.Marshal(obj)
		if err != nil {
			return err
		}
		r.cluster.DeleteResource("routes", obj.ID, data)
		return nil
	}
	log.Debugw("try to delete route",
		zap.String("id", obj.ID),
		zap.String("name", obj.Name),
		zap.String("cluster", r.cluster.name),
		zap.String("url", r.url),
	)
	if err := r.cluster.HasSynced(ctx); err != nil {
		return err
	}
	url := r.url + "/" + obj.ID
	if err := r.cluster.deleteResource(ctx, url, "route"); err != nil {
		return err
	}
	if err := r.cluster.cache.DeleteRoute(obj); err != nil {
		log.Errorf("failed to reflect route delete to cache: %s", err)
		if err != cache.ErrNotFound {
			return err
		}
	}
	if err := r.cluster.generatedObjCache.DeleteRoute(obj); err != nil {
		log.Errorf("failed to reflect route delete to generated cache: %s", err)
		if err != cache.ErrNotFound {
			return err
		}
	}
	return nil
}

func (r *routeClient) Update(ctx context.Context, obj *v1.Route, shouldCompare bool) (*v1.Route, error) {
	if r.cluster.adapter != nil {
		data, err := json.Marshal(obj)
		if err != nil {
			return nil, err
		}
		r.cluster.UpdateResource("routes", obj.ID, data)
		return obj, nil
	}
	if v, skip := skipRequest(r.cluster, shouldCompare, r.url, obj.ID, obj); skip {
		return v, nil
	}

	log.Debugw("try to update route",
		zap.String("id", obj.ID),
		zap.String("name", obj.Name),
		zap.String("cluster", r.cluster.name),
		zap.String("url", r.url),
	)
	if err := r.cluster.HasSynced(ctx); err != nil {
		return nil, err
	}
	body, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	url := r.url + "/" + obj.ID
	resp, err := r.cluster.updateResource(ctx, url, "route", body)
	if err != nil {
		return nil, err
	}
	route, err := resp.route()
	if err != nil {
		return nil, err
	}
	if err := r.cluster.cache.InsertRoute(route); err != nil {
		log.Errorf("failed to reflect route update to cache: %s", err)
		return nil, err
	}
	if err := r.cluster.generatedObjCache.InsertRoute(obj); err != nil {
		log.Errorf("failed to cache generated route object: %s", err)
		return nil, err
	}
	return route, nil
}

type routeMem struct {
	url string

	resource string
	cluster  *cluster
}

func newRouteMem(c *cluster) Route {
	return &routeMem{
		url:      c.baseURL + "/routes",
		resource: "routes",
		cluster:  c,
	}
}

func (r *routeMem) Get(ctx context.Context, name string) (*v1.Route, error) {
	log.Debugw("try to look up route",
		zap.String("name", name),
		zap.String("cluster", r.cluster.name),
	)
	rid := id.GenID(name)
	route, err := r.cluster.cache.GetRoute(rid)
	if err != nil && err != cache.ErrNotFound {
		log.Errorw("failed to find route in cache",
			zap.String("name", name),
			zap.Error(err),
		)
		return nil, err
	}
	return route, nil
}

// List is only used in cache warming up. So here just pass through
// to APISIX.
func (r *routeMem) List(ctx context.Context) ([]*v1.Route, error) {
	log.Debugw("try to list resource in APISIX",
		zap.String("cluster", r.cluster.name),
		zap.String("resource", r.resource),
	)
	routes, err := r.cluster.cache.ListRoutes()
	if err != nil {
		log.Errorf("failed to list %s: %s", r.resource, err)
		return nil, err
	}
	return routes, nil
}

func (r *routeMem) Create(ctx context.Context, obj *v1.Route, shouldCompare bool) (*v1.Route, error) {
	if shouldCompare && CompareResourceEqualFromCluster(r.cluster, obj.ID, obj) {
		return obj, nil
	}
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	r.cluster.CreateResource(r.resource, obj.ID, data)
	if err := r.cluster.cache.InsertRoute(obj); err != nil {
		log.Errorf("failed to reflect route create to cache: %s", err)
		return nil, err
	}
	return obj, nil
}

func (r *routeMem) Delete(ctx context.Context, obj *v1.Route) error {
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	r.cluster.DeleteResource(r.resource, obj.ID, data)
	if err := r.cluster.cache.DeleteRoute(obj); err != nil {
		log.Errorf("failed to reflect route delete to cache: %s", err)
		return nil
	}
	return nil
}

func (r *routeMem) Update(ctx context.Context, obj *v1.Route, shouldCompare bool) (*v1.Route, error) {
	if shouldCompare && CompareResourceEqualFromCluster(r.cluster, obj.ID, obj) {
		return obj, nil
	}
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	r.cluster.UpdateResource(r.resource, obj.ID, data)
	if err := r.cluster.cache.InsertRoute(obj); err != nil {
		log.Errorf("failed to reflect route update to cache: %s", err)
		return nil, err
	}
	return obj, nil
}
