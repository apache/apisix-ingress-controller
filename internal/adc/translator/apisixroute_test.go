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
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	adc "github.com/apache/apisix-ingress-controller/api/adc"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/provider"
)

func TestBuildRoute_HostsNotSet(t *testing.T) {
	translator := NewTranslator(logr.Discard(), "")

	ar := &apiv2.ApisixRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-route",
			Namespace: "default",
		},
	}

	service := &adc.Service{}
	rule := apiv2.ApisixRouteHTTP{
		Name: "rule1",
		Match: apiv2.ApisixRouteHTTPMatch{
			Hosts: []string{"example.com", "foo.com"},
			Paths: []string{"/api/*"},
		},
	}

	var enableWebsocket *bool
	translator.buildRoute(ar, service, rule, nil, nil, nil, &enableWebsocket)

	assert.Len(t, service.Routes, 1)
	route := service.Routes[0]
	// route.Hosts should NOT be set — hosts belong on Service, not Route.
	// Setting hosts on Route causes false diffs in backends that don't
	// support route-level hosts, triggering unnecessary PUT requests.
	assert.Nil(t, route.Hosts, "route.Hosts should not be set; hosts should only be on Service")
	assert.Equal(t, []string{"/api/*"}, route.Uris)
}

func TestBuildService_HostsSet(t *testing.T) {
	translator := NewTranslator(logr.Discard(), "")

	ar := &apiv2.ApisixRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-route",
			Namespace: "default",
		},
	}

	rule := apiv2.ApisixRouteHTTP{
		Name: "rule1",
		Match: apiv2.ApisixRouteHTTPMatch{
			Hosts: []string{"example.com", "foo.com"},
			Paths: []string{"/api/*"},
		},
	}

	service := translator.buildService(ar, rule, 0)

	// service.Hosts SHOULD be set — this is the canonical location for hosts.
	assert.Equal(t, []string{"example.com", "foo.com"}, service.Hosts)
}

func TestBuildService_HostsLowercased(t *testing.T) {
	translator := NewTranslator(logr.Discard(), "")

	ar := &apiv2.ApisixRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-route",
			Namespace: "default",
		},
	}

	rule := apiv2.ApisixRouteHTTP{
		Name: "rule1",
		Match: apiv2.ApisixRouteHTTPMatch{
			Hosts: []string{"MixedCase.example.com", "*.UPPER.example.com", "mixedcase.example.com"},
			Paths: []string{"/api/*"},
		},
	}

	service := translator.buildService(ar, rule, 0)

	// APISIX matches service hosts against nginx's $host, which is always lowercase,
	// and does not normalize service-level hosts itself. Entries that collide once
	// lowercased collapse: the APISIX service schema requires unique hosts.
	assert.Equal(t, []string{"mixedcase.example.com", "*.upper.example.com"}, service.Hosts)

	rule.Match.Hosts = nil
	assert.Nil(t, translator.buildService(ar, rule, 0).Hosts)
}

func TestBuildRoute_MetadataLabelsDoNotOverwriteControllerLabels(t *testing.T) {
	translator := NewTranslator(logr.Discard(), "")

	ar := &apiv2.ApisixRoute{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ApisixRoute",
			APIVersion: apiv2.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-route",
			Namespace: "default",
			Labels: map[string]string{
				"team":          "payments",
				"k8s/name":      "user-value",
				"manager-by":    "user-manager",
				"k8s/namespace": "user-namespace",
			},
		},
	}

	service := &adc.Service{}
	rule := apiv2.ApisixRouteHTTP{
		Name: "rule1",
		Match: apiv2.ApisixRouteHTTPMatch{
			Paths: []string{"/api/*"},
		},
	}

	var enableWebsocket *bool
	translator.buildRoute(ar, service, rule, nil, nil, nil, &enableWebsocket)

	assert.Len(t, service.Routes, 1)
	route := service.Routes[0]
	assert.Equal(t, "payments", route.Labels["team"])
	assert.Equal(t, ar.Name, route.Labels["k8s/name"])
	assert.Equal(t, ar.Namespace, route.Labels["k8s/namespace"])
	assert.Equal(t, "apisix-ingress-controller", route.Labels["manager-by"])
}

func TestTranslateApisixRouteStreamUpstreamScheme(t *testing.T) {
	const (
		namespace   = "default"
		serviceName = "backend"
		portName    = "tcp"
		portNumber  = int32(6000)
	)

	tests := []struct {
		scheme   string
		protocol string
	}{
		{scheme: apiv2.SchemeTLS, protocol: "TCP"},
		{scheme: apiv2.SchemeTCP, protocol: "TCP"},
		{scheme: apiv2.SchemeUDP, protocol: "UDP"},
	}

	for _, tt := range tests {
		t.Run(tt.scheme, func(t *testing.T) {
			translator := NewTranslator(logr.Discard(), "")
			tctx := provider.NewDefaultTranslateContext(context.Background())

			serviceKey := k8stypes.NamespacedName{Namespace: namespace, Name: serviceName}
			tctx.Services[serviceKey] = &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      serviceName,
					Namespace: namespace,
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{{
						Name: portName,
						Port: portNumber,
					}},
				},
			}
			tctx.EndpointSlices[serviceKey] = []discoveryv1.EndpointSlice{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      serviceName + "-1",
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
			tctx.Upstreams[serviceKey] = &apiv2.ApisixUpstream{
				ObjectMeta: metav1.ObjectMeta{
					Name:      serviceName,
					Namespace: namespace,
				},
				Spec: apiv2.ApisixUpstreamSpec{
					ApisixUpstreamConfig: apiv2.ApisixUpstreamConfig{
						Scheme: tt.scheme,
					},
				},
			}

			ar := &apiv2.ApisixRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-route",
					Namespace: namespace,
				},
				Spec: apiv2.ApisixRouteSpec{
					Stream: []apiv2.ApisixRouteStream{{
						Name:     "rule1",
						Protocol: tt.protocol,
						Match: apiv2.ApisixRouteStreamMatch{
							IngressPort: 8000,
						},
						Backend: apiv2.ApisixRouteStreamBackend{
							ServiceName: serviceName,
							ServicePort: intstr.FromInt32(portNumber),
						},
					}},
				},
			}

			result, err := translator.TranslateApisixRoute(tctx, ar)
			require.NoError(t, err)
			require.Len(t, result.Services, 1)
			require.NotNil(t, result.Services[0].Upstream)

			assert.Equal(t, tt.scheme, result.Services[0].Upstream.Scheme)
			assert.Equal(t, "10.0.0.1", result.Services[0].Upstream.Nodes[0].Host)
		})
	}
}
