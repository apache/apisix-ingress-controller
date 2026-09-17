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
	"fmt"

	"k8s.io/utils/ptr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	"github.com/apache/apisix-ingress-controller/internal/id"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/internal/types"
)

func (t *Translator) TranslateTLSRoute(tctx *provider.TranslateContext, tlsRoute *gatewayv1.TLSRoute) (*TranslateResult, error) {
	result := &TranslateResult{}
	rules := tlsRoute.Spec.Rules
	labels := label.GenLabel(tlsRoute)
	snis := tlsRouteSNIs(tctx, tlsRoute)
	for ruleIndex, rule := range rules {
		service := adctypes.NewDefaultService()
		service.Labels = labels
		service.Name = adctypes.ComposeServiceNameWithStream(tlsRoute.Namespace, tlsRoute.Name, fmt.Sprintf("%d", ruleIndex), "TLS")
		service.ID = id.GenID(service.Name)
		var (
			upstreams         = make([]*adctypes.Upstream, 0)
			weightedUpstreams = make([]adctypes.TrafficSplitConfigRuleWeightedUpstream, 0)
		)
		for _, backend := range rule.BackendRefs {
			if backend.Namespace == nil {
				namespace := gatewayv1.Namespace(tlsRoute.Namespace)
				backend.Namespace = &namespace
			}
			upstream := newDefaultUpstreamWithoutScheme()
			upNodes, _, err := t.translateBackendRef(tctx, backend, DefaultEndpointFilter)
			if err != nil {
				continue
			}
			if len(upNodes) == 0 {
				continue
			}
			// TODO: Confirm BackendTrafficPolicy attachment with e2e test case.
			t.AttachBackendTrafficPolicyToUpstream(backend, tctx.BackendTrafficPolicies, upstream, tctx.Services)
			upstream.Nodes = upNodes
			var (
				kind string
				port int32
			)
			if backend.Kind == nil {
				kind = types.KindService
			} else {
				kind = string(*backend.Kind)
			}
			if backend.Port != nil {
				port = *backend.Port
			}
			namespace := string(*backend.Namespace)
			name := string(backend.Name)
			upstreamName := adctypes.ComposeUpstreamNameForBackendRef(kind, namespace, name, port)
			upstream.Name = upstreamName
			upstream.ID = id.GenID(upstreamName)
			upstreams = append(upstreams, upstream)
		}

		// Handle multiple backends with traffic-split plugin
		if len(upstreams) == 0 {
			// Create a default upstream if no valid backends
			upstream := adctypes.NewDefaultUpstream()
			service.Upstream = upstream
		} else if len(upstreams) == 1 {
			// Single backend - use directly as service upstream
			service.Upstream = upstreams[0]
			// remove the id and name of the service.upstream, adc schema does not need id and name for it
			service.Upstream.ID = ""
			service.Upstream.Name = ""
		} else {
			// Multiple backends - use traffic-split plugin
			service.Upstream = upstreams[0]
			// remove the id and name of the service.upstream, adc schema does not need id and name for it
			service.Upstream.ID = ""
			service.Upstream.Name = ""

			upstreams = upstreams[1:]

			if len(upstreams) > 0 {
				service.Upstreams = upstreams
			}

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

		for _, port := range t.l4StreamRoutePorts(tctx) {
			streamRoute := adctypes.NewDefaultStreamRoute()
			ruleKey := fmt.Sprintf("%d", ruleIndex)
			if port != 0 {
				// Include the port in the name key so multiple listeners produce
				// distinct StreamRoute names/IDs instead of colliding.
				ruleKey = fmt.Sprintf("%d-%d", ruleIndex, port)
				streamRoute.ServerPort = port
			}
			streamRouteName := adctypes.ComposeStreamRouteName(tlsRoute.Namespace, tlsRoute.Name, ruleKey, "TLS")
			streamRoute.Name = streamRouteName
			streamRoute.ID = id.GenID(streamRouteName)
			// A single SNI keeps using the singular form: it is what every
			// APISIX version understands, and snis only earns its place once
			// there is more than one to match.
			if len(snis) == 1 {
				streamRoute.SNI = snis[0]
			} else {
				streamRoute.SNIs = snis
			}
			if tlsPassthroughOnPort(tctx.Listeners, port) {
				streamRoute.TLSPassthrough = ptr.To(true)
			}
			streamRoute.Labels = labels
			// Attach L4RoutePolicy plugins at the stream_route level: the APISIX stream proxy
			// applies plugins from the stream_route, not from the service. With multiple
			// listener ports each stream_route carries its own copy of the plugins.
			streamRoute.Plugins = make(adctypes.Plugins)
			t.AttachL4RoutePolicyPlugins(tctx.L4RoutePolicies, tlsRoute.Namespace, tlsRoute.Name, "TLSRoute", streamRoute.Plugins, tctx.Secrets)
			service.StreamRoutes = append(service.StreamRoutes, streamRoute)
		}

		result.Services = append(result.Services, service)
	}
	return result, nil
}

// tlsRouteSNIs returns the SNIs the route's stream routes match on.
//
// A TLSRoute without hostnames matches everything its listeners accept, so it
// falls back to the matched listener hostnames and, when those carry none
// either, to the catch-all "*". Emitting nothing - which is what the per
// hostname loop used to do - left such a route attached but unserved.
func tlsRouteSNIs(tctx *provider.TranslateContext, tlsRoute *gatewayv1.TLSRoute) []string {
	if len(tlsRoute.Spec.Hostnames) > 0 {
		snis := make([]string, 0, len(tlsRoute.Spec.Hostnames))
		for _, hostname := range tlsRoute.Spec.Hostnames {
			snis = append(snis, string(hostname))
		}
		return snis
	}

	snis := make([]string, 0, len(tctx.Listeners))
	seen := make(map[string]struct{}, len(tctx.Listeners))
	for _, listener := range tctx.Listeners {
		if listener.Hostname == nil || *listener.Hostname == "" {
			continue
		}
		hostname := string(*listener.Hostname)
		if _, ok := seen[hostname]; ok {
			continue
		}
		seen[hostname] = struct{}{}
		snis = append(snis, hostname)
	}
	if len(snis) == 0 {
		return []string{"*"}
	}
	return snis
}

// tlsPassthroughOnPort reports whether the stream routes bound to port must
// forward the connection untouched instead of having the gateway terminate it.
// port 0 means the StreamRoute carries no server_port match, so every matched
// listener applies.
//
// Every matched TLS listener on the port has to agree. Within one Gateway a
// port carrying both modes is already reported ProtocolConflict and attaches
// no routes; across Gateways the combination is unrepresentable, since the
// physical stream listen has a single mode - so the terminating behaviour wins
// rather than a guess.
func tlsPassthroughOnPort(listeners []gatewayv1.Listener, port int32) bool {
	matched := false
	for _, listener := range listeners {
		if listener.Protocol != gatewayv1.TLSProtocolType {
			continue
		}
		if port != 0 && listener.Port != port {
			continue
		}
		if listener.TLS == nil || listener.TLS.Mode == nil || *listener.TLS.Mode != gatewayv1.TLSModePassthrough {
			return false
		}
		matched = true
	}
	return matched
}
