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
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	internaltypes "github.com/apache/apisix-ingress-controller/internal/types"
)

func TestTranslateHTTPRouteUpstreamScheme(t *testing.T) {
	tests := []struct {
		name         string
		appProtocol  string
		policyScheme string
		wantScheme   string
	}{
		{
			name:         "preserves backend traffic policy scheme",
			appProtocol:  internaltypes.AppProtocolHTTP,
			policyScheme: apiv2.SchemeHTTPS,
			wantScheme:   apiv2.SchemeHTTPS,
		},
		{
			name:        "falls back to app protocol when scheme is unset",
			appProtocol: internaltypes.AppProtocolWSS,
			wantScheme:  apiv2.SchemeHTTPS,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			translator := NewTranslator(logr.Discard())
			tctx := provider.NewDefaultTranslateContext(context.Background())

			const (
				namespace   = "default"
				serviceName = "backend"
				portName    = "web"
				portNumber  = int32(8443)
			)

			serviceKey := types.NamespacedName{Namespace: namespace, Name: serviceName}
			tctx.Services[serviceKey] = &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      serviceName,
					Namespace: namespace,
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{{
						Name:        portName,
						Port:        portNumber,
						AppProtocol: ptr.To(tt.appProtocol),
					}},
				},
			}
			tctx.EndpointSlices[serviceKey] = []discoveryv1.EndpointSlice{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "backend-1",
					Namespace: namespace,
				},
				Ports: []discoveryv1.EndpointPort{{
					Name: ptr.To(portName),
					Port: ptr.To(portNumber),
				}},
				Endpoints: []discoveryv1.Endpoint{{
					Addresses: []string{"10.0.0.1"},
					Conditions: discoveryv1.EndpointConditions{
						Ready: ptr.To(true),
					},
				}},
			}}

			if tt.policyScheme != "" {
				tctx.BackendTrafficPolicies[serviceKey] = &v1alpha1.BackendTrafficPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "backend-policy",
						Namespace: namespace,
					},
					Spec: v1alpha1.BackendTrafficPolicySpec{
						TargetRefs: []v1alpha1.BackendPolicyTargetReferenceWithSectionName{{
							LocalPolicyTargetReference: gatewayv1alpha2.LocalPolicyTargetReference{
								Name: gatewayv1alpha2.ObjectName(serviceName),
								Kind: gatewayv1alpha2.Kind(internaltypes.KindService),
							},
						}},
						Scheme: tt.policyScheme,
					},
				}
			}

			route := &gatewayv1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "demo",
					Namespace: namespace,
				},
				Spec: gatewayv1.HTTPRouteSpec{
					Rules: []gatewayv1.HTTPRouteRule{{
						BackendRefs: []gatewayv1.HTTPBackendRef{{
							BackendRef: gatewayv1.BackendRef{
								BackendObjectReference: gatewayv1.BackendObjectReference{
									Name: gatewayv1.ObjectName(serviceName),
									Port: ptr.To(gatewayv1.PortNumber(portNumber)),
								},
							},
						}},
					}},
				},
			}

			result, err := translator.TranslateHTTPRoute(tctx, route)
			require.NoError(t, err)
			require.Len(t, result.Services, 1)
			require.NotNil(t, result.Services[0].Upstream)

			assert.Equal(t, tt.wantScheme, result.Services[0].Upstream.Scheme)
			assert.Equal(t, "10.0.0.1", result.Services[0].Upstream.Nodes[0].Host)
		})
	}
}

func TestAttachBackendTrafficPolicyHealthCheck(t *testing.T) {
	trueVal := true
	falseVal := false

	tests := []struct {
		name       string
		policy     *v1alpha1.BackendTrafficPolicy
		wantChecks *adctypes.UpstreamHealthCheck
	}{
		{
			name:       "nil health check produces no checks",
			policy:     &v1alpha1.BackendTrafficPolicy{},
			wantChecks: nil,
		},
		{
			name: "active health check with all fields",
			policy: &v1alpha1.BackendTrafficPolicy{
				Spec: v1alpha1.BackendTrafficPolicySpec{
					HealthCheck: &v1alpha1.HealthCheck{
						Active: &v1alpha1.ActiveHealthCheck{
							Type:           "http",
							Timeout:        metav1.Duration{Duration: 3 * time.Second},
							HTTPPath:       "/healthz",
							Concurrency:    10,
							Host:           "example.com",
							Port:           8080,
							StrictTLS:      &trueVal,
							RequestHeaders: []string{"X-Custom: value"},
							Healthy: &v1alpha1.ActiveHealthCheckHealthy{
								Interval: metav1.Duration{Duration: 5 * time.Second},
								PassiveHealthCheckHealthy: v1alpha1.PassiveHealthCheckHealthy{
									HTTPCodes: []int{200, 201},
									Successes: 3,
								},
							},
							Unhealthy: &v1alpha1.ActiveHealthCheckUnhealthy{
								Interval: metav1.Duration{Duration: 2 * time.Second},
								PassiveHealthCheckUnhealthy: v1alpha1.PassiveHealthCheckUnhealthy{
									HTTPCodes:    []int{500, 503},
									HTTPFailures: 5,
									TCPFailures:  2,
									Timeouts:     3,
								},
							},
						},
					},
				},
			},
			wantChecks: &adctypes.UpstreamHealthCheck{
				Active: &adctypes.UpstreamActiveHealthCheck{
					Type:                   "http",
					Timeout:                3,
					HTTPPath:               "/healthz",
					Concurrency:            10,
					Host:                   "example.com",
					Port:                   8080,
					HTTPSVerifyCertificate: true,
					HTTPRequestHeaders:     []string{"X-Custom: value"},
					Healthy: adctypes.UpstreamActiveHealthCheckHealthy{
						Interval: 5,
						UpstreamPassiveHealthCheckHealthy: adctypes.UpstreamPassiveHealthCheckHealthy{
							HTTPStatuses: []int{200, 201},
							Successes:    3,
						},
					},
					Unhealthy: adctypes.UpstreamActiveHealthCheckUnhealthy{
						Interval: 2,
						UpstreamPassiveHealthCheckUnhealthy: adctypes.UpstreamPassiveHealthCheckUnhealthy{
							HTTPStatuses: []int{500, 503},
							HTTPFailures: 5,
							TCPFailures:  2,
							Timeouts:     3,
						},
					},
				},
			},
		},
		{
			name: "strictTLS false disables certificate verification",
			policy: &v1alpha1.BackendTrafficPolicy{
				Spec: v1alpha1.BackendTrafficPolicySpec{
					HealthCheck: &v1alpha1.HealthCheck{
						Active: &v1alpha1.ActiveHealthCheck{
							StrictTLS: &falseVal,
							Healthy: &v1alpha1.ActiveHealthCheckHealthy{
								Interval: metav1.Duration{Duration: 1 * time.Second},
							},
						},
					},
				},
			},
			wantChecks: &adctypes.UpstreamHealthCheck{
				Active: &adctypes.UpstreamActiveHealthCheck{
					Type:                   "http",
					HTTPSVerifyCertificate: false,
					Healthy: adctypes.UpstreamActiveHealthCheckHealthy{
						Interval: 1,
					},
				},
			},
		},
		{
			name: "active and passive health checks together",
			policy: &v1alpha1.BackendTrafficPolicy{
				Spec: v1alpha1.BackendTrafficPolicySpec{
					HealthCheck: &v1alpha1.HealthCheck{
						Active: &v1alpha1.ActiveHealthCheck{
							Type: "tcp",
							Healthy: &v1alpha1.ActiveHealthCheckHealthy{
								Interval: metav1.Duration{Duration: 1 * time.Second},
							},
						},
						Passive: &v1alpha1.PassiveHealthCheck{
							Type: "http",
							Healthy: &v1alpha1.PassiveHealthCheckHealthy{
								HTTPCodes: []int{200},
								Successes: 2,
							},
							Unhealthy: &v1alpha1.PassiveHealthCheckUnhealthy{
								HTTPCodes:    []int{500},
								HTTPFailures: 3,
							},
						},
					},
				},
			},
			wantChecks: &adctypes.UpstreamHealthCheck{
				Active: &adctypes.UpstreamActiveHealthCheck{
					Type:                   "tcp",
					HTTPSVerifyCertificate: true,
					Healthy: adctypes.UpstreamActiveHealthCheckHealthy{
						Interval: 1,
					},
				},
				Passive: &adctypes.UpstreamPassiveHealthCheck{
					Type: "http",
					Healthy: adctypes.UpstreamPassiveHealthCheckHealthy{
						HTTPStatuses: []int{200},
						Successes:    2,
					},
					Unhealthy: adctypes.UpstreamPassiveHealthCheckUnhealthy{
						HTTPStatuses: []int{500},
						HTTPFailures: 3,
					},
				},
			},
		},
	}

	translator := &Translator{Log: logr.Discard()}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ups := adctypes.NewDefaultUpstream()
			translator.attachBackendTrafficPolicyToUpstream(tt.policy, ups)
			assert.Equal(t, tt.wantChecks, ups.Checks)
		})
	}
}
