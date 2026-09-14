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
	"testing"

	"github.com/stretchr/testify/assert"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestPortsWithConflictingTLSMode(t *testing.T) {
	tlsListener := func(name string, port gatewayv1.PortNumber, mode *gatewayv1.TLSModeType) gatewayv1.Listener {
		return gatewayv1.Listener{
			Name:     gatewayv1.SectionName(name),
			Port:     port,
			Protocol: gatewayv1.TLSProtocolType,
			TLS:      &gatewayv1.ListenerTLSConfig{Mode: mode},
		}
	}
	terminate := gatewayv1.TLSModeTerminate
	passthrough := gatewayv1.TLSModePassthrough

	for _, tc := range []struct {
		name      string
		listeners []gatewayv1.Listener
		conflicts []gatewayv1.PortNumber
	}{
		{
			name:      "single terminate listener",
			listeners: []gatewayv1.Listener{tlsListener("a", 443, &terminate)},
		},
		{
			name: "same mode on one port",
			listeners: []gatewayv1.Listener{
				tlsListener("a", 443, &passthrough),
				tlsListener("b", 443, &passthrough),
			},
		},
		{
			name: "distinct modes on distinct ports",
			listeners: []gatewayv1.Listener{
				tlsListener("a", 443, &terminate),
				tlsListener("b", 8443, &passthrough),
			},
		},
		{
			name: "mixed modes on one port",
			listeners: []gatewayv1.Listener{
				tlsListener("a", 8443, &terminate),
				tlsListener("b", 8443, &passthrough),
			},
			conflicts: []gatewayv1.PortNumber{8443},
		},
		{
			// An omitted mode defaults to Terminate, so this still conflicts.
			name: "omitted mode conflicts with explicit passthrough",
			listeners: []gatewayv1.Listener{
				tlsListener("a", 8443, nil),
				tlsListener("b", 8443, &passthrough),
			},
			conflicts: []gatewayv1.PortNumber{8443},
		},
		{
			// Non-TLS listeners never take part in tls.mode conflicts.
			name: "http listener sharing the port is ignored",
			listeners: []gatewayv1.Listener{
				{Name: "http", Port: 8443, Protocol: gatewayv1.HTTPProtocolType},
				tlsListener("tls", 8443, &passthrough),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gateway := &gatewayv1.Gateway{Spec: gatewayv1.GatewaySpec{Listeners: tc.listeners}}
			got := portsWithConflictingTLSMode(gateway)

			assert.Len(t, got, len(tc.conflicts))
			for _, port := range tc.conflicts {
				assert.True(t, got[port], "port %d should conflict", port)
			}
		})
	}
}
