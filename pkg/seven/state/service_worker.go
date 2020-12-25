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
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/gxthrj/apisix-types/pkg/apis/apisix/v1"

	"github.com/api7/ingress-controller/pkg/seven/apisix"
	"github.com/api7/ingress-controller/pkg/seven/db"
	"github.com/api7/ingress-controller/pkg/seven/utils"
)

const ApisixService = "ApisixService"

type serviceWorker struct {
	*v1.Service
	Event chan Event
	Quit  chan Quit
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
			case <-w.Quit:
				return
			}
		}
	}()
}

// trigger add to queue
func (w *serviceWorker) trigger(event Event, rwg *RouteWorkerGroup) error {
	glog.V(2).Infof("1.service trigger from %s, %s", event.Op, event.Kind)
	defer close(w.Quit)
	// consumer Event set upstreamID
	upstream := event.Obj.(*v1.Upstream)
	glog.V(2).Infof("2.service trigger from %s, %s", event.Op, *upstream.Name)

	w.UpstreamId = upstream.ID
	// add to queue
	services := []*v1.Service{w.Service}
	sqo := &ServiceQueueObj{Services: services, RouteWorkerGroup: *rwg}
	sqo.AddQueue()

	//op := Update
	//// padding
	//currentService, _ := apisix.FindCurrentService(*w.Service.Name)
	//paddingService(w.Service, currentService)
	//// diff
	//hasDiff, err := utils.HasDiff(w.Service, currentService)
	//// sync
	//if err != nil {
	//	return err
	//}
	//if hasDiff {
	//	if *w.Service.ID == strconv.Itoa(0) {
	//		op = Create
	//		// 1. sync apisix and get id
	//		if serviceResponse, err := apisix.AddService(w.Service, conf.BaseUrl); err != nil {
	//			// todo log error
	//			glog.Info(err.Error())
	//		}else {
	//			tmp := strings.Split(*serviceResponse.Service.Key, "/")
	//			*w.Service.ID = tmp[len(tmp) - 1]
	//		}
	//		// 2. sync memDB
	//		db := &db.ServiceDB{Services: []*v1.Service{w.Service}}
	//		db.Insert()
	//		glog.Infof("create service %s, %s", *w.Name, *w.UpstreamId)
	//	}else {
	//		op = Update
	//		// 1. sync memDB
	//		db := db.ServiceDB{Services: []*v1.Service{w.Service}}
	//		if err := db.UpdateService(); err != nil {
	//			// todo log error
	//		}
	//		// 2. sync apisix
	//		apisix.UpdateService(w.Service, conf.BaseUrl)
	//		glog.Infof("update service %s, %s", *w.Name, *w.UpstreamId)
	//	}
	//}
	//// broadcast to route
	//routeWorkers := (*rwg)[*w.Service.Name]
	//for _, rw := range routeWorkers{
	//	event := &Event{Kind: ServiceKind, Op: op, Obj: w.Service}
	//	glog.Infof("send event %s, %s, %s", event.Kind, event.Op, *w.Service.Name)
	//	rw.Event <- *event
	//}
	return nil
}

func SolverService(services []*v1.Service, rwg RouteWorkerGroup) error {
	for _, svc := range services {
		op := Update
		// padding
		currentService, _ := apisix.FindCurrentService(*svc.Group, *svc.Name, *svc.FullName)
		paddingService(svc, currentService)
		// diff
		hasDiff, err := utils.HasDiff(svc, currentService)
		// sync
		if err != nil {
			return err
		}
		if hasDiff {
			if *svc.ID == strconv.Itoa(0) {
				op = Create
				// 1. sync apisix and get id
				if serviceResponse, err := apisix.AddService(svc); err != nil {
					// todo log error
					glog.V(2).Info(err.Error())
				} else {
					tmp := strings.Split(*serviceResponse.Service.Key, "/")
					*svc.ID = tmp[len(tmp)-1]
				}
				// 2. sync memDB
				db := &db.ServiceDB{Services: []*v1.Service{svc}}
				db.Insert()
				glog.V(2).Infof("create service %s, %s", *svc.Name, *svc.UpstreamId)
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
					}
					// 2. sync apisix
					apisix.UpdateService(svc)
					glog.V(2).Infof("update service %s, %s", *svc.Name, *svc.UpstreamId)
				}

			}
		}
		// broadcast to route
		routeWorkers := rwg[*svc.Name]
		for _, rw := range routeWorkers {
			event := &Event{Kind: ServiceKind, Op: op, Obj: svc}
			glog.V(2).Infof("send event %s, %s, %s", event.Kind, event.Op, *svc.Name)
			rw.Event <- *event
		}
	}
	return nil
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
