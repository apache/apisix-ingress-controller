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

package cache

import (
	"cmp"
	"fmt"
	"sync"

	"github.com/go-logr/logr"
	"github.com/google/uuid"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	"github.com/apache/apisix-ingress-controller/internal/types"
)

type Store struct {
	cacheMap          map[string]Cache
	pluginMetadataMap map[string]adctypes.PluginMetadata

	sync.Mutex
	log logr.Logger
}

// Entity is a top-level ADC resource a cacheKey holds: a service, ssl or consumer.
type Entity struct {
	Type string
	ID   string
	// Name identifies the entity in a status message: a service's name, a consumer's
	// username, or the id for the other types.
	Name  string
	Owner types.NamespacedNameKind
	// Children are the routes and stream routes a service holds. A service whose
	// children are all dropped serves nothing, even though the service itself was never
	// rejected.
	Children []Entity
}

func NewStore(log logr.Logger) *Store {
	return &Store{
		cacheMap:          make(map[string]Cache),
		pluginMetadataMap: make(map[string]adctypes.PluginMetadata),
		log:               log.WithName("store"),
	}
}

func ownerFromLabels(labels map[string]string) types.NamespacedNameKind {
	return types.NamespacedNameKind{
		Kind:      labels[label.LabelKind],
		Namespace: labels[label.LabelNamespace],
		Name:      labels[label.LabelName],
	}
}

func childrenOf(service *adctypes.Service, owner types.NamespacedNameKind) []Entity {
	children := make([]Entity, 0, len(service.Routes)+len(service.StreamRoutes))
	for _, route := range service.Routes {
		children = append(children, Entity{Type: adctypes.TypeRoute, ID: route.ID, Name: cmp.Or(route.Name, route.ID), Owner: owner})
	}
	for _, streamRoute := range service.StreamRoutes {
		children = append(children, Entity{Type: adctypes.TypeStreamRoute, ID: streamRoute.ID, Name: cmp.Or(streamRoute.Name, streamRoute.ID), Owner: owner})
	}
	return children
}

func (s *Store) Insert(name string, resourceTypes []string, resources *adctypes.Resources, Labels map[string]string) error {
	s.Lock()
	defer s.Unlock()
	targetCache, ok := s.cacheMap[name]
	if !ok {
		db, err := NewMemDBCache()
		if err != nil {
			return err
		}
		s.cacheMap[name] = db
		targetCache = s.cacheMap[name]
	}
	s.log.V(1).Info("Inserting resources into cache", "name", name, "resourceTypes", resourceTypes, "Labels", Labels)
	selector := &KindLabelSelector{
		Kind:      Labels[label.LabelKind],
		Name:      Labels[label.LabelName],
		Namespace: Labels[label.LabelNamespace],
	}
	for _, resourceType := range resourceTypes {
		switch resourceType {
		case adctypes.TypeService:
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
		case adctypes.TypeConsumer:
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
		case adctypes.TypeSSL:
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
		case adctypes.TypeGlobalRule:
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
		case adctypes.TypePluginMetadata:
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
	selector := &KindLabelSelector{
		Kind:      Labels[label.LabelKind],
		Name:      Labels[label.LabelName],
		Namespace: Labels[label.LabelNamespace],
	}
	for _, resourceType := range resourceTypes {
		switch resourceType {
		case adctypes.TypeService:
			services, err := targetCache.ListServices(selector)
			if err != nil {
				s.log.Error(err, "failed to list services")
			}
			for _, service := range services {
				if err := targetCache.DeleteService(service); err != nil {
					s.log.Error(err, "failed to delete service", "service", service.ID)
				}
			}
		case adctypes.TypeSSL:
			ssls, err := targetCache.ListSSL(selector)
			if err != nil {
				s.log.Error(err, "failed to list ssl")
			}
			for _, ssl := range ssls {
				if err := targetCache.DeleteSSL(ssl); err != nil {
					s.log.Error(err, "failed to delete ssl", "ssl", ssl.ID)
				}
			}
		case adctypes.TypeConsumer:
			consumers, err := targetCache.ListConsumers(selector)
			if err != nil {
				s.log.Error(err, "failed to list consumers")
			}
			for _, consumer := range consumers {
				if err := targetCache.DeleteConsumer(consumer); err != nil {
					s.log.Error(err, "failed to delete consumer", "consumer", consumer.Username)
				}
			}
		case adctypes.TypeGlobalRule:
			globalRules, err := targetCache.ListGlobalRules(selector)
			if err != nil {
				s.log.Error(err, "failed to list global rules")
			}
			for _, globalRule := range globalRules {
				if err := targetCache.DeleteGlobalRule(globalRule); err != nil {
					s.log.Error(err, "failed to delete global rule", "global rule", globalRule.ID)
				}
			}
		case adctypes.TypePluginMetadata:
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
	s.log.V(1).Info("GetResources fetched global rule items", "itemCount", len(globalRuleItems), "pluginCount", len(globalrule))
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

func (s *Store) ListGlobalRules(name string) ([]*adctypes.GlobalRuleItem, error) {
	s.Lock()
	defer s.Unlock()
	targetCache, ok := s.cacheMap[name]
	if !ok {
		return nil, fmt.Errorf("cache not found for name: %s", name)
	}
	globalRules, err := targetCache.ListGlobalRules()
	if err != nil {
		return nil, fmt.Errorf("failed to list global rules: %w", err)
	}
	return globalRules, nil
}

func (s *Store) GetResourceLabel(name, resourceType string, id string) (map[string]string, error) {
	s.Lock()
	defer s.Unlock()
	targetCache, ok := s.cacheMap[name]
	if !ok {
		return nil, fmt.Errorf("cache not found for name: %s", name)
	}
	switch resourceType {
	case adctypes.TypeService:
		service, err := targetCache.GetService(id)
		if err != nil {
			return nil, fmt.Errorf("failed to get service: %w", err)
		}
		return service.Labels, nil
	case adctypes.TypeRoute:
		services, err := targetCache.ListServices()
		if err != nil {
			return nil, fmt.Errorf("failed to list services: %w", err)
		}
		for _, service := range services {
			for _, route := range service.Routes {
				if route.ID == id {
					// Return labels from the service that contains the route
					return route.GetLabels(), nil
				}
			}
		}
		return nil, fmt.Errorf("route not found: %s", id)
	case adctypes.TypeSSL:
		ssl, err := targetCache.GetSSL(id)
		if err != nil {
			return nil, err
		}
		if ssl != nil {
			return ssl.GetLabels(), nil
		}
	case adctypes.TypeConsumer:
		consumer, err := targetCache.GetConsumer(id)
		if err != nil {
			return nil, err
		}
		if consumer != nil {
			return consumer.Labels, nil
		}
	case adctypes.TypeGlobalRule:
		globalRule, err := targetCache.GetGlobalRule(id)
		if err != nil {
			return nil, err
		}
		if globalRule != nil {
			return globalRule.GetLabels(), nil
		}
	default:
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}
	return nil, nil
}

// Lookup finds the top-level entity of resourceType and id the cacheKey name holds,
// along with the Kubernetes resource that produced it, read straight from the entity's
// own stored labels (the same ones KindLabelSelector matches Insert/Delete against, so
// there is exactly one place this can ever disagree with what a selector would find). It
// covers service, ssl and consumer; global_rule and plugin_metadata still only expose
// GetResourceLabel, until their own storage grows the same per-entity owner this does.
func (s *Store) Lookup(name, resourceType, id string) (Entity, bool) {
	s.Lock()
	defer s.Unlock()
	targetCache, ok := s.cacheMap[name]
	if !ok {
		return Entity{}, false
	}
	switch resourceType {
	case adctypes.TypeService:
		service, err := targetCache.GetService(id)
		if err != nil {
			return Entity{}, false
		}
		return Entity{Type: resourceType, ID: id, Name: cmp.Or(service.Name, id), Owner: ownerFromLabels(service.GetLabels())}, true
	case adctypes.TypeSSL:
		ssl, err := targetCache.GetSSL(id)
		if err != nil {
			return Entity{}, false
		}
		return Entity{Type: resourceType, ID: id, Name: id, Owner: ownerFromLabels(ssl.GetLabels())}, true
	case adctypes.TypeConsumer:
		consumer, err := targetCache.GetConsumer(id)
		if err != nil {
			return Entity{}, false
		}
		return Entity{Type: resourceType, ID: id, Name: id, Owner: ownerFromLabels(consumer.GetLabels())}, true
	}
	return Entity{}, false
}

// OwnedEntities lists every service, ssl and consumer owner produced in the cacheKey
// name, via the same KindLabelSelector index Insert and Delete already use to find an
// owner's resources. See Lookup for why global_rule and plugin_metadata are absent.
func (s *Store) OwnedEntities(name string, owner types.NamespacedNameKind) []Entity {
	s.Lock()
	defer s.Unlock()
	targetCache, ok := s.cacheMap[name]
	if !ok {
		return nil
	}
	selector := &KindLabelSelector{Kind: owner.Kind, Namespace: owner.Namespace, Name: owner.Name}

	var entities []Entity
	if services, err := targetCache.ListServices(selector); err == nil {
		for _, service := range services {
			entities = append(entities, Entity{
				Type:     adctypes.TypeService,
				ID:       service.ID,
				Name:     cmp.Or(service.Name, service.ID),
				Owner:    owner,
				Children: childrenOf(service, owner),
			})
		}
	}
	if ssls, err := targetCache.ListSSL(selector); err == nil {
		for _, ssl := range ssls {
			entities = append(entities, Entity{Type: adctypes.TypeSSL, ID: ssl.ID, Name: ssl.ID, Owner: owner})
		}
	}
	if consumers, err := targetCache.ListConsumers(selector); err == nil {
		for _, consumer := range consumers {
			entities = append(entities, Entity{Type: adctypes.TypeConsumer, ID: consumer.Username, Name: consumer.Username, Owner: owner})
		}
	}
	return entities
}
