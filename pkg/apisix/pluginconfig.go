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
	"fmt"

	"go.uber.org/zap"

	"github.com/apache/apisix-ingress-controller/pkg/apisix/cache"
	"github.com/apache/apisix-ingress-controller/pkg/id"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	v1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

type pluginConfigClient struct {
	url     string
	cluster *cluster
}

func newPluginConfigClient(c *cluster) PluginConfig {
	return &pluginConfigClient{
		url:     c.baseURL + "/plugin_configs",
		cluster: c,
	}
}

// Get returns the v1.PluginConfig.
// FIXME, currently if caller pass a non-existent resource, the Get always passes
// through cache.
func (pc *pluginConfigClient) Get(ctx context.Context, name string) (*v1.PluginConfig, error) {
	log.Debugw("try to look up pluginConfig",
		zap.String("name", name),
		zap.String("url", pc.url),
		zap.String("cluster", pc.cluster.name),
	)
	rid := id.GenID(name)
	pluginConfig, err := pc.cluster.cache.GetPluginConfig(rid)
	if err == nil {
		return pluginConfig, nil
	}
	if err != cache.ErrNotFound {
		log.Errorw("failed to find pluginConfig in cache, will try to lookup from APISIX",
			zap.String("name", name),
			zap.Error(err),
		)
	} else {
		log.Debugw("pluginConfig not found in cache, will try to lookup from APISIX",
			zap.String("name", name),
			zap.Error(err),
		)
	}

	// TODO Add mutex here to avoid dog-pile effect.
	pluginConfig, err = pc.cluster.GetPluginConfig(ctx, pc.url, rid)
	if err != nil {
		return nil, err
	}

	if err := pc.cluster.cache.InsertPluginConfig(pluginConfig); err != nil {
		log.Errorf("failed to reflect pluginConfig create to cache: %s", err)
		return nil, err
	}
	return pluginConfig, nil
}

// List is only used in cache warming up. So here just pass through
// to APISIX.
func (pc *pluginConfigClient) List(ctx context.Context) ([]*v1.PluginConfig, error) {
	log.Debugw("try to list pluginConfig in APISIX",
		zap.String("cluster", pc.cluster.name),
		zap.String("url", pc.url),
	)
	pluginConfigItems, err := pc.cluster.listResource(ctx, pc.url, "pluginConfig")
	if err != nil {
		log.Errorf("failed to list pluginConfig: %s", err)
		return nil, err
	}

	var items []*v1.PluginConfig
	for i, item := range pluginConfigItems {
		pluginConfig, err := item.pluginConfig()
		if err != nil {
			log.Errorw("failed to convert pluginConfig item",
				zap.String("url", pc.url),
				zap.String("pluginConfig_key", item.Key),
				zap.String("pluginConfig_value", string(item.Value)),
				zap.Error(err),
			)
			return nil, err
		}

		items = append(items, pluginConfig)
		log.Debugf("list pluginConfig #%d, body: %s", i, string(item.Value))
	}

	return items, nil
}

func (pc *pluginConfigClient) Create(ctx context.Context, obj *v1.PluginConfig, shouldCompare bool) (*v1.PluginConfig, error) {
	if v, skip := skipRequest(pc.cluster, shouldCompare, pc.url, obj.ID, obj); skip {
		return v, nil
	}

	log.Debugw("try to create pluginConfig",
		zap.String("name", obj.Name),
		zap.Any("plugins", obj.Plugins),
		zap.String("cluster", pc.cluster.name),
		zap.String("url", pc.url),
	)

	if err := pc.cluster.HasSynced(ctx); err != nil {
		return nil, err
	}
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	url := pc.url + "/" + obj.ID
	log.Debugw("creating pluginConfig", zap.ByteString("body", data), zap.String("url", url))
	resp, err := pc.cluster.createResource(ctx, url, "pluginConfig", data)
	if err != nil {
		log.Errorf("failed to create pluginConfig: %s", err)
		return nil, err
	}

	pluginConfig, err := resp.pluginConfig()
	if err != nil {
		return nil, err
	}
	if err := pc.cluster.cache.InsertPluginConfig(pluginConfig); err != nil {
		log.Errorf("failed to reflect pluginConfig create to cache: %s", err)
		return nil, err
	}
	if err := pc.cluster.generatedObjCache.InsertPluginConfig(obj); err != nil {
		log.Errorf("failed to cache generated pluginConfig object: %s", err)
		return nil, err
	}
	return pluginConfig, nil
}

func (pc *pluginConfigClient) Delete(ctx context.Context, obj *v1.PluginConfig) error {
	log.Debugw("try to delete pluginConfig",
		zap.String("id", obj.ID),
		zap.String("name", obj.Name),
		zap.String("cluster", pc.cluster.name),
		zap.String("url", pc.url),
	)

	if err := pc.cluster.HasSynced(ctx); err != nil {
		return err
	}
	url := pc.url + "/" + obj.ID
	if err := pc.cluster.deleteResource(ctx, url, "pluginConfig"); err != nil {
		return err
	}
	if err := pc.cluster.cache.DeletePluginConfig(obj); err != nil {
		log.Errorf("failed to reflect pluginConfig delete to cache: %s", err)
		if err != cache.ErrNotFound {
			return err
		}
	}
	if err := pc.cluster.generatedObjCache.DeletePluginConfig(obj); err != nil {
		log.Errorf("failed to reflect pluginConfig delete to generated cache: %s", err)
		if err != cache.ErrNotFound {
			return err
		}
	}
	return nil
}

func (pc *pluginConfigClient) Update(ctx context.Context, obj *v1.PluginConfig, shouldCompare bool) (*v1.PluginConfig, error) {
	if v, skip := skipRequest(pc.cluster, shouldCompare, pc.url, obj.ID, obj); skip {
		return v, nil
	}

	log.Debugw("try to update pluginConfig",
		zap.String("id", obj.ID),
		zap.String("name", obj.Name),
		zap.Any("plugins", obj.Plugins),
		zap.String("cluster", pc.cluster.name),
		zap.String("url", pc.url),
	)
	if err := pc.cluster.HasSynced(ctx); err != nil {
		return nil, err
	}
	body, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	url := pc.url + "/" + obj.ID
	resp, err := pc.cluster.updateResource(ctx, url, "pluginConfig", body)
	if err != nil {
		return nil, err
	}
	pluginConfig, err := resp.pluginConfig()
	if err != nil {
		return nil, err
	}
	if err := pc.cluster.cache.InsertPluginConfig(pluginConfig); err != nil {
		log.Errorf("failed to reflect pluginConfig update to cache: %s", err)
		return nil, err
	}
	if err := pc.cluster.generatedObjCache.InsertPluginConfig(obj); err != nil {
		log.Errorf("failed to cache generated pluginConfig object: %s", err)
		return nil, err
	}
	return pluginConfig, nil
}

type pluginConfigMem struct {
	url string

	resource string
	cluster  *cluster
}

func newPluginConfigMem(c *cluster) PluginConfig {
	return &pluginConfigMem{
		url:      c.baseURL + "/plugin_configs",
		resource: "plugin_configs",
		cluster:  c,
	}
}

func (r *pluginConfigMem) Get(ctx context.Context, name string) (*v1.PluginConfig, error) {
	log.Debugw("try to look up pluginConfig",
		zap.String("name", name),
		zap.String("url", r.url),
		zap.String("cluster", r.cluster.name),
	)
	rid := id.GenID(name)
	pluginConfig, err := r.cluster.cache.GetPluginConfig(rid)
	if err != nil {
		log.Errorw("failed to find pluginConfig in cache, will try to lookup from APISIX",
			zap.String("name", name),
			zap.Error(err),
		)
	}
	return pluginConfig, nil
}

// List is only used in cache warming up. So here just pass through
// to APISIX.
func (r *pluginConfigMem) List(ctx context.Context) ([]*v1.PluginConfig, error) {
	log.Debugw("try to list resource in APISIX",
		zap.String("cluster", r.cluster.name),
		zap.String("resource", r.resource),
	)
	pluginConfigItems, err := r.cluster.cache.ListPluginConfigs()
	if err != nil {
		log.Errorf("failed to list %s: %s", r.resource, err)
		return nil, err
	}
	return pluginConfigItems, nil
}

func (r *pluginConfigMem) Create(ctx context.Context, obj *v1.PluginConfig, shouldCompare bool) (*v1.PluginConfig, error) {
	if shouldCompare && CompareResourceEqualFromCluster(r.cluster, obj.ID, obj) {
		return obj, nil
	}
	if ok, err := r.cluster.validator.ValidateHTTPPluginSchema(obj.Plugins); !ok {
		return nil, err
	}
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	r.cluster.CreateResource(r.resource, obj.ID, data)
	if err := r.cluster.cache.InsertPluginConfig(obj); err != nil {
		log.Errorf("failed to reflect plugin_config create to cache: %s", err)
		return nil, err
	}
	return obj, nil
}

func (r *pluginConfigMem) Delete(ctx context.Context, obj *v1.PluginConfig) error {
	if ok, err := r.deleteCheck(ctx, obj); !ok {
		log.Debug("failed to delete upstream", zap.Error(err))
		return cache.ErrStillInUse
	}
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	r.cluster.DeleteResource(r.resource, obj.ID, data)
	if err := r.cluster.cache.DeletePluginConfig(obj); err != nil {
		log.Errorf("failed to reflect plugin_config delete to cache: %s", err)
		return err
	}
	return nil
}

func (r *pluginConfigMem) Update(ctx context.Context, obj *v1.PluginConfig, shouldCompare bool) (*v1.PluginConfig, error) {
	if shouldCompare && CompareResourceEqualFromCluster(r.cluster, obj.ID, obj) {
		return obj, nil
	}
	if ok, err := r.cluster.validator.ValidateHTTPPluginSchema(obj.Plugins); !ok {
		return nil, err
	}
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	r.cluster.UpdateResource(r.resource, obj.ID, data)
	if err := r.cluster.cache.InsertPluginConfig(obj); err != nil {
		log.Errorf("failed to reflect plugin_config update to cache: %s", err)
		return nil, err
	}
	return obj, nil
}

// TODO: Maintain a reference count for each object without having to poll each time
func (u *pluginConfigMem) deleteCheck(ctx context.Context, obj *v1.PluginConfig) (bool, error) {
	routes, _ := u.cluster.route.List(ctx)
	if routes == nil {
		return true, nil
	}
	for _, route := range routes {
		if route.PluginConfigId == obj.ID {
			return false, fmt.Errorf("can not delete this plugin_config, route.id=%s is still using it now", route.ID)
		}
	}
	return true, nil
}
