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
	DefaultLBType      = "roundrobin"
	SSLREDIRECT        = "k8s.apisix.apache.org/ssl-redirect"
	WHITELIST          = "k8s.apisix.apache.org/whitelist-source-range"
	ENABLE_CORS        = "k8s.apisix.apache.org/enable-cors"
	CORS_ALLOW_ORIGIN  = "k8s.apisix.apache.org/cors-allow-origin"
	CORS_ALLOW_HEADERS = "k8s.apisix.apache.org/cors-allow-headers"
	CORS_ALLOW_METHODS = "k8s.apisix.apache.org/cors-allow-methods"
	INGRESS_CLASS      = "k8s.apisix.apache.org/ingress.class"
)

type ApisixRoute configv1.ApisixRoute

// Convert convert to  apisix.Route from ingress.ApisixRoute CRD
func (ar *ApisixRoute) Convert() ([]*apisix.Route, []*apisix.Service, []*apisix.Upstream, error) {
	ns := ar.Namespace
	// meta annotation
	plugins, group := BuildAnnotation(ar.Annotations)
	conf.AddGroup(group)
	// Host
	rules := ar.Spec.Rules
	routes := make([]*apisix.Route, 0)
	services := make([]*apisix.Service, 0)
	serviceMap := make(map[string]*apisix.Service)
	upstreams := make([]*apisix.Upstream, 0)
	upstreamMap := make(map[string]*apisix.Upstream)
	rv := ar.ObjectMeta.ResourceVersion
	for _, r := range rules {
		host := r.Host
		paths := r.Http.Paths
		for _, p := range paths {
			uri := p.Path
			svcName := p.Backend.ServiceName
			svcPort := strconv.Itoa(p.Backend.ServicePort)
			// apisix route name = host + path
			apisixRouteName := host + uri
			// apisix service name = namespace_svcName_svcPort
			apisixSvcName := ns + "_" + svcName + "_" + svcPort
			// apisix route name = namespace_svcName_svcPort = apisix service name
			apisixUpstreamName := ns + "_" + svcName + "_" + svcPort
			// plugins defined in Route Level
			pls := p.Plugins
			pluginRet := make(apisix.Plugins)
			// 1.add annotation plugins
			for k, v := range plugins {
				pluginRet[k] = v
			}
			// 2.add route plugins
			for _, p := range pls {
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
			// fullRouteName
			fullRouteName := apisixRouteName
			if group != "" {
				fullRouteName = group + "_" + apisixRouteName
			}

			// routes
			route := &apisix.Route{
				Group:           &group,
				FullName:        &fullRouteName,
				ResourceVersion: &rv,
				Name:            &apisixRouteName,
				Host:            &host,
				Path:            &uri,
				ServiceName:     &apisixSvcName,
				UpstreamName:    &apisixUpstreamName,
				Plugins:         &pluginRet,
			}
			routes = append(routes, route)
			// services
			// fullServiceName
			fullServiceName := apisixSvcName
			if group != "" {
				fullServiceName = group + "_" + apisixSvcName
			}

			service := &apisix.Service{
				FullName:        &fullServiceName,
				Group:           &group,
				Name:            &apisixSvcName,
				UpstreamName:    &apisixUpstreamName,
				ResourceVersion: &rv,
			}
			serviceMap[*service.FullName] = service

			// upstreams
			// fullServiceName
			fullUpstreamName := apisixUpstreamName
			if group != "" {
				fullUpstreamName = group + "_" + apisixUpstreamName
			}
			LBType := DefaultLBType
			port, _ := strconv.Atoi(svcPort)
			nodes := endpoint.BuildEps(ns, svcName, port)
			upstream := &apisix.Upstream{
				FullName:        &fullUpstreamName,
				Group:           &group,
				ResourceVersion: &rv,
				Name:            &apisixUpstreamName,
				Type:            &LBType,
				Nodes:           nodes,
			}
			upstreamMap[*upstream.FullName] = upstream
		}
	}
	for _, s := range serviceMap {
		services = append(services, s)
	}
	for _, u := range upstreamMap {
		upstreams = append(upstreams, u)
	}
	return routes, services, upstreams, nil
}
