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
	"k8s.io/apimachinery/pkg/util/intstr"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/apache/apisix-ingress-controller/pkg/id"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/providers/translation"
	"github.com/apache/apisix-ingress-controller/pkg/providers/utils"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

func (t *translator) TranslateGatewayTLSRouteV1Alpha2(tlsRoute *gatewayv1alpha2.TLSRoute) (*translation.TranslateContext, error) {
	ctx := translation.DefaultEmptyTranslateContext()

	// TODO: Handle ParentRefs

	var hosts []string
	for _, hostname := range tlsRoute.Spec.Hostnames {
		// TODO: calculate intersection of listeners
		hosts = append(hosts, string(hostname))
	}

	rules := tlsRoute.Spec.Rules

	for i, rule := range rules {
		backends := rule.BackendRefs
		if len(backends) == 0 {
			continue
		}

		var ruleUpstreams []*apisixv1.Upstream

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
				ns = tlsRoute.Namespace
			} else {
				ns = string(*backend.Namespace)
			}
			//if ns != tlsRoute.Namespace {
			// TODO: check gatewayv1alpha2.ReferencePolicy
			//}

			if backend.Port == nil {
				log.Warnw(fmt.Sprintf("ignore nil port at Rules[%v].BackendRefs[%v]", i, j),
					zap.String("kind", kind),
				)
				continue
			}

			ups, err := t.KubeTranslator.TranslateUpstream(ns, string(backend.Name), "", "", intstr.FromInt(int(*backend.Port)))
			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("failed to translate Rules[%v].BackendRefs[%v]", i, j))
			}
			name := apisixv1.ComposeUpstreamName(ns, string(backend.Name), "", int32(*backend.Port))

			ups.Labels["meta_namespace"] = utils.TruncateString(ns, 64)
			ups.Labels["meta_backend"] = utils.TruncateString(string(backend.Name), 64)
			ups.Labels["meta_port"] = fmt.Sprintf("%v", int32(*backend.Port))

			ups.ID = id.GenID(name)
			ctx.AddUpstream(ups)
			ruleUpstreams = append(ruleUpstreams, ups)
		}
		if len(ruleUpstreams) == 0 {
			log.Warnw(fmt.Sprintf("ignore all-failed backend refs at Rules[%v]", i),
				zap.Any("BackendRefs", rule.BackendRefs),
			)
			continue
		}

		for _, host := range hosts {
			route := apisixv1.NewDefaultStreamRoute()
			name := apisixv1.ComposeRouteName(tlsRoute.Namespace, tlsRoute.Name, fmt.Sprintf("%d-%s", i, host))
			route.ID = id.GenID(name)

			route.Labels["meta_namespace"] = utils.TruncateString(tlsRoute.Namespace, 64)
			route.Labels["meta_tlsroute"] = utils.TruncateString(tlsRoute.Name, 64)

			route.SNI = host

			route.UpstreamId = ruleUpstreams[0].ID
			if len(ruleUpstreams) > 1 {
				log.Warnw("ignore backends which is not the first one",
					zap.String("namespace", tlsRoute.Namespace),
					zap.String("tlsroute", tlsRoute.Name),
				)
			}
			ctx.AddStreamRoute(route)
		}
	}

	return ctx, nil
}
