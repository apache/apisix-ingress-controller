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

	v1 "github.com/api7/ingress-controller/pkg/types/apisix/v1"
)

type routeWorker struct {
	*v1.Route
	Event     chan Event
	Ctx       context.Context
	Wg        *sync.WaitGroup
	ErrorChan chan CRDStatus
}

// RouteWorkerGroup for broadcast from service to route
type RouteWorkerGroup map[string][]*routeWorker

// start start watch event
func (w *routeWorker) start() {
	w.Event = make(chan Event)
	go func() {
		for {
			select {
			case event := <-w.Event:
				w.trigger(event)
			case <-w.Ctx.Done():
				return
			}
		}
	}()
}

func (rg *RouteWorkerGroup) Add(key string, rw *routeWorker) {
	routes := (*rg)[key]
	if routes == nil {
		routes = make([]*routeWorker, 0)
	}
	routes = append(routes, rw)
	(*rg)[key] = routes
}

func (rg *RouteWorkerGroup) Delete(key string, route *routeWorker) {
	routes := (*rg)[key]
	result := make([]*routeWorker, 0)
	for _, r := range routes {
		if r.Name != route.Name {
			result = append(result, r)
		}
	}
	(*rg)[key] = result
}
