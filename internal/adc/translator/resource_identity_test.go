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
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/adc/cache"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	internaltypes "github.com/apache/apisix-ingress-controller/internal/types"
)

// Resources of different kinds that share a namespace and name must not share a
// generated service or route identity, or one replaces the other in the store.
func TestSameNameResourcesOfDifferentKindsKeepDistinctIdentities(t *testing.T) {
	const name = "foo"
	meta := func(kind string) (metav1.TypeMeta, metav1.ObjectMeta) {
		return metav1.TypeMeta{Kind: kind}, metav1.ObjectMeta{Namespace: extensionRefTestNamespace, Name: name}
	}

	tm, om := meta(internaltypes.KindApisixRoute)
	ar := &apiv2.ApisixRoute{TypeMeta: tm, ObjectMeta: om, Spec: apiv2.ApisixRouteSpec{HTTP: []apiv2.ApisixRouteHTTP{{
		// The rule name an HTTPRoute's first rule and match compose to.
		Name:  "0-0",
		Match: apiv2.ApisixRouteHTTPMatch{Paths: []string{"/ar"}},
	}}}}

	tm, om = meta(internaltypes.KindHTTPRoute)
	hr := &gatewayv1.HTTPRoute{TypeMeta: tm, ObjectMeta: om, Spec: gatewayv1.HTTPRouteSpec{Rules: []gatewayv1.HTTPRouteRule{{
		BackendRefs: []gatewayv1.HTTPBackendRef{{BackendRef: extensionRefTestBackendRef()}},
	}}}}

	tm, om = meta(internaltypes.KindGRPCRoute)
	gr := &gatewayv1.GRPCRoute{TypeMeta: tm, ObjectMeta: om, Spec: gatewayv1.GRPCRouteSpec{Rules: []gatewayv1.GRPCRouteRule{{
		BackendRefs: []gatewayv1.GRPCBackendRef{{BackendRef: extensionRefTestBackendRef()}},
	}}}}

	pathType := networkingv1.PathTypePrefix
	tm, om = meta(internaltypes.KindIngress)
	ing := &networkingv1.Ingress{TypeMeta: tm, ObjectMeta: om, Spec: networkingv1.IngressSpec{Rules: []networkingv1.IngressRule{{
		IngressRuleValue: networkingv1.IngressRuleValue{HTTP: &networkingv1.HTTPIngressRuleValue{
			Paths: []networkingv1.HTTPIngressPath{{
				Path:     "/ing",
				PathType: &pathType,
				Backend: networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{
					Name: extensionRefTestBackend,
					Port: networkingv1.ServiceBackendPort{Number: extensionRefTestPort},
				}},
			}},
		}},
	}}}}

	tr := NewTranslator(logr.Discard(), "")
	tctx := newExtensionRefTranslateContext()
	store := cache.NewStore(logr.Discard())
	serviceIDs := map[string]string{}
	routeIDs := map[string]string{}

	for _, obj := range []client.Object{ar, hr, gr, ing} {
		var (
			result *TranslateResult
			err    error
		)
		switch o := obj.(type) {
		case *apiv2.ApisixRoute:
			result, err = tr.TranslateApisixRoute(tctx, o)
		case *gatewayv1.HTTPRoute:
			result, err = tr.TranslateHTTPRoute(tctx, o)
		case *gatewayv1.GRPCRoute:
			result, err = tr.TranslateGRPCRoute(tctx, o)
		case *networkingv1.Ingress:
			result, err = tr.TranslateIngress(tctx, o)
		}
		kind := obj.GetObjectKind().GroupVersionKind().Kind
		require.NoError(t, err, kind)
		require.NotEmpty(t, result.Services, kind)

		for _, svc := range result.Services {
			if other, ok := serviceIDs[svc.ID]; ok {
				t.Errorf("%s service %q shares id %s with %s", kind, svc.Name, svc.ID, other)
			}
			serviceIDs[svc.ID] = kind
			for _, route := range svc.Routes {
				if other, ok := routeIDs[route.ID]; ok {
					t.Errorf("%s route %q shares id %s with %s", kind, route.Name, route.ID, other)
				}
				routeIDs[route.ID] = kind
			}
		}

		require.NoError(t, store.Insert("gateway-proxy", []string{adctypes.TypeService},
			&adctypes.Resources{Services: result.Services}, label.GenLabel(obj)))
	}

	resources, err := store.GetResources("gateway-proxy")
	require.NoError(t, err)
	kinds := map[string]bool{}
	for _, svc := range resources.Services {
		kinds[svc.Labels[label.LabelKind]] = true
	}
	assert.Equal(t, map[string]bool{
		internaltypes.KindApisixRoute: true,
		internaltypes.KindHTTPRoute:   true,
		internaltypes.KindGRPCRoute:   true,
		internaltypes.KindIngress:     true,
	}, kinds)
}
