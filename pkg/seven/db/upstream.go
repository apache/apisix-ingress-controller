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
	Upstream = "Upstream"
)

type UpstreamDB struct {
	Upstreams []*v1.Upstream
}

type UpstreamRequest struct {
	Group    string
	Name     string
	FullName string
}

func (ur *UpstreamRequest) FindByName() (*v1.Upstream, error) {
	txn := DB.Txn(false)
	defer txn.Abort()
	if raw, err := txn.First(Upstream, "id", ur.FullName); err != nil {
		return nil, err
	} else {
		if raw != nil {
			currentUpstream := raw.(*v1.Upstream)
			return currentUpstream, nil
		}
		return nil, utils.NotFound
	}
}

// insertUpstream insert upstream to memDB
func (upstreamDB *UpstreamDB) InsertUpstreams() error {
	txn := DB.Txn(true)
	defer txn.Abort()
	for _, u := range upstreamDB.Upstreams {
		if err := txn.Insert(Upstream, u); err != nil {
			return err
		}
	}
	txn.Commit()
	return nil
}

func (upstreamDB *UpstreamDB) UpdateUpstreams() error {
	txn := DB.Txn(true)
	defer txn.Abort()
	for _, u := range upstreamDB.Upstreams {
		// delete
		if _, err := txn.DeleteAll(Upstream, "id", *(u.FullName)); err != nil {
			return err
		}
		// insert
		if err := txn.Insert(Upstream, u); err != nil {
			return err
		}
	}
	txn.Commit()
	return nil
}

var upstreamSchema = &memdb.TableSchema{
	Name: Upstream,
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

//func indexer() *memdb.CompoundMultiIndex{
//	var idx = make([]memdb.Indexer, 0)
//	idx = append(idx, &memdb.StringFieldIndex{Field: "Group"})
//	idx = append(idx, &memdb.StringFieldIndex{Field: "Name"})
//	return &memdb.CompoundMultiIndex{Indexes: idx, AllowMissing: false}
//}
