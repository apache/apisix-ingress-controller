// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
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
	"github.com/hashicorp/go-memdb"
)

var (
	_schema = &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			"service": {
				Name: "service",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "ID"},
					},
					"name": {
						Name:         "name",
						Unique:       true,
						Indexer:      &memdb.StringFieldIndex{Field: "Name"},
						AllowMissing: true,
					},
					KindLabelIndex: {
						Name:         KindLabelIndex,
						Unique:       false,
						AllowMissing: true,
						Indexer:      &KindLabelIndexer,
					},
				},
			},
			"ssl": {
				Name: "ssl",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "ID"},
					},
					KindLabelIndex: {
						Name:         KindLabelIndex,
						Unique:       false,
						AllowMissing: true,
						Indexer:      &KindLabelIndexer,
					},
				},
			},
			"consumer": {
				Name: "consumer",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "Username"},
					},
					KindLabelIndex: {
						Name:         KindLabelIndex,
						Unique:       false,
						AllowMissing: true,
						Indexer:      &KindLabelIndexer,
					},
				},
			},
			"global_rule": {
				Name: "global_rule",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "ID"},
					},
					KindLabelIndex: {
						Name:         KindLabelIndex,
						Unique:       false,
						AllowMissing: true,
						Indexer:      &KindLabelIndexer,
					},
				},
			},
		},
	}
)
