// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package translator

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/controller/config"
	"github.com/apache/apisix-ingress-controller/internal/provider"
)

func TestTranslateGRPCRouteServerPortVarsByMode(t *testing.T) {
	singlePortVars := adctypes.Vars{
		{
			{StrVal: "server_port"},
			{StrVal: "=="},
			{StrVal: "9080"},
		},
	}
	multiPortVars := adctypes.Vars{
		{
			{StrVal: "server_port"},
			{StrVal: "in"},
			{SliceVal: []adctypes.StringOrSlice{
				{StrVal: "9080"},
				{StrVal: "9081"},
			}},
		},
	}

	tests := []struct {
		name string
		mode config.ListenerPortMatchMode
		// explicit is the provenance-aware signal the controller stores on tctx
		// (HasExplicitListenerMatch); the cross-Gateway sectionName/port
		// resolution that computes it is covered by the controller tests.
		explicit  bool
		listeners []gatewayv1.Listener
		expected  adctypes.Vars
	}{
		{
			name: "auto mode: no injection for single listener without explicit target",
			mode: config.ListenerPortMatchModeAuto,
			listeners: []gatewayv1.Listener{
				{Name: "grpc-main", Protocol: gatewayv1.HTTPProtocolType, Port: gatewayv1.PortNumber(9080)},
			},
			expected: nil,
		},
		{
			name:     "auto mode: inject for explicit sectionName target",
			mode:     config.ListenerPortMatchModeAuto,
			explicit: true,
			listeners: []gatewayv1.Listener{
				{Name: "grpc-main", Protocol: gatewayv1.HTTPProtocolType, Port: gatewayv1.PortNumber(9080)},
			},
			expected: singlePortVars,
		},
		{
			name: "auto mode: inject for multiple listener ports",
			mode: config.ListenerPortMatchModeAuto,
			listeners: []gatewayv1.Listener{
				{Name: "grpc-main", Protocol: gatewayv1.HTTPProtocolType, Port: gatewayv1.PortNumber(9081)},
				{Name: "grpc-alt", Protocol: gatewayv1.HTTPProtocolType, Port: gatewayv1.PortNumber(9080)},
			},
			expected: multiPortVars,
		},
		{
			name:     "explicit mode: inject for explicit target",
			mode:     config.ListenerPortMatchModeExplicit,
			explicit: true,
			listeners: []gatewayv1.Listener{
				{Name: "grpc-main", Protocol: gatewayv1.HTTPProtocolType, Port: gatewayv1.PortNumber(9080)},
			},
			expected: singlePortVars,
		},
		{
			name: "explicit mode: no injection for multiple listener ports without explicit target",
			mode: config.ListenerPortMatchModeExplicit,
			listeners: []gatewayv1.Listener{
				{Name: "grpc-main", Protocol: gatewayv1.HTTPProtocolType, Port: gatewayv1.PortNumber(9081)},
				{Name: "grpc-alt", Protocol: gatewayv1.HTTPProtocolType, Port: gatewayv1.PortNumber(9080)},
			},
			expected: nil,
		},
		{
			name:     "off mode: no injection even with explicit target",
			mode:     config.ListenerPortMatchModeOff,
			explicit: true,
			listeners: []gatewayv1.Listener{
				{Name: "grpc-main", Protocol: gatewayv1.HTTPProtocolType, Port: gatewayv1.PortNumber(9080)},
			},
			expected: nil,
		},
		{
			name: "off mode: no injection for multiple listener ports",
			mode: config.ListenerPortMatchModeOff,
			listeners: []gatewayv1.Listener{
				{Name: "grpc-main", Protocol: gatewayv1.HTTPProtocolType, Port: gatewayv1.PortNumber(9081)},
				{Name: "grpc-alt", Protocol: gatewayv1.HTTPProtocolType, Port: gatewayv1.PortNumber(9080)},
			},
			expected: nil,
		},
		{
			// An unset mode must not start injecting predicates behind the
			// operator's back, so it resolves to off rather than to auto.
			name:     "empty mode normalizes to off",
			mode:     "",
			explicit: true,
			listeners: []gatewayv1.Listener{
				{Name: "grpc-main", Protocol: gatewayv1.HTTPProtocolType, Port: gatewayv1.PortNumber(9080)},
			},
			expected: nil,
		},
		{
			// Same-port listeners differing by hostname are isolated by host, not by
			// port, so no server_port var must be emitted (it would pin the route to
			// the Gateway port and drop every request to 404).
			name:     "auto mode: no injection for same-port listeners differing by hostname",
			mode:     config.ListenerPortMatchModeAuto,
			explicit: true,
			listeners: []gatewayv1.Listener{
				{Name: "grpc-main", Protocol: gatewayv1.HTTPProtocolType, Port: gatewayv1.PortNumber(80), Hostname: ptr.To(gatewayv1.Hostname("bar.com"))},
				{Name: "grpc-alt", Protocol: gatewayv1.HTTPProtocolType, Port: gatewayv1.PortNumber(80), Hostname: ptr.To(gatewayv1.Hostname("foo.bar.com"))},
			},
			expected: nil,
		},
		{
			// A hostname-less sibling listener triggers a server_port var, but once
			// emitted it must cover every targeted port - including the hostname
			// listener's - or traffic arriving through the hostname listener is
			// silently dropped. So the predicate lists both ports, not just 9080.
			name:     "explicit mode: mixed hostname and hostname-less listeners keep every targeted port",
			mode:     config.ListenerPortMatchModeExplicit,
			explicit: true,
			listeners: []gatewayv1.Listener{
				{Name: "grpc-main", Protocol: gatewayv1.HTTPProtocolType, Port: gatewayv1.PortNumber(80), Hostname: ptr.To(gatewayv1.Hostname("bar.com"))},
				{Name: "grpc-alt", Protocol: gatewayv1.HTTPProtocolType, Port: gatewayv1.PortNumber(9080)},
			},
			expected: adctypes.Vars{
				{
					{StrVal: "server_port"},
					{StrVal: "in"},
					{SliceVal: []adctypes.StringOrSlice{
						{StrVal: "80"},
						{StrVal: "9080"},
					}},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tctx := provider.NewDefaultTranslateContext(context.Background())
			tctx.HasExplicitListenerMatch = tt.explicit
			tctx.Listeners = tt.listeners

			grpcRoute := &gatewayv1.GRPCRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "route",
					Namespace: "default",
				},
				Spec: gatewayv1.GRPCRouteSpec{
					Rules: []gatewayv1.GRPCRouteRule{
						{},
					},
				},
			}

			translator := NewTranslator(logr.Discard(), tt.mode)
			got, err := translator.TranslateGRPCRoute(tctx, grpcRoute)
			assert.NoError(t, err)
			if assert.Len(t, got.Services, 1) && assert.Len(t, got.Services[0].Routes, 1) {
				assert.Equal(t, tt.expected, got.Services[0].Routes[0].Vars)
			}
		})
	}
}
