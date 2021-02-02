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
	URI        string                 `json:"uri"`
	Desc       string                 `json:"desc"`
	Methods    []string               `json:"methods"`
	Plugins    map[string]interface{} `json:"plugins"`
}

// route decodes item.Value and converts it to v1.Route.
func (i *item) route(clusterName string) (*v1.Route, error) {
	log.Infof("got route: %s", string(i.Value))
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
			ID:         list[len(list)-1],
			FullName:   fullName,
			Group:      clusterName,
			Name:       route.Desc,
		},
		Host:       route.Host,
		Path:       route.URI,
		Methods:    route.Methods,
		UpstreamId: route.UpstreamId,
		ServiceId:  route.ServiceId,
		Plugins:    route.Plugins,
	}, nil
}

// upstream decodes item.Value and converts it to v1.Upstream.
func (i *item) upstream(clusterName string) (*v1.Upstream, error) {
	log.Infof("got upstream: %s", string(i.Value))
	list := strings.Split(i.Key, "/")
	if len(list) < 1 {
		return nil, fmt.Errorf("bad upstream config key: %s", i.Key)
	}

	var ups upstreamItem
	if err := json.Unmarshal(i.Value, &ups); err != nil {
		return nil, err
	}

	id := list[len(list)-1]
	name := ups.Desc
	LBType := ups.LBType
	key := i.Key

	var nodes []v1.Node
	for _, node := range ups.Nodes {
		nodes = append(nodes, v1.Node{
			IP:     node.Host,
			Port:   node.Port,
			Weight: node.Weight,
		})
	}

	fullName := genFullName(ups.Desc, clusterName)

	return &v1.Upstream{
		Metadata: v1.Metadata{
			ID:       id,
			FullName: fullName,
			Group:    clusterName,
			Name:     name,
		},
		Type:     LBType,
		Key:      key,
		Nodes:    nodes,
	}, nil
}

// service decodes item.Value and converts it to v1.Service.
func (i *item) service(clusterName string) (*v1.Service, error) {
	log.Infof("got service: %s", string(i.Value))
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
	log.Infof("got ssl: %s", string(i.Value))
	var ssl v1.Ssl
	if err := json.Unmarshal(i.Value, &ssl); err != nil {
		return nil, err
	}

	list := strings.Split(i.Key, "/")
	id := list[len(list)-1]
	ssl.ID = id
	ssl.Group = clusterName
	return &ssl, nil
}

func genFullName(name string, clusterName string) string {
	fullName := name
	if clusterName != "" {
		fullName = clusterName + "_" + fullName
	}
	return fullName
}
