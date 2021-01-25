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
	"errors"

	"go.uber.org/zap"

	"github.com/api7/ingress-controller/pkg/apisix/cache"
	"github.com/api7/ingress-controller/pkg/id"
	"github.com/api7/ingress-controller/pkg/log"
	v1 "github.com/api7/ingress-controller/pkg/types/apisix/v1"
)

type upstreamClient struct {
	clusterName string
	url         string
	cluster     *cluster
}

type upstreamNode struct {
	Host   string `json:"host,omitempty" yaml:"ip,omitempty"`
	Port   int    `json:"port,omitempty" yaml:"port,omitempty"`
	Weight int    `json:"weight,omitempty" yaml:"weight,omitempty"`
}

type upstreamNodes []upstreamNode

// items implements json.Unmarshaler interface.
// lua-cjson doesn't distinguish empty array and table,
// and by default empty array will be encoded as '{}'.
// We have to maintain the compatibility.
func (n *upstreamNodes) UnmarshalJSON(p []byte) error {
	if p[0] == '{' {
		if len(p) != 2 {
			return errors.New("unexpected non-empty object")
		}
		return nil
	}
	var data []upstreamNode
	if err := json.Unmarshal(p, &data); err != nil {
		return err
	}
	*n = data
	return nil
}

type upstreamReqBody struct {
	LBType *string       `json:"type"`
	HashOn *string       `json:"hash_on,omitempty"`
	Key    *string       `json:"key,omitempty"`
	Nodes  upstreamNodes `json:"nodes"`
	Desc   *string       `json:"desc"`
}

type upstreamItem struct {
	Nodes  upstreamNodes `json:"nodes"`
	Desc   *string       `json:"desc"`
	LBType *string       `json:"type"`
}

func newUpstreamClient(c *cluster) Upstream {
	return &upstreamClient{
		url:         c.baseURL + "/upstreams",
		cluster:     c,
		clusterName: c.name,
	}
}

func (u *upstreamClient) Get(ctx context.Context, fullname string) (*v1.Upstream, error) {
	log.Infow("try to look up upstream",
		zap.String("fullname", fullname),
		zap.String("url", u.url),
		zap.String("cluster", u.clusterName),
	)
	ups, err := u.cluster.cache.GetUpstream(fullname)
	if err == nil {
		return ups, nil
	}
	if err != cache.ErrNotFound {
		log.Errorw("failed to find upstream in cache, will try to lookup from APISIX",
			zap.String("fullname", fullname),
			zap.Error(err),
		)
	} else {
		log.Warnw("failed to find upstream in cache, will try to lookup from APISIX",
			zap.String("fullname", fullname),
			zap.Error(err),
		)
	}

	// TODO Add mutex here to avoid dog-pile effection.
	url := u.url + "/" + id.GenID(fullname)
	resp, err := u.cluster.getResource(ctx, url)
	if err != nil {
		if err == cache.ErrNotFound {
			log.Warnw("upstream not found",
				zap.String("fullname", fullname),
				zap.String("url", url),
				zap.String("cluster", u.clusterName),
			)
		} else {
			log.Errorw("failed to get upstream from APISIX",
				zap.String("fullname", fullname),
				zap.String("url", url),
				zap.String("cluster", u.clusterName),
				zap.Error(err),
			)
		}
		return nil, err
	}

	ups, err = resp.Item.upstream(u.clusterName)
	if err != nil {
		log.Errorw("failed to convert upstream item",
			zap.String("url", u.url),
			zap.String("ssl_key", resp.Item.Key),
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
	log.Infow("try to list upstreams in APISIX",
		zap.String("url", u.url),
		zap.String("cluster", u.clusterName),
	)

	upsItems, err := u.cluster.listResource(ctx, u.url)
	if err != nil {
		log.Errorf("failed to list upstreams: %s", err)
		return nil, err
	}

	var items []*v1.Upstream
	for i, item := range upsItems.Node.Items {
		ups, err := item.upstream(u.clusterName)
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
	log.Infow("try to create upstream",
		zap.String("fullname", *obj.FullName),
		zap.String("url", u.url),
		zap.String("cluster", u.clusterName),
	)

	if err := u.cluster.HasSynced(ctx); err != nil {
		return nil, err
	}

	nodes := make(upstreamNodes, 0, len(obj.Nodes))
	for _, node := range obj.Nodes {
		nodes = append(nodes, upstreamNode{
			Host:   *node.IP,
			Port:   *node.Port,
			Weight: *node.Weight,
		})
	}
	body, err := json.Marshal(upstreamReqBody{
		LBType: obj.Type,
		HashOn: obj.HashOn,
		Key:    obj.Key,
		Nodes:  nodes,
		Desc:   obj.Name,
	})
	if err != nil {
		return nil, err
	}
	url := u.url + "/" + *obj.ID
	log.Infow("creating upstream", zap.ByteString("body", body), zap.String("url", url))

	resp, err := u.cluster.createResource(ctx, url, bytes.NewReader(body))
	if err != nil {
		log.Errorf("failed to create upstream: %s", err)
		return nil, err
	}
	var clusterName string
	if obj.Group != nil {
		clusterName = *obj.Group
	}
	ups, err := resp.Item.upstream(clusterName)
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
	log.Infow("try to delete upstream",
		zap.String("id", *obj.ID),
		zap.String("fullname", *obj.FullName),
		zap.String("cluster", u.clusterName),
		zap.String("url", u.url),
	)

	if err := u.cluster.HasSynced(ctx); err != nil {
		return err
	}
	url := u.url + "/" + *obj.ID
	if err := u.cluster.deleteResource(ctx, url); err != nil {
		return err
	}
	if err := u.cluster.cache.DeleteUpstream(obj); err != nil {
		log.Errorf("failed to reflect upstream delete to cache: %s", err.Error())
		return err
	}
	return nil
}

func (u *upstreamClient) Update(ctx context.Context, obj *v1.Upstream) (*v1.Upstream, error) {
	log.Infow("try to update upstream",
		zap.String("id", *obj.ID),
		zap.String("fullname", *obj.FullName),
		zap.String("cluster", u.clusterName),
		zap.String("url", u.url),
	)

	if err := u.cluster.HasSynced(ctx); err != nil {
		return nil, err
	}

	nodes := make(upstreamNodes, 0, len(obj.Nodes))
	for _, node := range obj.Nodes {
		nodes = append(nodes, upstreamNode{
			Host:   *node.IP,
			Port:   *node.Port,
			Weight: *node.Weight,
		})
	}
	body, err := json.Marshal(upstreamReqBody{
		LBType: obj.Type,
		HashOn: obj.HashOn,
		Key:    obj.Key,
		Nodes:  nodes,
		Desc:   obj.Name,
	})
	if err != nil {
		return nil, err
	}

	url := u.url + "/" + *obj.ID
	log.Infow("updating upstream", zap.ByteString("body", body), zap.String("url", url))
	resp, err := u.cluster.updateResource(ctx, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	var clusterName string
	if obj.Group != nil {
		clusterName = *obj.Group
	}
	ups, err := resp.Item.upstream(clusterName)
	if err != nil {
		return nil, err
	}
	if err := u.cluster.cache.InsertUpstream(ups); err != nil {
		log.Errorf("failed to reflect upstraem update to cache: %s", err)
		return nil, err
	}
	return ups, err
}
