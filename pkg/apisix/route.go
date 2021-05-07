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
	"bytes"
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
		zap.String("cluster", "default"),
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

	// TODO Add mutex here to avoid dog-pile effection.
	url := r.url + "/" + rid
	resp, err := r.cluster.getResource(ctx, url)
	if err != nil {
		if err == cache.ErrNotFound {
			log.Warnw("route not found",
				zap.String("name", name),
				zap.String("url", url),
				zap.String("cluster", "default"),
			)
		} else {
			log.Errorw("failed to get route from APISIX",
				zap.String("name", name),
				zap.String("url", url),
				zap.String("cluster", "default"),
				zap.Error(err),
			)
		}
		return nil, err
	}

	route, err = resp.Item.route()
	if err != nil {
		log.Errorw("failed to convert route item",
			zap.String("url", r.url),
			zap.String("route_key", resp.Item.Key),
			zap.String("route_value", string(resp.Item.Value)),
			zap.Error(err),
		)
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
		zap.String("cluster", "default"),
		zap.String("url", r.url),
	)
	routeItems, err := r.cluster.listResource(ctx, r.url)
	if err != nil {
		log.Errorf("failed to list routes: %s", err)
		return nil, err
	}

	var items []*v1.Route
	for i, item := range routeItems.Node.Items {
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

func (r *routeClient) Create(ctx context.Context, obj *v1.Route) (*v1.Route, error) {
	log.Debugw("try to create route",
		zap.Strings("hosts", obj.Hosts),
		zap.String("name", obj.Name),
		zap.String("cluster", "default"),
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
	resp, err := r.cluster.createResource(ctx, url, bytes.NewReader(data))
	if err != nil {
		log.Errorf("failed to create route: %s", err)
		return nil, err
	}

	route, err := resp.Item.route()
	if err != nil {
		return nil, err
	}
	if err := r.cluster.cache.InsertRoute(route); err != nil {
		log.Errorf("failed to reflect route create to cache: %s", err)
		return nil, err
	}
	return route, nil
}

func (r *routeClient) Delete(ctx context.Context, obj *v1.Route) error {
	log.Debugw("try to delete route",
		zap.String("id", obj.ID),
		zap.String("name", obj.Name),
		zap.String("cluster", "default"),
		zap.String("url", r.url),
	)
	if err := r.cluster.HasSynced(ctx); err != nil {
		return err
	}
	url := r.url + "/" + obj.ID
	if err := r.cluster.deleteResource(ctx, url); err != nil {
		return err
	}
	if err := r.cluster.cache.DeleteRoute(obj); err != nil {
		log.Errorf("failed to reflect route delete to cache: %s", err)
		if err != cache.ErrNotFound {
			return err
		}
	}
	return nil
}

func (r *routeClient) Update(ctx context.Context, obj *v1.Route) (*v1.Route, error) {
	log.Debugw("try to update route",
		zap.String("id", obj.ID),
		zap.String("name", obj.Name),
		zap.String("cluster", "default"),
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
	log.Debugw("updating route", zap.ByteString("body", body), zap.String("url", url))
	resp, err := r.cluster.updateResource(ctx, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	route, err := resp.Item.route()
	if err != nil {
		return nil, err
	}
	if err := r.cluster.cache.InsertRoute(route); err != nil {
		log.Errorf("failed to reflect route update to cache: %s", err)
		return nil, err
	}
	return route, nil
}
