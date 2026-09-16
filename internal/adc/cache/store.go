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
	"encoding/json"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/go-logr/logr"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	"github.com/apache/apisix-ingress-controller/internal/types"
)

type Store struct {
	cacheMap map[string]Cache
	// owners maps each service, ssl and consumer a cacheKey holds to the Kubernetes
	// resource that produced it, recorded from Insert's labels as it is written. A
	// global_rule row carries its own owner, and plugin_metadata always belongs to the
	// GatewayProxy the cacheKey names.
	owners map[string]map[entityKey]types.NamespacedNameKind

	// revision increases on every write that changes what the store holds. lastChange
	// holds, per cacheKey, the revision of each owner's latest change, and resetAt the
	// revision the cacheKey was last wiped at. A write that leaves an owner's content as
	// it was is not a change. See ChangedSince.
	revision   uint64
	lastChange map[string]map[types.NamespacedNameKind]uint64
	resetAt    map[string]uint64

	sync.Mutex
	log logr.Logger
}

type entityKey struct {
	resourceType string
	id           string
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
}

func NewStore(log logr.Logger) *Store {
	return &Store{
		cacheMap:   make(map[string]Cache),
		owners:     make(map[string]map[entityKey]types.NamespacedNameKind),
		lastChange: make(map[string]map[types.NamespacedNameKind]uint64),
		resetAt:    make(map[string]uint64),
		log:        log.WithName("store"),
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

func (s *Store) setOwner(name, resourceType, id string, owner types.NamespacedNameKind) {
	if s.owners[name] == nil {
		s.owners[name] = make(map[entityKey]types.NamespacedNameKind)
	}
	s.owners[name][entityKey{resourceType, id}] = owner
}

func (s *Store) removeOwner(name, resourceType, id string) {
	delete(s.owners[name], entityKey{resourceType, id})
}

func (s *Store) recordChange(name string, owner types.NamespacedNameKind) {
	s.revision++
	if s.lastChange[name] == nil {
		s.lastChange[name] = make(map[types.NamespacedNameKind]uint64)
	}
	s.lastChange[name][owner] = s.revision
}

// contentOf renders items, ordered by id, the way they reach ADC, so two writes can be
// compared by what they would push.
func contentOf[T any](items []T, id func(T) string) string {
	if len(items) == 0 {
		return ""
	}
	sorted := slices.Clone(items)
	slices.SortFunc(sorted, func(a, b T) int { return strings.Compare(id(a), id(b)) })
	return canonicalJSON(sorted)
}

// canonicalJSON renders v as JSON with every object's keys sorted. A stored object's
// plugin configs have been through a JSON round trip while a freshly translated one may
// still hold typed structs, whose fields marshal in declaration order rather than sorted.
func canonicalJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	var generic any
	if err := json.Unmarshal(b, &generic); err != nil {
		return string(b)
	}
	b, _ = json.Marshal(generic)
	return string(b)
}

func serviceID(service *adctypes.Service) string    { return service.ID }
func consumerID(consumer *adctypes.Consumer) string { return consumer.Username }
func sslID(ssl *adctypes.SSL) string                { return ssl.ID }

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

// Insert replaces the services, consumers and ssls the resource identified by labels
// contributes to the cacheKey name with the ones in resources. global_rule and
// plugin_metadata are written through SetGlobalRules and SetPluginMetadata instead.
func (s *Store) Insert(name string, resourceTypes []string, resources *adctypes.Resources, labels map[string]string) error {
	s.Lock()
	defer s.Unlock()
	targetCache, err := s.cacheFor(name)
	if err != nil {
		return err
	}
	s.log.V(1).Info("Inserting resources into cache", "name", name, "resourceTypes", resourceTypes, "Labels", labels)
	selector := &KindLabelSelector{
		Kind:      labels[label.LabelKind],
		Name:      labels[label.LabelName],
		Namespace: labels[label.LabelNamespace],
	}
	owner := ownerFromLabels(labels)
	changed := false
	defer func() {
		if changed {
			s.recordChange(name, owner)
		}
	}()
	for _, resourceType := range resourceTypes {
		switch resourceType {
		case adctypes.TypeService:
			services, err := targetCache.ListServices(selector)
			if err != nil {
				return err
			}
			changed = changed || contentOf(services, serviceID) != contentOf(resources.Services, serviceID)
			for _, service := range services {
				if err := targetCache.DeleteService(service); err != nil {
					return err
				}
				s.removeOwner(name, adctypes.TypeService, service.ID)
			}
			for _, service := range resources.Services {
				if err := targetCache.InsertService(service); err != nil {
					return err
				}
				s.setOwner(name, adctypes.TypeService, service.ID, owner)
			}
		case adctypes.TypeConsumer:
			consumers, err := targetCache.ListConsumers(selector)
			if err != nil {
				return err
			}
			changed = changed || contentOf(consumers, consumerID) != contentOf(resources.Consumers, consumerID)
			for _, consumer := range consumers {
				if err := targetCache.DeleteConsumer(consumer); err != nil {
					return err
				}
				s.removeOwner(name, adctypes.TypeConsumer, consumer.Username)
			}
			for _, consumer := range resources.Consumers {
				if err := targetCache.InsertConsumer(consumer); err != nil {
					return err
				}
				s.setOwner(name, adctypes.TypeConsumer, consumer.Username, owner)
			}
		case adctypes.TypeSSL:
			ssls, err := targetCache.ListSSL(selector)
			if err != nil {
				return err
			}
			changed = changed || contentOf(ssls, sslID) != contentOf(resources.SSLs, sslID)
			for _, ssl := range ssls {
				if err := targetCache.DeleteSSL(ssl); err != nil {
					return err
				}
				s.removeOwner(name, adctypes.TypeSSL, ssl.ID)
			}
			for _, ssl := range resources.SSLs {
				if err := targetCache.InsertSSL(ssl); err != nil {
					return err
				}
				s.setOwner(name, adctypes.TypeSSL, ssl.ID, owner)
			}
		}
	}
	return nil
}

// Delete removes the services, consumers and ssls the resource identified by labels
// contributes to the cacheKey name. An empty resourceTypes deletes nothing; see
// DeleteAll for wiping a whole cacheKey.
func (s *Store) Delete(name string, resourceTypes []string, labels map[string]string) error {
	s.Lock()
	defer s.Unlock()
	targetCache, ok := s.cacheMap[name]
	if !ok || len(resourceTypes) == 0 {
		return nil
	}
	selector := &KindLabelSelector{
		Kind:      labels[label.LabelKind],
		Name:      labels[label.LabelName],
		Namespace: labels[label.LabelNamespace],
	}
	changed := false
	defer func() {
		if changed {
			s.recordChange(name, ownerFromLabels(labels))
		}
	}()
	for _, resourceType := range resourceTypes {
		switch resourceType {
		case adctypes.TypeService:
			services, err := targetCache.ListServices(selector)
			if err != nil {
				s.log.Error(err, "failed to list services")
			}
			changed = changed || len(services) > 0
			for _, service := range services {
				if err := targetCache.DeleteService(service); err != nil {
					s.log.Error(err, "failed to delete service", "service", service.ID)
				}
				s.removeOwner(name, adctypes.TypeService, service.ID)
			}
		case adctypes.TypeSSL:
			ssls, err := targetCache.ListSSL(selector)
			if err != nil {
				s.log.Error(err, "failed to list ssl")
			}
			changed = changed || len(ssls) > 0
			for _, ssl := range ssls {
				if err := targetCache.DeleteSSL(ssl); err != nil {
					s.log.Error(err, "failed to delete ssl", "ssl", ssl.ID)
				}
				s.removeOwner(name, adctypes.TypeSSL, ssl.ID)
			}
		case adctypes.TypeConsumer:
			consumers, err := targetCache.ListConsumers(selector)
			if err != nil {
				s.log.Error(err, "failed to list consumers")
			}
			changed = changed || len(consumers) > 0
			for _, consumer := range consumers {
				if err := targetCache.DeleteConsumer(consumer); err != nil {
					s.log.Error(err, "failed to delete consumer", "consumer", consumer.Username)
				}
				s.removeOwner(name, adctypes.TypeConsumer, consumer.Username)
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
	delete(s.owners, name)
	delete(s.lastChange, name)
	s.revision++
	s.resetAt[name] = s.revision
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
	rows, err := targetCache.ListGlobalRules(&OwnerSelector{Owner: owner})
	if err != nil {
		return err
	}
	existing := make(adctypes.Plugins, len(rows))
	for _, row := range rows {
		existing[row.ID] = row.Config
	}
	if pluginsContent(existing) != pluginsContent(adctypes.Plugins(plugins)) {
		defer s.recordChange(name, owner)
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

// SetPluginMetadata replaces all plugin_metadata the cacheKey name holds.
func (s *Store) SetPluginMetadata(name string, metadata adctypes.PluginMetadata) error {
	s.Lock()
	defer s.Unlock()
	targetCache, err := s.cacheFor(name)
	if err != nil {
		return err
	}
	rows, err := targetCache.ListPluginMetadata()
	if err != nil {
		return err
	}
	existing := make(adctypes.Plugins, len(rows))
	for _, row := range rows {
		existing[row.ID] = row.Config
	}
	if gatewayProxy, ok := gatewayProxyOf(name); ok && pluginsContent(existing) != pluginsContent(adctypes.Plugins(metadata)) {
		defer s.recordChange(name, gatewayProxy)
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

// GetResources returns everything the cacheKey name holds, together with the store
// revision it was read at.
func (s *Store) GetResources(name string) (*adctypes.Resources, uint64, error) {
	s.Lock()
	defer s.Unlock()
	targetCache, ok := s.cacheMap[name]
	if !ok {
		return &adctypes.Resources{}, s.revision, nil
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
	}, s.revision, nil
}

// Lookup finds the top-level entity of resourceType and id the cacheKey name holds,
// along with the Kubernetes resource that produced it.
func (s *Store) Lookup(name, resourceType, id string) (Entity, bool) {
	s.Lock()
	defer s.Unlock()
	targetCache, ok := s.cacheMap[name]
	if !ok {
		return Entity{}, false
	}
	entity := Entity{Type: resourceType, ID: id, Name: id}
	switch resourceType {
	case adctypes.TypeService, adctypes.TypeSSL, adctypes.TypeConsumer:
		owner, ok := s.owners[name][entityKey{resourceType, id}]
		if !ok {
			return Entity{}, false
		}
		entity.Owner = owner
		if resourceType == adctypes.TypeService {
			if service, err := targetCache.GetService(id); err == nil && service.Name != "" {
				entity.Name = service.Name
			}
		}
		return entity, true
	case adctypes.TypeGlobalRule:
		row, err := targetCache.GetGlobalRule(id)
		if err != nil {
			return Entity{}, false
		}
		entity.Owner = row.Owner
		return entity, true
	case adctypes.TypePluginMetadata:
		if _, err := targetCache.GetPluginMetadata(id); err != nil {
			return Entity{}, false
		}
		gatewayProxy, ok := gatewayProxyOf(name)
		if !ok {
			return Entity{}, false
		}
		entity.Owner = gatewayProxy
		return entity, true
	}
	return Entity{}, false
}

// OwnedEntities lists every top-level entity owner produced in the cacheKey name.
func (s *Store) OwnedEntities(name string, owner types.NamespacedNameKind) []Entity {
	s.Lock()
	defer s.Unlock()
	targetCache, ok := s.cacheMap[name]
	if !ok {
		return nil
	}
	var entities []Entity
	for key, o := range s.owners[name] {
		if o == owner {
			entities = append(entities, Entity{Type: key.resourceType, ID: key.id, Name: key.id, Owner: owner})
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

func pluginsContent(plugins adctypes.Plugins) string {
	if len(plugins) == 0 {
		return ""
	}
	return canonicalJSON(plugins)
}

// Revision returns the store's current revision.
func (s *Store) Revision() uint64 {
	s.Lock()
	defer s.Unlock()
	return s.revision
}

// ChangedSince reports whether owner's content in the cacheKey name, or the cacheKey as a
// whole, changed after revision: a sync result built at revision then no longer describes
// what the store holds for owner.
func (s *Store) ChangedSince(name string, owner types.NamespacedNameKind, revision uint64) bool {
	s.Lock()
	defer s.Unlock()
	return s.lastChange[name][owner] > revision || s.resetAt[name] > revision
}

// OwnerChangedSince reports whether owner's content changed after revision in any
// cacheKey.
func (s *Store) OwnerChangedSince(owner types.NamespacedNameKind, revision uint64) bool {
	s.Lock()
	defer s.Unlock()
	return slices.ContainsFunc(slices.Collect(maps.Values(s.lastChange)), func(byOwner map[types.NamespacedNameKind]uint64) bool {
		return byOwner[owner] > revision
	})
}
