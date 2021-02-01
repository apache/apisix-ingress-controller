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
	"sync"

	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/seven/conf"
	"github.com/apache/apisix-ingress-controller/pkg/seven/utils"
	v1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
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
				if err := w.trigger(event, rwg); err != nil {
					log.Errorf("failed to trigger event: %s", err)
				}
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
	log.Infof("2.service trigger from %s, %s", event.Op, upstream.Name)

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
	var cluster string
	if svc.Group != "" {
		cluster = svc.Group
	}
	currentService, _ := conf.Client.Cluster(cluster).Service().Get(context.TODO(), svc.FullName)
	if paddingService(svc, currentService) {
		op = Create
	}
	// diff
	hasDiff, err := utils.HasDiff(svc, currentService)
	// sync
	if err != nil {
		errNotify = err
		return
	}
	if hasDiff {
		if op == Create {
			if _, err := conf.Client.Cluster(cluster).Service().Create(context.TODO(), svc); err != nil {
				log.Errorf("failed to create service: %s", err)
				errNotify = err
				return
			}
			log.Infof("create service %s, %s", svc.Name, svc.UpstreamId)
		} else {
			needToUpdate := true
			if currentService.FromKind == ApisixService { // update from ApisixUpstream
				if svc.FromKind != ApisixService {
					// currentService > svc
					// set lb && health check
					needToUpdate = false
				}
			}
			if needToUpdate {
				if _, err := conf.Client.Cluster(cluster).Service().Update(context.TODO(), svc); err != nil {
					errNotify = err
					log.Errorf("failed to update service: %s, id:%s", err, svc.ID)
				} else {
					log.Infof("updated service, id:%s, upstream_id:%s", svc.ID, svc.UpstreamId)
				}
			}
		}
	}
	// broadcast to route
	for _, rw := range rwg[svc.Name] {
		event := &Event{Kind: ServiceKind, Op: op, Obj: svc}
		log.Infof("send event %s, %s, %s", event.Kind, event.Op, svc.Name)
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
