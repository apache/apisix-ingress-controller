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
	"strconv"

	"github.com/api7/ingress-controller/pkg/ingress/endpoint"
	configv1 "github.com/api7/ingress-controller/pkg/kube/apisix/apis/config/v1"
	"github.com/api7/ingress-controller/pkg/seven/conf"
	apisix "github.com/api7/ingress-controller/pkg/types/apisix/v1"
)

const (
	ApisixService = "ApisixService"
)

type ApisixServiceBuilder struct {
	CRD                 *configv1.ApisixService
	Ep                  endpoint.Endpoint
	EnableEndpointSlice bool
}

// Convert convert to  apisix.Service from ingress.ApisixService CRD
func (asb *ApisixServiceBuilder) Convert() ([]*apisix.Service, []*apisix.Upstream, error) {
	ar := asb.CRD
	ns := ar.Namespace
	name := ar.Name
	// meta annotation
	pluginsInAnnotation, group := BuildAnnotation(ar.Annotations)
	conf.AddGroup(group)
	services := make([]*apisix.Service, 0)
	upstreams := make([]*apisix.Upstream, 0)
	rv := ar.ObjectMeta.ResourceVersion
	port := ar.Spec.Port
	upstreamName := ar.Spec.Upstream
	// apisix upstream name = namespace_upstreamName_svcPort
	apisixUpstreamName := ns + "_" + upstreamName + "_" + strconv.Itoa(int(port))
	apisixServiceName := ns + "_" + name + "_" + strconv.Itoa(int(port))
	fromKind := ApisixService
	// plugins
	plugins := ar.Spec.Plugins
	pluginRet := apisix.Plugins{}
	// 1.from annotations
	for k, v := range pluginsInAnnotation {
		pluginRet[k] = v
	}
	// 2.from service plugins
	for _, p := range plugins {
		if p.Enable {
			if p.Config != nil {
				pluginRet[p.Name] = p.Config
			} else if p.ConfigSet != nil {
				pluginRet[p.Name] = p.ConfigSet
			} else {
				pluginRet[p.Name] = make(map[string]interface{})
			}
		}
	}
	// fullServiceName
	fullServiceName := apisixServiceName
	if group != "" {
		fullServiceName = group + "_" + apisixServiceName
	}

	service := &apisix.Service{
		FullName:        fullServiceName,
		Group:           group,
		ResourceVersion: rv,
		Name:            apisixServiceName,
		UpstreamName:    apisixUpstreamName,
		FromKind:        fromKind,
		Plugins:         pluginRet,
	}
	services = append(services, service)
	// upstream
	// fullUpstreamName
	fullUpstreamName := apisixUpstreamName
	if group != "" {
		fullUpstreamName = group + "_" + apisixUpstreamName
	}
	LBType := DefaultLBType
	var nodes []apisix.Node
	if asb.EnableEndpointSlice {
		nodes = endpoint.BuildEpss(ns, upstreamName, port)
	} else {
		nodes = endpoint.BuildEps(ns, upstreamName, port)
	}
	upstream := &apisix.Upstream{
		FullName:        fullUpstreamName,
		Group:           group,
		ResourceVersion: rv,
		Name:            apisixUpstreamName,
		Type:            LBType,
		Nodes:           nodes,
		FromKind:        fromKind,
	}
	upstreams = append(upstreams, upstream)
	return services, upstreams, nil
}
