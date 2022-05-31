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
	configv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	configv2beta2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta2"
	configv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta3"
	_const "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/const"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

func (t *translator) TranslateRouteV2beta2(ar *configv2beta2.ApisixRoute) (*TranslateContext, error) {
	ctx := DefaultEmptyTranslateContext()

	if err := t.translateHTTPRouteV2beta2(ctx, ar); err != nil {
		return nil, err
	}
	if err := t.translateStreamRouteV2beta2(ctx, ar); err != nil {
		return nil, err
	}
	return ctx, nil
}

func (t *translator) TranslateRouteV2beta2NotStrictly(ar *configv2beta2.ApisixRoute) (*TranslateContext, error) {
	ctx := DefaultEmptyTranslateContext()

	if err := t.translateHTTPRouteV2beta2NotStrictly(ctx, ar); err != nil {
		return nil, err
	}
	if err := t.translateStreamRouteNotStrictlyV2beta2(ctx, ar); err != nil {
		return nil, err
	}
	return ctx, nil
}

func (t *translator) TranslateRouteV2beta3(ar *configv2beta3.ApisixRoute) (*TranslateContext, error) {
	ctx := DefaultEmptyTranslateContext()

	if err := t.translateHTTPRouteV2beta3(ctx, ar); err != nil {
		return nil, err
	}
	if err := t.translateStreamRouteV2beta3(ctx, ar); err != nil {
		return nil, err
	}
	return ctx, nil
}

func (t *translator) TranslateRouteV2beta3NotStrictly(ar *configv2beta3.ApisixRoute) (*TranslateContext, error) {
	ctx := DefaultEmptyTranslateContext()

	if err := t.translateHTTPRouteV2beta3NotStrictly(ctx, ar); err != nil {
		return nil, err
	}
	if err := t.translateStreamRouteNotStrictlyV2beta3(ctx, ar); err != nil {
		return nil, err
	}
	return ctx, nil
}

func (t *translator) TranslateRouteV2(ar *configv2.ApisixRoute) (*TranslateContext, error) {
	ctx := DefaultEmptyTranslateContext()

	if err := t.translateHTTPRouteV2(ctx, ar); err != nil {
		return nil, err
	}
	if err := t.translateStreamRouteV2(ctx, ar); err != nil {
		return nil, err
	}
	return ctx, nil
}

func (t *translator) TranslateRouteV2NotStrictly(ar *configv2.ApisixRoute) (*TranslateContext, error) {
	ctx := DefaultEmptyTranslateContext()

	if err := t.translateHTTPRouteV2NotStrictly(ctx, ar); err != nil {
		return nil, err
	}
	if err := t.translateStreamRouteNotStrictlyV2(ctx, ar); err != nil {
		return nil, err
	}
	return ctx, nil
}

func (t *translator) translateHTTPRouteV2beta2(ctx *TranslateContext, ar *configv2beta2.ApisixRoute) error {
	ruleNameMap := make(map[string]struct{})
	for _, part := range ar.Spec.HTTP {
		if _, ok := ruleNameMap[part.Name]; ok {
			return errors.New("duplicated route rule name")
		}
		ruleNameMap[part.Name] = struct{}{}
		backends := part.Backends
		// Use the first backend as the default backend in Route,
		// others will be configured in traffic-split plugin.
		backend := backends[0]
		backends = backends[1:]

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
			backendPoints := make([]configv2.ApisixRouteHTTPBackend, 0)
			backendPoints = append(backendPoints, backends...)
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
		ctx.AddRoute(route)
		if !ctx.CheckUpstreamExist(upstreamName) {
			ups, err := t.translateUpstream(ar.Namespace, backend.ServiceName, backend.Subset, backend.ResolveGranularity, svcClusterIP, svcPort)
			if err != nil {
				return err
			}
			ctx.AddUpstream(ups)
		}
	}
	return nil
}

func (t *translator) translateHTTPRouteV2beta3(ctx *TranslateContext, ar *configv2beta3.ApisixRoute) error {
	ruleNameMap := make(map[string]struct{})
	for _, part := range ar.Spec.HTTP {
		if _, ok := ruleNameMap[part.Name]; ok {
			return errors.New("duplicated route rule name")
		}
		ruleNameMap[part.Name] = struct{}{}
		backends := part.Backends
		// Use the first backend as the default backend in Route,
		// others will be configured in traffic-split plugin.
		backend := backends[0]
		backends = backends[1:]

		svcClusterIP, svcPort, err := t.getServiceClusterIPAndPort(&backend, ar.Namespace)
		if err != nil {
			log.Errorw("failed to get service port in backend",
				zap.Any("backend", backend),
				zap.Any("apisix_route", ar),
				zap.Error(err),
			)
			return err
		}

		var timeout *apisixv1.UpstreamTimeout
		if part.Timeout != nil {
			timeout = &apisixv1.UpstreamTimeout{
				Connect: apisixv1.DefaultUpstreamTimeout,
				Read:    apisixv1.DefaultUpstreamTimeout,
				Send:    apisixv1.DefaultUpstreamTimeout,
			}
			if part.Timeout.Connect.Duration > 0 {
				timeout.Connect = int(part.Timeout.Connect.Seconds())
			}
			if part.Timeout.Read.Duration > 0 {
				timeout.Read = int(part.Timeout.Read.Seconds())
			}
			if part.Timeout.Send.Duration > 0 {
				timeout.Send = int(part.Timeout.Send.Seconds())
			}
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
			case "wolfRBAC":
				pluginMap["wolf-rbac"] = make(map[string]interface{})
			case "jwtAuth":
				pluginMap["jwt-auth"] = part.Authentication.JwtAuth
			case "hmacAuth":
				pluginMap["hmac-auth"] = make(map[string]interface{})
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
		route.Timeout = timeout
		if part.PluginConfigName != "" {
			route.PluginConfigId = id.GenID(apisixv1.ComposePluginConfigName(ar.Namespace, part.PluginConfigName))
		}

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
		ctx.AddRoute(route)
		if !ctx.CheckUpstreamExist(upstreamName) {
			ups, err := t.translateUpstream(ar.Namespace, backend.ServiceName, backend.Subset, backend.ResolveGranularity, svcClusterIP, svcPort)
			if err != nil {
				return err
			}
			ctx.AddUpstream(ups)
		}
	}
	return nil
}

func (t *translator) translateHTTPRouteV2(ctx *TranslateContext, ar *configv2.ApisixRoute) error {
	ruleNameMap := make(map[string]struct{})
	for _, part := range ar.Spec.HTTP {
		if _, ok := ruleNameMap[part.Name]; ok {
			return errors.New("duplicated route rule name")
		}
		ruleNameMap[part.Name] = struct{}{}
		backends := part.Backends
		// Use the first backend as the default backend in Route,
		// others will be configured in traffic-split plugin.
		backend := backends[0]
		backends = backends[1:]

		svcClusterIP, svcPort, err := t.getServiceClusterIPAndPort(&backend, ar.Namespace)
		if err != nil {
			log.Errorw("failed to get service port in backend",
				zap.Any("backend", backend),
				zap.Any("apisix_route", ar),
				zap.Error(err),
			)
			return err
		}

		var timeout *apisixv1.UpstreamTimeout
		if part.Timeout != nil {
			timeout = &apisixv1.UpstreamTimeout{
				Connect: apisixv1.DefaultUpstreamTimeout,
				Read:    apisixv1.DefaultUpstreamTimeout,
				Send:    apisixv1.DefaultUpstreamTimeout,
			}
			if part.Timeout.Connect.Duration > 0 {
				timeout.Connect = int(part.Timeout.Connect.Seconds())
			}
			if part.Timeout.Read.Duration > 0 {
				timeout.Read = int(part.Timeout.Read.Seconds())
			}
			if part.Timeout.Send.Duration > 0 {
				timeout.Send = int(part.Timeout.Send.Seconds())
			}
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
			case "wolfRBAC":
				pluginMap["wolf-rbac"] = make(map[string]interface{})
			case "jwtAuth":
				pluginMap["jwt-auth"] = part.Authentication.JwtAuth
			case "hmacAuth":
				pluginMap["hmac-auth"] = make(map[string]interface{})
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
		route.Timeout = timeout
		if part.PluginConfigName != "" {
			route.PluginConfigId = id.GenID(apisixv1.ComposePluginConfigName(ar.Namespace, part.PluginConfigName))
		}

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
		ctx.AddRoute(route)
		if !ctx.CheckUpstreamExist(upstreamName) {
			ups, err := t.translateUpstream(ar.Namespace, backend.ServiceName, backend.Subset, backend.ResolveGranularity, svcClusterIP, svcPort)
			if err != nil {
				return err
			}
			ctx.AddUpstream(ups)
		}
	}
	return nil
}

func (t *translator) translateRouteMatchExprs(nginxVars []configv2.ApisixRouteHTTPMatchExpr) ([][]apisixv1.StringOrSlice, error) {
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
		if expr.Subject.Name == "" && expr.Subject.Scope != _const.ScopePath {
			return nil, errors.New("empty subject name")
		}
		switch expr.Subject.Scope {
		case _const.ScopeQuery:
			subj = "arg_" + strings.ToLower(expr.Subject.Name)
		case _const.ScopeHeader:
			name := strings.ToLower(expr.Subject.Name)
			name = strings.ReplaceAll(name, "-", "_")
			subj = "http_" + name
		case _const.ScopeCookie:
			subj = "cookie_" + expr.Subject.Name
		case _const.ScopePath:
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
		case _const.OpEqual:
			op = "=="
		case _const.OpGreaterThan:
			op = ">"
		// TODO Implement "<=", ">=" operators after the
		// lua-resty-expr supports it. See
		// https://github.com/api7/lua-resty-expr/issues/28
		// for details.
		//case configv2alpha1.OpGreaterThanEqual:
		//	invert = true
		//	op = "<"
		case _const.OpIn:
			op = "in"
		case _const.OpLessThan:
			op = "<"
		//case configv2alpha1.OpLessThanEqual:
		//	invert = true
		//	op = ">"
		case _const.OpNotEqual:
			op = "~="
		case _const.OpNotIn:
			invert = true
			op = "in"
		case _const.OpRegexMatch:
			op = "~~"
		case _const.OpRegexMatchCaseInsensitive:
			op = "~*"
		case _const.OpRegexNotMatch:
			invert = true
			op = "~~"
		case _const.OpRegexNotMatchCaseInsensitive:
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
		if expr.Op == _const.OpIn || expr.Op == _const.OpNotIn {
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

// translateHTTPRouteV2beta2NotStrictly translates http route with a loose way, only generate ID and Name for delete Event.
func (t *translator) translateHTTPRouteV2beta2NotStrictly(ctx *TranslateContext, ar *configv2beta2.ApisixRoute) error {
	for _, part := range ar.Spec.HTTP {
		backends := part.Backends
		// Use the first backend as the default backend in Route,
		// others will be configured in traffic-split plugin.
		backend := backends[0]
		upstreamName := apisixv1.ComposeUpstreamName(ar.Namespace, backend.ServiceName, backend.Subset, backend.ServicePort.IntVal)
		route := apisixv1.NewDefaultRoute()
		route.Name = apisixv1.ComposeRouteName(ar.Namespace, ar.Name, part.Name)
		route.ID = id.GenID(route.Name)
		ctx.AddRoute(route)
		if !ctx.CheckUpstreamExist(upstreamName) {
			ups, err := t.translateUpstreamNotStrictly(ar.Namespace, backend.ServiceName, backend.Subset, backend.ServicePort.IntVal)
			if err != nil {
				return err
			}
			ctx.AddUpstream(ups)
		}
	}
	return nil
}

// translateHTTPRouteV2beta3NotStrictly translates http route with a loose way, only generate ID and Name for delete Event.
func (t *translator) translateHTTPRouteV2beta3NotStrictly(ctx *TranslateContext, ar *configv2beta3.ApisixRoute) error {
	for _, part := range ar.Spec.HTTP {
		backends := part.Backends
		// Use the first backend as the default backend in Route,
		// others will be configured in traffic-split plugin.
		backend := backends[0]

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
			case "wolfRBAC":
				pluginMap["wolf-rbac"] = make(map[string]interface{})
			case "jwtAuth":
				pluginMap["jwt-auth"] = part.Authentication.JwtAuth
			case "hmacAuth":
				pluginMap["hmac-auth"] = make(map[string]interface{})
			default:
				pluginMap["basic-auth"] = make(map[string]interface{})
			}
		}

		upstreamName := apisixv1.ComposeUpstreamName(ar.Namespace, backend.ServiceName, backend.Subset, backend.ServicePort.IntVal)
		route := apisixv1.NewDefaultRoute()
		route.Name = apisixv1.ComposeRouteName(ar.Namespace, ar.Name, part.Name)
		route.ID = id.GenID(route.Name)
		if part.PluginConfigName != "" {
			route.PluginConfigId = id.GenID(apisixv1.ComposePluginConfigName(ar.Namespace, part.PluginConfigName))
		}

		ctx.AddRoute(route)
		if !ctx.CheckUpstreamExist(upstreamName) {
			ups, err := t.translateUpstreamNotStrictly(ar.Namespace, backend.ServiceName, backend.Subset, backend.ServicePort.IntVal)
			if err != nil {
				return err
			}
			ctx.AddUpstream(ups)
		}
	}
	return nil
}

// translateHTTPRouteV2NotStrictly translates http route with a loose way, only generate ID and Name for delete Event.
func (t *translator) translateHTTPRouteV2NotStrictly(ctx *TranslateContext, ar *configv2.ApisixRoute) error {
	for _, part := range ar.Spec.HTTP {
		backends := part.Backends
		// Use the first backend as the default backend in Route,
		// others will be configured in traffic-split plugin.
		backend := backends[0]

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
			case "wolfRBAC":
				pluginMap["wolf-rbac"] = make(map[string]interface{})
			case "jwtAuth":
				pluginMap["jwt-auth"] = part.Authentication.JwtAuth
			case "hmacAuth":
				pluginMap["hmac-auth"] = make(map[string]interface{})
			default:
				pluginMap["basic-auth"] = make(map[string]interface{})
			}
		}

		upstreamName := apisixv1.ComposeUpstreamName(ar.Namespace, backend.ServiceName, backend.Subset, backend.ServicePort.IntVal)
		route := apisixv1.NewDefaultRoute()
		route.Name = apisixv1.ComposeRouteName(ar.Namespace, ar.Name, part.Name)
		route.ID = id.GenID(route.Name)
		if part.PluginConfigName != "" {
			route.PluginConfigId = id.GenID(apisixv1.ComposePluginConfigName(ar.Namespace, part.PluginConfigName))
		}

		ctx.AddRoute(route)
		if !ctx.CheckUpstreamExist(upstreamName) {
			ups, err := t.translateUpstreamNotStrictly(ar.Namespace, backend.ServiceName, backend.Subset, backend.ServicePort.IntVal)
			if err != nil {
				return err
			}
			ctx.AddUpstream(ups)
		}
	}
	return nil
}

func (t *translator) translateStreamRouteV2beta2(ctx *TranslateContext, ar *configv2beta2.ApisixRoute) error {
	ruleNameMap := make(map[string]struct{})
	for _, part := range ar.Spec.Stream {
		if _, ok := ruleNameMap[part.Name]; ok {
			return errors.New("duplicated route rule name")
		}
		ruleNameMap[part.Name] = struct{}{}
		backend := part.Backend
		svcClusterIP, svcPort, err := t.getStreamServiceClusterIPAndPortV2beta2(backend, ar.Namespace)
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
		ctx.AddStreamRoute(sr)
		if !ctx.CheckUpstreamExist(ups.Name) {
			ctx.AddUpstream(ups)
		}

	}
	return nil
}

func (t *translator) translateStreamRouteV2beta3(ctx *TranslateContext, ar *configv2beta3.ApisixRoute) error {
	ruleNameMap := make(map[string]struct{})
	for _, part := range ar.Spec.Stream {
		if _, ok := ruleNameMap[part.Name]; ok {
			return errors.New("duplicated route rule name")
		}
		ruleNameMap[part.Name] = struct{}{}
		backend := part.Backend
		svcClusterIP, svcPort, err := t.getStreamServiceClusterIPAndPortV2beta3(backend, ar.Namespace)
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
		ctx.AddStreamRoute(sr)
		if !ctx.CheckUpstreamExist(ups.Name) {
			ctx.AddUpstream(ups)
		}

	}
	return nil
}

func (t *translator) translateStreamRouteV2(ctx *TranslateContext, ar *configv2.ApisixRoute) error {
	ruleNameMap := make(map[string]struct{})
	for _, part := range ar.Spec.Stream {
		if _, ok := ruleNameMap[part.Name]; ok {
			return errors.New("duplicated route rule name")
		}
		ruleNameMap[part.Name] = struct{}{}
		backend := part.Backend
		svcClusterIP, svcPort, err := t.getStreamServiceClusterIPAndPortV2(backend, ar.Namespace)
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
		ctx.AddStreamRoute(sr)
		if !ctx.CheckUpstreamExist(ups.Name) {
			ctx.AddUpstream(ups)
		}

	}
	return nil
}

// translateStreamRouteNotStrictlyV2beta2 translates tcp route with a loose way, only generate ID and Name for delete Event.
func (t *translator) translateStreamRouteNotStrictlyV2beta2(ctx *TranslateContext, ar *configv2beta2.ApisixRoute) error {
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
		ctx.AddStreamRoute(sr)
		if !ctx.CheckUpstreamExist(ups.Name) {
			ctx.AddUpstream(ups)
		}
	}
	return nil
}

// translateStreamRouteNotStrictlyV2beta3 translates tcp route with a loose way, only generate ID and Name for delete Event.
func (t *translator) translateStreamRouteNotStrictlyV2beta3(ctx *TranslateContext, ar *configv2beta3.ApisixRoute) error {
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
		ctx.AddStreamRoute(sr)
		if !ctx.CheckUpstreamExist(ups.Name) {
			ctx.AddUpstream(ups)
		}
	}
	return nil
}

// translateStreamRouteNotStrictlyV2 translates tcp route with a loose way, only generate ID and Name for delete Event.
func (t *translator) translateStreamRouteNotStrictlyV2(ctx *TranslateContext, ar *configv2.ApisixRoute) error {
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
		ctx.AddStreamRoute(sr)
		if !ctx.CheckUpstreamExist(ups.Name) {
			ctx.AddUpstream(ups)
		}
	}
	return nil
}
