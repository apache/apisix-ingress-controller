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
	"k8s.io/utils/ptr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/internal/controller/config"
	"github.com/apache/apisix-ingress-controller/internal/provider"
)

func tlsModeListener(name string, port int32, mode *gatewayv1.TLSModeType, hostname string) gatewayv1.Listener {
	listener := gatewayv1.Listener{
		Name:     gatewayv1.SectionName(name),
		Protocol: gatewayv1.TLSProtocolType,
		Port:     port,
		TLS:      &gatewayv1.ListenerTLSConfig{Mode: mode},
	}
	if hostname != "" {
		listener.Hostname = ptr.To(gatewayv1.Hostname(hostname))
	}
	return listener
}

func translateTLSRoute(t *testing.T, listeners []gatewayv1.Listener, explicit bool, hostnames ...string) *TranslateResult {
	t.Helper()

	// listener_port_match_mode defaults to off; auto is what the e2e and
	// conformance runs use, and what makes the per-port fan-out observable.
	translator := NewTranslator(logr.Discard(), config.ListenerPortMatchModeAuto)
	tctx := provider.NewDefaultTranslateContext(context.Background())
	tctx.Listeners = listeners
	tctx.HasExplicitListenerMatch = explicit

	specHostnames := make([]gatewayv1.Hostname, 0, len(hostnames))
	for _, hostname := range hostnames {
		specHostnames = append(specHostnames, gatewayv1.Hostname(hostname))
	}

	result, err := translator.TranslateTLSRoute(tctx, &gatewayv1.TLSRoute{
		ObjectMeta: metav1.ObjectMeta{Name: "my-tls", Namespace: "default"},
		Spec: gatewayv1.TLSRouteSpec{
			Hostnames: specHostnames,
			Rules: []gatewayv1.TLSRouteRule{
				{BackendRefs: []gatewayv1.BackendRef{}},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, result.Services, 1)
	return result
}

func TestTranslateTLSRouteTLSPassthrough(t *testing.T) {
	terminate := gatewayv1.TLSModeTerminate
	passthrough := gatewayv1.TLSModePassthrough

	for _, tc := range []struct {
		name      string
		listeners []gatewayv1.Listener
		explicit  bool
		want      []*bool
	}{
		{
			name:      "passthrough listener asks the data plane to pass the stream through",
			listeners: []gatewayv1.Listener{tlsModeListener("tls", 9110, &passthrough, "")},
			explicit:  true,
			want:      []*bool{ptr.To(true)},
		},
		{
			name:      "terminate listener leaves the flag off",
			listeners: []gatewayv1.Listener{tlsModeListener("tls", 9110, &terminate, "")},
			explicit:  true,
			want:      []*bool{nil},
		},
		{
			// tls.mode is optional and defaults to Terminate.
			name:      "omitted mode is terminate",
			listeners: []gatewayv1.Listener{tlsModeListener("tls", 9110, nil, "")},
			explicit:  true,
			want:      []*bool{nil},
		},
		{
			name: "each port carries the mode of its own listener",
			listeners: []gatewayv1.Listener{
				tlsModeListener("terminate", 9110, &terminate, ""),
				tlsModeListener("passthrough", 9120, &passthrough, ""),
			},
			want: []*bool{nil, ptr.To(true)},
		},
		{
			// A physical stream listen has one mode, so a port whose matched
			// listeners disagree cannot be served both ways; terminating is
			// what the gateway did before passthrough existed.
			name: "listeners disagreeing on one port fall back to terminate",
			listeners: []gatewayv1.Listener{
				tlsModeListener("a", 9110, &passthrough, ""),
				tlsModeListener("b", 9110, &terminate, ""),
			},
			explicit: true,
			want:     []*bool{nil},
		},
		{
			// No listener port to pin to: the single portless StreamRoute takes
			// the mode every matched listener agrees on.
			name:      "portless stream route still follows the listener mode",
			listeners: []gatewayv1.Listener{tlsModeListener("tls", 9110, &passthrough, "")},
			want:      []*bool{ptr.To(true)},
		},
		{
			name:      "no matched listener leaves the flag off",
			listeners: nil,
			want:      []*bool{nil},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := translateTLSRoute(t, tc.listeners, tc.explicit, "example.com")

			streamRoutes := result.Services[0].StreamRoutes
			require.Len(t, streamRoutes, len(tc.want))
			for i, want := range tc.want {
				assert.Equal(t, want, streamRoutes[i].TLSPassthrough, "stream route %d", i)
			}
		})
	}
}

func TestTranslateTLSRouteSNIs(t *testing.T) {
	passthrough := gatewayv1.TLSModePassthrough

	t.Run("a single hostname uses the singular sni", func(t *testing.T) {
		result := translateTLSRoute(t, nil, false, "example.com")

		streamRoutes := result.Services[0].StreamRoutes
		require.Len(t, streamRoutes, 1)
		assert.Equal(t, "example.com", streamRoutes[0].SNI)
		assert.Empty(t, streamRoutes[0].SNIs)
	})

	t.Run("several hostnames share one stream route", func(t *testing.T) {
		result := translateTLSRoute(t, nil, false, "a.example.com", "b.example.com")

		// Regression: every hostname used to produce its own StreamRoute under
		// one name, so they collided on a single id and only the last survived.
		streamRoutes := result.Services[0].StreamRoutes
		require.Len(t, streamRoutes, 1)
		assert.Equal(t, []string{"a.example.com", "b.example.com"}, streamRoutes[0].SNIs)
		assert.Empty(t, streamRoutes[0].SNI)
	})

	t.Run("no hostname falls back to the listener hostnames", func(t *testing.T) {
		result := translateTLSRoute(t, []gatewayv1.Listener{
			tlsModeListener("tls", 9110, &passthrough, "*.example.com"),
		}, true)

		streamRoutes := result.Services[0].StreamRoutes
		require.Len(t, streamRoutes, 1)
		assert.Equal(t, "*.example.com", streamRoutes[0].SNI)
	})

	t.Run("no hostname anywhere falls back to the catch-all", func(t *testing.T) {
		result := translateTLSRoute(t, []gatewayv1.Listener{
			tlsModeListener("tls", 9110, &passthrough, ""),
		}, true)

		streamRoutes := result.Services[0].StreamRoutes
		require.Len(t, streamRoutes, 1)
		assert.Equal(t, "*", streamRoutes[0].SNI)
	})
}

func TestTranslateTLSRouteServerPort(t *testing.T) {
	passthrough := gatewayv1.TLSModePassthrough

	t.Run("each listener port gets its own stream route", func(t *testing.T) {
		result := translateTLSRoute(t, []gatewayv1.Listener{
			tlsModeListener("a", 9110, &passthrough, ""),
			tlsModeListener("b", 9120, &passthrough, ""),
		}, false, "example.com")

		// Regression: without a server_port match both listeners' traffic fell
		// onto one StreamRoute, and the two shared a name, hence an id.
		streamRoutes := result.Services[0].StreamRoutes
		require.Len(t, streamRoutes, 2)
		assert.Equal(t, int32(9110), streamRoutes[0].ServerPort)
		assert.Equal(t, int32(9120), streamRoutes[1].ServerPort)
		assert.NotEqual(t, streamRoutes[0].ID, streamRoutes[1].ID)
		assert.NotEqual(t, streamRoutes[0].Name, streamRoutes[1].Name)
	})

	t.Run("a single listener without explicit targeting stays portless", func(t *testing.T) {
		result := translateTLSRoute(t, []gatewayv1.Listener{
			tlsModeListener("a", 9110, &passthrough, ""),
		}, false, "example.com")

		streamRoutes := result.Services[0].StreamRoutes
		require.Len(t, streamRoutes, 1)
		assert.Zero(t, streamRoutes[0].ServerPort)
	})
}
