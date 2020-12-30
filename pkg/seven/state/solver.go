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
	"context"
	"errors"
	"sync"
	"time"

	"github.com/api7/ingress-controller/pkg/log"
	"github.com/api7/ingress-controller/pkg/seven/apisix"
	"github.com/api7/ingress-controller/pkg/seven/conf"
	"github.com/api7/ingress-controller/pkg/seven/db"
	v1 "github.com/api7/ingress-controller/pkg/types/apisix/v1"
)

var UpstreamQueue chan UpstreamQueueObj

func init() {
	UpstreamQueue = make(chan UpstreamQueueObj, 500)
	go WatchUpstream()
}

func WatchUpstream() {
	for {
		uqo := <-UpstreamQueue
		SolverUpstream(uqo.Upstreams, uqo.ServiceWorkerGroup, uqo.Wg, uqo.ErrorChan)
	}
}

// Solver
func (s *ApisixCombination) Solver() (string, error) {
	// define the result notify
	timeout := 10 * time.Second
	resultChan := make(chan CRDStatus)
	ctx := context.Background()
	ctx, _ = context.WithTimeout(ctx, timeout)
	go s.SyncWithGroup(ctx, "", resultChan)

	return WaitWorkerGroup("", resultChan)
}

func waitTimeout(ctx context.Context, wg *sync.WaitGroup, resultChan chan CRDStatus) {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		resultChan <- CRDStatus{Id: "", Status: "success", Err: nil}
	case <-ctx.Done():
		resultChan <- CRDStatus{Id: "", Status: "failure", Err: errors.New("timeout")}
	}
}

func (s *ApisixCombination) SyncWithGroup(ctx context.Context, id string, resultChan chan CRDStatus) {
	var wg sync.WaitGroup
	count := len(s.Routes) + len(s.Services) + len(s.Upstreams)
	wg.Add(count)
	// goroutine for sync route/service/upstream
	// route
	rwg := NewRouteWorkers(ctx, s.Routes, &wg, resultChan)
	// service
	swg := NewServiceWorkers(ctx, s.Services, &rwg, &wg, resultChan)
	// upstream
	uqo := &UpstreamQueueObj{Upstreams: s.Upstreams, ServiceWorkerGroup: swg, Wg: &wg, ErrorChan: resultChan}
	uqo.AddQueue()

	waitTimeout(ctx, &wg, resultChan)
}

func WaitWorkerGroup(id string, resultChan chan CRDStatus) (string, error) {
	r := <-resultChan
	return id, r.Err
}

// UpstreamQueueObj for upstream queue
type UpstreamQueueObj struct {
	Upstreams          []*v1.Upstream
	ServiceWorkerGroup ServiceWorkerGroup
	Wg                 *sync.WaitGroup
	ErrorChan          chan CRDStatus
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
				log.Errorf("failed to find route %s from memory DB: %s", *old.Name, err)
			} else {
				if err := conf.Client.Route().Delete(context.TODO(), route); err != nil {
					log.Errorf("failed to delete route %s from APISIX: %s", *route.Name, err)
				} else {
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
