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

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/adc/translator/annotations"
	"github.com/apache/apisix-ingress-controller/internal/provider"
)

func TestTranslateIngress_WithCORS(t *testing.T) {
	translator := &Translator{}

	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ingress",
			Namespace: "default",
			Annotations: map[string]string{
				annotations.AnnotationsEnableCors:       "true",
				annotations.AnnotationsCorsAllowOrigin:  "https://example.com",
				annotations.AnnotationsCorsAllowMethods: "GET,POST,PUT",
				annotations.AnnotationsCorsAllowHeaders: "Content-Type,Authorization",
			},
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: "test.example.com",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/api",
									PathType: func() *networkingv1.PathType { pt := networkingv1.PathTypePrefix; return &pt }(),
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "test-service",
											Port: networkingv1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Port: 80,
					Name: "http",
				},
			},
		},
	}

	endpointSlice := discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service-abc",
			Namespace: "default",
			Labels: map[string]string{
				discoveryv1.LabelServiceName: "test-service",
			},
		},
		Ports: []discoveryv1.EndpointPort{
			{
				Name: func() *string { s := "http"; return &s }(),
				Port: func() *int32 { p := int32(80); return &p }(),
			},
		},
		Endpoints: []discoveryv1.Endpoint{
			{
				Addresses: []string{"10.0.0.1"},
				Conditions: discoveryv1.EndpointConditions{
					Ready: func() *bool { b := true; return &b }(),
				},
			},
		},
	}

	tctx := &provider.TranslateContext{
		Services: map[types.NamespacedName]*corev1.Service{
			{Namespace: "default", Name: "test-service"}: svc,
		},
		EndpointSlices: map[types.NamespacedName][]discoveryv1.EndpointSlice{
			{Namespace: "default", Name: "test-service"}: {endpointSlice},
		},
	}

	result, err := translator.TranslateIngress(tctx, ingress)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Services, 1)

	service := result.Services[0]
	assert.NotNil(t, service)
	assert.NotNil(t, service.Plugins)

	// Verify CORS plugin is configured
	corsPlugin, exists := service.Plugins[adctypes.PluginCORS]
	assert.True(t, exists, "CORS plugin should be present")
	assert.NotNil(t, corsPlugin)

	corsConfig, ok := corsPlugin.(*adctypes.CorsConfig)
	assert.True(t, ok, "CORS plugin should be of type *adctypes.CorsConfig")
	assert.Equal(t, "https://example.com", corsConfig.AllowOrigins)
	assert.Equal(t, "GET,POST,PUT", corsConfig.AllowMethods)
	assert.Equal(t, "Content-Type,Authorization", corsConfig.AllowHeaders)
}

func TestTranslateIngress_WithoutCORS(t *testing.T) {
	translator := &Translator{}

	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ingress",
			Namespace: "default",
			// No CORS annotations
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: "test.example.com",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/api",
									PathType: func() *networkingv1.PathType { pt := networkingv1.PathTypePrefix; return &pt }(),
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: "test-service",
											Port: networkingv1.ServiceBackendPort{
												Number: 80,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Port: 80,
					Name: "http",
				},
			},
		},
	}

	tctx := &provider.TranslateContext{
		Services: map[types.NamespacedName]*corev1.Service{
			{Namespace: "default", Name: "test-service"}: svc,
		},
		EndpointSlices: map[types.NamespacedName][]discoveryv1.EndpointSlice{},
	}

	result, err := translator.TranslateIngress(tctx, ingress)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Services, 1)

	service := result.Services[0]
	assert.NotNil(t, service)
	assert.NotNil(t, service.Plugins)

	// Verify CORS plugin is NOT configured
	_, exists := service.Plugins[adctypes.PluginCORS]
	assert.False(t, exists, "CORS plugin should not be present")
}
