// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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

type streamRouteClient struct {
	url     string
	cluster *cluster
}

func newStreamRouteClient(c *cluster) StreamRoute {
	url := c.baseURL + "/stream_routes"
	_, err := c.listResource(context.Background(), url, "streamRoute")
	if err == ErrFunctionDisabled {
		log.Infow("resource stream_routes is disabled")
		return &noopClient{}
	}
	return &streamRouteClient{
		url:     url,
		cluster: c,
	}
}

// Get returns the StreamRoute.
// FIXME, currently if caller pass a non-existent resource, the Get always passes
// through cache.
func (r *streamRouteClient) Get(ctx context.Context, name string) (*v1.StreamRoute, error) {
	log.Debugw("try to look up stream_route",
		zap.String("name", name),
		zap.String("url", r.url),
		zap.String("cluster", r.cluster.name),
	)
	rid := id.GenID(name)
	streamRoute, err := r.cluster.cache.GetStreamRoute(rid)
	if err == nil {
		return streamRoute, nil
	}
	if err != cache.ErrNotFound {
		log.Errorw("failed to find stream_route in cache, will try to lookup from APISIX",
			zap.String("name", name),
			zap.Error(err),
		)
	} else {
		log.Debugw("failed to find stream_route in cache, will try to lookup from APISIX",
			zap.String("name", name),
			zap.Error(err),
		)
	}

	// TODO Add mutex here to avoid dog-pile effect.
	streamRoute, err = r.cluster.GetStreamRoute(ctx, r.url, rid)
	if err != nil {
		return nil, err
	}

	if err := r.cluster.cache.InsertStreamRoute(streamRoute); err != nil {
		log.Errorf("failed to reflect route create to cache: %s", err)
		return nil, err
	}
	return streamRoute, nil
}

// List is only used in cache warming up. So here just pass through
// to APISIX.
func (r *streamRouteClient) List(ctx context.Context) ([]*v1.StreamRoute, error) {
	log.Debugw("try to list stream_routes in APISIX",
		zap.String("cluster", r.cluster.name),
		zap.String("url", r.url),
	)
	streamRouteItems, err := r.cluster.listResource(ctx, r.url, "streamRoute")
	if err != nil {
		log.Errorf("failed to list stream_routes: %s", err)
		return nil, err
	}

	var items []*v1.StreamRoute
	for i, item := range streamRouteItems {
		streamRoute, err := item.streamRoute()
		if err != nil {
			log.Errorw("failed to convert stream_route item",
				zap.String("url", r.url),
				zap.String("stream_route_key", item.Key),
				zap.String("stream_route_value", string(item.Value)),
				zap.Error(err),
			)
			return nil, err
		}

		items = append(items, streamRoute)
		log.Debugf("list stream_route #%d, body: %s", i, string(item.Value))
	}
	return items, nil
}

func (r *streamRouteClient) Create(ctx context.Context, obj *v1.StreamRoute, shouldCompare bool) (*v1.StreamRoute, error) {
	if v, skip := skipRequest(r.cluster, shouldCompare, r.url, obj.ID, obj); skip {
		return v, nil
	}

	log.Debugw("try to create stream_route",
		zap.String("id", obj.ID),
		zap.Int32("server_port", obj.ServerPort),
		zap.String("cluster", r.cluster.name),
		zap.String("url", r.url),
		zap.String("sni", obj.SNI),
	)

	if err := r.cluster.HasSynced(ctx); err != nil {
		return nil, err
	}
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	url := r.url + "/" + obj.ID
	log.Debugw("creating stream_route", zap.ByteString("body", data), zap.String("url", url))
	resp, err := r.cluster.createResource(ctx, url, "streamRoute", data)
	if err != nil {
		log.Errorf("failed to create stream_route: %s", err)
		return nil, err
	}

	streamRoute, err := resp.streamRoute()
	if err != nil {
		return nil, err
	}
	if err := r.cluster.cache.InsertStreamRoute(streamRoute); err != nil {
		log.Errorf("failed to reflect stream_route create to cache: %s", err)
		return nil, err
	}
	if err := r.cluster.generatedObjCache.InsertStreamRoute(obj); err != nil {
		log.Errorf("failed to reflect generated stream_route create to cache: %s", err)
		return nil, err
	}
	return streamRoute, nil
}

func (r *streamRouteClient) Delete(ctx context.Context, obj *v1.StreamRoute) error {
	log.Debugw("try to delete stream_route",
		zap.String("id", obj.ID),
		zap.String("cluster", r.cluster.name),
		zap.String("url", r.url),
	)
	if err := r.cluster.HasSynced(ctx); err != nil {
		return err
	}
	url := r.url + "/" + obj.ID
	if err := r.cluster.deleteResource(ctx, url, "streamRoute"); err != nil {
		return err
	}
	if err := r.cluster.cache.DeleteStreamRoute(obj); err != nil {
		log.Errorf("failed to reflect stream_route delete to cache: %s", err)
		if err != cache.ErrNotFound {
			return err
		}
	}
	if err := r.cluster.generatedObjCache.DeleteStreamRoute(obj); err != nil {
		log.Errorf("failed to reflect stream_route delete to generated cache: %s", err)
		if err != cache.ErrNotFound {
			return err
		}
	}
	return nil
}

func (r *streamRouteClient) Update(ctx context.Context, obj *v1.StreamRoute, shouldCompare bool) (*v1.StreamRoute, error) {
	if v, skip := skipRequest(r.cluster, shouldCompare, r.url, obj.ID, obj); skip {
		return v, nil
	}

	log.Debugw("try to update stream_route",
		zap.String("id", obj.ID),
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
	resp, err := r.cluster.updateResource(ctx, url, "streamRoute", body)
	if err != nil {
		return nil, err
	}
	streamRoute, err := resp.streamRoute()
	if err != nil {
		return nil, err
	}
	if err := r.cluster.cache.InsertStreamRoute(streamRoute); err != nil {
		log.Errorf("failed to reflect stream_route update to cache: %s", err)
		return nil, err
	}
	if err := r.cluster.generatedObjCache.InsertStreamRoute(obj); err != nil {
		log.Errorf("failed to reflect generated stream_route update to cache: %s", err)
		return nil, err
	}
	return streamRoute, nil
}

type streamRouteMem struct {
	url string

	resource string
	cluster  *cluster
}

func newStreamRouteMem(c *cluster) StreamRoute {
	return &streamRouteMem{
		url:      c.baseURL + "/stream_routes",
		resource: "stream_routes",
		cluster:  c,
	}
}

func (r *streamRouteMem) Get(ctx context.Context, name string) (*v1.StreamRoute, error) {
	log.Debugw("try to look up route",
		zap.String("name", name),
		zap.String("cluster", r.cluster.name),
	)
	rid := id.GenID(name)
	route, err := r.cluster.cache.GetStreamRoute(rid)
	if err != nil {
		log.Errorw("failed to find route in cache, will try to lookup from APISIX",
			zap.String("name", name),
			zap.Error(err),
		)
		return nil, err
	}
	return route, nil
}

// List is only used in cache warming up. So here just pass through
// to APISIX.
func (r *streamRouteMem) List(ctx context.Context) ([]*v1.StreamRoute, error) {
	log.Debugw("try to list resource in APISIX",
		zap.String("cluster", r.cluster.name),
		zap.String("resource", r.resource),
	)
	streamRoutes, err := r.cluster.cache.ListStreamRoutes()
	if err != nil {
		log.Errorf("failed to list %s: %s", r.resource, err)
		return nil, err
	}
	return streamRoutes, nil
}

func (r *streamRouteMem) Create(ctx context.Context, obj *v1.StreamRoute, shouldCompare bool) (*v1.StreamRoute, error) {
	if shouldCompare && CompareResourceEqualFromCluster(r.cluster, obj.ID, obj) {
		return obj, nil
	}
	if ok, err := r.cluster.validator.ValidateSteamPluginSchema(obj.Plugins); !ok {
		return nil, err
	}
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	r.cluster.CreateResource(r.resource, obj.ID, data)
	if err := r.cluster.cache.InsertStreamRoute(obj); err != nil {
		log.Errorf("failed to reflect stream_route create to cache: %s", err)
		return nil, err
	}
	return obj, nil
}

func (r *streamRouteMem) Delete(ctx context.Context, obj *v1.StreamRoute) error {
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	r.cluster.DeleteResource(r.resource, obj.ID, data)
	if err := r.cluster.cache.DeleteStreamRoute(obj); err != nil {
		log.Errorf("failed to reflect stream_route delete to cache: %s", err)
		return err
	}
	return nil
}

func (r *streamRouteMem) Update(ctx context.Context, obj *v1.StreamRoute, shouldCompare bool) (*v1.StreamRoute, error) {
	if shouldCompare && CompareResourceEqualFromCluster(r.cluster, obj.ID, obj) {
		return obj, nil
	}
	if ok, err := r.cluster.validator.ValidateSteamPluginSchema(obj.Plugins); !ok {
		return nil, err
	}
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	r.cluster.UpdateResource(r.resource, obj.ID, data)
	if err := r.cluster.cache.InsertStreamRoute(obj); err != nil {
		log.Errorf("failed to reflect stream_route update to cache: %s", err)
		return nil, err
	}
	return obj, nil
}
