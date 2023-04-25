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
	"fmt"

	"go.uber.org/zap"

	"github.com/apache/apisix-ingress-controller/pkg/apisix/cache"
	"github.com/apache/apisix-ingress-controller/pkg/id"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	v1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

type upstreamClient struct {
	url     string
	cluster *cluster
}

func newUpstreamClient(c *cluster) Upstream {
	return &upstreamClient{
		url:     c.baseURL + "/upstreams",
		cluster: c,
	}
}

func (u *upstreamClient) Get(ctx context.Context, name string) (*v1.Upstream, error) {
	log.Debugw("try to look up upstream",
		zap.String("name", name),
		zap.String("url", u.url),
		zap.String("cluster", u.cluster.name),
	)
	uid := id.GenID(name)
	ups, err := u.cluster.cache.GetUpstream(uid)
	if err == nil {
		return ups, nil
	}
	if err != cache.ErrNotFound {
		log.Errorw("failed to find upstream in cache, will try to lookup from APISIX",
			zap.String("name", name),
			zap.Error(err),
		)
	} else {
		log.Debugw("failed to find upstream in cache, will try to lookup from APISIX",
			zap.String("name", name),
			zap.Error(err),
		)
	}

	// TODO Add mutex here to avoid dog-pile effect
	ups, err = u.cluster.GetUpstream(ctx, u.url, uid)
	if err != nil {
		return nil, err
	}

	if err := u.cluster.cache.InsertUpstream(ups); err != nil {
		log.Errorf("failed to reflect upstream create to cache: %s", err)
		return nil, err
	}
	return ups, nil
}

// List is only used in cache warming up. So here just pass through
// to APISIX.
func (u *upstreamClient) List(ctx context.Context) ([]*v1.Upstream, error) {
	log.Debugw("try to list upstreams in APISIX",
		zap.String("url", u.url),
		zap.String("cluster", u.cluster.name),
	)

	upsItems, err := u.cluster.listResource(ctx, u.url, "upstream")
	if err != nil {
		log.Errorf("failed to list upstreams: %s", err)
		return nil, err
	}

	var items []*v1.Upstream
	for i, item := range upsItems {
		ups, err := item.upstream()
		if err != nil {
			log.Errorw("failed to convert upstream item",
				zap.String("url", u.url),
				zap.String("upstream_key", item.Key),
				zap.Error(err),
			)
			return nil, err
		}
		items = append(items, ups)
		log.Debugf("list upstream #%d, body: %s", i, string(item.Value))
	}
	return items, nil
}

func (u *upstreamClient) Create(ctx context.Context, obj *v1.Upstream, shouldCompare bool) (*v1.Upstream, error) {
	if v, skip := skipRequest(u.cluster, shouldCompare, u.url, obj.ID, obj); skip {
		return v, nil
	}

	log.Debugw("try to create upstream",
		zap.String("name", obj.Name),
		zap.String("url", u.url),
		zap.String("cluster", u.cluster.name),
	)

	if err := u.cluster.upstreamServiceRelation.Create(ctx, obj.Name); err != nil {
		log.Errorf("failed to reflect upstreamService create to cache: %s", err)
	}
	if err := u.cluster.HasSynced(ctx); err != nil {
		return nil, err
	}

	body, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	url := u.url + "/" + obj.ID
	log.Debugw("creating upstream", zap.ByteString("body", body), zap.String("url", url))

	resp, err := u.cluster.createResource(ctx, url, "upstream", body)
	if err != nil {
		log.Errorf("failed to create upstream: %s", err)
		return nil, err
	}
	ups, err := resp.upstream()
	if err != nil {
		return nil, err
	}
	if err := u.cluster.cache.InsertUpstream(ups); err != nil {
		log.Errorf("failed to reflect upstream create to cache: %s", err)
		return nil, err
	}
	if err := u.cluster.generatedObjCache.InsertUpstream(obj); err != nil {
		log.Errorf("failed to reflect generated upstream create to cache: %s", err)
		return nil, err
	}
	return ups, err
}

func (u *upstreamClient) Delete(ctx context.Context, obj *v1.Upstream) error {
	log.Debugw("try to delete upstream",
		zap.String("id", obj.ID),
		zap.String("name", obj.Name),
		zap.String("cluster", u.cluster.name),
		zap.String("url", u.url),
	)

	if err := u.cluster.HasSynced(ctx); err != nil {
		return err
	}
	url := u.url + "/" + obj.ID
	if err := u.cluster.deleteResource(ctx, url, "upstream"); err != nil {
		return err
	}
	if err := u.cluster.cache.DeleteUpstream(obj); err != nil {
		log.Errorf("failed to reflect upstream delete to cache: %s", err.Error())
		if err != cache.ErrNotFound {
			return err
		}
	}
	if err := u.cluster.generatedObjCache.DeleteUpstream(obj); err != nil {
		log.Errorf("failed to reflect upstream delete to generated cache: %s", err.Error())
		if err != cache.ErrNotFound {
			return err
		}
	}
	return nil
}

func (u *upstreamClient) Update(ctx context.Context, obj *v1.Upstream, shouldCompare bool) (*v1.Upstream, error) {
	if v, skip := skipRequest(u.cluster, shouldCompare, u.url, obj.ID, obj); skip {
		return v, nil
	}

	log.Debugw("try to update upstream",
		zap.String("id", obj.ID),
		zap.String("name", obj.Name),
		zap.String("cluster", u.cluster.name),
		zap.String("url", u.url),
	)

	if err := u.cluster.upstreamServiceRelation.Create(ctx, obj.Name); err != nil {
		log.Errorf("failed to reflect upstreamService create to cache: %s", err)
	}
	if err := u.cluster.HasSynced(ctx); err != nil {
		return nil, err
	}

	body, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	url := u.url + "/" + obj.ID
	resp, err := u.cluster.updateResource(ctx, url, "upstream", body)
	if err != nil {
		return nil, err
	}
	ups, err := resp.upstream()
	if err != nil {
		return nil, err
	}
	if err := u.cluster.cache.InsertUpstream(ups); err != nil {
		log.Errorf("failed to reflect upstream update to cache: %s", err)
		return nil, err
	}
	if err := u.cluster.generatedObjCache.InsertUpstream(obj); err != nil {
		log.Errorf("failed to reflect generated upstream update to cache: %s", err)
		return nil, err
	}
	return ups, err
}

type upstreamMem struct {
	url string

	resource string
	cluster  *cluster
}

func newUpstreamMem(c *cluster) Upstream {
	return &upstreamMem{
		url:      c.baseURL + "/upstreams",
		resource: "upstreams",
		cluster:  c,
	}
}

func (r *upstreamMem) Get(ctx context.Context, name string) (*v1.Upstream, error) {
	log.Debugw("try to look up upstream",
		zap.String("name", name),
		zap.String("cluster", r.cluster.name),
	)
	rid := id.GenID(name)
	upstream, err := r.cluster.cache.GetUpstream(rid)
	if err != nil && err != cache.ErrNotFound {
		log.Errorw("failed to find upstream in cache",
			zap.String("name", name),
			zap.Error(err),
		)
		return nil, err
	}
	return upstream, nil
}

// List is only used in cache warming up. So here just pass through
// to APISIX.
func (r *upstreamMem) List(ctx context.Context) ([]*v1.Upstream, error) {
	log.Debugw("try to list resource in APISIX",
		zap.String("cluster", r.cluster.name),
		zap.String("resource", r.resource),
	)
	upstreams, err := r.cluster.cache.ListUpstreams()
	if err != nil {
		log.Errorf("failed to list %s: %s", r.resource, err)
		return nil, err
	}
	return upstreams, err
}

func (u *upstreamMem) Create(ctx context.Context, obj *v1.Upstream, shouldCompare bool) (*v1.Upstream, error) {
	if shouldCompare && CompareResourceEqualFromCluster(u.cluster, obj.ID, obj) {
		return obj, nil
	}
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	if err := u.cluster.upstreamServiceRelation.Create(ctx, obj.Name); err != nil {
		log.Errorf("failed to reflect upstreamService create to cache: %s", err)
	}
	u.cluster.CreateResource(u.resource, obj.ID, data)
	if err := u.cluster.cache.InsertUpstream(obj); err != nil {
		log.Errorf("failed to reflect upstream create to cache: %s", err)
		return nil, err
	}
	return obj, nil
}

func (u *upstreamMem) Delete(ctx context.Context, obj *v1.Upstream) error {
	if ok, err := u.deleteCheck(ctx, obj); !ok {
		log.Debug("failed to delete upstream", zap.Error(err))
		return cache.ErrStillInUse
	}
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	u.cluster.DeleteResource(u.resource, obj.ID, data)
	if err := u.cluster.cache.DeleteUpstream(obj); err != nil {
		log.Errorf("failed to reflect upstream delete to cache: %s", err)
		return err
	}
	return nil
}

func (u *upstreamMem) Update(ctx context.Context, obj *v1.Upstream, shouldCompare bool) (*v1.Upstream, error) {
	if shouldCompare && CompareResourceEqualFromCluster(u.cluster, obj.ID, obj) {
		return obj, nil
	}
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	if err := u.cluster.upstreamServiceRelation.Create(ctx, obj.Name); err != nil {
		log.Errorf("failed to reflect upstreamService update to cache: %s", err)
	}
	u.cluster.UpdateResource(u.resource, obj.ID, data)
	if err := u.cluster.cache.InsertUpstream(obj); err != nil {
		log.Errorf("failed to reflect upstream update to cache: %s", err)
		return nil, err
	}
	return obj, nil
}

// TODO: Maintain a reference count for each object without having to poll each time
func (u *upstreamMem) deleteCheck(ctx context.Context, obj *v1.Upstream) (bool, error) {
	routes, _ := u.cluster.route.List(ctx)
	sroutes, _ := u.cluster.cache.ListStreamRoutes()
	if routes == nil && sroutes == nil {
		return true, nil
	}
	for _, route := range routes {
		if route.UpstreamId == obj.ID {
			return false, fmt.Errorf("can not delete this upstream, route.id=%s is still using it now", route.ID)
		}
	}
	for _, sroute := range sroutes {
		if sroute.UpstreamId == obj.ID {
			return false, fmt.Errorf("can not delete this upstream, stream_route.id=%s is still using it now", sroute.ID)
		}
	}
	return true, nil
}
