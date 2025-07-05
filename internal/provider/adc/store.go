// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package adc

import (
	"sync"

	"github.com/api7/gopkg/pkg/log"
	"github.com/google/uuid"
	"go.uber.org/zap"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	"github.com/apache/apisix-ingress-controller/internal/provider/adc/cache"
)

type Store struct {
	cacheMap          map[string]cache.Cache
	pluginMetadataMap map[string]adctypes.PluginMetadata

	sync.Mutex
}

func NewStore() *Store {
	return &Store{
		cacheMap:          make(map[string]cache.Cache),
		pluginMetadataMap: make(map[string]adctypes.PluginMetadata),
	}
}

func (s *Store) Insert(name string, resourceTypes []string, resources adctypes.Resources, Labels map[string]string) error {
	s.Lock()
	defer s.Unlock()
	targetCache, ok := s.cacheMap[name]
	if !ok {
		db, err := cache.NewMemDBCache()
		if err != nil {
			return err
		}
		s.cacheMap[name] = db
		targetCache = s.cacheMap[name]
	}
	log.Debugw("Inserting resources into cache for", zap.String("name", name))
	selector := &cache.KindLabelSelector{
		Kind:      Labels[label.LabelKind],
		Name:      Labels[label.LabelName],
		Namespace: Labels[label.LabelNamespace],
	}
	for _, resourceType := range resourceTypes {
		switch resourceType {
		case "service":
			services, err := targetCache.ListServices(selector)
			if err != nil {
				return err
			}
			for _, service := range services {
				if err := targetCache.DeleteService(service); err != nil {
					return err
				}
			}
			for _, service := range resources.Services {
				if err := targetCache.InsertService(service); err != nil {
					return err
				}
			}
		case "consumer":
			consumers, err := targetCache.ListConsumers(selector)
			if err != nil {
				return err
			}
			for _, consumer := range consumers {
				if err := targetCache.DeleteConsumer(consumer); err != nil {
					return err
				}
			}
			for _, consumer := range resources.Consumers {
				if err := targetCache.InsertConsumer(consumer); err != nil {
					return err
				}
			}
		case "ssl":
			ssls, err := targetCache.ListSSL(selector)
			if err != nil {
				return err

			}
			for _, ssl := range ssls {
				if err := targetCache.DeleteSSL(ssl); err != nil {
					return err
				}
			}
			for _, ssl := range resources.SSLs {
				if err := targetCache.InsertSSL(ssl); err != nil {
					return err
				}
			}
		case "global_rule":
			// List existing global rules that match the selector
			globalRules, err := targetCache.ListGlobalRules(selector)
			if err != nil {
				return err
			}
			// Delete existing matching global rules
			for _, globalRule := range globalRules {
				if err := targetCache.DeleteGlobalRule(globalRule); err != nil {
					return err
				}
			}
			// Convert GlobalRule (Plugins) to GlobalRuleItem and insert
			if len(resources.GlobalRules) > 0 {
				id := name + "-" + uuid.NewString()
				globalRuleItem := &adctypes.GlobalRuleItem{
					Metadata: adctypes.Metadata{
						ID:     id,
						Name:   id,
						Labels: Labels,
					},
					Plugins: adctypes.Plugins(resources.GlobalRules),
				}
				if err := targetCache.InsertGlobalRule(globalRuleItem); err != nil {
					return err
				}
			}
		case "plugin_metadata":
			s.pluginMetadataMap[name] = resources.PluginMetadata
		default:
			continue
		}
	}
	return nil
}

func (s *Store) Delete(name string, resourceTypes []string, Labels map[string]string) error {
	s.Lock()
	defer s.Unlock()
	targetCache, ok := s.cacheMap[name]
	if !ok {
		return nil
	}
	selector := &cache.KindLabelSelector{
		Kind:      Labels[label.LabelKind],
		Name:      Labels[label.LabelName],
		Namespace: Labels[label.LabelNamespace],
	}
	for _, resourceType := range resourceTypes {
		switch resourceType {
		case "service":
			services, err := targetCache.ListServices(selector)
			if err != nil {
				log.Errorw("failed to list services", zap.Error(err))
			}
			for _, service := range services {
				if err := targetCache.DeleteService(service); err != nil {
					log.Errorw("failed to delete service", zap.Error(err), zap.String("service", service.ID))
				}
			}
		case "ssl":
			ssls, err := targetCache.ListSSL(selector)
			if err != nil {
				log.Errorw("failed to list ssl", zap.Error(err))
			}
			for _, ssl := range ssls {
				if err := targetCache.DeleteSSL(ssl); err != nil {
					log.Errorw("failed to delete ssl", zap.Error(err), zap.String("ssl", ssl.ID))
				}
			}
		case "consumer":
			consumers, err := targetCache.ListConsumers(selector)
			if err != nil {
				log.Errorw("failed to list consumers", zap.Error(err))
			}
			for _, consumer := range consumers {
				if err := targetCache.DeleteConsumer(consumer); err != nil {
					log.Errorw("failed to delete consumer", zap.Error(err), zap.String("consumer", consumer.Username))
				}
			}
		case "global_rule":
			globalRules, err := targetCache.ListGlobalRules(selector)
			if err != nil {
				log.Errorw("failed to list global rules", zap.Error(err))
			}
			for _, globalRule := range globalRules {
				if err := targetCache.DeleteGlobalRule(globalRule); err != nil {
					log.Errorw("failed to delete global rule", zap.Error(err), zap.String("global rule", globalRule.ID))
				}
			}
		case "plugin_metadata":
			delete(s.pluginMetadataMap, name)
		}
	}
	if len(resourceTypes) == 0 {
		delete(s.cacheMap, name)
	}
	return nil
}

func (s *Store) GetResources(name string) (*adctypes.Resources, error) {
	s.Lock()
	defer s.Unlock()
	targetCache, ok := s.cacheMap[name]
	if !ok {
		return &adctypes.Resources{}, nil
	}
	var globalrule adctypes.GlobalRule
	var metadata adctypes.PluginMetadata
	// Get all global rules from cache and merge them
	globalRuleItems, _ := targetCache.ListGlobalRules()
	if len(globalRuleItems) > 0 {
		merged := make(adctypes.Plugins)
		for _, item := range globalRuleItems {
			for k, v := range item.Plugins {
				merged[k] = v
			}
		}
		globalrule = adctypes.GlobalRule(merged)
	}
	log.Debugw("get resources global rule items", zap.Any("globalRuleItems", globalRuleItems))
	if meta, ok := s.pluginMetadataMap[name]; ok {
		metadata = meta.DeepCopy()
	}
	consumers, _ := targetCache.ListConsumers()
	services, _ := targetCache.ListServices()
	ssls, _ := targetCache.ListSSL()
	return &adctypes.Resources{
		Consumers:      consumers,
		Services:       services,
		SSLs:           ssls,
		GlobalRules:    globalrule,
		PluginMetadata: metadata,
	}, nil
}
