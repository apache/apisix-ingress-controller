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

	"github.com/apache/apisix-ingress-controller/pkg/log"
	v1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

type getResponse struct {
	Item item `json:"node"`
}

// listResponse is the unified LIST response mapping of APISIX.
type listResponse struct {
	Count string `json:"count"`
	Node  node   `json:"node"`
}

type createResponse struct {
	Action string `json:"action"`
	Item   item   `json:"node"`
}

type updateResponse = createResponse

type node struct {
	Key   string `json:"key"`
	Items items  `json:"nodes"`
}

type items []item

// items implements json.Unmarshaler interface.
// lua-cjson doesn't distinguish empty array and table,
// and by default empty array will be encoded as '{}'.
// We have to maintain the compatibility.
func (items *items) UnmarshalJSON(p []byte) error {
	if p[0] == '{' {
		if len(p) != 2 {
			return errors.New("unexpected non-empty object")
		}
		return nil
	}
	var data []item
	if err := json.Unmarshal(p, &data); err != nil {
		return err
	}
	*items = data
	return nil
}

type item struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

type routeItem struct {
	UpstreamId string                 `json:"upstream_id"`
	ServiceId  string                 `json:"service_id"`
	Host       string                 `json:"host"`
	Hosts      []string               `json:"hosts"`
	URI        string                 `json:"uri"`
	Vars       [][]v1.StringOrSlice   `json:"vars"`
	Uris       []string               `json:"uris"`
	Desc       string                 `json:"desc"`
	Methods    []string               `json:"methods"`
	Priority   int                    `json:"priority"`
	Plugins    map[string]interface{} `json:"plugins"`
}

// route decodes item.Value and converts it to v1.Route.
func (i *item) route(clusterName string) (*v1.Route, error) {
	log.Debugf("got route: %s", string(i.Value))
	list := strings.Split(i.Key, "/")
	if len(list) < 1 {
		return nil, fmt.Errorf("bad route config key: %s", i.Key)
	}

	var route routeItem
	if err := json.Unmarshal(i.Value, &route); err != nil {
		return nil, err
	}

	fullName := genFullName(route.Desc, clusterName)

	return &v1.Route{
		Metadata: v1.Metadata{
			ID:       list[len(list)-1],
			FullName: fullName,
			Group:    clusterName,
			Name:     route.Desc,
		},
		Host:       route.Host,
		Path:       route.URI,
		Uris:       route.Uris,
		Vars:       route.Vars,
		Methods:    route.Methods,
		UpstreamId: route.UpstreamId,
		ServiceId:  route.ServiceId,
		Plugins:    route.Plugins,
		Hosts:      route.Hosts,
		Priority:   route.Priority,
	}, nil
}

// upstream decodes item.Value and converts it to v1.Upstream.
func (i *item) upstream(clusterName string) (*v1.Upstream, error) {
	log.Debugf("got upstream: %s", string(i.Value))
	list := strings.Split(i.Key, "/")
	if len(list) < 1 {
		return nil, fmt.Errorf("bad upstream config key: %s", i.Key)
	}

	var ups upstreamItem
	if err := json.Unmarshal(i.Value, &ups); err != nil {
		return nil, err
	}

	var nodes []v1.UpstreamNode
	for _, node := range ups.Nodes {
		nodes = append(nodes, v1.UpstreamNode{
			IP:     node.Host,
			Port:   node.Port,
			Weight: node.Weight,
		})
	}

	// This is a work around scheme to avoid APISIX's
	// health check schema about the health checker intervals.
	if ups.Checks != nil && ups.Checks.Active != nil {
		if ups.Checks.Active.Healthy.Interval == 0 {
			ups.Checks.Active.Healthy.Interval = int(v1.ActiveHealthCheckMinInterval.Seconds())
		}
		if ups.Checks.Active.Unhealthy.Interval == 0 {
			ups.Checks.Active.Healthy.Interval = int(v1.ActiveHealthCheckMinInterval.Seconds())
		}
	}

	fullName := genFullName(ups.Desc, clusterName)

	return &v1.Upstream{
		Metadata: v1.Metadata{
			ID:       list[len(list)-1],
			FullName: fullName,
			Group:    clusterName,
			Name:     ups.Desc,
		},
		Type:    ups.LBType,
		Key:     ups.Key,
		HashOn:  ups.HashOn,
		Nodes:   nodes,
		Scheme:  ups.Scheme,
		Checks:  ups.Checks,
		Retries: ups.Retries,
		Timeout: ups.Timeout,
	}, nil
}

// service decodes item.Value and converts it to v1.Service.
func (i *item) service(clusterName string) (*v1.Service, error) {
	log.Debugf("got service: %s", string(i.Value))
	var svc serviceItem
	if err := json.Unmarshal(i.Value, &svc); err != nil {
		return nil, err
	}

	list := strings.Split(i.Key, "/")
	id := list[len(list)-1]
	var plugins v1.Plugins
	if svc.Plugins != nil {
		plugins := make(v1.Plugins, len(svc.Plugins))
		for k, v := range svc.Plugins {
			plugins[k] = v
		}
	}
	fullName := genFullName(svc.Desc, clusterName)

	return &v1.Service{
		ID:         id,
		FullName:   fullName,
		Group:      clusterName,
		Name:       svc.Desc,
		UpstreamId: svc.UpstreamId,
		Plugins:    plugins,
	}, nil
}

// ssl decodes item.Value and converts it to v1.Ssl.
func (i *item) ssl(clusterName string) (*v1.Ssl, error) {
	log.Debugf("got ssl: %s", string(i.Value))
	var ssl v1.Ssl
	if err := json.Unmarshal(i.Value, &ssl); err != nil {
		return nil, err
	}

	list := strings.Split(i.Key, "/")
	id := list[len(list)-1]
	ssl.ID = id
	ssl.Group = clusterName
	ssl.FullName = id
	return &ssl, nil
}

func genFullName(name string, clusterName string) string {
	fullName := name
	if clusterName != "" {
		fullName = clusterName + "_" + fullName
	}
	return fullName
}
