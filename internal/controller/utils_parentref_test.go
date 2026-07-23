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

package controller

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/apache/apisix-ingress-controller/internal/controller/config"
)

func parentRefTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, gatewayv1.Install(scheme))
	require.NoError(t, gatewayv1alpha2.Install(scheme))
	return scheme
}

func newParentRefGatewayClass() *gatewayv1.GatewayClass {
	return &gatewayv1.GatewayClass{
		ObjectMeta: metav1.ObjectMeta{Name: "apisix"},
		Spec: gatewayv1.GatewayClassSpec{
			ControllerName: gatewayv1.GatewayController(config.ControllerConfig.ControllerName),
		},
	}
}

// TestParseRouteParentRefs_ReasonOrderIndependent verifies the parent condition
// reason does not depend on the order of gateway.spec.listeners. A
// protocol-compatible listener that only fails hostname intersection must yield
// NoMatchingListenerHostname regardless of where an unrelated incompatible
// listener sits in the list.
func TestParseRouteParentRefs_ReasonOrderIndependent(t *testing.T) {
	scheme := parentRefTestScheme(t)

	httpListener := gatewayv1.Listener{
		Name:     "http",
		Port:     80,
		Protocol: gatewayv1.HTTPProtocolType,
		Hostname: ptr.To(gatewayv1.Hostname("foo.example.com")),
	}
	tcpListener := gatewayv1.Listener{
		Name:     "tcp",
		Port:     9000,
		Protocol: gatewayv1.TCPProtocolType,
	}
	route := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "r"},
		Spec: gatewayv1.HTTPRouteSpec{
			Hostnames: []gatewayv1.Hostname{"bar.com"},
		},
	}

	for _, tc := range []struct {
		name      string
		listeners []gatewayv1.Listener
	}{
		{"compatible-first", []gatewayv1.Listener{httpListener, tcpListener}},
		{"incompatible-first", []gatewayv1.Listener{tcpListener, httpListener}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gw := &gatewayv1.Gateway{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "gw"},
				Spec: gatewayv1.GatewaySpec{
					GatewayClassName: "apisix",
					Listeners:        tc.listeners,
				},
			}
			cli := fake.NewClientBuilder().WithScheme(scheme).
				WithObjects(newParentRefGatewayClass(), gw).Build()

			got, err := ParseRouteParentRefs(context.Background(), cli, logr.Discard(), route,
				[]gatewayv1.ParentReference{{Name: "gw"}})
			require.NoError(t, err)
			require.Len(t, got, 1)
			cond := got[0].Conditions[0]
			assert.Equal(t, metav1.ConditionFalse, cond.Status)
			assert.Equal(t, string(gatewayv1.RouteReasonNoMatchingListenerHostname), cond.Reason,
				"hostname mismatch on the compatible listener must win regardless of listener order")
		})
	}
}

// TestParseRouteParentRefs_ConflictingTLSModePort verifies a route does not
// attach to a listener whose port carries a conflicting tls.mode: such a listener
// is not programmable, so the route must not be Accepted.
func TestParseRouteParentRefs_ConflictingTLSModePort(t *testing.T) {
	scheme := parentRefTestScheme(t)
	terminate := gatewayv1.TLSModeTerminate
	passthrough := gatewayv1.TLSModePassthrough

	gw := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "gw"},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: "apisix",
			Listeners: []gatewayv1.Listener{
				{Name: "tls-a", Port: 443, Protocol: gatewayv1.TLSProtocolType, TLS: &gatewayv1.ListenerTLSConfig{Mode: &terminate}},
				{Name: "tls-b", Port: 443, Protocol: gatewayv1.TLSProtocolType, TLS: &gatewayv1.ListenerTLSConfig{Mode: &passthrough}},
			},
		},
	}
	route := &gatewayv1alpha2.TLSRoute{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "tr"},
	}
	cli := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(newParentRefGatewayClass(), gw).Build()

	got, err := ParseRouteParentRefs(context.Background(), cli, logr.Discard(), route,
		[]gatewayv1.ParentReference{{Name: "gw"}})
	require.NoError(t, err)
	require.Len(t, got, 1)
	cond := got[0].Conditions[0]
	assert.Equal(t, metav1.ConditionFalse, cond.Status,
		"route must not attach to a conflicting-tls-mode port")
	assert.Equal(t, string(gatewayv1.RouteReasonNotAllowedByListeners), cond.Reason)
}
