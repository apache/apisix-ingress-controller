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
	v1 "github.com/api7/ingress-controller/pkg/types/apisix/v1"
)

type ApisixCombination struct {
	Routes    []*v1.Route
	Services  []*v1.Service
	Upstreams []*v1.Upstream
}

type RouteCompare struct {
	OldRoutes []*v1.Route
	NewRoutes []*v1.Route
}

type Quit struct {
	Err error
}

const (
	RouteKind    = "route"
	ServiceKind  = "service"
	UpstreamKind = "upstream"
	Create       = "create"
	Update       = "update"
	Delete       = "delete"
)

type Event struct {
	Kind string      // route/service/upstream
	Op   string      // create update delete
	Obj  interface{} // the obj of kind
}
