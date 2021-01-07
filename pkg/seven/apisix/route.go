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
package apisix

import (
	"context"
	"fmt"

	"github.com/api7/ingress-controller/pkg/seven/conf"
	sevendb "github.com/api7/ingress-controller/pkg/seven/db"
	"github.com/api7/ingress-controller/pkg/seven/utils"
	v1 "github.com/api7/ingress-controller/pkg/types/apisix/v1"
)

// FindCurrentRoute find current route in memDB
func FindCurrentRoute(route *v1.Route) (*v1.Route, error) {
	var cluster string
	if route.Group != nil {
		cluster = *route.Group
	}
	db := &sevendb.RouteRequest{Group: *route.Group, Name: *route.Name, FullName: *route.FullName}
	currentRoute, _ := db.FindByName()
	if currentRoute != nil {
		return currentRoute, nil
	} else {
		// find from apisix
		if routes, err := conf.Client.Cluster(cluster).Route().List(context.TODO()); err != nil {
			return nil, fmt.Errorf("list routes from etcd failed, err: %+v", err)
		} else {
			for _, r := range routes {
				if r.Name != nil && *r.Name == *route.Name {
					// insert to memDB
					db := &sevendb.RouteDB{Routes: []*v1.Route{r}}
					db.Insert()
					// return
					return r, nil
				}
			}
		}

	}
	return nil, utils.ErrNotFound
}
