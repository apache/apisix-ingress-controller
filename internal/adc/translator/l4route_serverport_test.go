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
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/apache/apisix-ingress-controller/internal/controller/config"
	"github.com/apache/apisix-ingress-controller/internal/provider"
)

func tcpListener(name string, port int32) gatewayv1.Listener {
	return gatewayv1.Listener{
		Name:     gatewayv1.SectionName(name),
		Protocol: gatewayv1.TCPProtocolType,
		Port:     port,
	}
}

func udpListener(name string, port int32) gatewayv1.Listener {
	return gatewayv1.Listener{
		Name:     gatewayv1.SectionName(name),
		Protocol: gatewayv1.UDPProtocolType,
		Port:     port,
	}
}

func TestTranslateTCPRouteServerPort(t *testing.T) {
	tests := []struct {
		name string
		// listeners the controller would have matched for this route's parentRefs
		listeners []gatewayv1.Listener
		// explicit is the provenance-aware signal the controller stores on tctx
		// (HasExplicitListenerMatch) when the route targeted a listener by an
		// explicit sectionName or port.
		explicit    bool
		wantPorts   []int32
		wantNoMatch bool
	}{
		{
			name:      "explicit sectionName injects the matching listener port",
			listeners: []gatewayv1.Listener{tcpListener("tcp-a", 9100)},
			explicit:  true,
			wantPorts: []int32{9100},
		},
		{
			name:      "multiple listener ports fan out even without explicit targeting",
			listeners: []gatewayv1.Listener{tcpListener("tcp-a", 9100), tcpListener("tcp-b", 9101)},
			wantPorts: []int32{9100, 9101},
		},
		{
			name:      "duplicate ports across gateways are de-duplicated",
			listeners: []gatewayv1.Listener{tcpListener("tcp-a", 9100), tcpListener("tcp-a2", 9100)},
			explicit:  true,
			wantPorts: []int32{9100},
		},
		{
			name:        "single listener without explicit targeting keeps a portless StreamRoute",
			listeners:   []gatewayv1.Listener{tcpListener("tcp-a", 9100)},
			wantNoMatch: true,
		},
		{
			name:        "no matched listener falls back to a single portless StreamRoute",
			listeners:   nil,
			explicit:    true,
			wantNoMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// listener_port_match_mode defaults to off; these cases assert the
			// injection behavior, so exercise the translator in auto mode.
			translator := NewTranslator(logr.Discard(), config.ListenerPortMatchModeAuto)
			tctx := provider.NewDefaultTranslateContext(context.Background())
			tctx.Listeners = tt.listeners
			tctx.HasExplicitListenerMatch = tt.explicit

			route := &gatewayv1alpha2.TCPRoute{
				ObjectMeta: metav1.ObjectMeta{Name: "my-tcp", Namespace: "default"},
				Spec: gatewayv1alpha2.TCPRouteSpec{
					Rules: []gatewayv1alpha2.TCPRouteRule{
						{BackendRefs: []gatewayv1alpha2.BackendRef{}},
					},
				},
			}

			result, err := translator.TranslateTCPRoute(tctx, route)
			require.NoError(t, err)
			require.Len(t, result.Services, 1)
			streamRoutes := result.Services[0].StreamRoutes

			if tt.wantNoMatch {
				require.Len(t, streamRoutes, 1)
				assert.Zero(t, streamRoutes[0].ServerPort)
				return
			}

			require.Len(t, streamRoutes, len(tt.wantPorts))
			gotPorts := make([]int32, 0, len(streamRoutes))
			ids := make(map[string]struct{})
			names := make(map[string]struct{})
			for _, sr := range streamRoutes {
				gotPorts = append(gotPorts, sr.ServerPort)
				ids[sr.ID] = struct{}{}
				names[sr.Name] = struct{}{}
			}
			assert.ElementsMatch(t, tt.wantPorts, gotPorts)
			// Distinct name/ID per listener port so StreamRoutes do not collide.
			assert.Len(t, ids, len(streamRoutes))
			assert.Len(t, names, len(streamRoutes))
		})
	}
}

func TestTranslateUDPRouteServerPort(t *testing.T) {
	tests := []struct {
		name        string
		listeners   []gatewayv1.Listener
		explicit    bool
		wantPorts   []int32
		wantNoMatch bool
	}{
		{
			name:      "explicit sectionName injects the matching listener port",
			listeners: []gatewayv1.Listener{udpListener("udp-a", 9200)},
			explicit:  true,
			wantPorts: []int32{9200},
		},
		{
			name:      "two listeners on different ports produce distinct StreamRoutes",
			listeners: []gatewayv1.Listener{udpListener("udp-a", 9200), udpListener("udp-b", 9201)},
			wantPorts: []int32{9200, 9201},
		},
		{
			name:        "single listener without explicit targeting keeps a portless StreamRoute",
			listeners:   []gatewayv1.Listener{udpListener("udp-a", 9200)},
			wantNoMatch: true,
		},
		{
			name:        "no matched listener falls back to a single portless StreamRoute",
			listeners:   nil,
			explicit:    true,
			wantNoMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// listener_port_match_mode defaults to off; these cases assert the
			// injection behavior, so exercise the translator in auto mode.
			translator := NewTranslator(logr.Discard(), config.ListenerPortMatchModeAuto)
			tctx := provider.NewDefaultTranslateContext(context.Background())
			tctx.Listeners = tt.listeners
			tctx.HasExplicitListenerMatch = tt.explicit

			route := &gatewayv1alpha2.UDPRoute{
				ObjectMeta: metav1.ObjectMeta{Name: "my-udp", Namespace: "default"},
				Spec: gatewayv1alpha2.UDPRouteSpec{
					Rules: []gatewayv1alpha2.UDPRouteRule{
						{BackendRefs: []gatewayv1alpha2.BackendRef{}},
					},
				},
			}

			result, err := translator.TranslateUDPRoute(tctx, route)
			require.NoError(t, err)
			require.Len(t, result.Services, 1)
			streamRoutes := result.Services[0].StreamRoutes

			if tt.wantNoMatch {
				require.Len(t, streamRoutes, 1)
				assert.Zero(t, streamRoutes[0].ServerPort)
				return
			}

			require.Len(t, streamRoutes, len(tt.wantPorts))
			gotPorts := make([]int32, 0, len(streamRoutes))
			ids := make(map[string]struct{})
			for _, sr := range streamRoutes {
				gotPorts = append(gotPorts, sr.ServerPort)
				ids[sr.ID] = struct{}{}
			}
			assert.ElementsMatch(t, tt.wantPorts, gotPorts)
			assert.Len(t, ids, len(streamRoutes))
		})
	}
}
