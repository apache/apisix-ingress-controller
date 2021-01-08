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
	"fmt"
	"sync"
	"time"

	"github.com/api7/ingress-controller/pkg/log"
	"github.com/api7/ingress-controller/pkg/seven/conf"
	"github.com/api7/ingress-controller/pkg/seven/db"
	"github.com/api7/ingress-controller/pkg/seven/utils"
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

func (s *ApisixCombination) Remove() error {
	// services
	for _, svc := range s.Services {
		if err := RemoveService(svc); err != nil {
			return err
		}
	}

	// upstreams
	for _, up := range s.Upstreams {
		if err := RemoveUpstream(up); err != nil {
			return err
		}
	}
	return nil
}

func RemoveService(svc *v1.Service) error {
	// find ref route
	routeRequest := db.RouteRequest{ServiceId: *svc.ID}
	if route, err := routeRequest.ExistByServiceId(); err != nil {
		if !errors.Is(err, utils.ErrNotFound) {
			// except ErrNotFound, need to retry
			return err
		} else {
			// do delete svc
			var cluster string
			if route.Group != nil {
				cluster = *route.Group
			}
			if err := conf.Client.Cluster(cluster).Service().Delete(context.TODO(), svc); err != nil {
				log.Errorf("failed to delete svc %s from APISIX: %s", *svc.FullName, err)
				return err
			} else {
				db := db.ServiceDB{Services: []*v1.Service{svc}}
				db.DeleteService()
				return nil
			}
		}
	} else {
		return fmt.Errorf("svc %s is still referenced by route %s", *svc.FullName, *route.FullName)
	}
}

func RemoveUpstream(up *v1.Upstream) error {
	serviceRequest := db.ServiceRequest{UpstreamId: *up.ID}
	if svc, err := serviceRequest.ExistByUpstreamId(); err != nil {
		if !errors.Is(err, utils.ErrNotFound) {
			// except ErrNotFound, need to retry
			return err
		} else {
			// do delete upstream
			var cluster string
			if svc.Group != nil {
				cluster = *svc.Group
			}
			if err := conf.Client.Cluster(cluster).Upstream().Delete(context.TODO(), up); err != nil {
				log.Errorf("failed to delete upstream %s from APISIX: %s", *up.FullName, err)
				return err
			} else {
				db := db.UpstreamDB{Upstreams: []*v1.Upstream{up}}
				db.DeleteUpstream()
				return nil
			}
		}
	} else {
		return fmt.Errorf("upstream %s is still referenced by service %s", *up.FullName, *svc.FullName)
	}
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
				var cluster string
				if route.Group != nil {
					cluster = *route.Group
				}
				if err := conf.Client.Cluster(cluster).Route().Delete(context.TODO(), route); err != nil {
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
	var cluster string
	if ssl.Group != nil {
		cluster = *ssl.Group
	}
	switch method {
	case Create:
		_, err := conf.Client.Cluster(cluster).SSL().Create(context.TODO(), ssl)
		return err
	case Update:
		_, err := conf.Client.Cluster(cluster).SSL().Update(context.TODO(), ssl)
		return err
	case Delete:
		return conf.Client.Cluster(cluster).SSL().Delete(context.TODO(), ssl)
	}
	return nil
}
