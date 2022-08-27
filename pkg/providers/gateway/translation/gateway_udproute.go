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
	"github.com/apache/apisix-ingress-controller/pkg/kube/translation"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

func (t *translator) TranslateGatewayUDPRouteV1Alpha2(udpRoute *gatewayv1alpha2.UDPRoute) (*translation.TranslateContext, error) {
	ctx := translation.DefaultEmptyTranslateContext()

	// TODO: handle UDPRoute.Spec.ParentRef
	for i, rule := range udpRoute.Spec.Rules {

		for j, backend := range rule.BackendRefs {
			// Spec validation
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
				ns = udpRoute.Namespace
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

			// create apisix Upstream
			sr := apisixv1.NewDefaultStreamRoute()
			name := apisixv1.ComposeStreamRouteName(ns, udpRoute.Name, fmt.Sprintf("%d-%d", i, j))
			sr.ID = id.GenID(name)
			ups, err := t.KubeTranslator.TranslateUpstream(ns, string(backend.Name), "", int32(*backend.Port))
			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("failed to translate Rules[%v].BackendRefs[%v]", i, j))
			}
			ups.Scheme = apisixv1.SchemeUDP
			name = apisixv1.ComposeUpstreamName(ns, string(backend.Name), "", int32(*backend.Port))
			ups.ID = id.GenID(name)
			sr.UpstreamId = ups.ID
			ctx.AddStreamRoute(sr)
			if !ctx.CheckUpstreamExist(ups.Name) {
				ctx.AddUpstream(ups)
			}

			//if backend.Weight == nil {
			// TODO: set Upstream.Nodes roundrobin by BackendRef.Weight
			//}
		}
	}
	return ctx, nil
}
