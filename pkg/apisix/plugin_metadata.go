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

	"github.com/apache/apisix-ingress-controller/pkg/log"
	v1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

type pluginMetadataClient struct {
	url     string
	cluster *cluster
}

func newPluginMetadataClient(c *cluster) *pluginMetadataClient {
	return &pluginMetadataClient{
		url:     c.baseURL + "/plugin_metadata",
		cluster: c,
	}
}

func (r *pluginMetadataClient) Get(ctx context.Context, name string) (*v1.PluginMetadata, error) {
	log.Debugw("try to look up pluginMetadata",
		zap.String("name", name),
		zap.String("url", r.url),
		zap.String("cluster", "default"),
	)

	// TODO Add mutex here to avoid dog-pile effect.
	url := r.url + "/" + name
	resp, err := r.cluster.getResource(ctx, url, "pluginMetadata")
	r.cluster.metricsCollector.IncrAPISIXRequest("pluginMetadata")
	if err != nil {
		log.Errorw("failed to get pluginMetadata from APISIX",
			zap.String("name", name),
			zap.String("url", url),
			zap.String("cluster", "default"),
			zap.Error(err),
		)
		return nil, err
	}

	pluginMetadata, err := resp.pluginMetadata()
	if err != nil {
		log.Errorw("failed to convert pluginMetadata item",
			zap.String("url", r.url),
			zap.String("pluginMetadata_key", resp.Key),
			zap.String("pluginMetadata_value", string(resp.Value)),
			zap.Error(err),
		)
		return nil, err
	}
	return pluginMetadata, nil
}

func (r *pluginMetadataClient) List(ctx context.Context) ([]*v1.PluginMetadata, error) {
	log.Debugw("try to list pluginMetadatas in APISIX",
		zap.String("cluster", "default"),
		zap.String("url", r.url),
	)
	pluginMetadataItems, err := r.cluster.listResource(ctx, r.url, "pluginMetadata")
	r.cluster.metricsCollector.IncrAPISIXRequest("pluginMetadata")
	if err != nil {
		log.Errorf("failed to list pluginMetadatas: %s", err)
		return nil, err
	}

	var items []*v1.PluginMetadata
	for i, item := range pluginMetadataItems {
		pluginMetadata, err := item.pluginMetadata()
		if err != nil {
			log.Errorw("failed to convert pluginMetadata item",
				zap.String("url", r.url),
				zap.String("pluginMetadata_key", item.Key),
				zap.String("pluginMetadata_value", string(item.Value)),
				zap.Error(err),
			)
			return nil, err
		}

		items = append(items, pluginMetadata)
		log.Debugf("list pluginMetadata #%d, body: %s", i, string(item.Value))
	}

	return items, nil
}

func (r *pluginMetadataClient) Delete(ctx context.Context, obj *v1.PluginMetadata) error {
	log.Debugw("try to delete pluginMetadata",
		zap.String("name", obj.Name),
		zap.Any("metadata", obj.Metadata),
		zap.String("cluster", "default"),
		zap.String("url", r.url),
	)
	if err := r.cluster.HasSynced(ctx); err != nil {
		return err
	}
	url := r.url + "/" + obj.Name
	if err := r.cluster.deleteResource(ctx, url, "pluginMetadata"); err != nil {
		r.cluster.metricsCollector.IncrAPISIXRequest("pluginMetadata")
		return err
	}
	r.cluster.metricsCollector.IncrAPISIXRequest("pluginMetadata")
	return nil
}

func (r *pluginMetadataClient) Update(ctx context.Context, obj *v1.PluginMetadata) (*v1.PluginMetadata, error) {
	log.Debugw("try to update pluginMetadata",
		zap.String("name", obj.Name),
		zap.Any("metadata", obj.Metadata),
		zap.String("cluster", "default"),
		zap.String("url", r.url),
	)
	if err := r.cluster.HasSynced(ctx); err != nil {
		return nil, err
	}
	body, err := json.Marshal(obj.Metadata)
	if err != nil {
		return nil, err
	}
	url := r.url + "/" + obj.Name
	resp, err := r.cluster.updateResource(ctx, url, "pluginMetadata", body)
	r.cluster.metricsCollector.IncrAPISIXRequest("pluginMetadata")
	if err != nil {
		return nil, err
	}
	pluginMetadata, err := resp.pluginMetadata()
	if err != nil {
		return nil, err
	}
	return pluginMetadata, nil
}
