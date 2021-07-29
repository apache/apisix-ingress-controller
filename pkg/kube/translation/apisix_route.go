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
	configv2beta1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta1"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

func (t *translator) TranslateRouteV1(ar *configv1.ApisixRoute) (*TranslateContext, error) {
	ctx := &TranslateContext{
		upstreamMap: make(map[string]struct{}),
	}
	plugins := t.translateAnnotations(ar.Annotations)

	for _, r := range ar.Spec.Rules {
		for _, p := range r.Http.Paths {
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

			upstreamName := apisixv1.ComposeUpstreamName(ar.Namespace, p.Backend.ServiceName, "", int32(p.Backend.ServicePort))
			route := apisixv1.NewDefaultRoute()
			route.Name = r.Host + p.Path
			route.ID = id.GenID(route.Name)
			route.Host = r.Host
			route.Uri = p.Path
			route.Plugins = pluginMap
			route.UpstreamId = id.GenID(upstreamName)

			if !ctx.checkUpstreamExist(upstreamName) {
				ups, err := t.TranslateUpstream(ar.Namespace, p.Backend.ServiceName, "", int32(p.Backend.ServicePort))
				if err != nil {
					return nil, err
				}
				ups.ID = route.UpstreamId
				ups.Name = upstreamName
				ctx.addUpstream(ups)
			}
			ctx.addRoute(route)
		}
	}
	return ctx, nil
}

// TranslateRouteV2alpha1NotStrictly translates route v2alpha1 with a loose way, only generate ID and Name for delete Event.
func (t *translator) TranslateRouteV2alpha1NotStrictly(ar *configv2alpha1.ApisixRoute) (*TranslateContext, error) {
	ctx := &TranslateContext{
		upstreamMap: make(map[string]struct{}),
	}

	if err := t.translateHTTPRouteNotStrictly(ctx, ar); err != nil {
		return nil, err
	}
	if err := t.translateTCPRouteNotStrictly(ctx, ar); err != nil {
		return nil, err
	}
	return ctx, nil
}

func (t *translator) TranslateRouteV2alpha1(ar *configv2alpha1.ApisixRoute) (*TranslateContext, error) {
	ctx := &TranslateContext{
		upstreamMap: make(map[string]struct{}),
	}

	if err := t.translateHTTPRoute(ctx, ar); err != nil {
		return nil, err
	}
	if err := t.translateTCPRoute(ctx, ar); err != nil {
		return nil, err
	}
	return ctx, nil
}

// translateHTTPRouteNotStrictly translates http route with a loose way, only generate ID and Name for delete Event.
func (t *translator) translateHTTPRouteNotStrictly(ctx *TranslateContext, ar *configv2alpha1.ApisixRoute) error {
	for _, part := range ar.Spec.HTTP {
		backends := part.Backends
		backend := part.Backend
		if len(backends) > 0 {
			// Use the first backend as the default backend in Route,
			// others will be configured in traffic-split plugin.
			backend = backends[0]
		} // else use the deprecated Backend.
		upstreamName := apisixv1.ComposeUpstreamName(ar.Namespace, backend.ServiceName, backend.Subset, backend.ServicePort.IntVal)
		route := apisixv1.NewDefaultRoute()
		route.Name = apisixv1.ComposeRouteName(ar.Namespace, ar.Name, part.Name)
		route.ID = id.GenID(route.Name)
		ctx.addRoute(route)
		if !ctx.checkUpstreamExist(upstreamName) {
			ups, err := t.translateUpstreamNotStrictly(ar.Namespace, backend.ServiceName, backend.Subset, backend.ServicePort.IntVal)
			if err != nil {
				return err
			}
			ctx.addUpstream(ups)
		}
	}
	return nil
}

func (t *translator) TranslateRouteV2beta1(ar *configv2beta1.ApisixRoute) (*TranslateContext, error) {
	ctx := &TranslateContext{
		upstreamMap: make(map[string]struct{}),
	}

	if err := t.translateHTTPRouteV2beta1(ctx, ar); err != nil {
		return nil, err
	}
	if err := t.translateStreamRoute(ctx, ar); err != nil {
		return nil, err
	}
	return ctx, nil
}

func (t *translator) TranslateRouteV2beta1NotStrictly(ar *configv2beta1.ApisixRoute) (*TranslateContext, error) {
	ctx := &TranslateContext{
		upstreamMap: make(map[string]struct{}),
	}

	if err := t.translateHTTPRouteV2beta1NotStrictly(ctx, ar); err != nil {
		return nil, err
	}
	if err := t.translateStreamRouteNotStrictly(ctx, ar); err != nil {
		return nil, err
	}
	return ctx, nil
}

func (t *translator) translateHTTPRouteV2beta1(ctx *TranslateContext, ar *configv2beta1.ApisixRoute) error {
	ruleNameMap := make(map[string]struct{})
	for _, part := range ar.Spec.HTTP {
		if _, ok := ruleNameMap[part.Name]; ok {
			return errors.New("duplicated route rule name")
		}
		ruleNameMap[part.Name] = struct{}{}
		backends := part.Backends
		backend := part.Backend
		if len(backends) > 0 {
			// Use the first backend as the default backend in Route,
			// others will be configured in traffic-split plugin.
			backend = backends[0]
			backends = backends[1:]
		} // else use the deprecated Backend.

		svcClusterIP, svcPort, err := t.getServiceClusterIPAndPort(&backend, ar.Namespace)
		if err != nil {
			log.Errorw("failed to get service port in backend",
				zap.Any("backend", backend),
				zap.Any("apisix_route", ar),
				zap.Error(err),
			)
			return err
		}

		pluginMap := make(apisixv1.Plugins)
		// add route plugins
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

		// add KeyAuth and basicAuth plugin
		if part.Authentication.Enable {
			switch part.Authentication.Type {
			case "keyAuth":
				pluginMap["key-auth"] = part.Authentication.KeyAuth
			case "basicAuth":
				pluginMap["basic-auth"] = make(map[string]interface{})
			default:
				pluginMap["basic-auth"] = make(map[string]interface{})
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
				return err
			}
		}
		if err := validateRemoteAddrs(part.Match.RemoteAddrs); err != nil {
			log.Errorw("ApisixRoute with invalid remote addrs",
				zap.Error(err),
				zap.Strings("remote_addrs", part.Match.RemoteAddrs),
				zap.Any("ApisixRoute", ar),
			)
			return err
		}

		upstreamName := apisixv1.ComposeUpstreamName(ar.Namespace, backend.ServiceName, backend.Subset, svcPort)
		route := apisixv1.NewDefaultRoute()
		route.Name = apisixv1.ComposeRouteName(ar.Namespace, ar.Name, part.Name)
		route.ID = id.GenID(route.Name)
		route.Priority = part.Priority
		route.RemoteAddrs = part.Match.RemoteAddrs
		route.Vars = exprs
		route.Hosts = part.Match.Hosts
		route.Uris = part.Match.Paths
		route.Methods = part.Match.Methods
		route.UpstreamId = id.GenID(upstreamName)
		route.EnableWebsocket = part.Websocket
		route.Plugins = pluginMap

		if len(backends) > 0 {
			weight := _defaultWeight
			if backend.Weight != nil {
				weight = *backend.Weight
			}
			backendPoints := make([]*configv2alpha1.ApisixRouteHTTPBackend, 0)
			for _, b := range backends {
				backendPoints = append(backendPoints, &b)
			}
			plugin, err := t.translateTrafficSplitPlugin(ctx, ar.Namespace, weight, backendPoints)
			if err != nil {
				log.Errorw("failed to translate traffic-split plugin",
					zap.Error(err),
					zap.Any("ApisixRoute", ar),
				)
				return err
			}
			route.Plugins["traffic-split"] = plugin
		}
		ctx.addRoute(route)
		if !ctx.checkUpstreamExist(upstreamName) {
			ups, err := t.translateUpstream(ar.Namespace, backend.ServiceName, backend.Subset, backend.ResolveGranularity, svcClusterIP, svcPort)
			if err != nil {
				return err
			}
			ctx.addUpstream(ups)
		}
	}
	return nil
}

func (t *translator) translateHTTPRoute(ctx *TranslateContext, ar *configv2alpha1.ApisixRoute) error {
	ruleNameMap := make(map[string]struct{})
	for _, part := range ar.Spec.HTTP {
		if _, ok := ruleNameMap[part.Name]; ok {
			return errors.New("duplicated route rule name")
		}
		ruleNameMap[part.Name] = struct{}{}
		backends := part.Backends
		backend := part.Backend
		if len(backends) > 0 {
			// Use the first backend as the default backend in Route,
			// others will be configured in traffic-split plugin.
			backend = backends[0]
			backends = backends[1:]
		} // else use the deprecated Backend.

		svcClusterIP, svcPort, err := t.getServiceClusterIPAndPort(backend, ar.Namespace)
		if err != nil {
			log.Errorw("failed to get service port in backend",
				zap.Any("backend", backend),
				zap.Any("apisix_route", ar),
				zap.Error(err),
			)
			return err
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

		// add KeyAuth and basicAuth plugin
		if part.Authentication != nil && part.Authentication.Enable {
			switch part.Authentication.Type {
			case "keyAuth":
				pluginMap["key-auth"] = part.Authentication.KeyAuth
			case "basicAuth":
				pluginMap["basic-auth"] = make(map[string]interface{})
			default:
				pluginMap["basic-auth"] = make(map[string]interface{})
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
				return err
			}
		}
		if err := validateRemoteAddrs(part.Match.RemoteAddrs); err != nil {
			log.Errorw("ApisixRoute with invalid remote addrs",
				zap.Error(err),
				zap.Strings("remote_addrs", part.Match.RemoteAddrs),
				zap.Any("ApisixRoute", ar),
			)
			return err
		}

		upstreamName := apisixv1.ComposeUpstreamName(ar.Namespace, backend.ServiceName, backend.Subset, svcPort)
		route := apisixv1.NewDefaultRoute()
		route.Name = apisixv1.ComposeRouteName(ar.Namespace, ar.Name, part.Name)
		route.ID = id.GenID(route.Name)
		route.Priority = part.Priority
		route.RemoteAddrs = part.Match.RemoteAddrs
		route.Vars = exprs
		route.Hosts = part.Match.Hosts
		route.Uris = part.Match.Paths
		route.Methods = part.Match.Methods
		route.UpstreamId = id.GenID(upstreamName)
		route.EnableWebsocket = part.Websocket
		route.Plugins = pluginMap

		if len(backends) > 0 {
			weight := _defaultWeight
			if backend.Weight != nil {
				weight = *backend.Weight
			}
			plugin, err := t.translateTrafficSplitPlugin(ctx, ar.Namespace, weight, backends)
			if err != nil {
				log.Errorw("failed to translate traffic-split plugin",
					zap.Error(err),
					zap.Any("ApisixRoute", ar),
				)
				return err
			}
			route.Plugins["traffic-split"] = plugin
		}
		ctx.addRoute(route)
		if !ctx.checkUpstreamExist(upstreamName) {
			ups, err := t.translateUpstream(ar.Namespace, backend.ServiceName, backend.Subset, backend.ResolveGranularity, svcClusterIP, svcPort)
			if err != nil {
				return err
			}
			ctx.addUpstream(ups)
		}
	}
	return nil
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
			subj = "cookie_" + expr.Subject.Name
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

// translateTCPRouteNotStrictly translates tcp route with a loose way, only generate ID and Name for delete Event.
func (t *translator) translateTCPRouteNotStrictly(ctx *TranslateContext, ar *configv2alpha1.ApisixRoute) error {
	for _, part := range ar.Spec.TCP {
		backend := &part.Backend
		sr := apisixv1.NewDefaultStreamRoute()
		name := apisixv1.ComposeStreamRouteName(ar.Namespace, ar.Name, part.Name)
		sr.ID = id.GenID(name)
		sr.ServerPort = part.Match.IngressPort
		ups, err := t.translateUpstreamNotStrictly(ar.Namespace, backend.ServiceName, backend.Subset, backend.ServicePort.IntVal)
		if err != nil {
			return err
		}
		sr.UpstreamId = ups.ID
		ctx.addStreamRoute(sr)
		if !ctx.checkUpstreamExist(ups.Name) {
			ctx.addUpstream(ups)
		}
	}
	return nil
}

func (t *translator) translateStreamRoute(ctx *TranslateContext, ar *configv2beta1.ApisixRoute) error {
	ruleNameMap := make(map[string]struct{})
	for _, part := range ar.Spec.Stream {
		if _, ok := ruleNameMap[part.Name]; ok {
			return errors.New("duplicated route rule name")
		}
		ruleNameMap[part.Name] = struct{}{}
		backend := part.Backend
		svcClusterIP, svcPort, err := t.getStreamServiceClusterIPAndPort(backend, ar.Namespace)
		if err != nil {
			log.Errorw("failed to get service port in backend",
				zap.Any("backend", backend),
				zap.Any("apisix_route", ar),
				zap.Error(err),
			)
			return err
		}
		sr := apisixv1.NewDefaultStreamRoute()
		name := apisixv1.ComposeStreamRouteName(ar.Namespace, ar.Name, part.Name)
		sr.ID = id.GenID(name)
		sr.ServerPort = part.Match.IngressPort
		ups, err := t.translateUpstream(ar.Namespace, backend.ServiceName, backend.Subset, backend.ResolveGranularity, svcClusterIP, svcPort)
		if err != nil {
			return err
		}
		sr.UpstreamId = ups.ID
		ctx.addStreamRoute(sr)
		if !ctx.checkUpstreamExist(ups.Name) {
			ctx.addUpstream(ups)
		}

	}
	return nil
}

func (t *translator) translateTCPRoute(ctx *TranslateContext, ar *configv2alpha1.ApisixRoute) error {
	ruleNameMap := make(map[string]struct{})
	for _, part := range ar.Spec.TCP {
		if _, ok := ruleNameMap[part.Name]; ok {
			return errors.New("duplicated route rule name")
		}
		ruleNameMap[part.Name] = struct{}{}
		backend := &part.Backend
		svcClusterIP, svcPort, err := t.getTCPServiceClusterIPAndPort(backend, ar)
		if err != nil {
			log.Errorw("failed to get service port in backend",
				zap.Any("backend", backend),
				zap.Any("apisix_route", ar),
				zap.Error(err),
			)
			return err
		}
		sr := apisixv1.NewDefaultStreamRoute()
		name := apisixv1.ComposeStreamRouteName(ar.Namespace, ar.Name, part.Name)
		sr.ID = id.GenID(name)
		sr.ServerPort = part.Match.IngressPort
		ups, err := t.translateUpstream(ar.Namespace, backend.ServiceName, backend.Subset, backend.ResolveGranularity, svcClusterIP, svcPort)
		if err != nil {
			return err
		}
		sr.UpstreamId = ups.ID
		ctx.addStreamRoute(sr)
		if !ctx.checkUpstreamExist(ups.Name) {
			ctx.addUpstream(ups)
		}

	}
	return nil
}

// translateHTTPRouteV2beta1NotStrictly translates http route with a loose way, only generate ID and Name for delete Event.
func (t *translator) translateHTTPRouteV2beta1NotStrictly(ctx *TranslateContext, ar *configv2beta1.ApisixRoute) error {
	for _, part := range ar.Spec.HTTP {
		backends := part.Backends
		backend := part.Backend
		if len(backends) > 0 {
			// Use the first backend as the default backend in Route,
			// others will be configured in traffic-split plugin.
			backend = backends[0]
		} // else use the deprecated Backend.
		upstreamName := apisixv1.ComposeUpstreamName(ar.Namespace, backend.ServiceName, backend.Subset, backend.ServicePort.IntVal)
		route := apisixv1.NewDefaultRoute()
		route.Name = apisixv1.ComposeRouteName(ar.Namespace, ar.Name, part.Name)
		route.ID = id.GenID(route.Name)
		ctx.addRoute(route)
		if !ctx.checkUpstreamExist(upstreamName) {
			ups, err := t.translateUpstreamNotStrictly(ar.Namespace, backend.ServiceName, backend.Subset, backend.ServicePort.IntVal)
			if err != nil {
				return err
			}
			ctx.addUpstream(ups)
		}
	}
	return nil
}

// translateStreamRouteNotStrictly translates tcp route with a loose way, only generate ID and Name for delete Event.
func (t *translator) translateStreamRouteNotStrictly(ctx *TranslateContext, ar *configv2beta1.ApisixRoute) error {
	for _, part := range ar.Spec.Stream {
		backend := &part.Backend
		sr := apisixv1.NewDefaultStreamRoute()
		name := apisixv1.ComposeStreamRouteName(ar.Namespace, ar.Name, part.Name)
		sr.ID = id.GenID(name)
		sr.ServerPort = part.Match.IngressPort
		ups, err := t.translateUpstreamNotStrictly(ar.Namespace, backend.ServiceName, backend.Subset, backend.ServicePort.IntVal)
		if err != nil {
			return err
		}
		sr.UpstreamId = ups.ID
		ctx.addStreamRoute(sr)
		if !ctx.checkUpstreamExist(ups.Name) {
			ctx.addUpstream(ups)
		}
	}
	return nil
}
