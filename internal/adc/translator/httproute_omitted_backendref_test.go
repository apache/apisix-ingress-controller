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
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/internal/provider"
)

// A rule with omitted or empty backendRefs must explicitly respond with 500,
// per the Gateway API HTTPRouteNoBackendRefs conformance test.
func TestTranslateHTTPRouteOmittedBackendRefs(t *testing.T) {
	for _, tc := range []struct {
		name string
		rule gatewayv1.HTTPRouteRule
	}{
		{name: "omitted backendRefs", rule: gatewayv1.HTTPRouteRule{}},
		{name: "empty backendRefs", rule: gatewayv1.HTTPRouteRule{BackendRefs: []gatewayv1.HTTPBackendRef{}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			translator := NewTranslator(logr.Discard(), "")
			tctx := provider.NewDefaultTranslateContext(context.Background())

			route := &gatewayv1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{Name: "omitted", Namespace: "default"},
				Spec:       gatewayv1.HTTPRouteSpec{Rules: []gatewayv1.HTTPRouteRule{tc.rule}},
			}

			result, err := translator.TranslateHTTPRoute(tctx, route)
			require.NoError(t, err)
			require.Len(t, result.Services, 1)

			plugins := result.Services[0].Plugins
			require.NotNil(t, plugins, "a rule with no backendRefs must carry a fault-injection plugin")
			fi, ok := plugins["fault-injection"].(map[string]any)
			require.True(t, ok, "expected fault-injection plugin")
			abort, ok := fi["abort"].(map[string]any)
			require.True(t, ok, "expected fault-injection.abort")
			assert.Equal(t, 500, abort["http_status"])
		})
	}
}

// A rule with no backendRefs but a RequestRedirect filter answers via the
// filter, so it must NOT be turned into a fault-injection 500. Regression guard
// for HTTPRouteRedirectPort / RedirectScheme / RedirectHostAndStatus.
func TestTranslateHTTPRouteRedirectNoFaultInjection(t *testing.T) {
	translator := NewTranslator(logr.Discard(), "")
	tctx := provider.NewDefaultTranslateContext(context.Background())

	port := gatewayv1.PortNumber(8083)
	route := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{Name: "redirect", Namespace: "default"},
		Spec: gatewayv1.HTTPRouteSpec{
			Rules: []gatewayv1.HTTPRouteRule{{
				Filters: []gatewayv1.HTTPRouteFilter{{
					Type:            gatewayv1.HTTPRouteFilterRequestRedirect,
					RequestRedirect: &gatewayv1.HTTPRequestRedirectFilter{Port: &port},
				}},
			}},
		},
	}

	result, err := translator.TranslateHTTPRoute(tctx, route)
	require.NoError(t, err)
	require.Len(t, result.Services, 1)

	if plugins := result.Services[0].Plugins; plugins != nil {
		_, hasFI := plugins["fault-injection"]
		assert.False(t, hasFI, "redirect rule must not carry a fault-injection plugin")
	}
}
