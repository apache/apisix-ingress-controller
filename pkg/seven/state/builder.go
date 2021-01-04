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
	"strconv"
	"strings"
	"sync"

	"github.com/api7/ingress-controller/pkg/log"
	"github.com/api7/ingress-controller/pkg/seven/apisix"
	"github.com/api7/ingress-controller/pkg/seven/db"
	"github.com/api7/ingress-controller/pkg/seven/utils"
	v1 "github.com/api7/ingress-controller/pkg/types/apisix/v1"
)

const (
	ApisixUpstream = "ApisixUpstream"
	WatchFromKind  = "watch"
)

//// InitDB insert object into memDB first time
//func InitDB(){
//	routes, _ := apisix.ListRoute()
//	upstreams, _ := apisix.ListUpstream()
//	apisix.InsertRoute(routes)
//	apisix.InsertUpstreams(upstreams)
//}
//
//// LoadTargetState load targetState from ... maybe k8s CRD
//func LoadTargetState(routes []*v1.Route, upstreams []*v1.Upstream){
//
//	// 1.diff
//	// 2.send event
//}

// paddingRoute padding route from memDB
func paddingRoute(route *v1.Route, currentRoute *v1.Route) {
	// padding object, just id
	if currentRoute == nil {
		// NOT FOUND : set Id = 0
		id := strconv.Itoa(0)
		route.ID = &id
	} else {
		route.ID = currentRoute.ID
	}
}

// padding service from memDB
func paddingService(service *v1.Service, currentService *v1.Service) {
	if currentService == nil {
		id := strconv.Itoa(0)
		service.ID = &id
	} else {
		service.ID = currentService.ID
	}
}

// paddingUpstream padding upstream from memDB
func paddingUpstream(upstream *v1.Upstream, currentUpstream *v1.Upstream) {
	// padding id
	if currentUpstream == nil {
		// NOT FOUND : set Id = 0
		id := strconv.Itoa(0)
		upstream.ID = &id
	} else {
		upstream.ID = currentUpstream.ID
	}
	// todo padding nodes ? or sync nodes from crd ?
}

// NewRouteWorkers make routeWrokers group by service per CRD
// 1.make routes group by (1_2_3) it may be a map like map[1_2_3][]Route;
// 2.route is listenning Event from the ready of 1_2_3;
func NewRouteWorkers(ctx context.Context, routes []*v1.Route, wg *sync.WaitGroup, errorChan chan CRDStatus) RouteWorkerGroup {
	rwg := make(RouteWorkerGroup)
	for _, r := range routes {
		rw := &routeWorker{Route: r, Ctx: ctx, Wg: wg, ErrorChan: errorChan}
		rw.start()
		rwg.Add(*r.ServiceName, rw)
	}
	return rwg
}

// 3.route get the Event and trigger a padding for object,then diff,sync;
func (r *routeWorker) trigger(event Event) {
	var errNotify error
	defer func() {
		if errNotify != nil {
			r.ErrorChan <- CRDStatus{Id: "", Status: "failure", Err: errNotify}
		}
		r.Wg.Done()
	}()
	// consumer Event
	service := event.Obj.(*v1.Service)
	r.ServiceId = service.ID
	log.Infof("trigger routeWorker %s from %s, %s", *r.Name, event.Op, *service.Name)

	// padding
	currentRoute, err := apisix.FindCurrentRoute(r.Route)
	if err != nil && !errors.Is(err, utils.NotFound) {
		errNotify = err
		return
	}
	paddingRoute(r.Route, currentRoute)
	// diff
	hasDiff, err := utils.HasDiff(r.Route, currentRoute)
	// sync
	if err != nil {
		errNotify = err
		return
	}
	if hasDiff {
		err := r.sync()
		if err != nil {
			errNotify = err
			return
		}
	}
}

// sync
func (r *routeWorker) sync() error {
	if *r.Route.ID != strconv.Itoa(0) {
		// 1. sync memDB
		db := &db.RouteDB{Routes: []*v1.Route{r.Route}}
		if err := db.UpdateRoute(); err != nil {
			log.Errorf("update route failed, route: %#v, err: %+v", r.Route, err)
			return err
		}
		// 2. sync apisix
		if err := apisix.UpdateRoute(r.Route); err != nil {
			return err
		}
		log.Infof("update route %s, %s", *r.Name, *r.ServiceId)
	} else {
		// 1. sync apisix and get id
		if res, err := apisix.AddRoute(r.Route); err != nil {
			log.Errorf("add route failed, route: %#v, err: %+v", r.Route, err)
			return err
		} else {
			key := res.Route.Key
			tmp := strings.Split(*key, "/")
			*r.ID = tmp[len(tmp)-1]
		}
		// 2. sync memDB
		db := &db.RouteDB{Routes: []*v1.Route{r.Route}}
		if err := db.Insert(); err != nil {
			return err
		}
		log.Infof("create route %s, %s", *r.Name, *r.ServiceId)
	}
	return nil
}

// service
func NewServiceWorkers(ctx context.Context, services []*v1.Service, rwg *RouteWorkerGroup, wg *sync.WaitGroup, errorChan chan CRDStatus) ServiceWorkerGroup {
	swg := make(ServiceWorkerGroup)
	for _, s := range services {
		rw := &serviceWorker{Service: s, Ctx: ctx, Wg: wg, ErrorChan: errorChan}
		//rw.Wg.Add(1)
		rw.start(rwg)
		swg.Add(*s.UpstreamName, rw)
	}
	return swg
}

// upstream
func SolverUpstream(upstreams []*v1.Upstream, swg ServiceWorkerGroup, wg *sync.WaitGroup, errorChan chan CRDStatus) {
	for _, u := range upstreams {
		go SolverSingleUpstream(u, swg, wg, errorChan)
	}
}

func SolverSingleUpstream(u *v1.Upstream, swg ServiceWorkerGroup, wg *sync.WaitGroup, errorChan chan CRDStatus) {
	var errNotify error
	defer func() {
		if errNotify != nil {
			errorChan <- CRDStatus{Id: "", Status: "failure", Err: errNotify}
		}
		wg.Done()
	}()
	op := Update
	if currentUpstream, err := apisix.FindCurrentUpstream(*u.Group, *u.Name, *u.FullName); err != nil {
		log.Errorf("solver upstream failed, find upstream from etcd failed, upstream: %+v, err: %+v", u, err)
		errNotify = err
		return
	} else {
		paddingUpstream(u, currentUpstream)
		// diff
		hasDiff, err := utils.HasDiff(u, currentUpstream)
		if err != nil {
			errNotify = err
			return
		}
		if hasDiff {
			if *u.ID != strconv.Itoa(0) {
				op = Update
				// 0.field check
				needToUpdate := true
				if currentUpstream.FromKind != nil && *(currentUpstream.FromKind) == ApisixUpstream { // update from ApisixUpstream
					if u.FromKind == nil || (u.FromKind != nil && *(u.FromKind) != ApisixUpstream) {
						// currentUpstream > u
						// set lb && health check
						needToUpdate = false
					}
				}
				if needToUpdate {
					// 1.sync memDB
					upstreamDB := &db.UpstreamDB{Upstreams: []*v1.Upstream{u}}
					if err := upstreamDB.UpdateUpstreams(); err != nil {
						log.Errorf("solver upstream failed, update upstream to local db failed, err: %s", err.Error())
						errNotify = err
						return
					}
					// 2.sync apisix
					if err = apisix.UpdateUpstream(u); err != nil {
						log.Errorf("solver upstream failed, update upstream to etcd failed, err: %+v", err)
						errNotify = err
						return
					}
				}
				// if fromKind == WatchFromKind
				if u.FromKind != nil && *u.FromKind == WatchFromKind {
					// 1.update nodes
					if err = apisix.PatchNodes(u, u.Nodes); err != nil {
						log.Errorf("solver upstream failed, patch node info to etcd failed, err: %+v", err)
						errNotify = err
						return
					}
					// 2. sync memDB
					us := []*v1.Upstream{u}
					if !needToUpdate {
						currentUpstream.Nodes = u.Nodes
						us = []*v1.Upstream{currentUpstream}
					}
					upstreamDB := &db.UpstreamDB{Upstreams: us}
					if err := upstreamDB.UpdateUpstreams(); err != nil {
						log.Errorf("solver upstream failed, update upstream to local db failed, err: %s", err.Error())
						errNotify = err
						return
					}
				}
			} else {
				op = Create
				// 1.sync apisix and get response
				if upstreamResponse, err := apisix.AddUpstream(u); err != nil {
					log.Errorf("solver upstream failed, update upstream to etcd failed, err: %+v", err)
					errNotify = err
					return
				} else {
					tmp := strings.Split(*upstreamResponse.Upstream.Key, "/")
					*u.ID = tmp[len(tmp)-1]
				}
				// 2.sync memDB
				//apisix.InsertUpstreams([]*v1.Upstream{u})
				upstreamDB := &db.UpstreamDB{Upstreams: []*v1.Upstream{u}}
				upstreamDB.InsertUpstreams()
			}
		}
	}
	log.Infof("solver upstream %s:%s", op, *u.Name)
	// anyway, broadcast to service
	serviceWorkers := swg[*u.Name]
	for _, sw := range serviceWorkers {
		event := &Event{Kind: UpstreamKind, Op: op, Obj: u}
		sw.Event <- *event
	}
}
