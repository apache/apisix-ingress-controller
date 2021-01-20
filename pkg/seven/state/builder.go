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
	"sync"

	"github.com/api7/ingress-controller/pkg/apisix/cache"
	"github.com/api7/ingress-controller/pkg/log"
	"github.com/api7/ingress-controller/pkg/seven/conf"
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
func NewRouteWorkers(ctx context.Context,
	routes []*v1.Route, wg *sync.WaitGroup, errorChan chan CRDStatus) RouteWorkerGroup {

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
	var cluster string
	if r.Route.Group != nil {
		cluster = *r.Route.Group
	}
	currentRoute, err := conf.Client.Cluster(cluster).Route().Get(context.TODO(), *r.Route.FullName)
	if err != nil && !errors.Is(err, cache.ErrNotFound) {
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
	var cluster string
	if r.Group != nil {
		cluster = *r.Group
	}
	if *r.Route.ID != strconv.Itoa(0) {
		if _, err := conf.Client.Cluster(cluster).Route().Update(context.TODO(), r.Route); err != nil {
			log.Errorf("failed to update route %s: %s, ", *r.Name, err)
			return err
		}
		log.Infof("update route %s, %s", *r.Name, *r.ServiceId)
	} else {
		route, err := conf.Client.Cluster(cluster).Route().Create(context.TODO(), r.Route)
		if err != nil {
			log.Errorf("failed to create route: %s", err.Error())
			return err
		}
		*r.ID = *route.ID
	}
	log.Infof("create route %s, %s", *r.Name, *r.ServiceId)
	return nil
}

// service
func NewServiceWorkers(ctx context.Context,
	services []*v1.Service, rwg *RouteWorkerGroup, wg *sync.WaitGroup, errorChan chan CRDStatus) ServiceWorkerGroup {
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
	var (
		op        string
		errNotify error
	)
	defer func() {
		if errNotify != nil {
			errorChan <- CRDStatus{Id: "", Status: "failure", Err: errNotify}
		}
		wg.Done()
	}()
	var cluster string
	if u.Group != nil {
		cluster = *u.Group
	}
	if currentUpstream, err := conf.Client.Cluster(cluster).Upstream().Get(context.TODO(), *u.FullName); err != nil && err != cache.ErrNotFound {
		log.Errorf("failed to find upstream %s: %s", *u.FullName, err)
		errNotify = err
		return
	} else {
		paddingUpstream(u, currentUpstream)
		if currentUpstream == nil {
			if u.FromKind != nil && *u.FromKind == WatchFromKind {
				// We don't have a pre-defined upstream and the current upstream updating from
				// endpoints.
				return
			}
			op = Create
			if _, err := conf.Client.Cluster(cluster).Upstream().Create(context.TODO(), u); err != nil {
				log.Errorf("failed to create upstream %s: %s", *u.FullName, err)
				return
			}
		} else {
			// diff
			hasDiff, err := utils.HasDiff(u, currentUpstream)
			if err != nil {
				errNotify = err
				return
			}
			if hasDiff {
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
				if needToUpdate || (u.FromKind != nil && *u.FromKind == WatchFromKind) {
					if u.FromKind != nil && *u.FromKind == WatchFromKind {
						currentUpstream.Nodes = u.Nodes
					} else { // due to CRD update
						currentUpstream = u
					}
					if _, err = conf.Client.Cluster(cluster).Upstream().Update(context.TODO(), currentUpstream); err != nil {
						log.Errorf("failed to update upstream %s: %s", *u.FullName, err)
						return
					}
				}
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
