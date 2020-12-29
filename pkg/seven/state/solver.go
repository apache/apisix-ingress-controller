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
	"errors"
	"github.com/api7/ingress-controller/pkg/seven/apisix"
	"github.com/api7/ingress-controller/pkg/seven/db"
	v1 "github.com/api7/ingress-controller/pkg/types/apisix/v1"
	"sync"
	"time"
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
func (s *ApisixCombination) Solver() (string, error) {
	// 1.route workers
	rwg := NewRouteWorkers(s.Routes)
	// 2.service workers
	swg := NewServiceWorkers(s.Services, &rwg)
	//sqo := &ServiceQueueObj{Services: s.Services, RouteWorkerGroup: rwg}
	//sqo.AddQueue()
	// 3.upstream workers
	uqo := &UpstreamQueueObj{Upstreams: s.Upstreams, ServiceWorkerGroup: swg}
	uqo.AddQueue()
	// add timeout after 5s
	return s.Status("", rwg, swg, 5*time.Second)

}

func (s *ApisixCombination) Status(id string, rwg RouteWorkerGroup, swg ServiceWorkerGroup, timeout time.Duration) (string, error) {
	count := len(s.Routes) + len(s.Services) + len(s.Upstreams)
	resultChan := make(chan CRDStatus)

	var wg *sync.WaitGroup
	return WaitWorkerGroup(id, wg, count, resultChan, rwg, swg, timeout)
}

func WaitWorkerGroup(id string, wg *sync.WaitGroup, count int, result chan CRDStatus, rwg RouteWorkerGroup, swg ServiceWorkerGroup, timeout time.Duration) (string, error) {
	go func() {
		wg.Add(count)
		wg.Wait()
		result <- CRDStatus{Id: "", Status: "success", Err: nil}
	}()

	resourceChan := make(chan ResourceStatus, count)
	for {
		select {
		case r := <-resourceChan:
			Done(wg, r, result)
		case r := <-result:
			return id, r.Err
		case <-time.After(timeout):
			quit := &Quit{Err: errors.New("timeout")}
			// clean route workers
			for _, routeWorkers := range rwg {
				for _, rw := range routeWorkers {
					rw.Quit <- *quit
				}
			}
			// clean service workers
			for _, serviceWorkers := range swg {
				for _, sw := range serviceWorkers {
					sw.Quit <- *quit
				}
			}
			return id, errors.New("timeout")
		}
	}
}

func Done(wg *sync.WaitGroup, r ResourceStatus, result chan CRDStatus) {
	if r.Err != nil {
		result <- CRDStatus{Id: "", Status: "failed", Err: r.Err}
	}
	wg.Done()
}

// UpstreamQueueObj for upstream queue
type UpstreamQueueObj struct {
	Upstreams          []*v1.Upstream
	ServiceWorkerGroup ServiceWorkerGroup
}

type CRDStatus struct {
	Id     string `json:"id"`
	Status string `json:"status"`
	Err    error  `json:"err"`
}

type ResourceStatus struct {
	Kind string `json:"kind"`
	Id   string `json:"id"`
	Err  error  `json:"err"`
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
