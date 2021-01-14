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

package cache

import (
	"errors"

	"github.com/hashicorp/go-memdb"

	v1 "github.com/api7/ingress-controller/pkg/types/apisix/v1"
)

var (
	_allowedNamespace = map[string]struct{}{
		"route":    {},
		"service":  {},
		"ssl":      {},
		"upstream": {},
	}
)

type dbCache struct {
	db *memdb.MemDB
}

// NewMemDBCache creates a Cache object backs with a memory DB.
func NewMemDBCache() (NamespacingCache, error) {
	db, err := memdb.NewMemDB(_schema)
	if err != nil {
		return nil, err
	}
	return &dbCache{
		db: db,
	}, nil
}

func (c *dbCache) checkNamespace(ns string) error {
	_, ok := _allowedNamespace[ns]
	if !ok {
		return errors.New("invalid namespace")
	}
	return nil
}

func (c *dbCache) Insert(namespace string, obj interface{}) error {
	if err := c.checkNamespace(namespace); err != nil {
		return err
	}
	if err := c.checkObject(namespace, obj); err != nil {
		return err
	}
	txn := c.db.Txn(true)
	defer txn.Abort()
	if err := txn.Insert(namespace, obj); err != nil {
		return err
	}
	txn.Commit()
	return nil
}

func (c *dbCache) Get(namespace, key string) (interface{}, error) {
	if err := c.checkNamespace(namespace); err != nil {
		return nil, err
	}
	txn := c.db.Txn(false)
	defer txn.Abort()
	obj, err := txn.First(namespace, "id", key)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (c *dbCache) List(namespace string) ([]interface{}, error) {
	if err := c.checkNamespace(namespace); err != nil {
		return nil, err
	}
	txn := c.db.Txn(false)
	defer txn.Abort()
	iter, err := txn.Get(namespace, "id")
	if err != nil {
		return nil, err
	}
	var objs []interface{}
	for obj := iter.Next(); obj != nil; obj = iter.Next() {
		objs = append(objs, obj)
	}
	return objs, nil
}

func (c *dbCache) Delete(namespace string, obj interface{}) error {
	if err := c.checkNamespace(namespace); err != nil {
		return err
	}
	if err := c.checkObject(namespace, obj); err != nil {
		return err
	}
	txn := c.db.Txn(true)
	defer txn.Abort()
	if err := txn.Delete(namespace, obj); err != nil {
		return err
	}
	txn.Commit()
	return nil
}

func (c *dbCache) checkObject(namespace string, obj interface{}) error {
	var ok bool
	switch namespace {
	case "route":
		_, ok = obj.(*v1.Route)
	case "upstream":
		_, ok = obj.(*v1.Upstream)
	case "ssl":
		_, ok = obj.(*v1.Ssl)
	case "service":
		_, ok = obj.(*v1.Service)
	default:
		panic("invalid namespace")
	}
	if !ok {
		return errors.New("bad resource")
	}
	return nil
}

func (c *dbCache) checkReference(namespace string, obj interface{}) (bool, error) {
	switch namespace {
	case "route":
		return true, nil
	case "ssl":
		return true, nil
	case "service":
		return c.checkServiceReference(obj.(*v1.Service))
	case "upstream":
		return c.checkUpstreamReference(obj.(*v1.Upstream))
	default:
		// Impossible to here
		panic("invalid namespace")
	}
}

func (c *dbCache) checkServiceReference(s *v1.Service) (bool, error) {
	// Service is referenced by Route.
	txn := c.db.Txn(false)
	defer txn.Abort()
	_, err := txn.First("route", "service_id", s.FullName)
	if err != nil {
		if err == memdb.ErrNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (c *dbCache) checkUpstreamReference(u *v1.Upstream) (bool, error) {
	// Upstream is referenced by Service.
	txn := c.db.Txn(false)
	defer txn.Abort()
	_, err := txn.First("service", "upstream_id", u.FullName)
	if err != nil {
		if err == memdb.ErrNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
