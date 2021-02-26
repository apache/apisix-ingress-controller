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

	"github.com/apache/apisix-ingress-controller/pkg/apisix/cache"
	"github.com/apache/apisix-ingress-controller/pkg/id"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/seven/conf"
	"github.com/apache/apisix-ingress-controller/pkg/seven/utils"
	v1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

const (
	ApisixUpstream = "ApisixUpstream"
	WatchFromKind  = "watch"
)

// paddingRoute fills route through currentRoute, it returns a boolean
// value to indicate whether the route is a new created one.
func paddingRoute(route *v1.Route, currentRoute *v1.Route) bool {
	if currentRoute == nil {
		route.ID = id.GenID(route.FullName)
		return true
	}
	route.ID = currentRoute.ID
	return false
}

// paddingService fills service through currentService, it returns a boolean
// value to indicate whether the service is a new created one.
func paddingService(service *v1.Service, currentService *v1.Service) bool {
	if currentService == nil {
		service.ID = id.GenID(service.FullName)
		return true
	}
	service.ID = currentService.ID
	return false
}

// paddingUpstream fills upstream through currentUpstream, it returns a boolean
// value to indicate whether the upstream is a new created one.
func paddingUpstream(upstream *v1.Upstream, currentUpstream *v1.Upstream) bool {
	if currentUpstream == nil {
		upstream.ID = id.GenID(upstream.FullName)
		return true
	}
	upstream.ID = currentUpstream.ID
	return false
}

// NewRouteWorkers make routeWorkers group by service per CRD
// 1.make routes group by (1_2_3) it may be a map like map[1_2_3][]Route;
// 2.route is listening Event from the ready of 1_2_3;
func NewRouteWorkers(ctx context.Context,
	routes []*v1.Route, wg *sync.WaitGroup, errorChan chan CRDStatus) RouteWorkerGroup {

	rwg := make(RouteWorkerGroup)
	for _, r := range routes {
		rw := &routeWorker{Route: r, Ctx: ctx, Wg: wg, ErrorChan: errorChan}
		rw.start()
		rwg.Add(r.UpstreamName, rw)
	}
	return rwg
}

// 3.route get the Event and trigger a padding for object,then diff,sync;
func (r *routeWorker) trigger() {
	var (
		op        string
		errNotify error
	)
	defer func() {
		if errNotify != nil {
			r.ErrorChan <- CRDStatus{Id: "", Status: "failure", Err: errNotify}
		}
		r.Wg.Done()
	}()

	// padding
	var cluster string
	if r.Route.Group != "" {
		cluster = r.Route.Group
	}
	currentRoute, err := conf.Client.Cluster(cluster).Route().Get(context.TODO(), r.Route.FullName)
	if err != nil && !errors.Is(err, cache.ErrNotFound) {
		errNotify = err
		return
	}

	if paddingRoute(r.Route, currentRoute) {
		op = Create
	} else {
		op = Update
	}

	hasDiff, err := utils.HasDiff(r.Route, currentRoute)
	if err != nil {
		errNotify = err
		return
	}
	if hasDiff {
		err := r.sync(op)
		if err != nil {
			errNotify = err
			return
		}
	}
}

// sync
func (r *routeWorker) sync(op string) error {
	var cluster string
	if r.Group != "" {
		cluster = r.Group
	}
	if op == Update {
		if _, err := conf.Client.Cluster(cluster).Route().Update(context.TODO(), r.Route); err != nil {
			log.Errorf("failed to update route %s: %s, ", r.Name, err)
			return err
		}
		log.Infof("update route %s, %s", r.Name, r.ServiceId)
	} else {
		route, err := conf.Client.Cluster(cluster).Route().Create(context.TODO(), r.Route)
		if err != nil {
			log.Errorf("failed to create route: %s", err.Error())
			return err
		}
		r.ID = route.ID
	}
	log.Infof("create route %s, %s", r.Name, r.ServiceId)
	return nil
}

// upstream
func SolverUpstream(upstreams []*v1.Upstream, rwg RouteWorkerGroup, wg *sync.WaitGroup, errorChan chan CRDStatus) {
	for _, u := range upstreams {
		go SolverSingleUpstream(u, rwg, wg, errorChan)
	}
}

func SolverSingleUpstream(u *v1.Upstream, rwg RouteWorkerGroup, wg *sync.WaitGroup, errorChan chan CRDStatus) {
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
	if u.Group != "" {
		cluster = u.Group
	}
	if currentUpstream, err := conf.Client.Cluster(cluster).Upstream().Get(context.TODO(), u.FullName); err != nil && err != cache.ErrNotFound {
		log.Errorf("failed to find upstream %s: %s", u.FullName, err)
		errNotify = err
		return
	} else {
		if paddingUpstream(u, currentUpstream) {
			op = Create
		} else {
			op = Update
		}

		if op == Create {
			if u.FromKind == WatchFromKind {
				// We don't have a pre-defined upstream and the current upstream updating from
				// endpoints.
				return
			}
			if _, err := conf.Client.Cluster(cluster).Upstream().Create(context.TODO(), u); err != nil {
				log.Errorf("failed to create upstream %s: %s", u.FullName, err)
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
				if currentUpstream.FromKind == ApisixUpstream { // update from ApisixUpstream
					if u.FromKind != ApisixUpstream {
						// currentUpstream > u
						// set lb && health check
						needToUpdate = false
					}
				}
				if needToUpdate || u.FromKind == WatchFromKind {
					if u.FromKind == WatchFromKind {
						currentUpstream.Nodes = u.Nodes
					} else { // due to CRD update
						currentUpstream = u
					}
					if _, err = conf.Client.Cluster(cluster).Upstream().Update(context.TODO(), currentUpstream); err != nil {
						log.Errorf("failed to update upstream %s: %s", u.FullName, err)
						return
					}
				}
			}
		}
	}
	log.Infof("solver upstream %s:%s", op, u.Name)
	// anyway, broadcast to route
	for _, sw := range rwg[u.Name] {
		sw.Event <- Event{}
	}
}
