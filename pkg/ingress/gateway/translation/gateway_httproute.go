// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
//
package gateway_translation

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/apache/apisix-ingress-controller/pkg/id"
	"github.com/apache/apisix-ingress-controller/pkg/ingress/utils"
	"github.com/apache/apisix-ingress-controller/pkg/kube/translation"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

func (t *translator) TranslateGatewayHTTPRouteV1Alpha2(httpRoute *gatewayv1alpha2.HTTPRoute) (*translation.TranslateContext, error) {
	ctx := translation.DefaultEmptyTranslateContext()

	var hosts []string
	for _, hostname := range httpRoute.Spec.Hostnames {
		hosts = append(hosts, string(hostname))
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
			// TODO: check gatewayv1alpha2.ReferencePolicy
			//}

			if backend.Port == nil {
				log.Warnw(fmt.Sprintf("ignore nil port at Rules[%v].BackendRefs[%v]", i, j),
					zap.String("kind", kind),
				)
				continue
			}

			ups, err := t.KubeTranslator.TranslateUpstream(ns, string(backend.Name), "", int32(*backend.Port))
			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("failed to translate Rules[%v].BackendRefs[%v]", i, j))
			}
			name := apisixv1.ComposeUpstreamName(ns, string(backend.Name), "", int32(*backend.Port))

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
			defaultType := gatewayv1alpha2.PathMatchPathPrefix
			defaultValue := "/"
			matches = []gatewayv1alpha2.HTTPRouteMatch{
				{
					Path: &gatewayv1alpha2.HTTPPathMatch{
						Type:  &defaultType,
						Value: &defaultValue,
					},
				},
			}
		}

		for j, match := range matches {
			route, err := t.translateGatewayHTTPRouteMatch(&match)
			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("failed to translate Rules[%v].Matches[%v]", i, j))
			}

			name := apisixv1.ComposeRouteName(httpRoute.Namespace, httpRoute.Name, fmt.Sprintf("%d-%d", i, j))
			route.ID = id.GenID(name)
			route.Hosts = hosts

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

func (t *translator) translateGatewayHTTPRouteMatch(match *gatewayv1alpha2.HTTPRouteMatch) (*apisixv1.Route, error) {
	route := apisixv1.NewDefaultRoute()

	if match.Path != nil {
		switch *match.Path.Type {
		case gatewayv1alpha2.PathMatchExact:
			route.Uri = *match.Path.Value
		case gatewayv1alpha2.PathMatchPathPrefix:
			route.Uri = *match.Path.Value + "*"
		case gatewayv1alpha2.PathMatchRegularExpression:
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
			case gatewayv1alpha2.HeaderMatchExact:
				this = append(this, apisixv1.StringOrSlice{
					StrVal: "==",
				})
			case gatewayv1alpha2.HeaderMatchRegularExpression:
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
			case gatewayv1alpha2.QueryParamMatchExact:
				this = append(this, apisixv1.StringOrSlice{
					StrVal: "==",
				})
			case gatewayv1alpha2.QueryParamMatchRegularExpression:
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
