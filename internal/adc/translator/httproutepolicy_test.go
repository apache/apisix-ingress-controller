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
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/provider"
)

const validPolicyVar = `["remote_addr","==","10.0.0.1"]`

func policyWithVars(vars ...string) v1alpha1.HTTPRoutePolicy {
	policy := v1alpha1.HTTPRoutePolicy{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "policy"},
		Spec: v1alpha1.HTTPRoutePolicySpec{
			TargetRefs: []gatewayv1.LocalPolicyTargetReferenceWithSectionName{{
				LocalPolicyTargetReference: gatewayv1.LocalPolicyTargetReference{
					Group: gatewayv1.GroupName,
					Kind:  "HTTPRoute",
					Name:  "route",
				},
			}},
		},
	}
	for _, v := range vars {
		policy.Spec.Vars = append(policy.Spec.Vars, apiextensionsv1.JSON{Raw: []byte(v)})
	}
	return policy
}

func TestParseHTTPRoutePolicyVars(t *testing.T) {
	policy := policyWithVars(validPolicyVar)
	vars, err := ParseHTTPRoutePolicyVars(&policy)
	require.NoError(t, err)
	assert.Equal(t, adctypes.Vars{{{StrVal: "remote_addr"}, {StrVal: "=="}, {StrVal: "10.0.0.1"}}}, vars)

	for _, malformed := range []string{`{"remote_addr":"10.0.0.0/8"}`, `"remote_addr"`, `[{"a":"b"}]`, `[1]`} {
		t.Run(malformed, func(t *testing.T) {
			policy := policyWithVars(validPolicyVar, malformed)
			_, err := ParseHTTPRoutePolicyVars(&policy)
			assert.ErrorContains(t, err, "invalid spec.vars[1]")
		})
	}
}

func TestTranslateHTTPRoute_HTTPRoutePolicyVars(t *testing.T) {
	pathType := gatewayv1.PathMatchPathPrefix
	pathValue := "/"
	httpRoute := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "route"},
		Spec: gatewayv1.HTTPRouteSpec{
			Rules: []gatewayv1.HTTPRouteRule{{
				Matches: []gatewayv1.HTTPRouteMatch{{
					Path: &gatewayv1.HTTPPathMatch{Type: &pathType, Value: &pathValue},
				}},
			}},
		},
	}

	t.Run("valid vars are applied", func(t *testing.T) {
		tctx := provider.NewDefaultTranslateContext(context.Background())
		tctx.HTTPRoutePolicies = []v1alpha1.HTTPRoutePolicy{policyWithVars(validPolicyVar)}

		result, err := NewTranslator(logr.Discard(), "").TranslateHTTPRoute(tctx, httpRoute)
		require.NoError(t, err)
		require.Len(t, result.Services, 1)
		require.Len(t, result.Services[0].Routes, 1)
		assert.Contains(t, result.Services[0].Routes[0].Vars,
			[]adctypes.StringOrSlice{{StrVal: "remote_addr"}, {StrVal: "=="}, {StrVal: "10.0.0.1"}})
	})

	t.Run("malformed var fails translation", func(t *testing.T) {
		tctx := provider.NewDefaultTranslateContext(context.Background())
		tctx.HTTPRoutePolicies = []v1alpha1.HTTPRoutePolicy{
			policyWithVars(validPolicyVar, `{"remote_addr":"10.0.0.0/8"}`),
		}

		_, err := NewTranslator(logr.Discard(), "").TranslateHTTPRoute(tctx, httpRoute)
		assert.ErrorContains(t, err, "invalid spec.vars[1]")
	})
}

func TestTranslateIngress_HTTPRoutePolicyMalformedVars(t *testing.T) {
	pathType := networkingv1.PathTypePrefix
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "ingress"},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{{
				IngressRuleValue: networkingv1.IngressRuleValue{HTTP: &networkingv1.HTTPIngressRuleValue{
					Paths: []networkingv1.HTTPIngressPath{{
						Path:     "/",
						PathType: &pathType,
						Backend: networkingv1.IngressBackend{
							Service: &networkingv1.IngressServiceBackend{
								Name: "svc",
								Port: networkingv1.ServiceBackendPort{Number: 80},
							},
						},
					}},
				}},
			}},
		},
	}
	tctx := &provider.TranslateContext{
		Services: map[types.NamespacedName]*corev1.Service{
			{Namespace: "default", Name: "svc"}: {
				ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "svc"},
				Spec:       corev1.ServiceSpec{Ports: []corev1.ServicePort{{Port: 80}}},
			},
		},
		HTTPRoutePolicies: []v1alpha1.HTTPRoutePolicy{
			policyWithVars(validPolicyVar, `{"remote_addr":"10.0.0.0/8"}`),
		},
	}

	_, err := NewTranslator(logr.Discard(), "").TranslateIngress(tctx, ingress)
	assert.ErrorContains(t, err, "invalid spec.vars[1]")
}
