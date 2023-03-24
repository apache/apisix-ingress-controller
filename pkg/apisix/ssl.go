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
	"context"
	"encoding/json"

	"go.uber.org/zap"

	"github.com/apache/apisix-ingress-controller/pkg/apisix/cache"
	"github.com/apache/apisix-ingress-controller/pkg/id"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	v1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

type sslClient struct {
	url     string
	cluster *cluster
}

func newSSLClient(c *cluster) SSL {
	if c.adminVersion == "v3" {
		return &sslClient{
			url:     c.baseURL + "/ssls",
			cluster: c,
		}
	}

	return &sslClient{
		url:     c.baseURL + "/ssl",
		cluster: c,
	}
}

// name is namespace_sslname
func (s *sslClient) Get(ctx context.Context, name string) (*v1.Ssl, error) {
	log.Debugw("try to look up ssl",
		zap.String("name", name),
		zap.String("url", s.url),
		zap.String("cluster", s.cluster.name),
	)
	sid := id.GenID(name)
	ssl, err := s.cluster.cache.GetSSL(sid)
	if err == nil {
		return ssl, nil
	}
	if err != cache.ErrNotFound {
		log.Errorw("failed to find ssl in cache, will try to lookup from APISIX",
			zap.String("name", name),
			zap.Error(err),
		)
	} else {
		log.Debugw("failed to find ssl in cache, will try to lookup from APISIX",
			zap.String("name", name),
			zap.Error(err),
		)
	}

	// TODO Add mutex here to avoid dog-pile effect.
	ssl, err = s.cluster.GetSSL(ctx, s.url, sid)
	if err != nil {
		return nil, err
	}

	if err := s.cluster.cache.InsertSSL(ssl); err != nil {
		log.Errorf("failed to reflect ssl create to cache: %s", err)
		return nil, err
	}
	return ssl, nil
}

// List is only used in cache warming up. So here just pass through
// to APISIX.
func (s *sslClient) List(ctx context.Context) ([]*v1.Ssl, error) {
	log.Debugw("try to list ssl in APISIX",
		zap.String("url", s.url),
		zap.String("cluster", s.cluster.name),
	)

	sslItems, err := s.cluster.listResource(ctx, s.url, "ssl")
	if err != nil {
		log.Errorf("failed to list ssl: %s", err)
		return nil, err
	}

	var items []*v1.Ssl
	for i, item := range sslItems {
		ssl, err := item.ssl()
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

func (s *sslClient) Create(ctx context.Context, obj *v1.Ssl, shouldCompare bool) (*v1.Ssl, error) {
	if v, skip := skipRequest(s.cluster, shouldCompare, s.url, obj.ID, obj); skip {
		return v, nil
	}

	log.Debugw("try to create ssl",
		zap.String("cluster", s.cluster.name),
		zap.String("url", s.url),
		zap.String("id", obj.ID),
	)
	if err := s.cluster.HasSynced(ctx); err != nil {
		return nil, err
	}
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	url := s.url + "/" + obj.ID
	log.Debugw("creating ssl", zap.ByteString("body", data), zap.String("url", url))
	resp, err := s.cluster.createResource(ctx, url, "ssl", data)
	if err != nil {
		log.Errorf("failed to create ssl: %s", err)
		return nil, err
	}

	ssl, err := resp.ssl()
	if err != nil {
		return nil, err
	}
	if err := s.cluster.cache.InsertSSL(ssl); err != nil {
		log.Errorf("failed to reflect ssl create to cache: %s", err)
		return nil, err
	}
	if err := s.cluster.generatedObjCache.InsertSSL(obj); err != nil {
		log.Errorf("failed to reflect generated ssl create to cache: %s", err)
		return nil, err
	}
	return ssl, nil
}

func (s *sslClient) Delete(ctx context.Context, obj *v1.Ssl) error {
	log.Debugw("try to delete ssl",
		zap.String("id", obj.ID),
		zap.String("cluster", s.cluster.name),
		zap.String("url", s.url),
	)
	if err := s.cluster.HasSynced(ctx); err != nil {
		return err
	}
	url := s.url + "/" + obj.ID
	if err := s.cluster.deleteResource(ctx, url, "ssl"); err != nil {
		return err
	}
	if err := s.cluster.cache.DeleteSSL(obj); err != nil {
		log.Errorf("failed to reflect ssl delete to cache: %s", err)
		if err != cache.ErrNotFound {
			return err
		}
	}
	if err := s.cluster.generatedObjCache.DeleteSSL(obj); err != nil {
		log.Errorf("failed to reflect ssl delete to generated cache: %s", err)
		if err != cache.ErrNotFound {
			return err
		}
	}
	return nil
}

func (s *sslClient) Update(ctx context.Context, obj *v1.Ssl, shouldCompare bool) (*v1.Ssl, error) {
	if v, skip := skipRequest(s.cluster, shouldCompare, s.url, obj.ID, obj); skip {
		return v, nil
	}

	log.Debugw("try to update ssl",
		zap.String("id", obj.ID),
		zap.String("cluster", s.cluster.name),
		zap.String("url", s.url),
	)
	if err := s.cluster.HasSynced(ctx); err != nil {
		return nil, err
	}
	url := s.url + "/" + obj.ID
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	resp, err := s.cluster.updateResource(ctx, url, "ssl", data)
	if err != nil {
		return nil, err
	}
	ssl, err := resp.ssl()
	if err != nil {
		return nil, err
	}
	if err := s.cluster.cache.InsertSSL(ssl); err != nil {
		log.Errorf("failed to reflect ssl update to cache: %s", err)
		return nil, err
	}
	if err := s.cluster.generatedObjCache.InsertSSL(obj); err != nil {
		log.Errorf("failed to reflect generated ssl update to cache: %s", err)
		return nil, err
	}
	return ssl, nil
}
