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
	"strings"

	"go.uber.org/zap"

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
			upsId := id.GenID(upstreamName)
			route := &apisixv1.Route{
				Metadata: apisixv1.Metadata{
					ID:              id.GenID(routeName),
					FullName:        routeName,
					ResourceVersion: ar.ResourceVersion,
					Name:            routeName,
				},
				Host:         r.Host,
				Path:         p.Path,
				UpstreamName: upstreamName,
				UpstreamId:   upsId,
				Plugins:      pluginMap,
			}
			routes = append(routes, route)

			if _, ok := upstreamMap[upstreamName]; !ok {
				ups, err := t.TranslateUpstream(ar.Namespace, p.Backend.ServiceName, int32(p.Backend.ServicePort))
				if err != nil {
					return nil, nil, err
				}
				ups.FullName = upstreamName
				ups.ID = upsId
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
		if part.Backend != nil && len(part.Backends) > 1 {
			return nil, nil, errors.New("backend and backends are exclusive")
		}
		if part.Backend == nil && len(part.Backends) == 0 {
			return nil, nil, errors.New("no specified backend")
		}
		backends := part.Backends
		backend := part.Backend
		if len(backends) > 0 {
			// Use the first backend as the default backend in Route,
			// others will be configured in traffic-split plugin.
			backend = backends[0]
			backends = backends[1:]
		} // else use the deprecated Backend.

		svcClusterIP, svcPort, err := t.getServiceClusterIPAndPort(backend, ar)
		if err != nil {
			log.Errorw("failed to get service port in backend",
				zap.Any("backend", backend),
				zap.Any("apisix_route", ar),
				zap.Error(err),
			)
			return nil, nil, err
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
		var exprs [][]apisixv1.StringOrSlice
		if part.Match.NginxVars != nil {
			exprs, err = t.translateRouteMatchExprs(part.Match.NginxVars)
			if err != nil {
				log.Errorw("ApisixRoute with bad nginxVars",
					zap.Error(err),
					zap.Any("ApisixRoute", ar),
				)
				return nil, nil, err
			}
		}

		routeName := apisixv1.ComposeRouteName(ar.Namespace, ar.Name, part.Name)
		upstreamName := apisixv1.ComposeUpstreamName(ar.Namespace, backend.ServiceName, svcPort)
		upsId := id.GenID(upstreamName)
		route := &apisixv1.Route{
			Metadata: apisixv1.Metadata{
				FullName:        routeName,
				Name:            routeName,
				ID:              id.GenID(routeName),
				ResourceVersion: ar.ResourceVersion,
			},
			Vars:         exprs,
			Hosts:        part.Match.Hosts,
			Uris:         part.Match.Paths,
			Methods:      part.Match.Methods,
			UpstreamName: upstreamName,
			UpstreamId:   upsId,
			Plugins:      pluginMap,
		}

		if len(backends) > 0 {
			weight := _defaultWeight
			if backend.Weight != 0 {
				weight = backend.Weight
			}
			ups, plugin, err := t.translateTrafficSplitPlugin(ar, weight, backends)
			if err != nil {
				log.Errorw("failed to translate traffic-split plugin",
					zap.Error(err),
					zap.Any("ApisixRoute", ar),
				)
				return nil, nil, err
			}
			for _, u := range ups {
				if _, ok := upstreamMap[u.FullName]; !ok {
					upstreamMap[u.FullName] = u
				}
			}
			route.Plugins["traffic-split"] = plugin
		}

		routes = append(routes, route)
		if _, ok := upstreamMap[upstreamName]; !ok {
			ups, err := t.translateUpstream(ar.Namespace, backend.ServiceName, backend.ResolveGranularity, svcClusterIP, svcPort)
			if err != nil {
				return nil, nil, err
			}
			upstreamMap[ups.FullName] = ups
		}
	}

	for _, ups := range upstreamMap {
		upstreams = append(upstreams, ups)
	}
	return routes, upstreams, nil
}

func (t *translator) translateRouteMatchExprs(nginxVars []configv2alpha1.ApisixRouteHTTPMatchExpr) ([][]apisixv1.StringOrSlice, error) {
	var (
		vars [][]apisixv1.StringOrSlice
		op   string
	)
	for _, expr := range nginxVars {
		var (
			invert bool
			subj   string
			this   []apisixv1.StringOrSlice
		)
		if expr.Subject.Name == "" && expr.Subject.Scope != configv2alpha1.ScopePath {
			return nil, errors.New("empty subject name")
		}
		switch expr.Subject.Scope {
		case configv2alpha1.ScopeQuery:
			subj = "arg_" + strings.ToLower(expr.Subject.Name)
		case configv2alpha1.ScopeHeader:
			name := strings.ToLower(expr.Subject.Name)
			name = strings.ReplaceAll(name, "-", "_")
			subj = "http_" + name
		case configv2alpha1.ScopeCookie:
			name := strings.ToLower(expr.Subject.Name)
			name = strings.ReplaceAll(name, "-", "_")
			subj = "cookie_" + name
		case configv2alpha1.ScopePath:
			subj = "uri"
		default:
			return nil, errors.New("bad subject name")
		}
		if expr.Subject.Scope == "" {
			return nil, errors.New("empty nginxVar subject")
		}
		this = append(this, apisixv1.StringOrSlice{
			StrVal: subj,
		})

		switch expr.Op {
		case configv2alpha1.OpEqual:
			op = "=="
		case configv2alpha1.OpGreaterThan:
			op = ">"
		// TODO Implement "<=", ">=" operators after the
		// lua-resty-expr supports it. See
		// https://github.com/api7/lua-resty-expr/issues/28
		// for details.
		//case configv2alpha1.OpGreaterThanEqual:
		//	invert = true
		//	op = "<"
		case configv2alpha1.OpIn:
			op = "in"
		case configv2alpha1.OpLessThan:
			op = "<"
		//case configv2alpha1.OpLessThanEqual:
		//	invert = true
		//	op = ">"
		case configv2alpha1.OpNotEqual:
			op = "~="
		case configv2alpha1.OpNotIn:
			invert = true
			op = "in"
		case configv2alpha1.OpRegexMatch:
			op = "~~"
		case configv2alpha1.OpRegexMatchCaseInsensitive:
			op = "~*"
		case configv2alpha1.OpRegexNotMatch:
			invert = true
			op = "~~"
		case configv2alpha1.OpRegexNotMatchCaseInsensitive:
			invert = true
			op = "~*"
		default:
			return nil, errors.New("unknown operator")
		}
		if invert {
			this = append(this, apisixv1.StringOrSlice{
				StrVal: "!",
			})
		}
		this = append(this, apisixv1.StringOrSlice{
			StrVal: op,
		})
		if expr.Op == configv2alpha1.OpIn || expr.Op == configv2alpha1.OpNotIn {
			if expr.Set == nil {
				return nil, errors.New("empty set value")
			}
			this = append(this, apisixv1.StringOrSlice{
				SliceVal: expr.Set,
			})
		} else if expr.Value != nil {
			this = append(this, apisixv1.StringOrSlice{
				StrVal: *expr.Value,
			})
		} else {
			return nil, errors.New("neither set nor value is provided")
		}
		vars = append(vars, this)
	}

	return vars, nil
}
