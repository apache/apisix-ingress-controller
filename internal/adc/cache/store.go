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
	"sync"

	"github.com/go-logr/logr"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	"github.com/apache/apisix-ingress-controller/internal/types"
)

type Store struct {
	cacheMap map[string]Cache

	sync.Mutex
	log logr.Logger
}

// Entity is a top-level ADC resource a cacheKey holds: a service, ssl, consumer,
// global_rule or plugin_metadata.
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
		cacheMap: make(map[string]Cache),
		log:      log.WithName("store"),
	}
}

func (s *Store) cacheFor(name string) (Cache, error) {
	if c, ok := s.cacheMap[name]; ok {
		return c, nil
	}
	db, err := NewMemDBCache()
	if err != nil {
		return nil, err
	}
	s.cacheMap[name] = db
	return db, nil
}

func ownerFromLabels(labels map[string]string) types.NamespacedNameKind {
	return types.NamespacedNameKind{
		Kind:      labels[label.LabelKind],
		Namespace: labels[label.LabelNamespace],
		Name:      labels[label.LabelName],
	}
}

// gatewayProxyOf returns the GatewayProxy a cacheKey names.
func gatewayProxyOf(name string) (types.NamespacedNameKind, bool) {
	var gatewayProxy types.NamespacedNameKind
	if err := gatewayProxy.FromString(name); err != nil {
		return types.NamespacedNameKind{}, false
	}
	return gatewayProxy, true
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

// routeOwner finds the Kubernetes resource that produced the route id, by scanning
// every service the cacheKey holds. A route's own labels can differ from its service's
// (a traffic-split service can combine rules several ApisixRoutes each contributed),
// so this can't reuse the service's own KindLabelSelector match.
func routeOwner(targetCache Cache, id string) (types.NamespacedNameKind, bool) {
	services, err := targetCache.ListServices()
	if err != nil {
		return types.NamespacedNameKind{}, false
	}
	for _, service := range services {
		for _, route := range service.Routes {
			if route.ID == id {
				return ownerFromLabels(route.GetLabels()), true
			}
		}
	}
	return types.NamespacedNameKind{}, false
}

// setGlobalRules is Insert and SetGlobalRules' shared implementation. Callers must
// already hold s.Lock.
func (s *Store) setGlobalRules(targetCache Cache, owner types.NamespacedNameKind, plugins adctypes.GlobalRule) error {
	rows, err := targetCache.ListGlobalRules(&OwnerSelector{Owner: owner})
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err := targetCache.DeleteGlobalRule(row); err != nil {
			return err
		}
	}
	for pluginName, config := range plugins {
		if err := targetCache.InsertGlobalRule(&GlobalRuleRow{ID: pluginName, Owner: owner, Config: config}); err != nil {
			return err
		}
	}
	return nil
}

// setPluginMetadata is Insert and SetPluginMetadata's shared implementation. Callers
// must already hold s.Lock.
func (s *Store) setPluginMetadata(targetCache Cache, metadata adctypes.PluginMetadata) error {
	rows, err := targetCache.ListPluginMetadata()
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err := targetCache.DeletePluginMetadata(row); err != nil {
			return err
		}
	}
	for pluginName, config := range metadata {
		if err := targetCache.InsertPluginMetadata(&PluginMetadataRow{ID: pluginName, Config: config}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) Insert(name string, resourceTypes []string, resources *adctypes.Resources, Labels map[string]string) error {
	s.Lock()
	defer s.Unlock()
	targetCache, err := s.cacheFor(name)
	if err != nil {
		return err
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
			if err := s.setGlobalRules(targetCache, ownerFromLabels(Labels), resources.GlobalRules); err != nil {
				return err
			}
		case adctypes.TypePluginMetadata:
			if err := s.setPluginMetadata(targetCache, resources.PluginMetadata); err != nil {
				return err
			}
		default:
			continue
		}
	}
	return nil
}

// Delete removes the services, consumers, ssls and global_rules the resource identified
// by Labels contributes to the cacheKey name. An empty resourceTypes deletes nothing;
// see DeleteAll for wiping a whole cacheKey.
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
			if err := s.setGlobalRules(targetCache, ownerFromLabels(Labels), nil); err != nil {
				s.log.Error(err, "failed to delete global rules")
			}
		case adctypes.TypePluginMetadata:
			if err := s.setPluginMetadata(targetCache, nil); err != nil {
				s.log.Error(err, "failed to delete plugin metadata")
			}
		}
	}
	return nil
}

// DeleteAll wipes everything the cacheKey name holds.
func (s *Store) DeleteAll(name string) {
	s.Lock()
	defer s.Unlock()
	delete(s.cacheMap, name)
}

func (s *Store) GetResources(name string) (*adctypes.Resources, error) {
	s.Lock()
	defer s.Unlock()
	targetCache, ok := s.cacheMap[name]
	if !ok {
		return &adctypes.Resources{}, nil
	}
	var globalRules adctypes.GlobalRule
	globalRuleRows, _ := targetCache.ListGlobalRules()
	if len(globalRuleRows) > 0 {
		globalRules = make(adctypes.GlobalRule, len(globalRuleRows))
		for _, row := range globalRuleRows {
			globalRules[row.ID] = row.Config
		}
	}
	var pluginMetadata adctypes.PluginMetadata
	pluginMetadataRows, _ := targetCache.ListPluginMetadata()
	if len(pluginMetadataRows) > 0 {
		pluginMetadata = make(adctypes.PluginMetadata, len(pluginMetadataRows))
		for _, row := range pluginMetadataRows {
			pluginMetadata[row.ID] = row.Config
		}
	}
	consumers, _ := targetCache.ListConsumers()
	services, _ := targetCache.ListServices()
	ssls, _ := targetCache.ListSSL()
	return &adctypes.Resources{
		Consumers:      consumers,
		Services:       services,
		SSLs:           ssls,
		GlobalRules:    globalRules,
		PluginMetadata: pluginMetadata,
	}, nil
}

// SetGlobalRules replaces the global_rules plugins owner declares in the cacheKey name
// with plugins; an empty plugins removes all of them. A plugin name can only be
// configured once, so another owner's plugin of the same name is overwritten: which
// declaration wins is undefined.
func (s *Store) SetGlobalRules(name string, owner types.NamespacedNameKind, plugins adctypes.GlobalRule) error {
	s.Lock()
	defer s.Unlock()
	targetCache, err := s.cacheFor(name)
	if err != nil {
		return err
	}
	return s.setGlobalRules(targetCache, owner, plugins)
}

// SetPluginMetadata replaces all plugin_metadata the cacheKey name holds.
func (s *Store) SetPluginMetadata(name string, metadata adctypes.PluginMetadata) error {
	s.Lock()
	defer s.Unlock()
	targetCache, err := s.cacheFor(name)
	if err != nil {
		return err
	}
	return s.setPluginMetadata(targetCache, metadata)
}

// Lookup finds the top-level entity of resourceType and id the cacheKey name holds,
// along with the Kubernetes resource that produced it, read straight from the entity's
// own stored labels (the same ones KindLabelSelector matches Insert/Delete against, so
// there is exactly one place this can ever disagree with what a selector would find). A
// route is not top-level, but is looked up the same way pending its own nested-entity
// tracking.
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
	case adctypes.TypeRoute:
		owner, ok := routeOwner(targetCache, id)
		if !ok {
			return Entity{}, false
		}
		return Entity{Type: resourceType, ID: id, Name: id, Owner: owner}, true
	case adctypes.TypeGlobalRule:
		row, err := targetCache.GetGlobalRule(id)
		if err != nil {
			return Entity{}, false
		}
		return Entity{Type: resourceType, ID: id, Name: id, Owner: row.Owner}, true
	case adctypes.TypePluginMetadata:
		if _, err := targetCache.GetPluginMetadata(id); err != nil {
			return Entity{}, false
		}
		gatewayProxy, ok := gatewayProxyOf(name)
		if !ok {
			return Entity{}, false
		}
		return Entity{Type: resourceType, ID: id, Name: id, Owner: gatewayProxy}, true
	}
	return Entity{}, false
}

// OwnedEntities lists every top-level entity owner produced in the cacheKey name, via
// the same KindLabelSelector index Insert and Delete already use for service, ssl and
// consumer.
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
	rows, _ := targetCache.ListGlobalRules(&OwnerSelector{Owner: owner})
	for _, row := range rows {
		entities = append(entities, Entity{Type: adctypes.TypeGlobalRule, ID: row.ID, Name: row.ID, Owner: owner})
	}
	if gatewayProxy, ok := gatewayProxyOf(name); ok && gatewayProxy == owner {
		rows, _ := targetCache.ListPluginMetadata()
		for _, row := range rows {
			entities = append(entities, Entity{Type: adctypes.TypePluginMetadata, ID: row.ID, Name: row.ID, Owner: owner})
		}
	}
	return entities
}
