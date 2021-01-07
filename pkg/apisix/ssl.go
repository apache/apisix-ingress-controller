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
	"go.uber.org/zap"

	v1 "github.com/api7/ingress-controller/pkg/types/apisix/v1"
)

type sslClient struct {
	url         string
	clusterName string
	cluster     *cluster
}

func newSSLClient(c *cluster) SSL {
	return &sslClient{
		url:         c.baseURL + "/ssl",
		cluster:     c,
		clusterName: c.name,
	}
}

func (s *sslClient) List(ctx context.Context) ([]*v1.Ssl, error) {
	log.Infow("try to list ssl in APISIX", zap.String("url", s.url))

	sslItems, err := s.cluster.listResource(ctx, s.url)
	if err != nil {
		log.Errorf("failed to list ssl: %s", err)
		return nil, err
	}

	var items []*v1.Ssl
	for i, item := range sslItems.Node.Items {
		ssl, err := item.ssl(s.clusterName)
		if err != nil {
			log.Errorw("failed to convert ssl item",
				zap.String("url", s.url),
				zap.String("ssl_key", item.Key),
				zap.Error(err),
			)
			return nil, err
		}
		items = append(items, ssl)
		log.Infof("list ssl #%d, body: %s", i, string(item.Value))
	}

	return items, nil
}

func (s *sslClient) Create(ctx context.Context, obj *v1.Ssl) (*v1.Ssl, error) {
	log.Info("try to create ssl")
	data, err := json.Marshal(v1.Ssl{
		Snis:   obj.Snis,
		Cert:   obj.Cert,
		Key:    obj.Key,
		Status: obj.Status,
	})
	if err != nil {
		return nil, err
	}
	resp, err := s.cluster.createResource(ctx, s.url, bytes.NewReader(data))
	if err != nil {
		log.Errorf("failed to create ssl: %s", err)
		return nil, err
	}

	var clusterName string
	if obj.Group != nil {
		clusterName = *obj.Group
	}

	return resp.Item.ssl(clusterName)
}

func (s *sslClient) Delete(ctx context.Context, obj *v1.Ssl) error {
	log.Infof("delete ssl, id:%s", *obj.ID)
	url := s.url + "/" + *obj.ID
	return s.cluster.deleteResource(ctx, url)
}

func (s *sslClient) Update(ctx context.Context, obj *v1.Ssl) (*v1.Ssl, error) {
	log.Infof("update ssl, id:%s", *obj.ID)
	url := s.url + "/" + *obj.ID
	data, err := json.Marshal(v1.Ssl{
		ID:     obj.ID,
		Snis:   obj.Snis,
		Cert:   obj.Cert,
		Key:    obj.Key,
		Status: obj.Status,
	})
	if err != nil {
		return nil, err
	}
	resp, err := s.cluster.updateResource(ctx, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	var clusterName string
	if obj.Group != nil {
		clusterName = *obj.Group
	}
	return resp.Item.ssl(clusterName)
}
