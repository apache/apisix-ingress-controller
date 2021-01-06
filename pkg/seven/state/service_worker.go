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
	"strconv"
	"strings"
	"sync"

	"github.com/api7/ingress-controller/pkg/log"
	"github.com/api7/ingress-controller/pkg/seven/apisix"
	"github.com/api7/ingress-controller/pkg/seven/db"
	"github.com/api7/ingress-controller/pkg/seven/utils"
	v1 "github.com/api7/ingress-controller/pkg/types/apisix/v1"
)

const ApisixService = "ApisixService"

type serviceWorker struct {
	*v1.Service
	Event     chan Event
	Ctx       context.Context
	Wg        *sync.WaitGroup
	ErrorChan chan CRDStatus
}

// ServiceWorkerGroup for broadcast from upstream to service
type ServiceWorkerGroup map[string][]*serviceWorker

// start start watch event
func (w *serviceWorker) start(rwg *RouteWorkerGroup) {
	w.Event = make(chan Event)
	go func() {
		for {
			select {
			case event := <-w.Event:
				w.trigger(event, rwg)
			case <-w.Ctx.Done():
				return
			}
		}
	}()
}

// trigger add to queue
func (w *serviceWorker) trigger(event Event, rwg *RouteWorkerGroup) error {
	log.Infof("1.service trigger from %s, %s", event.Op, event.Kind)
	// consumer Event set upstreamID
	upstream := event.Obj.(*v1.Upstream)
	log.Infof("2.service trigger from %s, %s", event.Op, *upstream.Name)

	w.UpstreamId = upstream.ID
	// add to queue
	services := []*v1.Service{w.Service}
	sqo := &ServiceQueueObj{Services: services, RouteWorkerGroup: *rwg}
	//sqo.AddQueue()

	SolverService(sqo.Services, sqo.RouteWorkerGroup, w.Wg, w.ErrorChan)
	return nil
}

func SolverService(services []*v1.Service, rwg RouteWorkerGroup, wg *sync.WaitGroup, errorChan chan CRDStatus) {
	for _, svc := range services {
		go SolverSingleService(svc, rwg, wg, errorChan)
	}
}

func SolverSingleService(svc *v1.Service, rwg RouteWorkerGroup, wg *sync.WaitGroup, errorChan chan CRDStatus) {
	var errNotify error
	defer func() {
		if errNotify != nil {
			errorChan <- CRDStatus{Id: "", Status: "failure", Err: errNotify}
		}
		wg.Done()
	}()

	op := Update
	// padding
	currentService, _ := apisix.FindCurrentService(*svc.Group, *svc.Name, *svc.FullName)
	paddingService(svc, currentService)
	// diff
	hasDiff, err := utils.HasDiff(svc, currentService)
	// sync
	if err != nil {
		errNotify = err
		return
	}
	if hasDiff {
		if *svc.ID == strconv.Itoa(0) {
			op = Create
			// 1. sync apisix and get id
			if serviceResponse, err := apisix.AddService(svc); err != nil {
				log.Info(err.Error())
				errNotify = err
				return
			} else {
				tmp := strings.Split(*serviceResponse.Service.Key, "/")
				*svc.ID = tmp[len(tmp)-1]
			}
			// 2. sync memDB
			db := &db.ServiceDB{Services: []*v1.Service{svc}}
			if err := db.Insert(); err != nil {
				errNotify = err
				return
			}
			log.Infof("create service %s, %s", *svc.Name, *svc.UpstreamId)
		} else {
			op = Update
			needToUpdate := true
			if currentService.FromKind != nil && *(currentService.FromKind) == ApisixService { // update from ApisixUpstream
				if svc.FromKind == nil || (svc.FromKind != nil && *(svc.FromKind) != ApisixService) {
					// currentService > svc
					// set lb && health check
					needToUpdate = false
				}
			}
			if needToUpdate {
				// 1. sync memDB
				db := db.ServiceDB{Services: []*v1.Service{svc}}
				if err := db.UpdateService(); err != nil {
					// todo log error
					errNotify = err
					return
				}
				// 2. sync apisix
				if _, err := apisix.UpdateService(svc); err != nil {
					errNotify = err
					return
				}
				log.Infof("update service %s, %s", *svc.Name, *svc.UpstreamId)
			}

		}
	}
	// broadcast to route
	routeWorkers := rwg[*svc.Name]
	for _, rw := range routeWorkers {
		event := &Event{Kind: ServiceKind, Op: op, Obj: svc}
		log.Infof("send event %s, %s, %s", event.Kind, event.Op, *svc.Name)
		rw.Event <- *event
	}
}

func (swg *ServiceWorkerGroup) Add(key string, s *serviceWorker) {
	sws := (*swg)[key]
	if sws == nil {
		sws = make([]*serviceWorker, 0)
	}
	sws = append(sws, s)
	(*swg)[key] = sws
}

func (swg *ServiceWorkerGroup) Delete(key string, s *serviceWorker) {
	sws := (*swg)[key]
	result := make([]*serviceWorker, 0)
	for _, r := range sws {
		if r.Name != s.Name {
			result = append(result, r)
		}
	}
	(*swg)[key] = result
}
