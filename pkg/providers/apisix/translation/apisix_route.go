// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package translation

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/id"
	"github.com/apache/apisix-ingress-controller/pkg/kube"
	configv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	_const "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/const"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/providers/translation"
	"github.com/apache/apisix-ingress-controller/pkg/providers/utils"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

func (t *translator) TranslateRouteV2(ar *configv2.ApisixRoute) (*translation.TranslateContext, error) {
	ctx := translation.DefaultEmptyTranslateContext()

	if err := t.translateHTTPRouteV2(ctx, ar); err != nil {
		return nil, err
	}
	if err := t.translateStreamRouteV2(ctx, ar); err != nil {
		return nil, err
	}
	return ctx, nil
}

func (t *translator) GenerateRouteV2DeleteMark(ar *configv2.ApisixRoute) (*translation.TranslateContext, error) {
	ctx := translation.DefaultEmptyTranslateContext()

	if err := t.generateHTTPRouteV2DeleteMark(ctx, ar); err != nil {
		return nil, err
	}
	if err := t.generateStreamRouteDeleteMarkV2(ctx, ar); err != nil {
		return nil, err
	}
	return ctx, nil
}

func (t *translator) translateHTTPRouteV2(ctx *translation.TranslateContext, ar *configv2.ApisixRoute) error {
	ruleNameMap := make(map[string]struct{})
	for _, part := range ar.Spec.HTTP {
		if _, ok := ruleNameMap[part.Name]; ok {
			return errors.New("duplicated route rule name")
		}
		ruleNameMap[part.Name] = struct{}{}

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
				if plugin.SecretRef != "" {
					sec, err := t.SecretLister.Secrets(ar.Namespace).Get(plugin.SecretRef)
					if err != nil {
						log.Errorw("The config secretRef is invalid",
							zap.Any("plugin", plugin.Name),
							zap.String("secretRef", plugin.SecretRef))
						break
					}
					log.Debugw("Add new items, then override items with the same plugin key",
						zap.Any("plugin", plugin.Name),
						zap.String("secretRef", plugin.SecretRef))

					for key, value := range sec.Data {
						utils.InsertKeyInMap(key, string(value), plugin.Config)
					}
				}
				pluginMap[plugin.Name] = plugin.Config
			} else {
				pluginMap[plugin.Name] = make(map[string]interface{})
			}
		}

		// add Authentication plugins
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
			case "ldapAuth":
				pluginMap["ldap-auth"] = part.Authentication.LDAPAuth
			default:
				pluginMap["basic-auth"] = make(map[string]interface{})
			}
		}

		var (
			exprs [][]apisixv1.StringOrSlice
			err   error
		)
		if part.Match.NginxVars != nil {
			exprs, err = t.TranslateRouteMatchExprs(part.Match.NginxVars)
			if err != nil {
				log.Errorw("ApisixRoute with bad nginxVars",
					zap.Error(err),
					zap.Any("ApisixRoute", ar),
				)
				return err
			}
		}
		if err := translation.ValidateRemoteAddrs(part.Match.RemoteAddrs); err != nil {
			log.Errorw("ApisixRoute with invalid remote addrs",
				zap.Error(err),
				zap.Strings("remote_addrs", part.Match.RemoteAddrs),
				zap.Any("ApisixRoute", ar),
			)
			return err
		}

		route := apisixv1.NewDefaultRoute()
		route.Name = apisixv1.ComposeRouteName(ar.Namespace, ar.Name, part.Name)
		route.ID = id.GenID(route.Name)
		route.Priority = part.Priority
		route.RemoteAddrs = part.Match.RemoteAddrs
		route.Vars = exprs
		route.Hosts = part.Match.Hosts
		route.Uris = part.Match.Paths
		route.Methods = part.Match.Methods
		route.EnableWebsocket = part.Websocket
		route.Plugins = pluginMap
		route.Timeout = timeout
		route.FilterFunc = part.Match.FilterFunc

		if part.PluginConfigName != "" {
			route.PluginConfigId = id.GenID(apisixv1.ComposePluginConfigName(ar.Namespace, part.PluginConfigName))
		}

		for k, v := range ar.ObjectMeta.Labels {
			route.Metadata.Labels[k] = v
		}

		ctx.AddRoute(route)

		// --- translate "Backends" ---
		backends := part.Backends
		if len(backends) > 0 {
			// Use the first backend as the default backend in Route,
			// others will be configured in traffic-split plugin.
			backend := backends[0]
			backends = backends[1:]

			svcClusterIP, svcPort, err := t.GetServiceClusterIPAndPort(&backend, ar.Namespace)
			if err != nil {
				log.Errorw("failed to get service port in backend",
					zap.Any("backend", backend),
					zap.Any("apisix_route", ar),
					zap.Error(err),
				)
				return err
			}

			upstreamName := apisixv1.ComposeUpstreamName(ar.Namespace, backend.ServiceName, backend.Subset, svcPort, backend.ResolveGranularity)
			route.UpstreamId = id.GenID(upstreamName)

			if len(backends) > 0 {
				weight := translation.DefaultWeight
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
			if !ctx.CheckUpstreamExist(upstreamName) {
				ups, err := t.translateService(ar.Namespace, backend.ServiceName, backend.Subset, backend.ResolveGranularity, svcClusterIP, svcPort)
				if err != nil {
					return err
				}
				ctx.AddUpstream(ups)
			}
		}

		if len(part.Backends) == 0 && len(part.Upstreams) > 0 {
			// Only have Upstreams
			upName := apisixv1.ComposeExternalUpstreamName(ar.Namespace, part.Upstreams[0].Name)
			route.UpstreamId = id.GenID(upName)
		}
		// --- translate Upstreams ---
		var ups []*apisixv1.Upstream
		for i, au := range part.Upstreams {
			up, err := t.translateExternalApisixUpstream(ar.Namespace, au.Name)
			if err != nil {
				log.Errorw(fmt.Sprintf("failed to translate ApisixUpstream at Upstream[%v]", i),
					zap.Error(err),
					zap.String("apisix_upstream", ar.Namespace+"/"+au.Name),
				)
				continue
			}
			if au.Weight != nil {
				up.Labels["meta_weight"] = strconv.Itoa(*au.Weight)
			} else {
				up.Labels["meta_weight"] = strconv.Itoa(translation.DefaultWeight)
			}
			ups = append(ups, up)
		}

		if len(ups) == 0 {
			continue
		}

		var wups []apisixv1.TrafficSplitConfigRuleWeightedUpstream
		if len(part.Backends) == 0 {
			if len(ups) > 1 {
				for i, up := range ups {
					weight, err := strconv.Atoi(up.Labels["meta_weight"])
					if err != nil {
						// shouldn't happen
						log.Errorw(fmt.Sprintf("failed to parse translated upstream weight at %v", i),
							zap.Error(err),
							zap.String("meta_weight", up.Labels["meta_weight"]),
						)
						continue
					}
					if i == 0 {
						// set as default
						wups = append(wups, apisixv1.TrafficSplitConfigRuleWeightedUpstream{
							Weight: weight,
						})
					} else {
						wups = append(wups, apisixv1.TrafficSplitConfigRuleWeightedUpstream{
							UpstreamID: ups[i].ID,
							Weight:     weight,
						})
					}
				}
			}
		} else {
			// Mixed backends and upstreams
			if cfg, ok := route.Plugins["traffic-split"]; ok {
				if tsCfg, ok := cfg.(*apisixv1.TrafficSplitConfig); ok {
					wups = tsCfg.Rules[0].WeightedUpstreams
				}
			}
			if len(wups) == 0 {
				// append the default upstream in the route.
				weight := translation.DefaultWeight
				if part.Backends[0].Weight != nil {
					weight = *part.Backends[0].Weight
				}
				wups = append(wups, apisixv1.TrafficSplitConfigRuleWeightedUpstream{
					Weight: weight,
				})
			}
			for i, up := range ups {
				weight, err := strconv.Atoi(up.Labels["meta_weight"])
				if err != nil {
					// shouldn't happen
					log.Errorw(fmt.Sprintf("failed to parse translated upstream weight at %v", i),
						zap.Error(err),
						zap.String("meta_weight", up.Labels["meta_weight"]),
					)
					continue
				}
				wups = append(wups, apisixv1.TrafficSplitConfigRuleWeightedUpstream{
					UpstreamID: ups[i].ID,
					Weight:     weight,
				})
			}
		}
		if len(wups) > 0 {
			route.Plugins["traffic-split"] = &apisixv1.TrafficSplitConfig{
				Rules: []apisixv1.TrafficSplitConfigRule{
					{
						WeightedUpstreams: wups,
					},
				},
			}
		}

		for _, up := range ups {
			ctx.AddUpstream(up)
		}
	}
	return nil
}

func (t *translator) TranslateRouteMatchExprs(nginxVars []configv2.ApisixRouteHTTPMatchExpr) ([][]apisixv1.StringOrSlice, error) {
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
			subj = "arg_" + expr.Subject.Name
		case _const.ScopeHeader:
			name := strings.ToLower(expr.Subject.Name)
			name = strings.ReplaceAll(name, "-", "_")
			subj = "http_" + name
		case _const.ScopeCookie:
			subj = "cookie_" + expr.Subject.Name
		case _const.ScopePath:
			subj = "uri"
		case _const.ScopeVariable:
			subj = expr.Subject.Name
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
		case _const.OpGreaterThanEqual:
			op = ">="
		case _const.OpIn:
			op = "in"
		case _const.OpLessThan:
			op = "<"
		case _const.OpLessThanEqual:
			op = "<="
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

// generateHTTPRouteV2DeleteMark translates http route with a loose way, only generate ID and Name for delete Event.
func (t *translator) generateHTTPRouteV2DeleteMark(ctx *translation.TranslateContext, ar *configv2.ApisixRoute) error {
	for _, part := range ar.Spec.HTTP {
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
			case "ldapAuth":
				pluginMap["ldap-auth"] = part.Authentication.LDAPAuth
			default:
				pluginMap["basic-auth"] = make(map[string]interface{})
			}
		}

		route := apisixv1.NewDefaultRoute()
		route.Name = apisixv1.ComposeRouteName(ar.Namespace, ar.Name, part.Name)
		route.ID = id.GenID(route.Name)
		if part.PluginConfigName != "" {
			route.PluginConfigId = id.GenID(apisixv1.ComposePluginConfigName(ar.Namespace, part.PluginConfigName))
		}

		ctx.AddRoute(route)

		if len(part.Backends) > 0 {
			backends := part.Backends
			// Use the first backend as the default backend in Route,
			// others will be configured in traffic-split plugin.
			backend := backends[0]

			upstreamName := apisixv1.ComposeUpstreamName(ar.Namespace, backend.ServiceName, backend.Subset, backend.ServicePort.IntVal, backend.ResolveGranularity)
			if !ctx.CheckUpstreamExist(upstreamName) {
				ups, err := t.generateUpstreamDeleteMark(ar.Namespace, backend.ServiceName, backend.Subset, backend.ServicePort.IntVal, backend.ResolveGranularity)
				if err != nil {
					return err
				}
				ctx.AddUpstream(ups)
			}
		}
		if len(part.Upstreams) > 0 {
			upstreams := part.Upstreams
			for _, upstream := range upstreams {
				upstreamName := apisixv1.ComposeExternalUpstreamName(ar.Namespace, upstream.Name)
				if !ctx.CheckUpstreamExist(upstreamName) {
					ups := &apisixv1.Upstream{}
					ups.Name = upstreamName
					ups.ID = id.GenID(ups.Name)
					ctx.AddUpstream(ups)
				}
			}
		}
	}
	return nil
}

func (t *translator) translateStreamRouteV2(ctx *translation.TranslateContext, ar *configv2.ApisixRoute) error {
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

		// add stream route plugins
		pluginMap := make(apisixv1.Plugins)
		for _, plugin := range part.Plugins {
			if !plugin.Enable {
				continue
			}
			if plugin.Config != nil {
				if plugin.SecretRef != "" {
					sec, err := t.SecretLister.Secrets(ar.Namespace).Get(plugin.SecretRef)
					if err != nil {
						log.Errorw("The config secretRef is invalid",
							zap.Any("plugin", plugin.Name),
							zap.String("secretRef", plugin.SecretRef))
						break
					}
					log.Debugw("Add new items, then override items with the same plugin key",
						zap.Any("plugin", plugin.Name),
						zap.String("secretRef", plugin.SecretRef))
					for key, value := range sec.Data {
						utils.InsertKeyInMap(key, string(value), plugin.Config)
					}
				}
				pluginMap[plugin.Name] = plugin.Config
			} else {
				pluginMap[plugin.Name] = make(map[string]interface{})
			}
		}

		sr := apisixv1.NewDefaultStreamRoute()
		name := apisixv1.ComposeStreamRouteName(ar.Namespace, ar.Name, part.Name)
		sr.ID = id.GenID(name)
		sr.ServerPort = part.Match.IngressPort
		sr.SNI = part.Match.Host
		ups, err := t.translateService(ar.Namespace, backend.ServiceName, backend.Subset, backend.ResolveGranularity, svcClusterIP, svcPort)
		if err != nil {
			return err
		}
		sr.UpstreamId = ups.ID
		sr.Plugins = pluginMap
		ctx.AddStreamRoute(sr)
		if !ctx.CheckUpstreamExist(ups.Name) {
			ctx.AddUpstream(ups)
		}

	}
	return nil
}

// generateStreamRouteDeleteMarkV2 translates tcp route with a loose way, only generate ID and Name for delete Event.
func (t *translator) generateStreamRouteDeleteMarkV2(ctx *translation.TranslateContext, ar *configv2.ApisixRoute) error {
	for _, part := range ar.Spec.Stream {
		backend := &part.Backend
		sr := apisixv1.NewDefaultStreamRoute()
		name := apisixv1.ComposeStreamRouteName(ar.Namespace, ar.Name, part.Name)
		sr.ID = id.GenID(name)
		sr.ServerPort = part.Match.IngressPort
		sr.SNI = part.Match.Host
		ups, err := t.generateUpstreamDeleteMark(ar.Namespace, backend.ServiceName, backend.Subset, backend.ServicePort.IntVal, backend.ResolveGranularity)
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

func (t *translator) GetServiceClusterIPAndPort(backend *configv2.ApisixRouteHTTPBackend, ns string) (string, int32, error) {
	svc, err := t.ServiceLister.Services(ns).Get(backend.ServiceName)
	if err != nil {
		return "", 0, err
	}
	svcPort := int32(-1)
	if backend.ResolveGranularity == "service" && svc.Spec.ClusterIP == "" {
		log.Errorw("ApisixRoute refers to a headless service but want to use the service level resolve granularity",
			zap.Any("namespace", ns),
			zap.Any("service", svc),
		)
		return "", 0, errors.New("conflict headless service and backend resolve granularity")
	}
loop:
	for _, port := range svc.Spec.Ports {
		switch backend.ServicePort.Type {
		case intstr.Int:
			if backend.ServicePort.IntVal == port.Port {
				svcPort = port.Port
				break loop
			}
		case intstr.String:
			if backend.ServicePort.StrVal == port.Name {
				svcPort = port.Port
				break loop
			}
		}
	}
	if svcPort == -1 {
		log.Errorw("ApisixRoute refers to non-existent Service port",
			zap.String("namespace", ns),
			zap.String("port", backend.ServicePort.String()),
		)
		return "", 0, err
	}

	return svc.Spec.ClusterIP, svcPort, nil
}

// getStreamServiceClusterIPAndPortV2 is for v2 streamRoute
func (t *translator) getStreamServiceClusterIPAndPortV2(backend configv2.ApisixRouteStreamBackend, ns string) (string, int32, error) {
	svc, err := t.ServiceLister.Services(ns).Get(backend.ServiceName)
	if err != nil {
		return "", 0, err
	}
	svcPort := int32(-1)
	if backend.ResolveGranularity == "service" && svc.Spec.ClusterIP == "" {
		log.Errorw("ApisixRoute refers to a headless service but want to use the service level resolve granularity",
			zap.String("ApisixRoute namespace", ns),
			zap.Any("service", svc),
		)
		return "", 0, errors.New("conflict headless service and backend resolve granularity")
	}
loop:
	for _, port := range svc.Spec.Ports {
		switch backend.ServicePort.Type {
		case intstr.Int:
			if backend.ServicePort.IntVal == port.Port {
				svcPort = port.Port
				break loop
			}
		case intstr.String:
			if backend.ServicePort.StrVal == port.Name {
				svcPort = port.Port
				break loop
			}
		}
	}
	if svcPort == -1 {
		log.Errorw("ApisixRoute refers to non-existent Service port",
			zap.String("ApisixRoute namespace", ns),
			zap.String("port", backend.ServicePort.String()),
		)
		return "", 0, err
	}

	return svc.Spec.ClusterIP, svcPort, nil
}

func (t *translator) TranslateOldRoute(ar kube.ApisixRoute) (*translation.TranslateContext, error) {
	switch ar.GroupVersion() {
	case config.ApisixV2:
		return t.translateOldRouteV2(ar.V2())
	default:
		return nil, fmt.Errorf("translator: source group version not supported: %s", ar.GroupVersion())
	}
}

func (t *translator) translateOldRouteV2(ar *configv2.ApisixRoute) (*translation.TranslateContext, error) {
	oldCtx := translation.DefaultEmptyTranslateContext()

	for _, part := range ar.Spec.Stream {
		name := apisixv1.ComposeStreamRouteName(ar.Namespace, ar.Name, part.Name)
		sr, err := t.Apisix.Cluster(t.ClusterName).StreamRoute().Get(context.Background(), name)
		if err != nil || sr == nil {
			continue
		}
		if sr.UpstreamId != "" {
			ups := apisixv1.NewDefaultUpstream()
			ups.ID = sr.UpstreamId
			oldCtx.AddUpstream(ups)
		}
		oldCtx.AddStreamRoute(sr)
	}
	for _, part := range ar.Spec.HTTP {
		name := apisixv1.ComposeRouteName(ar.Namespace, ar.Name, part.Name)
		r, err := t.Apisix.Cluster(t.ClusterName).Route().Get(context.Background(), name)
		if err != nil || r == nil {
			continue
		}
		if r.UpstreamId != "" {
			ups := apisixv1.NewDefaultUpstream()
			ups.ID = r.UpstreamId
			oldCtx.AddUpstream(ups)
		}
		oldCtx.AddRoute(r)
	}
	return oldCtx, nil
}
