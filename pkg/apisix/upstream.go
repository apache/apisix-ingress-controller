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
	"net"
	"strconv"

	"go.uber.org/zap"

	"github.com/api7/ingress-controller/pkg/log"
	v1 "github.com/api7/ingress-controller/pkg/types/apisix/v1"
)

type upstreamClient struct {
	url  string
	stub *stub
}

type upstreamReqBody struct {
	LBType *string          `json:"type"`
	HashOn *string          `json:"hash_on,omitempty"`
	Key    *string          `json:"key,omitempty"`
	Nodes  map[string]int64 `json:"nodes"`
	Desc   *string          `json:"desc"`
}

type upstreamItem struct {
	Nodes  map[string]int64 `json:"nodes"`
	Desc   *string          `json:"desc"`
	LBType *string          `json:"type"`
}

func newUpstreamClient(stub *stub) Upstream {
	return &upstreamClient{
		url:  stub.baseURL + "/upstreams",
		stub: stub,
	}
}

func (r *upstreamClient) List(ctx context.Context, group string) ([]*v1.Upstream, error) {
	log.Infow("try to list upstreams in APISIX", zap.String("url", r.url))

	upsItems, err := r.stub.listResource(ctx, r.url)
	if err != nil {
		log.Errorf("failed to list upstreams: %s", err)
		return nil, err
	}

	var items []*v1.Upstream
	for i, item := range upsItems.Node.Items {
		ups, err := item.upstream(group)
		if err != nil {
			log.Errorw("failed to convert upstream item",
				zap.String("url", r.url),
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

func (r *upstreamClient) Create(ctx context.Context, obj *v1.Upstream) (*v1.Upstream, error) {
	log.Infow("try to create upstream",
		zap.String("full_name", *obj.FullName),
	)

	// TODO Just pass the node array.
	nodes := make(map[string]int64, len(obj.Nodes))
	for _, n := range obj.Nodes {
		ep := net.JoinHostPort(*n.IP, strconv.Itoa(*n.Port))
		nodes[ep] = int64(*n.Weight)
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

	resp, err := r.stub.createResource(ctx, r.url, bytes.NewReader(body))
	if err != nil {
		log.Errorf("failed to create upstream: %s", err)
		return nil, err
	}
	var group string
	if obj.Group != nil {
		group = *obj.Group
	}
	return resp.Item.upstream(group)
}

func (r *upstreamClient) Delete(ctx context.Context, obj *v1.Upstream) error {
	log.Infof("delete upstream, id:%s", *obj.ID)
	url := r.url + "/" + *obj.ID
	return r.stub.deleteResource(ctx, url)
}

func (r *upstreamClient) Update(ctx context.Context, obj *v1.Upstream) error {
	log.Infof("update upstream, id:%s", *obj.ID)

	// TODO Just pass the node array.
	nodes := make(map[string]int64, len(obj.Nodes))
	for _, n := range obj.Nodes {
		ep := net.JoinHostPort(*n.IP, strconv.Itoa(*n.Port))
		nodes[ep] = int64(*n.Weight)
	}

	body, err := json.Marshal(upstreamReqBody{
		LBType: obj.Type,
		HashOn: obj.HashOn,
		Key:    obj.Key,
		Nodes:  nodes,
		Desc:   obj.Name,
	})
	if err != nil {
		return err
	}

	url := r.url + "/" + *obj.ID
	return r.stub.updateResource(ctx, url, bytes.NewReader(body))
}
