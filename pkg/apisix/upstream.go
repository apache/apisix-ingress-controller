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

type upstreamReqBody struct {
	LBType *string        `json:"type"`
	HashOn *string        `json:"hash_on,omitempty"`
	Key    *string        `json:"key,omitempty"`
	Nodes  []upstreamNode `json:"nodes"`
	Desc   *string        `json:"desc"`
}

type upstreamItem struct {
	Nodes  map[string]int64 `json:"nodes"`
	Desc   *string          `json:"desc"`
	LBType *string          `json:"type"`
}

func newUpstreamClient(c *cluster) Upstream {
	return &upstreamClient{
		url:         c.baseURL + "/upstreams",
		cluster:     c,
		clusterName: c.name,
	}
}

func (u *upstreamClient) List(ctx context.Context) ([]*v1.Upstream, error) {
	log.Infow("try to list upstreams in APISIX", zap.String("url", u.url))

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
		log.Infof("list upstream #%d, body: %s", i, string(item.Value))
	}
	return items, nil
}

func (u *upstreamClient) Create(ctx context.Context, obj *v1.Upstream) (*v1.Upstream, error) {
	log.Infow("try to create upstream",
		zap.String("full_name", *obj.FullName),
	)

	nodes := make([]upstreamNode, 0, len(obj.Nodes))
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
	log.Infow("creating upstream", zap.ByteString("body", body), zap.String("url", u.url))

	resp, err := u.cluster.createResource(ctx, u.url, bytes.NewReader(body))
	if err != nil {
		log.Errorf("failed to create upstream: %s", err)
		return nil, err
	}
	var clusterName string
	if obj.Group != nil {
		clusterName = *obj.Group
	}
	return resp.Item.upstream(clusterName)
}

func (u *upstreamClient) Delete(ctx context.Context, obj *v1.Upstream) error {
	log.Infof("delete upstream, id:%s", *obj.ID)
	url := u.url + "/" + *obj.ID
	return u.cluster.deleteResource(ctx, url)
}

func (u *upstreamClient) Update(ctx context.Context, obj *v1.Upstream) (*v1.Upstream, error) {
	log.Infof("update upstream, id:%s", *obj.ID)

	nodes := make([]upstreamNode, 0, len(obj.Nodes))
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
	log.Infow("upating upstream", zap.ByteString("body", body), zap.String("url", url))
	resp, err := u.cluster.updateResource(ctx, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	var clusterName string
	if obj.Group != nil {
		clusterName = *obj.Group
	}
	return resp.Item.upstream(clusterName)
}
