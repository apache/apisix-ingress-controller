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
package apisix

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/gxthrj/apisix-types/pkg/apis/apisix/v1"

	"github.com/api7/ingress-controller/pkg/seven/conf"
	sevendb "github.com/api7/ingress-controller/pkg/seven/db"
	"github.com/api7/ingress-controller/pkg/seven/utils"
)

// FindCurrentRoute find current route in memDB
func FindCurrentRoute(route *v1.Route) (*v1.Route, error) {
	db := &sevendb.RouteRequest{Group: *route.Group, Name: *route.Name, FullName: *route.FullName}
	currentRoute, _ := db.FindByName()
	if currentRoute != nil {
		return currentRoute, nil
	} else {
		// find from apisix
		if routes, err := ListRoute(*route.Group); err != nil {
			return nil, fmt.Errorf("list routes from etcd failed, err: %+v", err)
		} else {
			for _, r := range routes {
				if r.Name != nil && *r.Name == *route.Name {
					// insert to memDB
					db := &sevendb.RouteDB{Routes: []*v1.Route{r}}
					db.Insert()
					// return
					return r, nil
				}
			}
		}

	}
	return nil, fmt.Errorf("NOT FOUND")
}

// ListRoute list route from etcd , convert to v1.Route
func ListRoute(group string) ([]*v1.Route, error) {
	baseUrl := conf.FindUrl(group)
	url := baseUrl + "/routes"
	ret, err := Get(url)
	if err != nil {
		return nil, fmt.Errorf("http get failed, url: %s, err: %+v", url, err)
	}
	var routesResponse RoutesResponse
	if err := json.Unmarshal(ret, &routesResponse); err != nil {
		return nil, fmt.Errorf("json unmarshal failed, err: %+v", err)
	} else {
		routes := make([]*v1.Route, 0)
		for _, u := range routesResponse.Routes.Routes {
			if n, err := u.convert(group); err == nil {
				routes = append(routes, n)
			} else {
				return nil, fmt.Errorf("upstream: %s 转换失败, %s", *u.Value.Desc, err.Error())
			}
		}
		return routes, nil
	}
}

func AddRoute(route *v1.Route) (*RouteResponse, error) {
	baseUrl := conf.FindUrl(*route.Group)
	url := fmt.Sprintf("%s/routes", baseUrl)
	rr := convert2RouteRequest(route)
	if b, err := json.Marshal(rr); err != nil {
		return nil, err
	} else {
		if res, err := utils.Post(url, b); err != nil {
			return nil, err
		} else {
			var routeResp RouteResponse
			if err = json.Unmarshal(res, &routeResp); err != nil {
				return nil, err
			} else {
				if routeResp.Route.Key != nil {
					return &routeResp, nil
				} else {
					return nil, fmt.Errorf("apisix route not expected response")
				}

			}
		}
	}
}

func UpdateRoute(route *v1.Route) error {
	baseUrl := conf.FindUrl(*route.Group)
	url := fmt.Sprintf("%s/routes/%s", baseUrl, *route.ID)
	rr := convert2RouteRequest(route)
	if b, err := json.Marshal(rr); err != nil {
		return err
	} else {
		if _, err := utils.Patch(url, b); err != nil {
			return err
		} else {
			return nil
		}
	}
}

func DeleteRoute(route *v1.Route) error {
	baseUrl := conf.FindUrl(*route.Group)
	url := fmt.Sprintf("%s/routes/%s", baseUrl, *route.ID)
	if _, err := utils.Delete(url); err != nil {
		return err
	} else {
		return nil
	}
}

type Redirect struct {
	RetCode int64  `json:"ret_code"`
	Uri     string `json:"uri"`
}

func convert2RouteRequest(route *v1.Route) *RouteRequest {
	return &RouteRequest{
		Desc:      *route.Name,
		Host:      *route.Host,
		Uri:       *route.Path,
		ServiceId: *route.ServiceId,
		Plugins:   route.Plugins,
	}
}

// convert apisix RouteResponse -> apisix-types v1.Route
func (r *Route) convert(group string) (*v1.Route, error) {
	// id
	key := r.Key
	ks := strings.Split(*key, "/")
	id := ks[len(ks)-1]
	// name
	name := r.Value.Desc
	// host
	host := r.Value.Host
	// path
	path := r.Value.Uri
	// method
	methods := r.Value.Methods
	// upstreamId
	upstreamId := r.Value.UpstreamId
	// serviceId
	serviceId := r.Value.ServiceId
	// plugins
	var plugins v1.Plugins
	plugins = r.Value.Plugins

	// fullName
	fullName := "unknown"
	if name != nil {
		fullName = *name
	}
	if group != "" {
		fullName = group + "_" + fullName
	}

	return &v1.Route{
		ID:         &id,
		Group:      &group,
		FullName:   &fullName,
		Name:       name,
		Host:       host,
		Path:       path,
		Methods:    methods,
		UpstreamId: upstreamId,
		ServiceId:  serviceId,
		Plugins:    &plugins,
	}, nil
}

type RoutesResponse struct {
	Routes Routes `json:"node"`
}

type Routes struct {
	Key    string   `json:"key"`
	Routes RouteSet `json:"nodes"`
}

type RouteSet []Route

// RouteSet.UnmarshalJSON implements json.Unmarshaler interface.
// lua-cjson doesn't distinguish empty array and table,
// and by default empty array will be encoded as '{}'.
// We have to maintain the compatibility.
func (set *RouteSet) UnmarshalJSON(p []byte) error {
	if p[0] == '{' {
		if len(p) != 2 {
			return errors.New("unexpected non-empty object")
		}
		return nil
	}
	var route []Route
	if err := json.Unmarshal(p, &route); err != nil {
		return err
	}
	*set = route
	return nil
}

type RouteResponse struct {
	Action string `json:"action"`
	Route  Route  `json:"node"`
}

type Route struct {
	Key   *string `json:"key"`   // route key
	Value Value   `json:"value"` // route content
}

type Value struct {
	UpstreamId *string                `json:"upstream_id"`
	ServiceId  *string                `json:"service_id"`
	Plugins    map[string]interface{} `json:"plugins"`
	Host       *string                `json:"host,omitempty"`
	Uri        *string                `json:"uri"`
	Desc       *string                `json:"desc"`
	Methods    []*string              `json:"methods,omitempty"`
}

type RouteRequest struct {
	Desc      string      `json:"desc,omitempty"`
	Uri       string      `json:"uri,omitempty"`
	Host      string      `json:"host,omitempty"`
	ServiceId string      `json:"service_id,omitempty"`
	Plugins   *v1.Plugins `json:"plugins,omitempty"`
}
