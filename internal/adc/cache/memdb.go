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
	"errors"

	"github.com/hashicorp/go-memdb"

	types "github.com/apache/apisix-ingress-controller/api/adc"
)

var (
	// ErrStillInUse means an object is still in use.
	ErrStillInUse = errors.New("still in use")
	// ErrNotFound is returned when the requested item is not found.
	ErrNotFound = memdb.ErrNotFound
)

type dbCache struct {
	db *memdb.MemDB
}

// NewMemDBCache creates a Cache object backs with a memory DB.
func NewMemDBCache() (Cache, error) {
	db, err := memdb.NewMemDB(_schema)
	if err != nil {
		return nil, err
	}
	return &dbCache{
		db: db,
	}, nil
}

func (c *dbCache) Insert(obj any) error {
	switch t := obj.(type) {
	case *types.SSL:
		return c.InsertSSL(t)
	case *types.Service:
		return c.InsertService(t)
	case *types.Consumer:
		return c.InsertConsumer(t)
	case *types.GlobalRuleItem:
		return c.InsertGlobalRule(t)
	default:
		return errors.New("unsupported type")
	}

}
func (c *dbCache) Delete(obj any) error {
	switch t := obj.(type) {
	case *types.Route:
		return c.DeleteRoute(t)
	case *types.SSL:
		return c.DeleteSSL(t)
	case *types.Service:
		return c.DeleteService(t)
	case *types.Consumer:
		return c.DeleteConsumer(t)
	case *types.GlobalRuleItem:
		return c.DeleteGlobalRule(t)
	default:
		return errors.New("unsupported type")
	}
}

func (c *dbCache) InsertRoute(r *types.Route) error {
	route := r.DeepCopy()
	return c.insert("route", route)
}

func (c *dbCache) InsertSSL(ssl *types.SSL) error {
	return c.insert("ssl", ssl.DeepCopy())
}

func (c *dbCache) InsertService(u *types.Service) error {
	return c.insert("service", u.DeepCopy())
}

func (c *dbCache) InsertConsumer(consumer *types.Consumer) error {
	return c.insert("consumer", consumer.DeepCopy())
}

func (c *dbCache) InsertGlobalRule(globalRule *types.GlobalRuleItem) error {
	return c.insert("global_rule", globalRule.DeepCopy())
}

func (c *dbCache) insert(table string, obj any) error {
	txn := c.db.Txn(true)
	defer txn.Abort()
	if err := txn.Insert(table, obj); err != nil {
		return err
	}
	txn.Commit()
	return nil
}

func (c *dbCache) GetRoute(id string) (*types.Route, error) {
	obj, err := c.get("route", id)
	if err != nil {
		return nil, err
	}
	return obj.(*types.Route).DeepCopy(), nil
}

func (c *dbCache) GetSSL(id string) (*types.SSL, error) {
	obj, err := c.get("ssl", id)
	if err != nil {
		return nil, err
	}
	return obj.(*types.SSL).DeepCopy(), nil
}

func (c *dbCache) GetService(id string) (*types.Service, error) {
	obj, err := c.get("service", id)
	if err != nil {
		return nil, err
	}
	return obj.(*types.Service).DeepCopy(), nil
}

func (c *dbCache) GetConsumer(username string) (*types.Consumer, error) {
	obj, err := c.get("consumer", username)
	if err != nil {
		return nil, err
	}
	return obj.(*types.Consumer).DeepCopy(), nil
}

func (c *dbCache) GetGlobalRule(id string) (*types.GlobalRuleItem, error) {
	obj, err := c.get("global_rule", id)
	if err != nil {
		return nil, err
	}
	return obj.(*types.GlobalRuleItem).DeepCopy(), nil
}

func (c *dbCache) GetStreamRoute(id string) (*types.StreamRoute, error) {
	obj, err := c.get("stream_route", id)
	if err != nil {
		return nil, err
	}
	return obj.(*types.StreamRoute).DeepCopy(), nil
}

func (c *dbCache) get(table, id string) (any, error) {
	txn := c.db.Txn(false)
	defer txn.Abort()
	obj, err := txn.First(table, "id", id)
	if err != nil {
		if err == memdb.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if obj == nil {
		return nil, ErrNotFound
	}
	return obj, nil
}

func (c *dbCache) ListRoutes(opts ...ListOption) ([]*types.Route, error) {
	raws, err := c.list("route", opts...)
	if err != nil {
		return nil, err
	}
	routes := make([]*types.Route, 0, len(raws))
	for _, raw := range raws {
		routes = append(routes, raw.(*types.Route).DeepCopy())
	}
	return routes, nil
}

func (c *dbCache) ListSSL(opts ...ListOption) ([]*types.SSL, error) {
	raws, err := c.list("ssl", opts...)
	if err != nil {
		return nil, err
	}
	ssl := make([]*types.SSL, 0, len(raws))
	for _, raw := range raws {
		ssl = append(ssl, raw.(*types.SSL).DeepCopy())
	}
	return ssl, nil
}

func (c *dbCache) ListServices(opts ...ListOption) ([]*types.Service, error) {
	raws, err := c.list("service", opts...)
	if err != nil {
		return nil, err
	}
	services := make([]*types.Service, 0, len(raws))
	for _, raw := range raws {
		services = append(services, raw.(*types.Service).DeepCopy())
	}
	return services, nil
}

func (c *dbCache) ListConsumers(opts ...ListOption) ([]*types.Consumer, error) {
	raws, err := c.list("consumer", opts...)
	if err != nil {
		return nil, err
	}
	consumers := make([]*types.Consumer, 0, len(raws))
	for _, raw := range raws {
		consumers = append(consumers, raw.(*types.Consumer).DeepCopy())
	}
	return consumers, nil
}

func (c *dbCache) ListGlobalRules(opts ...ListOption) ([]*types.GlobalRuleItem, error) {
	raws, err := c.list("global_rule", opts...)
	if err != nil {
		return nil, err
	}
	globalRules := make([]*types.GlobalRuleItem, 0, len(raws))
	for _, raw := range raws {
		globalRules = append(globalRules, raw.(*types.GlobalRuleItem).DeepCopy())
	}
	return globalRules, nil
}

func (c *dbCache) list(table string, opts ...ListOption) ([]any, error) {
	txn := c.db.Txn(false)
	defer txn.Abort()
	listOpts := &ListOptions{}
	listOpts.ApplyOptions(opts)
	index := "id"
	var args []any
	if listOpts.KindLabelSelector != nil {
		index = KindLabelIndex
		args = []any{listOpts.KindLabelSelector.Kind, listOpts.KindLabelSelector.Namespace, listOpts.KindLabelSelector.Name}
	}
	iter, err := txn.Get(table, index, args...)
	if err != nil {
		return nil, err
	}
	var objs []any
	for obj := iter.Next(); obj != nil; obj = iter.Next() {
		objs = append(objs, obj)
	}
	return objs, nil
}

func (c *dbCache) DeleteRoute(r *types.Route) error {
	return c.delete("route", r)
}

func (c *dbCache) DeleteSSL(ssl *types.SSL) error {
	return c.delete("ssl", ssl)
}

func (c *dbCache) DeleteService(u *types.Service) error {
	return c.delete("service", u)
}

func (c *dbCache) DeleteConsumer(consumer *types.Consumer) error {
	return c.delete("consumer", consumer)
}

func (c *dbCache) DeleteGlobalRule(globalRule *types.GlobalRuleItem) error {
	return c.delete("global_rule", globalRule)
}

func (c *dbCache) delete(table string, obj any) error {
	txn := c.db.Txn(true)
	defer txn.Abort()
	if err := txn.Delete(table, obj); err != nil {
		if err == memdb.ErrNotFound {
			return ErrNotFound
		}
		return err
	}
	txn.Commit()
	return nil
}
