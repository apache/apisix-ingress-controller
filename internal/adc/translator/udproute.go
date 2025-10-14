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

	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	"github.com/apache/apisix-ingress-controller/internal/id"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/internal/types"
)

func (t *Translator) TranslateUDPRoute(tctx *provider.TranslateContext, udpRoute *gatewayv1alpha2.UDPRoute) (*TranslateResult, error) {
	result := &TranslateResult{}
	rules := udpRoute.Spec.Rules
	labels := label.GenLabel(udpRoute)
	for ruleIndex, rule := range rules {
		service := adctypes.NewDefaultService()
		service.Labels = labels
		service.Name = adctypes.ComposeServiceNameWithStream(udpRoute.Namespace, udpRoute.Name, fmt.Sprintf("%d", ruleIndex), "UDP")
		service.ID = id.GenID(service.Name)
		var (
			upstreams         = make([]*adctypes.Upstream, 0)
			weightedUpstreams = make([]adctypes.TrafficSplitConfigRuleWeightedUpstream, 0)
		)
		for _, backend := range rule.BackendRefs {
			if backend.Namespace == nil {
				namespace := gatewayv1.Namespace(udpRoute.Namespace)
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
			t.AttachBackendTrafficPolicyToUpstream(backend, tctx.BackendTrafficPolicies, upstream)
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
				port = int32(*backend.Port)
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
		streamRoute := adctypes.NewDefaultStreamRoute()
		streamRouteName := adctypes.ComposeStreamRouteName(udpRoute.Namespace, udpRoute.Name, fmt.Sprintf("%d", ruleIndex), "UDP")
		streamRoute.Name = streamRouteName
		streamRoute.ID = id.GenID(streamRouteName)
		streamRoute.Labels = labels
		// TODO: support remote_addr, server_addr, sni, server_port
		service.StreamRoutes = append(service.StreamRoutes, streamRoute)
		result.Services = append(result.Services, service)
	}
	return result, nil
}
