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

var httpsSchemeVar = []adctypes.StringOrSlice{
	{StrVal: "scheme"},
	{StrVal: "=="},
	{StrVal: "https"},
}

// A route attached only to HTTPS listeners must not answer plaintext requests for
// the same host and path. $scheme is evaluated against the connection APISIX
// accepted, so this holds whatever port mapping sits in front of the data plane,
// and it is therefore independent of listener_port_match_mode.
func TestTranslateHTTPRouteSchemeVar(t *testing.T) {
	pathMatchType := gatewayv1.PathMatchPathPrefix
	pathValue := "/"

	https := func(port int32, hostname *gatewayv1.Hostname) gatewayv1.Listener {
		return gatewayv1.Listener{
			Name:     "https",
			Protocol: gatewayv1.HTTPSProtocolType,
			Port:     gatewayv1.PortNumber(port),
			Hostname: hostname,
		}
	}
	plain := func(port int32) gatewayv1.Listener {
		return gatewayv1.Listener{
			Name:     "http",
			Protocol: gatewayv1.HTTPProtocolType,
			Port:     gatewayv1.PortNumber(port),
		}
	}

	tests := []struct {
		name       string
		mode       config.ListenerPortMatchMode
		listeners  []gatewayv1.Listener
		wantScheme bool
	}{
		{
			name:       "https listener pins the scheme with the default mode",
			mode:       config.ListenerPortMatchModeOff,
			listeners:  []gatewayv1.Listener{https(443, nil)},
			wantScheme: true,
		},
		{
			// The declared 443 need not be the port APISIX listens on, which is what
			// makes server_port unusable here and the scheme var necessary.
			name:       "https listener with a hostname is pinned too",
			mode:       config.ListenerPortMatchModeOff,
			listeners:  []gatewayv1.Listener{https(443, ptr.To(gatewayv1.Hostname("secure.example")))},
			wantScheme: true,
		},
		{
			name:       "a plaintext listener in the set keeps the route on both schemes",
			mode:       config.ListenerPortMatchModeOff,
			listeners:  []gatewayv1.Listener{https(443, nil), plain(80)},
			wantScheme: false,
		},
		{
			name:       "plaintext only is left alone",
			mode:       config.ListenerPortMatchModeOff,
			listeners:  []gatewayv1.Listener{plain(80)},
			wantScheme: false,
		},
		{
			name:       "no listener means nothing to pin",
			mode:       config.ListenerPortMatchModeOff,
			listeners:  nil,
			wantScheme: false,
		},
		{
			name:       "auto mode still pins the scheme",
			mode:       config.ListenerPortMatchModeAuto,
			listeners:  []gatewayv1.Listener{https(9443, nil)},
			wantScheme: true,
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
			if tt.wantScheme {
				assert.Contains(t, vars, httpsSchemeVar,
					"a route attached only to HTTPS listeners must be pinned to the https scheme")
				return
			}
			assert.NotContains(t, vars, httpsSchemeVar,
				"a route that can be reached over plaintext must not be pinned to https")
		})
	}
}
