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
package translation

import (
	"errors"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/apache/apisix-ingress-controller/pkg/id"
	configv1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v1"
	configv2alpha1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2alpha1"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

func (t *translator) TranslateRouteV1(ar *configv1.ApisixRoute) ([]*apisixv1.Route, []*apisixv1.Upstream, error) {
	var (
		routes    []*apisixv1.Route
		upstreams []*apisixv1.Upstream
	)

	plugins := t.TranslateAnnotations(ar.Annotations)
	upstreamMap := make(map[string]*apisixv1.Upstream)

	for _, r := range ar.Spec.Rules {
		for _, p := range r.Http.Paths {
			routeName := r.Host + p.Path
			upstreamName := apisixv1.ComposeUpstreamName(ar.Namespace, p.Backend.ServiceName, int32(p.Backend.ServicePort))

			pluginMap := make(apisixv1.Plugins)
			// 1.add annotation plugins
			for k, v := range plugins {
				pluginMap[k] = v
			}
			// 2.add route plugins
			for _, plugin := range p.Plugins {
				if !plugin.Enable {
					continue
				}
				if plugin.Config != nil {
					pluginMap[plugin.Name] = plugin.Config
				} else if plugin.ConfigSet != nil {
					pluginMap[plugin.Name] = plugin.ConfigSet
				} else {
					pluginMap[plugin.Name] = make(map[string]interface{})
				}
			}

			route := &apisixv1.Route{
				Metadata: apisixv1.Metadata{
					FullName:        routeName,
					ResourceVersion: ar.ResourceVersion,
					Name:            routeName,
				},
				Host:         r.Host,
				Path:         p.Path,
				UpstreamName: upstreamName,
				UpstreamId:   id.GenID(upstreamName),
				Plugins:      pluginMap,
			}
			routes = append(routes, route)

			if _, ok := upstreamMap[upstreamName]; !ok {
				ups, err := t.TranslateUpstream(ar.Namespace, p.Backend.ServiceName, int32(p.Backend.ServicePort))
				if err != nil {
					return nil, nil, err
				}
				ups.FullName = upstreamName
				ups.ResourceVersion = ar.ResourceVersion
				ups.Name = upstreamName
				upstreamMap[ups.FullName] = ups
			}
		}
	}
	for _, ups := range upstreamMap {
		upstreams = append(upstreams, ups)
	}
	return routes, upstreams, nil
}

func (t *translator) TranslateRouteV2alpha1(ar *configv2alpha1.ApisixRoute) ([]*apisixv1.Route, []*apisixv1.Upstream, error) {
	var (
		routes    []*apisixv1.Route
		upstreams []*apisixv1.Upstream
	)

	upstreamMap := make(map[string]*apisixv1.Upstream)

	for _, part := range ar.Spec.HTTP {
		if part.Match == nil {
			return nil, nil, errors.New("empty route match section")
		}
		if len(part.Match.Paths) < 1 {
			return nil, nil, errors.New("empty route paths match")
		}
		svc, err := t.ServiceLister.Services(ar.Namespace).Get(part.Backend.ServiceName)
		if err != nil {
			return nil, nil, err
		}
		svcPort := int32(-1)
	loop:
		for _, port := range svc.Spec.Ports {
			switch part.Backend.ServicePort.Type {
			case intstr.Int:
				if part.Backend.ServicePort.IntVal == port.Port {
					svcPort = port.Port
					break loop
				}
			case intstr.String:
				if part.Backend.ServicePort.StrVal == port.Name {
					svcPort = port.Port
					break loop
				}
			}
		}
		if svcPort == -1 {
			log.Errorw("ApisixRoute refers to non-existent Service port",
				zap.Any("ApisixRoute", ar),
				zap.String("port", part.Backend.ServicePort.String()),
			)
			return nil, nil, err
		}

		if part.Backend.ResolveGranularity == "service" && svc.Spec.ClusterIP == "" {
			log.Errorw("ApisixRoute refers to a headless service but want to use the service level resolve granualrity",
				zap.Any("ApisixRoute", ar),
				zap.Any("service", svc),
			)
			return nil, nil, errors.New("conflict headless service and backend resolve granularity")
		}

		pluginMap := make(apisixv1.Plugins)
		// 2.add route plugins
		for _, plugin := range part.Plugins {
			if !plugin.Enable {
				continue
			}
			if plugin.Config != nil {
				pluginMap[plugin.Name] = plugin.Config
			} else {
				pluginMap[plugin.Name] = make(map[string]interface{})
			}
		}

		routeName := apisixv1.ComposeRouteName(ar.Namespace, ar.Name, part.Name)
		upstreamName := apisixv1.ComposeUpstreamName(ar.Namespace, part.Backend.ServiceName, svcPort)
		route := &apisixv1.Route{
			Metadata: apisixv1.Metadata{
				FullName:        routeName,
				Name:            routeName,
				ID:              id.GenID(routeName),
				ResourceVersion: ar.ResourceVersion,
			},
			Hosts:        part.Match.Hosts,
			Uris:         part.Match.Paths,
			Methods:      part.Match.Methods,
			UpstreamName: upstreamName,
			UpstreamId:   id.GenID(upstreamName),
			Plugins:      pluginMap,
		}

		routes = append(routes, route)

		if _, ok := upstreamMap[upstreamName]; !ok {
			ups, err := t.TranslateUpstream(ar.Namespace, part.Backend.ServiceName, svcPort)
			if err != nil {
				return nil, nil, err
			}
			if part.Backend.ResolveGranularity == "service" {
				ups.Nodes = []apisixv1.UpstreamNode{
					{
						IP:     svc.Spec.ClusterIP,
						Port:   int(svcPort),
						Weight: _defaultWeight,
					},
				}
			}
			ups.FullName = upstreamName
			ups.ResourceVersion = ar.ResourceVersion
			ups.Name = upstreamName
			upstreamMap[ups.FullName] = ups
		}
	}

	for _, ups := range upstreamMap {
		upstreams = append(upstreams, ups)
	}
	return routes, upstreams, nil
}
