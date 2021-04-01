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

type routeReqBody struct {
	Desc       string               `json:"desc,omitempty"`
	Name       string               `json:"name,omitempty"`
	URI        string               `json:"uri,omitempty"`
	Uris       []string             `json:"uris,omitempty"`
	Vars       [][]v1.StringOrSlice `json:"vars,omitempty"`
	Host       string               `json:"host,omitempty"`
	ServiceId  string               `json:"service_id,omitempty"`
	UpstreamId string               `json:"upstream_id,omitempty"`
	Plugins    v1.Plugins           `json:"plugins,omitempty"`
}

type routeClient struct {
	clusterName string
	url         string
	cluster     *cluster
}

func newRouteClient(c *cluster) Route {
	return &routeClient{
		clusterName: c.name,
		url:         c.baseURL + "/routes",
		cluster:     c,
	}
}

// FIXME, currently if caller pass a non-existent resource, the Get always passes
// through cache.
func (r *routeClient) Get(ctx context.Context, fullname string) (*v1.Route, error) {
	log.Debugw("try to look up route",
		zap.String("fullname", fullname),
		zap.String("url", r.url),
		zap.String("cluster", r.clusterName),
	)
	route, err := r.cluster.cache.GetRoute(fullname)
	if err == nil {
		return route, nil
	}
	if err != cache.ErrNotFound {
		log.Errorw("failed to find route in cache, will try to lookup from APISIX",
			zap.String("fullname", fullname),
			zap.Error(err),
		)
	} else {
		log.Debugw("failed to find route in cache, will try to lookup from APISIX",
			zap.String("fullname", fullname),
			zap.Error(err),
		)
	}

	// TODO Add mutex here to avoid dog-pile effection.
	url := r.url + "/" + id.GenID(fullname)
	resp, err := r.cluster.getResource(ctx, url)
	if err != nil {
		if err == cache.ErrNotFound {
			log.Warnw("route not found",
				zap.String("fullname", fullname),
				zap.String("url", url),
				zap.String("cluster", r.clusterName),
			)
		} else {
			log.Errorw("failed to get route from APISIX",
				zap.String("fullname", fullname),
				zap.String("url", url),
				zap.String("cluster", r.clusterName),
				zap.Error(err),
			)
		}
		return nil, err
	}

	route, err = resp.Item.route(r.clusterName)
	if err != nil {
		log.Errorw("failed to convert route item",
			zap.String("url", r.url),
			zap.String("route_key", resp.Item.Key),
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
		zap.String("cluster", r.clusterName),
		zap.String("url", r.url),
	)
	routeItems, err := r.cluster.listResource(ctx, r.url)
	if err != nil {
		log.Errorf("failed to list routes: %s", err)
		return nil, err
	}

	var items []*v1.Route
	for i, item := range routeItems.Node.Items {
		route, err := item.route(r.clusterName)
		if err != nil {
			log.Errorw("failed to convert route item",
				zap.String("url", r.url),
				zap.String("route_key", item.Key),
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
		zap.String("host", obj.Host),
		zap.String("fullname", obj.FullName),
		zap.String("cluster", r.clusterName),
		zap.String("url", r.url),
	)

	if err := r.cluster.HasSynced(ctx); err != nil {
		return nil, err
	}
	data, err := json.Marshal(routeReqBody{
		Desc:       obj.Name,
		Name:       obj.Name,
		URI:        obj.Path,
		Host:       obj.Host,
		ServiceId:  obj.ServiceId,
		UpstreamId: obj.UpstreamId,
		Uris:       obj.Uris,
		Plugins:    obj.Plugins,
		Vars:       obj.Vars,
	})
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

	var clusterName string
	if obj.Group != "" {
		clusterName = obj.Group
	}
	route, err := resp.Item.route(clusterName)
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
		zap.String("fullname", obj.FullName),
		zap.String("cluster", r.clusterName),
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
		return err
	}
	return nil
}

func (r *routeClient) Update(ctx context.Context, obj *v1.Route) (*v1.Route, error) {
	log.Debugw("try to update route",
		zap.String("id", obj.ID),
		zap.String("fullname", obj.FullName),
		zap.String("cluster", r.clusterName),
		zap.String("url", r.url),
	)
	if err := r.cluster.HasSynced(ctx); err != nil {
		return nil, err
	}
	body, err := json.Marshal(routeReqBody{
		Desc:      obj.Name,
		Name:      obj.Name,
		Host:      obj.Host,
		URI:       obj.Path,
		ServiceId: obj.ServiceId,
		Plugins:   obj.Plugins,
		Vars:      obj.Vars,
	})
	if err != nil {
		return nil, err
	}
	url := r.url + "/" + obj.ID
	log.Debugw("updating route", zap.ByteString("body", body), zap.String("url", r.url))
	resp, err := r.cluster.updateResource(ctx, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	var clusterName string
	if obj.Group != "" {
		clusterName = obj.Group
	}
	route, err := resp.Item.route(clusterName)
	if err != nil {
		return nil, err
	}
	if err := r.cluster.cache.InsertRoute(route); err != nil {
		log.Errorf("failed to reflect route update to cache: %s", err)
		return nil, err
	}
	return route, nil
}
