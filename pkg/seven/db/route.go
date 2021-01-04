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
package db

import (
	"github.com/hashicorp/go-memdb"

	"github.com/api7/ingress-controller/pkg/seven/utils"
	"github.com/api7/ingress-controller/pkg/types/apisix/v1"
)

const (
	Route = "Route"
)

type RouteRequest struct {
	Group    string
	Name     string
	FullName string
}

func (rr *RouteRequest) FindByName() (*v1.Route, error) {
	txn := DB.Txn(false)
	defer txn.Abort()
	if raw, err := txn.First(Route, "id", rr.FullName); err != nil {
		return nil, err
	} else {
		if raw != nil {
			currentRoute := raw.(*v1.Route)
			return currentRoute, nil
		}
		return nil, utils.NotFound
	}
}

type RouteDB struct {
	Routes []*v1.Route
}

// InsertRoute insert route to memDB
func (db *RouteDB) Insert() error {
	txn := DB.Txn(true)
	defer txn.Abort()
	for _, r := range db.Routes {
		if err := txn.Insert(Route, r); err != nil {
			return err
		}
	}
	txn.Commit()
	return nil
}

func (db *RouteDB) UpdateRoute() error {
	txn := DB.Txn(true)
	defer txn.Abort()
	for _, r := range db.Routes {
		// 1. delete
		if _, err := txn.DeleteAll(Route, "id", *(r.FullName)); err != nil {
			return err
		}
		// 2. insert
		if err := txn.Insert(Route, r); err != nil {
			return err
		}
	}
	txn.Commit()
	return nil
}

func (db *RouteDB) DeleteRoute() error {
	txn := DB.Txn(true)
	defer txn.Abort()
	for _, r := range db.Routes {
		//if _, err := txn.DeleteAll(Route, "id", *(r.ID)); err != nil {
		if _, err := txn.DeleteAll(Route, "id", *(r.FullName)); err != nil {
			return err
		}
	}
	txn.Commit()
	return nil
}

var routeSchema = &memdb.TableSchema{
	Name: Route,
	Indexes: map[string]*memdb.IndexSchema{
		"id": {
			Name:    "id",
			Unique:  true,
			Indexer: &memdb.StringFieldIndex{Field: "FullName"},
		},
		"name": {
			Name:         "name",
			Unique:       true,
			Indexer:      &memdb.StringFieldIndex{Field: "Name"},
			AllowMissing: true,
		},
	},
}
