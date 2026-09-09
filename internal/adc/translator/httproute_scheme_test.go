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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/controller/config"
	"github.com/apache/apisix-ingress-controller/internal/provider"
)

func schemeVar(scheme string) []adctypes.StringOrSlice {
	return []adctypes.StringOrSlice{
		{StrVal: "scheme"},
		{StrVal: "=="},
		{StrVal: scheme},
	}
}

// A route answers only the schemes its listeners accept. $scheme is evaluated
// against the connection APISIX accepted, so this holds whatever port mapping sits
// in front of the data plane, and it is independent of listener_port_match_mode.
func TestTranslateHTTPRouteSchemeVar(t *testing.T) {
	pathMatchType := gatewayv1.PathMatchPathPrefix
	pathValue := "/"

	https := func(port gatewayv1.PortNumber, hostname *gatewayv1.Hostname) gatewayv1.Listener {
		return gatewayv1.Listener{
			Name:     "https",
			Protocol: gatewayv1.HTTPSProtocolType,
			Port:     port,
			Hostname: hostname,
		}
	}
	plain := func(port gatewayv1.PortNumber) gatewayv1.Listener {
		return gatewayv1.Listener{
			Name:     "http",
			Protocol: gatewayv1.HTTPProtocolType,
			Port:     port,
		}
	}

	tests := []struct {
		name      string
		mode      config.ListenerPortMatchMode
		listeners []gatewayv1.Listener
		// want is the scheme the route must be pinned to, or "" for no pinning.
		want string
	}{
		{
			name:      "https listener pins https with the default mode",
			mode:      config.ListenerPortMatchModeOff,
			listeners: []gatewayv1.Listener{https(443, nil)},
			want:      "https",
		},
		{
			// The declared 443 need not be the port APISIX listens on, which is what
			// makes server_port unusable here and the scheme var necessary.
			name:      "https listener with a hostname is pinned too",
			mode:      config.ListenerPortMatchModeOff,
			listeners: []gatewayv1.Listener{https(443, ptr.To(gatewayv1.Hostname("secure.example")))},
			want:      "https",
		},
		{
			name:      "http listener pins http",
			mode:      config.ListenerPortMatchModeOff,
			listeners: []gatewayv1.Listener{plain(80)},
			want:      "http",
		},
		{
			name:      "several listeners of the same protocol still pin it",
			mode:      config.ListenerPortMatchModeOff,
			listeners: []gatewayv1.Listener{https(443, nil), https(8443, ptr.To(gatewayv1.Hostname("secure.example")))},
			want:      "https",
		},
		{
			name:      "a route attached to both protocols serves both",
			mode:      config.ListenerPortMatchModeOff,
			listeners: []gatewayv1.Listener{https(443, nil), plain(80)},
			want:      "",
		},
		{
			name:      "no listener means nothing to pin",
			mode:      config.ListenerPortMatchModeOff,
			listeners: nil,
			want:      "",
		},
		{
			name:      "auto mode does not change the scheme predicate",
			mode:      config.ListenerPortMatchModeAuto,
			listeners: []gatewayv1.Listener{https(9443, nil)},
			want:      "https",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tctx := provider.NewDefaultTranslateContext(context.Background())
			tctx.Listeners = tt.listeners

			httpRoute := &gatewayv1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{Name: "route", Namespace: "default"},
				Spec: gatewayv1.HTTPRouteSpec{
					Rules: []gatewayv1.HTTPRouteRule{{
						Matches: []gatewayv1.HTTPRouteMatch{{
							Path: &gatewayv1.HTTPPathMatch{Type: &pathMatchType, Value: &pathValue},
						}},
					}},
				},
			}

			got, err := NewTranslator(logr.Discard(), tt.mode).TranslateHTTPRoute(tctx, httpRoute)
			assert.NoError(t, err)
			if !assert.Len(t, got.Services, 1) || !assert.Len(t, got.Services[0].Routes, 1) {
				return
			}
			vars := got.Services[0].Routes[0].Vars
			if tt.want != "" {
				assert.Contains(t, vars, schemeVar(tt.want),
					"a route whose listeners agree on a scheme must be pinned to it")
				return
			}
			assert.NotContains(t, vars, schemeVar("http"))
			assert.NotContains(t, vars, schemeVar("https"),
				"a route whose listeners disagree must serve both schemes")
		})
	}
}

// The L4 protocols carry TLSRoute, TCPRoute and UDPRoute, which have no request
// scheme. listenerScheme must refuse to pin those rather than guess, so that a
// listener set containing one leaves the route alone.
func TestListenerScheme(t *testing.T) {
	listener := func(protocol gatewayv1.ProtocolType) gatewayv1.Listener {
		return gatewayv1.Listener{Name: gatewayv1.SectionName(protocol), Protocol: protocol, Port: 443}
	}

	for name, tt := range map[string]struct {
		listeners []gatewayv1.Listener
		want      string
	}{
		"http":              {[]gatewayv1.Listener{listener(gatewayv1.HTTPProtocolType)}, "http"},
		"https":             {[]gatewayv1.Listener{listener(gatewayv1.HTTPSProtocolType)}, "https"},
		"tls":               {[]gatewayv1.Listener{listener(gatewayv1.TLSProtocolType)}, ""},
		"tcp":               {[]gatewayv1.Listener{listener(gatewayv1.TCPProtocolType)}, ""},
		"udp":               {[]gatewayv1.Listener{listener(gatewayv1.UDPProtocolType)}, ""},
		"https beside tls":  {[]gatewayv1.Listener{listener(gatewayv1.HTTPSProtocolType), listener(gatewayv1.TLSProtocolType)}, ""},
		"http beside https": {[]gatewayv1.Listener{listener(gatewayv1.HTTPProtocolType), listener(gatewayv1.HTTPSProtocolType)}, ""},
		"none":              {nil, ""},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.want, listenerScheme(tt.listeners))
		})
	}
}
