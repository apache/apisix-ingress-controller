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

	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/apache/apisix-ingress-controller/pkg/id"
	"github.com/apache/apisix-ingress-controller/pkg/providers/translation"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

func (t *translator) TranslateGatewayTCPRouteV1Alpha2(tcpRoute *gatewayv1alpha2.TCPRoute) (*translation.TranslateContext, error) {
	ctx := translation.DefaultEmptyTranslateContext()
	var ns string

	for i, rule := range tcpRoute.Spec.Rules {
		for j, backend := range rule.BackendRefs {
			if backend.Namespace != nil {
				ns = string(*backend.Namespace)
			} else {
				ns = tcpRoute.Namespace
			}
			sr := apisixv1.NewDefaultStreamRoute()
			name := apisixv1.ComposeStreamRouteName(tcpRoute.Namespace, tcpRoute.Name, fmt.Sprintf("%d-%d", i, j))
			sr.ID = id.GenID(name)
			ups, err := t.KubeTranslator.TranslateService(ns, string(backend.Name), "", int32(*backend.Port))
			if err != nil {
				return nil, err
			}
			ups.Name = apisixv1.ComposeUpstreamName(ns, string(backend.Name), "", int32(*backend.Port), "")
			ups.ID = id.GenID(ups.Name)
			sr.UpstreamId = ups.ID
			ctx.AddStreamRoute(sr)
			if !ctx.CheckUpstreamExist(ups.Name) {
				ctx.AddUpstream(ups)
			}
		}
	}
	return ctx, nil
}
