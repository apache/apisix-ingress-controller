// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
package translation

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/apache/apisix-ingress-controller/pkg/id"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/providers/translation"
	"github.com/apache/apisix-ingress-controller/pkg/providers/utils"
	"github.com/apache/apisix-ingress-controller/pkg/types"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

func (t *translator) generatePluginsFromHTTPRouteFilter(namespace string, filters []gatewayv1beta1.HTTPRouteFilter) apisixv1.Plugins {
	plugins := apisixv1.Plugins{}
	for _, filter := range filters {
		switch filter.Type {
		case gatewayv1beta1.HTTPRouteFilterRequestHeaderModifier:
			t.generatePluginFromHTTPRequestHeaderFilter(plugins, filter.RequestHeaderModifier)
		case gatewayv1beta1.HTTPRouteFilterRequestRedirect:
			t.generatePluginFromHTTPRequestRedirectFilter(plugins, filter.RequestRedirect)
		case gatewayv1beta1.HTTPRouteFilterRequestMirror:
			t.generatePluginFromHTTPRequestMirrorFilter(namespace, plugins, filter.RequestMirror)
		case gatewayv1beta1.HTTPRouteFilterURLRewrite:
			// TODO: It is not yet supported by v1beta1 CRDs.
		}
	}
	return plugins
}

func (t *translator) generatePluginFromHTTPRequestHeaderFilter(plugins apisixv1.Plugins, reqHeaderModifier *gatewayv1beta1.HTTPRequestHeaderFilter) {
	if reqHeaderModifier == nil {
		return
	}
	headers := map[string]any{}
	// TODO: The current apisix plugin does not conform to the specification.
	for _, header := range reqHeaderModifier.Add {
		headers[string(header.Name)] = header.Value
	}
	for _, header := range reqHeaderModifier.Set {
		headers[string(header.Name)] = header.Value
	}
	for _, header := range reqHeaderModifier.Remove {
		headers[header] = ""
	}

	plugins["proxy-rewrite"] = apisixv1.RewriteConfig{
		Headers: headers,
	}
}

func (t *translator) generatePluginFromHTTPRequestMirrorFilter(namespace string, plugins apisixv1.Plugins, reqMirror *gatewayv1beta1.HTTPRequestMirrorFilter) {
	if reqMirror == nil {
		return
	}

	var (
		port int    = 80
		ns   string = namespace
	)
	if reqMirror.BackendRef.Port != nil {
		port = int(*reqMirror.BackendRef.Port)
	}
	if reqMirror.BackendRef.Namespace != nil {
		ns = string(*reqMirror.BackendRef.Namespace)
	}
	// TODO 1: Need to support https.
	// TODO 2: https://github.com/apache/apisix/issues/8351 APISIX 3.0 support {service.namespace} and {service.namespace.svc}, but APISIX <= 2.15 version is not supported.
	host := fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", reqMirror.BackendRef.Name, ns, port)

	plugins["proxy-mirror"] = apisixv1.RequestMirror{
		Host: host,
	}
}

func (t *translator) generatePluginFromHTTPRequestRedirectFilter(plugins apisixv1.Plugins, reqRedirect *gatewayv1beta1.HTTPRequestRedirectFilter) {
	if reqRedirect == nil {
		return
	}

	var uri string

	code := 302
	if reqRedirect.StatusCode != nil {
		code = *reqRedirect.StatusCode
	}

	hostname := "$host"
	if reqRedirect.Hostname != nil {
		hostname = string(*reqRedirect.Hostname)
	}

	scheme := "$scheme"
	if reqRedirect.Scheme != nil {
		scheme = *reqRedirect.Scheme
	}

	if reqRedirect.Port != nil {
		uri = fmt.Sprintf("%s://%s:%d$request_uri", scheme, hostname, int(*reqRedirect.Port))
	} else {
		uri = fmt.Sprintf("%s://%s$request_uri", scheme, hostname)
	}

	plugins["redirect"] = apisixv1.RedirectConfig{
		RetCode: code,
		URI:     uri,
	}
}

func (t *translator) TranslateGatewayHTTPRouteV1beta1(httpRoute *gatewayv1beta1.HTTPRoute) (*translation.TranslateContext, error) {
	ctx := translation.DefaultEmptyTranslateContext()

	var hosts []string
	for _, hostname := range httpRoute.Spec.Hostnames {
		hosts = append(hosts, string(hostname))

		// TODO: See the document of gatewayv1beta1.Listener.Hostname
		_ = gatewayv1beta1.Listener{}.Hostname
		// For HTTPRoute and TLSRoute resources, there is an interaction with the
		// `spec.hostnames` array. When both listener and route specify hostnames,
		// there MUST be an intersection between the values for a Route to be
		// accepted. For more information, refer to the Route specific Hostnames
		// documentation.
	}

	rules := httpRoute.Spec.Rules

	for i, rule := range rules {
		backends := rule.BackendRefs
		if len(backends) == 0 {
			continue
		}

		var ruleUpstreams []*apisixv1.Upstream
		var weightedUpstreams []apisixv1.TrafficSplitConfigRuleWeightedUpstream

		for j, backend := range backends {
			//TODO: Support filters
			//filters := backend.Filters
			var kind string
			if backend.Kind == nil {
				kind = "service"
			} else {
				kind = strings.ToLower(string(*backend.Kind))
			}
			if kind != "service" {
				log.Warnw(fmt.Sprintf("ignore non-service kind at Rules[%v].BackendRefs[%v]", i, j),
					zap.String("kind", kind),
				)
				continue
			}

			var ns string
			if backend.Namespace == nil {
				ns = httpRoute.Namespace
			} else {
				ns = string(*backend.Namespace)
			}
			//if ns != httpRoute.Namespace {
			// TODO: check gatewayv1beta1.ReferencePolicy
			//}

			if backend.Port == nil {
				log.Warnw(fmt.Sprintf("ignore nil port at Rules[%v].BackendRefs[%v]", i, j),
					zap.String("kind", kind),
				)
				continue
			}

			ups, err := t.KubeTranslator.TranslateService(ns, string(backend.Name), "", int32(*backend.Port))
			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("failed to translate Rules[%v].BackendRefs[%v]", i, j))
			}
			name := apisixv1.ComposeUpstreamName(ns, string(backend.Name), "", int32(*backend.Port), types.ResolveGranularity.Endpoint)

			// APISIX limits max length of label value
			// https://github.com/apache/apisix/blob/5b95b85faea3094d5e466ee2d39a52f1f805abbb/apisix/schema_def.lua#L85
			ups.Labels["meta_namespace"] = utils.TruncateString(ns, 64)
			ups.Labels["meta_backend"] = utils.TruncateString(string(backend.Name), 64)
			ups.Labels["meta_port"] = fmt.Sprintf("%v", int32(*backend.Port))

			ups.ID = id.GenID(name)
			ctx.AddUpstream(ups)
			ruleUpstreams = append(ruleUpstreams, ups)

			if backend.Weight == nil {
				weightedUpstreams = append(weightedUpstreams, apisixv1.TrafficSplitConfigRuleWeightedUpstream{
					UpstreamID: ups.ID,
					Weight:     1, // 1 is default value of BackendRef
				})
			} else {
				weightedUpstreams = append(weightedUpstreams, apisixv1.TrafficSplitConfigRuleWeightedUpstream{
					UpstreamID: ups.ID,
					Weight:     int(*backend.Weight),
				})
			}
		}
		if len(ruleUpstreams) == 0 {
			log.Warnw(fmt.Sprintf("ignore all-failed backend refs at Rules[%v]", i),
				zap.Any("BackendRefs", rule.BackendRefs),
			)
			continue
		}

		matches := rule.Matches
		if len(matches) == 0 {
			defaultType := gatewayv1beta1.PathMatchPathPrefix
			defaultValue := "/"
			matches = []gatewayv1beta1.HTTPRouteMatch{
				{
					Path: &gatewayv1beta1.HTTPPathMatch{
						Type:  &defaultType,
						Value: &defaultValue,
					},
				},
			}
		}
		plugins := t.generatePluginsFromHTTPRouteFilter(httpRoute.Namespace, rule.Filters)

		for j, match := range matches {
			route, err := t.translateGatewayHTTPRouteMatch(&match)
			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("failed to translate Rules[%v].Matches[%v]", i, j))
			}

			name := apisixv1.ComposeRouteName(httpRoute.Namespace, httpRoute.Name, fmt.Sprintf("%d-%d", i, j))
			route.ID = id.GenID(name)
			route.Hosts = hosts
			route.Plugins = plugins

			// Bind Upstream
			if len(ruleUpstreams) == 1 {
				route.UpstreamId = ruleUpstreams[0].ID
			} else if len(ruleUpstreams) > 0 {
				route.Plugins["traffic-split"] = &apisixv1.TrafficSplitConfig{
					Rules: []apisixv1.TrafficSplitConfigRule{
						{
							WeightedUpstreams: weightedUpstreams,
						},
					},
				}
			}

			ctx.AddRoute(route)
		}

		//TODO: Support filters
		//filters := rule.Filters
	}

	return ctx, nil
}

func (t *translator) translateGatewayHTTPRouteMatch(match *gatewayv1beta1.HTTPRouteMatch) (*apisixv1.Route, error) {
	route := apisixv1.NewDefaultRoute()

	if match.Path != nil {
		switch *match.Path.Type {
		case gatewayv1beta1.PathMatchExact:
			route.Uri = *match.Path.Value
		case gatewayv1beta1.PathMatchPathPrefix:
			route.Uri = *match.Path.Value + "*"
		case gatewayv1beta1.PathMatchRegularExpression:
			var this []apisixv1.StringOrSlice
			this = append(this, apisixv1.StringOrSlice{
				StrVal: "uri",
			})
			this = append(this, apisixv1.StringOrSlice{
				StrVal: "~~",
			})
			this = append(this, apisixv1.StringOrSlice{
				StrVal: *match.Path.Value,
			})

			route.Vars = append(route.Vars, this)
		default:
			return nil, errors.New("unknown path match type " + string(*match.Path.Type))
		}
	}

	if match.Headers != nil && len(match.Headers) > 0 {
		for _, header := range match.Headers {
			name := strings.ToLower(string(header.Name))
			name = strings.ReplaceAll(name, "-", "_")

			var this []apisixv1.StringOrSlice
			this = append(this, apisixv1.StringOrSlice{
				StrVal: "http_" + name,
			})

			switch *header.Type {
			case gatewayv1beta1.HeaderMatchExact:
				this = append(this, apisixv1.StringOrSlice{
					StrVal: "==",
				})
			case gatewayv1beta1.HeaderMatchRegularExpression:
				this = append(this, apisixv1.StringOrSlice{
					StrVal: "~~",
				})
			default:
				return nil, errors.New("unknown header match type " + string(*header.Type))
			}

			this = append(this, apisixv1.StringOrSlice{
				StrVal: header.Value,
			})

			route.Vars = append(route.Vars, this)
		}
	}

	if match.QueryParams != nil && len(match.QueryParams) > 0 {
		for _, query := range match.QueryParams {
			var this []apisixv1.StringOrSlice
			this = append(this, apisixv1.StringOrSlice{
				StrVal: "arg_" + strings.ToLower(query.Name),
			})

			switch *query.Type {
			case gatewayv1beta1.QueryParamMatchExact:
				this = append(this, apisixv1.StringOrSlice{
					StrVal: "==",
				})
			case gatewayv1beta1.QueryParamMatchRegularExpression:
				this = append(this, apisixv1.StringOrSlice{
					StrVal: "~~",
				})
			default:
				return nil, errors.New("unknown query match type " + string(*query.Type))
			}

			this = append(this, apisixv1.StringOrSlice{
				StrVal: query.Value,
			})

			route.Vars = append(route.Vars, this)
		}
	}

	if match.Method != nil {
		route.Methods = []string{
			string(*match.Method),
		}
	}

	return route, nil
}
