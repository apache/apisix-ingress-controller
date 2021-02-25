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
	"github.com/apache/apisix-ingress-controller/pkg/id"
	configv1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v1"
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
