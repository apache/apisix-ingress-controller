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

	"github.com/golang/glog"

	"github.com/api7/ingress-controller/pkg/seven/conf"
	sevendb "github.com/api7/ingress-controller/pkg/seven/db"
	v1 "github.com/api7/ingress-controller/pkg/types/apisix/v1"
)

// FindCurrentService find service from memDB,
// if Not Found, find service from apisix
func FindCurrentService(group, name, fullName string) (*v1.Service, error) {
	db := sevendb.ServiceRequest{Group: group, Name: name, FullName: fullName}
	currentService, _ := db.FindByName()
	if currentService != nil {
		return currentService, nil
	} else {
		// find service from apisix
		if services, err := conf.Client.Service().List(context.TODO(), group); err != nil {
			glog.Errorf("list services in etcd failed, group: %s, err: %+v", group, err)
			return nil, fmt.Errorf("list services failed, err: %+v", err)
		} else {
			for _, s := range services {
				if s.Name != nil && *(s.Name) == name {
					// and save to memDB
					db := &sevendb.ServiceDB{Services: []*v1.Service{s}}
					db.Insert()
					// return
					return s, nil
				}
			}
		}
	}
	return nil, nil
}
