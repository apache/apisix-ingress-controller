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

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	"github.com/apache/apisix-ingress-controller/internal/id"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

func (t *Translator) TranslateTCPRoute(tctx *provider.TranslateContext, tcpRoute *gatewayv1alpha2.TCPRoute) (*TranslateResult, error) {
	fmt.Println("ADITI: Translating TCPRoute: ", tcpRoute.Name)
	result := &TranslateResult{}

	rules := tcpRoute.Spec.Rules

	labels := label.GenLabel(tcpRoute)

	for ruleIndex, rule := range rules {
		service := adctypes.NewDefaultService()
		service.Labels = labels
		service.Name = adctypes.ComposeServiceNameWithStream(tcpRoute.Namespace, tcpRoute.Name, fmt.Sprintf("%d", ruleIndex))
		service.ID = id.GenID(service.Name)
		var (
			upstreams = make([]*adctypes.Upstream, 0)
		)

		for _, backend := range rule.BackendRefs {
			if backend.Namespace == nil {
				namespace := gatewayv1.Namespace(tcpRoute.Namespace)
				backend.Namespace = &namespace
			}
			upstream := adctypes.NewDefaultUpstream()
			upNodes, err := t.translateBackendRef(tctx, backend, DefaultEndpointFilter)
			if err != nil {
				continue
			}
			if len(upNodes) == 0 {
				continue
			}

			t.AttachBackendTrafficPolicyToUpstream(backend, tctx.BackendTrafficPolicies, upstream)
			upstream.Nodes = upNodes

			var (
				kind string
				port int32
			)
			if backend.Kind == nil {
				kind = "Service"
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
			service.Upstreams = upstreams
		}
		streamRoute := adctypes.NewDefaultStreamRoute()
		streamRouteName := adctypes.ComposeStreamRouteName(tcpRoute.Namespace, tcpRoute.Name, fmt.Sprintf("%d", ruleIndex))
		streamRoute.Name = streamRouteName
		streamRoute.ID = id.GenID(streamRouteName)
		streamRoute.Labels = labels
		//TODO: support remote_addr, server_adrr, sni, server_port
		result.StreamRoutes = append(result.StreamRoutes, streamRoute)
		result.Services = append(result.Services, service)
		fmt.Println("Service: ", *service)
		fmt.Println("StreamRoute: ", *streamRoute)
	}
	return result, nil
}
