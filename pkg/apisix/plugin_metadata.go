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
		zap.String("cluster", r.cluster.name),
	)

	// TODO Add mutex here to avoid dog-pile effect.
	url := r.url + "/" + name
	resp, err := r.cluster.getResource(ctx, url, "pluginMetadata")
	if err != nil {
		log.Errorw("failed to get pluginMetadata from APISIX",
			zap.String("name", name),
			zap.String("url", url),
			zap.String("cluster", r.cluster.name),
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
		zap.String("cluster", r.cluster.name),
		zap.String("url", r.url),
	)
	pluginMetadataItems, err := r.cluster.listResource(ctx, r.url, "pluginMetadata")
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
		zap.String("cluster", r.cluster.name),
		zap.String("url", r.url),
	)
	if err := r.cluster.HasSynced(ctx); err != nil {
		return err
	}
	url := r.url + "/" + obj.Name
	if err := r.cluster.deleteResource(ctx, url, "pluginMetadata"); err != nil {
		return err
	}
	return nil
}

func (r *pluginMetadataClient) Update(ctx context.Context, obj *v1.PluginMetadata, shouldCompare bool) (*v1.PluginMetadata, error) {
	log.Debugw("try to update pluginMetadata",
		zap.String("name", obj.Name),
		zap.Any("metadata", obj.Metadata),
		zap.String("cluster", r.cluster.name),
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
	if err != nil {
		return nil, err
	}
	pluginMetadata, err := resp.pluginMetadata()
	if err != nil {
		return nil, err
	}
	return pluginMetadata, nil
}

func (r *pluginMetadataClient) Create(ctx context.Context, obj *v1.PluginMetadata, shouldCompare bool) (*v1.PluginMetadata, error) {
	log.Debugw("try to create pluginMetadata",
		zap.String("name", obj.Name),
		zap.Any("metadata", obj.Metadata),
		zap.String("cluster", r.cluster.name),
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
	if err != nil {
		return nil, err
	}
	pluginMetadata, err := resp.pluginMetadata()
	if err != nil {
		return nil, err
	}
	return pluginMetadata, nil
}

type pluginMetadataMem struct {
	url string

	resource string
	cluster  *cluster
}

func newPluginMetadataMem(c *cluster) PluginMetadata {
	return &pluginMetadataMem{
		url:      c.baseURL + "/plugin_metadata",
		resource: "plugin_metadata",
		cluster:  c,
	}
}

func (r *pluginMetadataMem) Get(ctx context.Context, name string) (*v1.PluginMetadata, error) {
	log.Debugw("try to look up pluginMetadata",
		zap.String("name", name),
		zap.String("url", r.url),
		zap.String("cluster", r.cluster.name),
	)

	// TODO Add mutex here to avoid dog-pile effect.
	url := r.url + "/" + name
	resp, err := r.cluster.getResource(ctx, url, "pluginMetadata")
	if err != nil {
		log.Errorw("failed to get pluginMetadata from APISIX",
			zap.String("name", name),
			zap.String("url", url),
			zap.String("cluster", r.cluster.name),
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

// List is only used in cache warming up. So here just pass through
// to APISIX.
func (r *pluginMetadataMem) List(ctx context.Context) ([]*v1.PluginMetadata, error) {
	log.Debugw("try to list resource in APISIX",
		zap.String("cluster", r.cluster.name),
		zap.String("url", r.url),
		zap.String("resource", r.resource),
	)
	pluginMetadataItems, err := r.cluster.listResource(ctx, r.url, r.resource)
	if err != nil {
		log.Errorf("failed to list %s: %s", r.resource, err)
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

func (r *pluginMetadataMem) Create(ctx context.Context, obj *v1.PluginMetadata, shouldCompare bool) (*v1.PluginMetadata, error) {
	data, err := json.Marshal(obj.Metadata)
	if err != nil {
		return nil, err
	}
	r.cluster.CreateResource(r.resource, obj.Name, data)
	return obj, nil
}

func (r *pluginMetadataMem) Delete(ctx context.Context, obj *v1.PluginMetadata) error {
	data, err := json.Marshal(obj.Metadata)
	if err != nil {
		return err
	}
	r.cluster.DeleteResource(r.resource, obj.Name, data)
	return nil
}

func (r *pluginMetadataMem) Update(ctx context.Context, obj *v1.PluginMetadata, shouldCompare bool) (*v1.PluginMetadata, error) {
	data, err := json.Marshal(obj.Metadata)
	if err != nil {
		return nil, err
	}
	r.cluster.UpdateResource(r.resource, obj.Name, data)
	return obj, nil
}
