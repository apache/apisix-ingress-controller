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
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/apache/apisix-ingress-controller/internal/provider"
)

func TestTranslateIngress_ImplementationSpecificPathWithoutAnnotations(t *testing.T) {
	translator := NewTranslator(logr.Discard(), "")
	pathType := networkingv1.PathTypeImplementationSpecific

	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test-ingress",
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{{
				Host: "example.com",
				IngressRuleValue: networkingv1.IngressRuleValue{HTTP: &networkingv1.HTTPIngressRuleValue{
					Paths: []networkingv1.HTTPIngressPath{{
						Path:     "/api/(.*)",
						PathType: &pathType,
						Backend: networkingv1.IngressBackend{
							Service: &networkingv1.IngressServiceBackend{
								Name: "test-svc",
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
			{Namespace: "default", Name: "test-svc"}: {
				ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "test-svc"},
				Spec:       corev1.ServiceSpec{Ports: []corev1.ServicePort{{Port: 80}}},
			},
		},
	}

	result, err := translator.TranslateIngress(tctx, ingress)
	require.NoError(t, err)
	require.Len(t, result.Services, 1)

	route := result.Services[0].Routes[0]
	assert.Equal(t, []string{"/api/(.*)"}, route.Uris)
	assert.Empty(t, route.Vars)
}

func TestTranslateIngress_PrefixPathURIs(t *testing.T) {
	// "/foo" and "/foo/" both match requests to "/foo" and "/foo/"
	// (networking.k8s.io/v1 PathTypePrefix), so the trailing slash of a
	// Prefix path must not stop it from matching the path without it.
	tests := []struct {
		name string
		path string
		uris []string
	}{
		{name: "root", path: "/", uris: []string{"/", "/*"}},
		{name: "no trailing slash", path: "/foo", uris: []string{"/foo", "/foo/*"}},
		{name: "trailing slash", path: "/foo/", uris: []string{"/foo/", "/foo", "/foo/*"}},
		{name: "nested trailing slash", path: "/foo/bar/", uris: []string{"/foo/bar/", "/foo/bar", "/foo/bar/*"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			translator := NewTranslator(logr.Discard(), "")
			pathType := networkingv1.PathTypePrefix

			ingress := &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-ingress",
				},
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{{
						Host: "example.com",
						IngressRuleValue: networkingv1.IngressRuleValue{HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{{
								Path:     tt.path,
								PathType: &pathType,
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: "test-svc",
										Port: networkingv1.ServiceBackendPort{Number: 80},
									},
								},
							}},
						}},
					}},
				},
			}

			result, err := translator.TranslateIngress(&provider.TranslateContext{}, ingress)
			require.NoError(t, err)
			require.Len(t, result.Services, 1)
			require.Len(t, result.Services[0].Routes, 1)
			assert.Equal(t, tt.uris, result.Services[0].Routes[0].Uris)
		})
	}
}
