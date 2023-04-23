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

type globalRuleClient struct {
	url     string
	cluster *cluster
}

func newGlobalRuleClient(c *cluster) GlobalRule {
	return &globalRuleClient{
		url:     c.baseURL + "/global_rules",
		cluster: c,
	}
}

// Get returns the GlobalRule.
// FIXME, currently if caller pass a non-existent resource, the Get always passes
// through cache.
func (r *globalRuleClient) Get(ctx context.Context, id string) (*v1.GlobalRule, error) {
	log.Debugw("try to look up global_rule",
		zap.String("id", id),
		zap.String("url", r.url),
		zap.String("cluster", r.cluster.name),
	)
	globalRule, err := r.cluster.cache.GetGlobalRule(id)
	if err == nil {
		return globalRule, nil
	}
	if err != cache.ErrNotFound {
		log.Errorw("failed to find global_rule in cache, will try to lookup from APISIX",
			zap.String("id", id),
			zap.Error(err),
		)
	} else {
		log.Debugw("failed to find global_rule in cache, will try to lookup from APISIX",
			zap.String("id", id),
			zap.Error(err),
		)
	}

	// TODO Add mutex here to avoid dog-pile effect.
	globalRule, err = r.cluster.GetGlobalRule(ctx, r.url, id)
	if err != nil {
		return nil, err
	}

	if err := r.cluster.cache.InsertGlobalRule(globalRule); err != nil {
		log.Errorf("failed to reflect global_rule create to cache: %s", err)
		return nil, err
	}
	return globalRule, nil
}

// List is only used in cache warming up. So here just pass through
// to APISIX.
func (r *globalRuleClient) List(ctx context.Context) ([]*v1.GlobalRule, error) {
	log.Debugw("try to list global_rules in APISIX",
		zap.String("cluster", r.cluster.name),
		zap.String("url", r.url),
	)
	globalRuleItems, err := r.cluster.listResource(ctx, r.url, "globalRule")
	if err != nil {
		log.Errorf("failed to list global_rules: %s", err)
		return nil, err
	}

	var items []*v1.GlobalRule
	for i, item := range globalRuleItems {
		globalRule, err := item.globalRule()
		if err != nil {
			log.Errorw("failed to convert global_rule item",
				zap.String("url", r.url),
				zap.String("global_rule_key", item.Key),
				zap.String("global_rule_value", string(item.Value)),
				zap.Error(err),
			)
			return nil, err
		}

		items = append(items, globalRule)
		log.Debugf("list global_rule #%d, body: %s", i, string(item.Value))
	}

	return items, nil
}

func (r *globalRuleClient) Create(ctx context.Context, obj *v1.GlobalRule, shouldCompare bool) (*v1.GlobalRule, error) {
	if v, skip := skipRequest(r.cluster, shouldCompare, r.url, obj.ID, obj); skip {
		return v, nil
	}

	log.Debugw("try to create global_rule",
		zap.String("id", obj.ID),
		zap.Any("plugins", obj.Plugins),
		zap.String("cluster", r.cluster.name),
		zap.String("url", r.url),
	)

	if err := r.cluster.HasSynced(ctx); err != nil {
		return nil, err
	}
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	url := r.url + "/" + obj.ID
	log.Debugw("creating global_rule", zap.ByteString("body", data), zap.String("url", url))
	resp, err := r.cluster.createResource(ctx, url, "globalRule", data)
	if err != nil {
		log.Errorf("failed to create global_rule: %s", err)
		return nil, err
	}

	globalRules, err := resp.globalRule()
	if err != nil {
		return nil, err
	}
	if err := r.cluster.cache.InsertGlobalRule(globalRules); err != nil {
		log.Errorf("failed to reflect global_rules create to cache: %s", err)
		return nil, err
	}
	if err := r.cluster.generatedObjCache.InsertGlobalRule(obj); err != nil {
		log.Errorf("failed to cache generated global_rule object: %s", err)
		return nil, err
	}
	return globalRules, nil
}

func (r *globalRuleClient) Delete(ctx context.Context, obj *v1.GlobalRule) error {
	log.Debugw("try to delete global_rule",
		zap.String("id", obj.ID),
		zap.String("cluster", r.cluster.name),
		zap.String("url", r.url),
	)
	if err := r.cluster.HasSynced(ctx); err != nil {
		return err
	}
	url := r.url + "/" + obj.ID
	if err := r.cluster.deleteResource(ctx, url, "globalRule"); err != nil {
		return err
	}
	if err := r.cluster.cache.DeleteGlobalRule(obj); err != nil {
		log.Errorf("failed to reflect global_rule delete to cache: %s", err)
		if err != cache.ErrNotFound {
			return err
		}
	}
	if err := r.cluster.generatedObjCache.DeleteGlobalRule(obj); err != nil {
		log.Errorf("failed to reflect global_rule delete to generated cache: %s", err)
		if err != cache.ErrNotFound {
			return err
		}
	}
	return nil
}

func (r *globalRuleClient) Update(ctx context.Context, obj *v1.GlobalRule, shouldCompare bool) (*v1.GlobalRule, error) {
	if v, skip := skipRequest(r.cluster, shouldCompare, r.url, obj.ID, obj); skip {
		return v, nil
	}

	log.Debugw("try to update global_rule",
		zap.String("id", obj.ID),
		zap.Any("plugins", obj.Plugins),
		zap.String("cluster", r.cluster.name),
		zap.String("url", r.url),
	)
	if err := r.cluster.HasSynced(ctx); err != nil {
		return nil, err
	}
	body, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	url := r.url + "/" + obj.ID
	resp, err := r.cluster.updateResource(ctx, url, "globalRule", body)
	if err != nil {
		return nil, err
	}
	globalRule, err := resp.globalRule()
	if err != nil {
		return nil, err
	}
	if err := r.cluster.cache.InsertGlobalRule(globalRule); err != nil {
		log.Errorf("failed to reflect global_rule update to cache: %s", err)
		return nil, err
	}
	if err := r.cluster.generatedObjCache.InsertGlobalRule(obj); err != nil {
		log.Errorf("failed to cache generated global_rule object: %s", err)
		return nil, err
	}
	return globalRule, nil
}

type globalRuleMem struct {
	url string

	resource string
	cluster  *cluster
}

func newGlobalRuleMem(c *cluster) GlobalRule {
	return &globalRuleMem{
		url:      c.baseURL + "/global_rules",
		resource: "global_rules",
		cluster:  c,
	}
}

func (r *globalRuleMem) Get(ctx context.Context, name string) (*v1.GlobalRule, error) {
	log.Debugw("try to look up globalRule",
		zap.String("name", name),
		zap.String("cluster", r.cluster.name),
	)
	rid := id.GenID(name)
	globalRule, err := r.cluster.cache.GetGlobalRule(rid)
	if err != nil && err != cache.ErrNotFound {
		log.Errorw("failed to find globalRule in cache, will try to lookup from APISIX",
			zap.String("name", name),
			zap.Error(err),
		)
		return nil, err
	}
	return globalRule, nil
}

// List is only used in cache warming up. So here just pass through
// to APISIX.
func (r *globalRuleMem) List(ctx context.Context) ([]*v1.GlobalRule, error) {
	log.Debugw("try to list resource in APISIX",
		zap.String("cluster", r.cluster.name),
		zap.String("url", r.url),
		zap.String("resource", r.resource),
	)
	globalRules, err := r.cluster.cache.ListGlobalRules()
	if err != nil {
		log.Errorf("failed to list %s: %s", r.resource, err)
		return nil, err
	}
	return globalRules, nil
}

func (r *globalRuleMem) Create(ctx context.Context, obj *v1.GlobalRule, shouldCompare bool) (*v1.GlobalRule, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	r.cluster.CreateResource(r.resource, obj.ID, data)
	if err := r.cluster.cache.InsertGlobalRule(obj); err != nil {
		log.Errorf("failed to reflect global_rule create to cache: %s", err)
		return nil, err
	}
	return obj, nil
}

func (r *globalRuleMem) Delete(ctx context.Context, obj *v1.GlobalRule) error {
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	r.cluster.DeleteResource(r.resource, obj.ID, data)
	if err := r.cluster.cache.DeleteGlobalRule(obj); err != nil {
		log.Errorf("failed to reflect global_rule delete to cache: %s", err)
		return err
	}
	return nil
}

func (r *globalRuleMem) Update(ctx context.Context, obj *v1.GlobalRule, shouldCompare bool) (*v1.GlobalRule, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	r.cluster.UpdateResource(r.resource, obj.ID, data)
	if err := r.cluster.cache.InsertGlobalRule(obj); err != nil {
		log.Errorf("failed to reflect global_rule update to cache: %s", err)
		return nil, err
	}
	return obj, nil
}
