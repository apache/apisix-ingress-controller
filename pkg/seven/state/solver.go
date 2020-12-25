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
package state

import (
	"github.com/api7/ingress-controller/pkg/seven/apisix"
	"github.com/api7/ingress-controller/pkg/seven/db"
	v1 "github.com/api7/ingress-controller/pkg/types/apisix/v1"
)

var UpstreamQueue chan UpstreamQueueObj
var ServiceQueue chan ServiceQueueObj

func init() {
	UpstreamQueue = make(chan UpstreamQueueObj, 500)
	ServiceQueue = make(chan ServiceQueueObj, 500)
	go WatchUpstream()
	go WatchService()
}

func WatchService() {
	for {
		sqo := <-ServiceQueue
		// solver service
		SolverService(sqo.Services, sqo.RouteWorkerGroup)
	}
}

func WatchUpstream() {
	for {
		uqo := <-UpstreamQueue
		SolverUpstream(uqo.Upstreams, uqo.ServiceWorkerGroup)
	}
}

// Solver
func (s *ApisixCombination) Solver() (bool, error) {
	// 1.route workers
	rwg := NewRouteWorkers(s.Routes)
	// 2.service workers
	swg := NewServiceWorkers(s.Services, &rwg)
	//sqo := &ServiceQueueObj{Services: s.Services, RouteWorkerGroup: rwg}
	//sqo.AddQueue()
	// 3.upstream workers
	uqo := &UpstreamQueueObj{Upstreams: s.Upstreams, ServiceWorkerGroup: swg}
	uqo.AddQueue()
	return true, nil
}

// UpstreamQueueObj for upstream queue
type UpstreamQueueObj struct {
	Upstreams          []*v1.Upstream
	ServiceWorkerGroup ServiceWorkerGroup
}

// AddQueue make upstreams in order
// upstreams is group by CRD
func (uqo *UpstreamQueueObj) AddQueue() {
	UpstreamQueue <- *uqo
}

type ServiceQueueObj struct {
	Services         []*v1.Service
	RouteWorkerGroup RouteWorkerGroup
}

// AddQueue make upstreams in order
// upstreams is group by CRD
func (sqo *ServiceQueueObj) AddQueue() {
	ServiceQueue <- *sqo
}

// Sync remove from apisix
func (rc *RouteCompare) Sync() error {
	for _, old := range rc.OldRoutes {
		needToDel := true
		for _, nr := range rc.NewRoutes {
			if old.Name == nr.Name {
				needToDel = false
				break
			}
		}
		if needToDel {
			fullName := *old.Name
			if *old.Group != "" {
				fullName = *old.Group + "_" + *old.Name
			}
			request := db.RouteRequest{Name: *old.Name, FullName: fullName}

			if route, err := request.FindByName(); err != nil {
				// log error
			} else {
				if err = apisix.DeleteRoute(route); err == nil {
					db := db.RouteDB{Routes: []*v1.Route{route}}
					db.DeleteRoute()
				}
			}
		}
	}
	return nil
}

func SyncSsl(ssl *v1.Ssl, method string) error {
	switch method {
	case Create:
		_, err := apisix.AddOrUpdateSsl(ssl)
		return err
	case Update:
		_, err := apisix.AddOrUpdateSsl(ssl)
		return err
	case Delete:
		err := apisix.DeleteSsl(ssl)
		return err
	}
	return nil
}
