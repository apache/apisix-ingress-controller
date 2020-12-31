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

	"github.com/api7/ingress-controller/pkg/log"
	v1 "github.com/api7/ingress-controller/pkg/types/apisix/v1"
	"go.uber.org/zap"
)

type serviceClient struct {
	url  string
	stub *stub
}

type serviceItem struct {
	UpstreamId *string                 `json:"upstream_id,omitempty"`
	Plugins    *map[string]interface{} `json:"plugins"`
	Desc       *string                 `json:"desc,omitempty"`
}

func newServiceClient(stub *stub) Service {
	return &serviceClient{
		url:  stub.baseURL + "/services",
		stub: stub,
	}
}

func (s *serviceClient) List(ctx context.Context, group string) ([]*v1.Service, error) {
	log.Infow("try to list services in APISIX", zap.String("url", s.url))

	upsItems, err := s.stub.listResource(ctx, s.url)
	if err != nil {
		log.Errorf("failed to list upstreams: %s", err)
		return nil, err
	}

	var items []*v1.Service
	for i, item := range upsItems.Node.Items {
		svc, err := item.service(group)
		if err != nil {
			log.Errorw("failed to convert service item",
				zap.String("url", s.url),
				zap.String("service_key", item.Key),
				zap.Error(err),
			)
			return nil, err
		}
		items = append(items, svc)
		log.Infof("list service #%d, body: %s", i, string(item.Value))
	}
	return items, nil
}

func (s *serviceClient) Create(ctx context.Context, obj *v1.Service) (*v1.Service, error) {
	log.Infow("try to create service", zap.String("full_name", *obj.FullName))

	body, err := json.Marshal(serviceItem{
		UpstreamId: obj.UpstreamId,
		Plugins:    (*map[string]interface{})(obj.Plugins),
		Desc:       obj.Name,
	})
	if err != nil {
		return nil, err
	}

	resp, err := s.stub.createResource(ctx, s.url, bytes.NewReader(body))
	if err != nil {
		log.Errorf("failed to create service: %s", err)
		return nil, err
	}
	var group string
	if obj.Group != nil {
		group = *obj.Group
	}
	return resp.Item.service(group)
}

func (s *serviceClient) Delete(ctx context.Context, obj *v1.Service) error {
	log.Infof("delete service, id:%s", *obj.ID)
	url := s.url + "/" + *obj.ID
	return s.stub.deleteResource(ctx, url)
}

func (s *serviceClient) Update(ctx context.Context, obj *v1.Service) error {
	log.Infof("update service, id:%s", *obj.ID)

	body, err := json.Marshal(serviceItem{
		UpstreamId: obj.UpstreamId,
		Plugins:    (*map[string]interface{})(obj.Plugins),
		Desc:       obj.Name,
	})
	if err != nil {
		return err
	}

	url := s.url + "/" + *obj.ID
	return s.stub.updateResource(ctx, url, bytes.NewReader(body))
}
