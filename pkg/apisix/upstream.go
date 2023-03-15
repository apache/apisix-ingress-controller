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

	// TODO Add mutex here to avoid dog-pile effection.
	url := u.url + "/" + uid
	resp, err := u.cluster.getResource(ctx, url, "upstream")
	if err != nil {
		if err == cache.ErrNotFound {
			log.Warnw("upstream not found",
				zap.String("name", name),
				zap.String("url", url),
				zap.String("cluster", u.cluster.name),
			)
		} else {
			log.Errorw("failed to get upstream from APISIX",
				zap.String("name", name),
				zap.String("url", url),
				zap.String("cluster", u.cluster.name),
				zap.Error(err),
			)
		}
		return nil, err
	}

	ups, err = resp.upstream()
	if err != nil {
		log.Errorw("failed to convert upstream item",
			zap.String("url", u.url),
			zap.String("ssl_key", resp.Key),
			zap.Error(err),
		)
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
	u.cluster.metricsCollector.IncrAPISIXRequest("upstream")
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

func (u *upstreamClient) Create(ctx context.Context, obj *v1.Upstream) (*v1.Upstream, error) {
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
	u.cluster.metricsCollector.IncrAPISIXRequest("upstream")
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
	if err := u.cluster.cache.DeleteUpstream(obj); err != nil {
		log.Errorf("failed to reflect upstream delete to cache: %s", err.Error())
		if err != cache.ErrNotFound {
			return err
		}
	}
	url := u.url + "/" + obj.ID
	if err := u.cluster.deleteResource(ctx, url, "upstream"); err != nil {
		u.cluster.metricsCollector.IncrAPISIXRequest("upstream")
		return err
	}
	u.cluster.metricsCollector.IncrAPISIXRequest("upstream")
	return nil
}

func (u *upstreamClient) Update(ctx context.Context, obj *v1.Upstream) (*v1.Upstream, error) {
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
	u.cluster.metricsCollector.IncrAPISIXRequest("upstream")
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
	return ups, err
}
