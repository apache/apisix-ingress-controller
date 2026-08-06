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
	"k8s.io/utils/ptr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	internaltypes "github.com/apache/apisix-ingress-controller/internal/types"
)

func TestTranslateTCPRouteWithL4RoutePolicy(t *testing.T) {
	tests := []struct {
		name          string
		policy        *v1alpha1.L4RoutePolicy
		wantPlugins   []string
		wantNoPlugins bool
	}{
		{
			name: "attaches plugins from matching L4RoutePolicy",
			policy: makeL4RoutePolicy("default", "tcp-policy", "TCPRoute", "my-tcp", []v1alpha1.Plugin{
				{Name: "limit-conn", Config: mustJSON(map[string]any{"conn": 100})},
				{Name: "ip-restriction", Config: mustJSON(map[string]any{"whitelist": []string{"10.0.0.0/8"}})},
			}),
			wantPlugins: []string{"limit-conn", "ip-restriction"},
		},
		{
			name: "does not attach plugins from policy targeting different route kind",
			policy: makeL4RoutePolicy("default", "udp-policy", "UDPRoute", "my-tcp", []v1alpha1.Plugin{
				{Name: "limit-conn", Config: mustJSON(map[string]any{"conn": 100})},
			}),
			wantNoPlugins: true,
		},
		{
			name: "does not attach plugins from policy targeting different route name",
			policy: makeL4RoutePolicy("default", "tcp-policy", "TCPRoute", "other-tcp", []v1alpha1.Plugin{
				{Name: "limit-conn", Config: mustJSON(map[string]any{"conn": 100})},
			}),
			wantNoPlugins: true,
		},
		{
			name:          "succeeds with no policy in context",
			policy:        nil,
			wantNoPlugins: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			translator := NewTranslator(logr.Discard(), "")
			tctx := provider.NewDefaultTranslateContext(context.Background())

			if tt.policy != nil {
				key := k8stypes.NamespacedName{Namespace: tt.policy.Namespace, Name: tt.policy.Name}
				tctx.L4RoutePolicies[key] = tt.policy
			}

			route := &gatewayv1alpha2.TCPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-tcp",
					Namespace: "default",
				},
				Spec: gatewayv1alpha2.TCPRouteSpec{
					Rules: []gatewayv1alpha2.TCPRouteRule{
						{BackendRefs: []gatewayv1alpha2.BackendRef{}},
					},
				},
			}

			result, err := translator.TranslateTCPRoute(tctx, route)
			require.NoError(t, err)
			require.Len(t, result.Services, 1)
			require.NotEmpty(t, result.Services[0].StreamRoutes)

			plugins := result.Services[0].StreamRoutes[0].Plugins
			if tt.wantNoPlugins {
				assert.Empty(t, plugins)
			} else {
				for _, name := range tt.wantPlugins {
					assert.Contains(t, plugins, name, "expected plugin %q to be attached", name)
				}
			}
		})
	}
}

func TestTranslateUDPRouteWithL4RoutePolicy(t *testing.T) {
	tests := []struct {
		name          string
		policy        *v1alpha1.L4RoutePolicy
		wantPlugins   []string
		wantNoPlugins bool
	}{
		{
			name: "attaches plugins from matching L4RoutePolicy",
			policy: makeL4RoutePolicy("default", "udp-policy", "UDPRoute", "my-udp", []v1alpha1.Plugin{
				{Name: "limit-conn", Config: mustJSON(map[string]any{"conn": 50})},
			}),
			wantPlugins: []string{"limit-conn"},
		},
		{
			name: "does not attach plugins from policy targeting TCPRoute",
			policy: makeL4RoutePolicy("default", "tcp-policy", "TCPRoute", "my-udp", []v1alpha1.Plugin{
				{Name: "limit-conn", Config: mustJSON(map[string]any{"conn": 50})},
			}),
			wantNoPlugins: true,
		},
		{
			name:          "succeeds with no policy in context",
			policy:        nil,
			wantNoPlugins: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			translator := NewTranslator(logr.Discard(), "")
			tctx := provider.NewDefaultTranslateContext(context.Background())

			if tt.policy != nil {
				key := k8stypes.NamespacedName{Namespace: tt.policy.Namespace, Name: tt.policy.Name}
				tctx.L4RoutePolicies[key] = tt.policy
			}

			route := &gatewayv1alpha2.UDPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-udp",
					Namespace: "default",
				},
				Spec: gatewayv1alpha2.UDPRouteSpec{
					Rules: []gatewayv1alpha2.UDPRouteRule{
						{BackendRefs: []gatewayv1alpha2.BackendRef{}},
					},
				},
			}

			result, err := translator.TranslateUDPRoute(tctx, route)
			require.NoError(t, err)
			require.Len(t, result.Services, 1)
			require.NotEmpty(t, result.Services[0].StreamRoutes)

			plugins := result.Services[0].StreamRoutes[0].Plugins
			if tt.wantNoPlugins {
				assert.Empty(t, plugins)
			} else {
				for _, name := range tt.wantPlugins {
					assert.Contains(t, plugins, name, "expected plugin %q to be attached", name)
				}
			}
		})
	}
}

func TestTranslateTLSRouteWithL4RoutePolicy(t *testing.T) {
	tests := []struct {
		name          string
		policy        *v1alpha1.L4RoutePolicy
		hostnames     []string
		wantPlugins   []string
		wantNoPlugins bool
	}{
		{
			name: "attaches plugins from matching L4RoutePolicy",
			policy: makeL4RoutePolicy("default", "tls-policy", "TLSRoute", "my-tls", []v1alpha1.Plugin{
				{Name: "ip-restriction", Config: mustJSON(map[string]any{"whitelist": []string{"192.168.0.0/16"}})},
			}),
			hostnames:   []string{"example.com"},
			wantPlugins: []string{"ip-restriction"},
		},
		{
			name: "plugins attached once per rule even with multiple SNI hostnames",
			policy: makeL4RoutePolicy("default", "tls-policy", "TLSRoute", "my-tls", []v1alpha1.Plugin{
				{Name: "limit-conn", Config: mustJSON(map[string]any{"conn": 20})},
			}),
			hostnames:   []string{"foo.example.com", "bar.example.com"},
			wantPlugins: []string{"limit-conn"},
		},
		{
			name: "does not attach plugins from policy targeting TCPRoute",
			policy: makeL4RoutePolicy("default", "tcp-policy", "TCPRoute", "my-tls", []v1alpha1.Plugin{
				{Name: "limit-conn", Config: mustJSON(map[string]any{"conn": 20})},
			}),
			hostnames:     []string{"example.com"},
			wantNoPlugins: true,
		},
		{
			name:          "succeeds with no policy in context",
			policy:        nil,
			hostnames:     []string{"example.com"},
			wantNoPlugins: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			translator := NewTranslator(logr.Discard(), "")
			tctx := provider.NewDefaultTranslateContext(context.Background())

			if tt.policy != nil {
				key := k8stypes.NamespacedName{Namespace: tt.policy.Namespace, Name: tt.policy.Name}
				tctx.L4RoutePolicies[key] = tt.policy
			}

			hostnames := make([]gatewayv1alpha2.Hostname, 0, len(tt.hostnames))
			for _, h := range tt.hostnames {
				hostnames = append(hostnames, gatewayv1alpha2.Hostname(h))
			}

			route := &gatewayv1alpha2.TLSRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-tls",
					Namespace: "default",
				},
				Spec: gatewayv1alpha2.TLSRouteSpec{
					Hostnames: hostnames,
					Rules: []gatewayv1alpha2.TLSRouteRule{
						{BackendRefs: []gatewayv1alpha2.BackendRef{}},
					},
				},
			}

			result, err := translator.TranslateTLSRoute(tctx, route)
			require.NoError(t, err)
			require.Len(t, result.Services, 1)

			// Verify stream routes are created per SNI hostname
			if len(tt.hostnames) > 0 {
				assert.Len(t, result.Services[0].StreamRoutes, len(tt.hostnames))
			}

			// Plugins are attached at the stream_route level so the APISIX stream proxy
			// applies them; with multiple SNIs each stream_route carries its own copy.
			require.NotEmpty(t, result.Services[0].StreamRoutes)
			plugins := result.Services[0].StreamRoutes[0].Plugins
			if tt.wantNoPlugins {
				assert.Empty(t, plugins)
			} else {
				for _, streamRoute := range result.Services[0].StreamRoutes {
					for _, name := range tt.wantPlugins {
						assert.Contains(t, streamRoute.Plugins, name, "expected plugin %q to be attached", name)
					}
				}
			}
		})
	}
}

func TestTranslateTCPRouteUpstreamScheme(t *testing.T) {
	const (
		namespace   = "default"
		serviceName = "backend"
		portName    = "tcp"
		portNumber  = int32(6000)
	)

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
	tctx.BackendTrafficPolicies[serviceKey] = &v1alpha1.BackendTrafficPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "backend-policy",
			Namespace: namespace,
		},
		Spec: v1alpha1.BackendTrafficPolicySpec{
			TargetRefs: []v1alpha1.BackendPolicyTargetReferenceWithSectionName{{
				LocalPolicyTargetReference: gatewayv1.LocalPolicyTargetReference{
					Name: gatewayv1.ObjectName(serviceName),
					Kind: gatewayv1.Kind(internaltypes.KindService),
				},
			}},
			Scheme: apiv2.SchemeTLS,
		},
	}

	route := &gatewayv1alpha2.TCPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-tcp",
			Namespace: namespace,
		},
		Spec: gatewayv1alpha2.TCPRouteSpec{
			Rules: []gatewayv1alpha2.TCPRouteRule{{
				BackendRefs: []gatewayv1alpha2.BackendRef{{
					BackendObjectReference: gatewayv1.BackendObjectReference{
						Name: gatewayv1.ObjectName(serviceName),
						Port: ptr.To(portNumber),
					},
				}},
			}},
		},
	}

	result, err := translator.TranslateTCPRoute(tctx, route)
	require.NoError(t, err)
	require.Len(t, result.Services, 1)
	require.NotNil(t, result.Services[0].Upstream)

	assert.Equal(t, apiv2.SchemeTLS, result.Services[0].Upstream.Scheme)
	assert.Equal(t, "10.0.0.1", result.Services[0].Upstream.Nodes[0].Host)
}
