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
	Service = "Service"
)

type ServiceRequest struct {
	Group      string
	Name       string
	FullName   string
	UpstreamId string
}

func (sr *ServiceRequest) FindByName() (*v1.Service, error) {
	txn := DB.Txn(false)
	defer txn.Abort()
	if raw, err := txn.First(Service, "id", sr.FullName); err != nil {
		return nil, err
	} else {
		if raw != nil {
			currentService := raw.(*v1.Service)
			return currentService, nil
		}
		return nil, utils.ErrNotFound
	}
}

func (db *ServiceDB) Insert() error {
	txn := DB.Txn(true)
	defer txn.Abort()
	for _, s := range db.Services {
		if err := txn.Insert(Service, s); err != nil {
			return err
		}
	}
	txn.Commit()
	return nil
}

type ServiceDB struct {
	Services []*v1.Service
}

func (db *ServiceDB) UpdateService() error {
	txn := DB.Txn(true)
	defer txn.Abort()
	for _, s := range db.Services {
		// 1. delete
		if _, err := txn.DeleteAll(Service, "id", *(s.FullName)); err != nil {
			return err
		}
		// 2. insert
		if err := txn.Insert(Service, s); err != nil {
			return err
		}
	}

	txn.Commit()
	return nil
}

func (db *ServiceDB) DeleteService() error {
	txn := DB.Txn(true)
	defer txn.Abort()
	for _, r := range db.Services {
		if _, err := txn.DeleteAll(Service, "id", *(r.FullName)); err != nil {
			return err
		}
	}
	txn.Commit()
	return nil
}

func (rr *ServiceRequest) ExistByUpstreamId() (*v1.Service, error) {
	txn := DB.Txn(false)
	defer txn.Abort()
	if raw, err := txn.First(Service, "upstream_id", rr.UpstreamId); err != nil {
		return nil, err
	} else {
		if raw != nil {
			firstService := raw.(*v1.Service)
			return firstService, nil
		}
		return nil, utils.ErrNotFound
	}
}

var serviceSchema = &memdb.TableSchema{
	Name: Service,
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
		"upstream_id": {
			Name:         "upstream_id",
			Unique:       false,
			Indexer:      &memdb.StringFieldIndex{Field: "UpstreamId"},
			AllowMissing: true,
		},
	},
}
