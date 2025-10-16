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

package translator

import (
	"cmp"
	"fmt"
	"strings"

	"k8s.io/utils/ptr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	"github.com/apache/apisix-ingress-controller/internal/id"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	internaltypes "github.com/apache/apisix-ingress-controller/internal/types"
)

func (t *Translator) fillPluginsFromGRPCRouteFilters(
	plugins adctypes.Plugins,
	namespace string,
	filters []gatewayv1.GRPCRouteFilter,
	tctx *provider.TranslateContext,
) {
	for _, filter := range filters {
		switch filter.Type {
		case gatewayv1.GRPCRouteFilterRequestHeaderModifier:
			t.fillPluginFromHTTPRequestHeaderFilter(plugins, filter.RequestHeaderModifier)
		case gatewayv1.GRPCRouteFilterRequestMirror:
			t.fillPluginFromHTTPRequestMirrorFilter(plugins, namespace, filter.RequestMirror, apiv2.SchemeGRPC)
		case gatewayv1.GRPCRouteFilterResponseHeaderModifier:
			t.fillPluginFromHTTPResponseHeaderFilter(plugins, filter.ResponseHeaderModifier)
		case gatewayv1.GRPCRouteFilterExtensionRef:
			t.fillPluginFromExtensionRef(plugins, namespace, filter.ExtensionRef, tctx)
		}
	}
}

func calculateGRPCRoutePriority(match *gatewayv1.GRPCRouteMatch, ruleIndex int, hosts []string) uint64 {
	const (
		// PreciseHostnameShiftBits assigns bit 31-38 for the length of hostname(max length=253).
		// which has 8 bits, so the max length of hostname is 2^8-1 = 255.
		PreciseHostnameShiftBits = 31

		// HostnameLengthShiftBits assigns bits 23-30 for the length of hostname(max length=253).
		// which has 8 bits, so the max length of hostname is 2^8-1 = 255.
		HostnameLengthShiftBits = 23

		// ServiceMatchShiftBits assigns bits 19-22 for the length of service name.
		ServiceMatchShiftBits = 19

		// MethodMatchShiftBits assigns bits 15-18 for the length of method name.
		MethodMatchShiftBits = 15

		// HeaderNumberShiftBits assign bits 10-14 to number of headers. (max number of headers = 16)
		HeaderNumberShiftBits = 10

		// RuleIndexShiftBits assigns bits 5-9 to rule index. (max number of rules = 16)
		RuleIndexShiftBits = 5
	)

	var (
		priority uint64 = 0
		// Handle hostname priority
		// 1. Non-wildcard hostname priority
		// 2. Hostname length priority
		maxNonWildcardLength = 0
		maxHostnameLength    = 0
	)

	for _, host := range hosts {
		isNonWildcard := !strings.Contains(host, "*")

		if isNonWildcard && len(host) > maxNonWildcardLength {
			maxNonWildcardLength = len(host)
		}

		if len(host) > maxHostnameLength {
			maxHostnameLength = len(host)
		}
	}

	// If there is a non-wildcard hostname, set the PreciseHostnameShiftBits bit
	if maxNonWildcardLength > 0 {
		priority |= (uint64(maxNonWildcardLength) << PreciseHostnameShiftBits)
	}

	if maxHostnameLength > 0 {
		priority |= (uint64(maxHostnameLength) << HostnameLengthShiftBits)
	}

	// Service and Method matching - this is the key difference from HTTPRoute
	serviceLength := 0
	methodLength := 0

	if match.Method != nil {
		// Service matching
		if match.Method.Service != nil {
			serviceLength = len(*match.Method.Service)
			priority |= (uint64(serviceLength) << ServiceMatchShiftBits)
		}

		// Method matching
		if match.Method.Method != nil {
			methodLength = len(*match.Method.Method)
			priority |= (uint64(methodLength) << MethodMatchShiftBits)
		}
	}

	// HeaderNumberShiftBits - GRPCRoute also supports header matching
	headerCount := 0
	if match.Headers != nil {
		headerCount = len(match.Headers)
	}
	priority |= (uint64(headerCount) << HeaderNumberShiftBits)

	// RuleIndexShiftBits - lower index has higher priority
	// We invert the index so that rule 0 gets highest priority (16), rule 1 gets 15, etc.
	index := 16 - ruleIndex
	if index < 0 {
		index = 0
	}
	if index > 16 {
		index = 16
	}
	priority |= (uint64(index) << RuleIndexShiftBits)

	return priority
}

func (t *Translator) TranslateGRPCRoute(tctx *provider.TranslateContext, grpcRoute *gatewayv1.GRPCRoute) (*TranslateResult, error) {
	result := &TranslateResult{}

	hosts := make([]string, 0, len(grpcRoute.Spec.Hostnames))
	for _, hostname := range grpcRoute.Spec.Hostnames {
		hosts = append(hosts, string(hostname))
	}

	for _, listener := range tctx.Listeners {
		if listener.Hostname != nil {
			hosts = append(hosts, string(*listener.Hostname))
		}
	}

	rules := grpcRoute.Spec.Rules

	labels := label.GenLabel(grpcRoute)

	for ruleIndex, rule := range rules {
		service := adctypes.NewDefaultService()
		service.Labels = labels

		service.Name = adctypes.ComposeGRPCServiceNameWithRule(grpcRoute.Namespace, grpcRoute.Name, fmt.Sprintf("%d", ruleIndex))
		service.ID = id.GenID(service.Name)
		service.Hosts = hosts

		var (
			upstreams         = make([]*adctypes.Upstream, 0)
			weightedUpstreams = make([]adctypes.TrafficSplitConfigRuleWeightedUpstream, 0)
			backendErr        error
		)

		for _, backend := range rule.BackendRefs {
			if backend.Namespace == nil {
				namespace := gatewayv1.Namespace(grpcRoute.Namespace)
				backend.Namespace = &namespace
			}
			upstream := adctypes.NewDefaultUpstream()
			upNodes, _, err := t.translateBackendRef(tctx, backend.BackendRef, DefaultEndpointFilter)
			if err != nil {
				backendErr = err
				continue
			}
			if len(upNodes) == 0 {
				continue
			}

			t.AttachBackendTrafficPolicyToUpstream(backend.BackendRef, tctx.BackendTrafficPolicies, upstream)
			upstream.Nodes = upNodes

			var (
				kind string
				port int32
			)
			if backend.Kind == nil {
				kind = internaltypes.KindService
			} else {
				kind = string(*backend.Kind)
			}
			if backend.Port != nil {
				port = int32(*backend.Port)
			}
			namespace := string(*backend.Namespace)
			name := string(backend.Name)
			upstreamName := adctypes.ComposeUpstreamNameForBackendRef(kind, namespace, name, port)
			upstream.Name = upstreamName
			upstream.ID = id.GenID(upstreamName)
			upstream.Scheme = cmp.Or(upstream.Scheme, apiv2.SchemeGRPC)
			upstreams = append(upstreams, upstream)
		}

		// Handle multiple backends with traffic-split plugin
		if len(upstreams) == 0 {
			// Create a default upstream if no valid backends
			upstream := adctypes.NewDefaultUpstream()
			upstream.Scheme = apiv2.SchemeGRPC
			service.Upstream = upstream
		} else {
			// Multiple backends - use traffic-split plugin
			service.Upstream = upstreams[0]
			// remove the id and name of the service.upstream, adc schema does not need id and name for it
			service.Upstream.ID = ""
			service.Upstream.Name = ""

			upstreams = upstreams[1:]

			if len(upstreams) > 0 {
				service.Upstreams = upstreams

				// Set weight in traffic-split for the default upstream
				weight := apiv2.DefaultWeight
				if rule.BackendRefs[0].Weight != nil {
					weight = int(*rule.BackendRefs[0].Weight)
				}
				weightedUpstreams = append(weightedUpstreams, adctypes.TrafficSplitConfigRuleWeightedUpstream{
					Weight: weight,
				})

				// Set other upstreams in traffic-split using upstream_id
				for i, upstream := range upstreams {
					weight := apiv2.DefaultWeight
					// get weight from the backend refs starting from the second backend
					if i+1 < len(rule.BackendRefs) && rule.BackendRefs[i+1].Weight != nil {
						weight = int(*rule.BackendRefs[i+1].Weight)
					}
					weightedUpstreams = append(weightedUpstreams, adctypes.TrafficSplitConfigRuleWeightedUpstream{
						UpstreamID: upstream.ID,
						Weight:     weight,
					})
				}

				if len(weightedUpstreams) > 0 {
					if service.Plugins == nil {
						service.Plugins = make(map[string]any)
					}
					service.Plugins["traffic-split"] = &adctypes.TrafficSplitConfig{
						Rules: []adctypes.TrafficSplitConfigRule{
							{
								WeightedUpstreams: weightedUpstreams,
							},
						},
					}
				}
			}
		}

		if backendErr != nil && (service.Upstream == nil || len(service.Upstream.Nodes) == 0) {
			if service.Plugins == nil {
				service.Plugins = make(map[string]any)
			}
			service.Plugins["fault-injection"] = map[string]any{
				"abort": map[string]any{
					"http_status": 500,
					"body":        "No existing backendRef provided",
				},
			}
		}

		t.fillPluginsFromGRPCRouteFilters(service.Plugins, grpcRoute.GetNamespace(), rule.Filters, tctx)

		matches := rule.Matches
		if len(matches) == 0 {
			matches = []gatewayv1.GRPCRouteMatch{{}}
		}

		routes := []*adctypes.Route{}
		for j, match := range matches {
			route, err := t.translateGatewayGRPCRouteMatch(&match)
			if err != nil {
				return nil, err
			}

			name := adctypes.ComposeRouteName(grpcRoute.Namespace, grpcRoute.Name, fmt.Sprintf("%d-%d", ruleIndex, j))
			route.Name = name
			route.ID = id.GenID(name)
			route.Labels = labels

			// Set the route priority
			priority := calculateGRPCRoutePriority(&match, ruleIndex, hosts)
			route.Priority = ptr.To(int64(priority))

			routes = append(routes, route)
		}
		service.Routes = routes

		result.Services = append(result.Services, service)
	}

	return result, nil
}

func (t *Translator) translateGatewayGRPCRouteMatch(match *gatewayv1.GRPCRouteMatch) (*adctypes.Route, error) {
	route := &adctypes.Route{}

	var (
		service string
		method  string
	)
	if match.Method != nil {
		service = ptr.Deref(match.Method.Service, "")
		method = ptr.Deref(match.Method.Method, "")
		matchType := ptr.Deref(match.Method.Type, gatewayv1.GRPCMethodMatchExact)
		if matchType == gatewayv1.GRPCMethodMatchExact &&
			service == "" && method == "" {
			return nil, fmt.Errorf("service and method cannot both be empty for exact match type")
		}
	}

	uri := t.translateGRPCURI(service, method)
	route.Uris = append(route.Uris, uri)

	if match.Headers != nil {
		for _, header := range match.Headers {
			this, err := t.translateGRPCRouteHeaderMatchToVars(header)
			if err != nil {
				return nil, err
			}
			route.Vars = append(route.Vars, this)
		}
	}
	return route, nil
}

func (t *Translator) translateGRPCURI(service, method string) string {
	var uri string
	if service == "" {
		uri = "/*"
	} else {
		uri = fmt.Sprintf("/%s", service)
	}
	if method != "" {
		uri = uri + fmt.Sprintf("/%s", method)
	} else if service != "" {
		uri = uri + "/*"
	}
	return uri
}

func (t *Translator) translateGRPCRouteHeaderMatchToVars(header gatewayv1.GRPCHeaderMatch) ([]adctypes.StringOrSlice, error) {
	var matchType string
	if header.Type != nil {
		matchType = string(*header.Type)
	}
	return HeaderMatchToVars(matchType, string(header.Name), header.Value)
}
