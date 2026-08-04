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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/apache/apisix-ingress-controller/internal/controller/indexer"
)

// TestGetListenerStatus_Conflicted covers both ends of the Conflicted condition.
// The no-conflict case is the one worth pinning: NoConflicts is only a valid
// reason on Status=False, and reporting it on Status=True made every ordinary
// listener look conflicted while the conflict branch below was a no-op.
func TestGetListenerStatus_Conflicted(t *testing.T) {
	scheme := parentRefTestScheme(t)
	tlsListener := func(name string, port gatewayv1.PortNumber, mode gatewayv1.TLSModeType) gatewayv1.Listener {
		return gatewayv1.Listener{
			Name:     gatewayv1.SectionName(name),
			Port:     port,
			Protocol: gatewayv1.TLSProtocolType,
			TLS:      &gatewayv1.ListenerTLSConfig{Mode: ptr.To(mode)},
		}
	}

	for _, tc := range []struct {
		name       string
		listeners  []gatewayv1.Listener
		wantStatus metav1.ConditionStatus
		wantReason gatewayv1.ListenerConditionReason
	}{
		{
			name:       "no conflict",
			listeners:  []gatewayv1.Listener{tlsListener("tls", 443, gatewayv1.TLSModePassthrough)},
			wantStatus: metav1.ConditionFalse,
			wantReason: gatewayv1.ListenerReasonNoConflicts,
		},
		{
			// Two listeners disagreeing on tls.mode for one port cannot both be
			// programmed, so both report the conflict.
			name: "tls mode conflict on one port",
			listeners: []gatewayv1.Listener{
				tlsListener("passthrough", 8443, gatewayv1.TLSModePassthrough),
				tlsListener("terminate", 8443, gatewayv1.TLSModeTerminate),
			},
			wantStatus: metav1.ConditionTrue,
			wantReason: gatewayv1.ListenerReasonProtocolConflict,
		},
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
				WithObjects(newParentRefGatewayClass(), gw).
				// Counting attached routes goes through the parentRefs index; no
				// route exists here, only the index has to be known.
				WithIndex(&gatewayv1alpha2.TLSRoute{}, indexer.ParentRefs,
					func(client.Object) []string { return nil }).
				Build()

			statuses, err := getListenerStatus(context.Background(), cli, gw)
			require.NoError(t, err)
			require.Len(t, statuses, len(tc.listeners))

			for _, status := range statuses {
				var conflicted *metav1.Condition
				for i := range status.Conditions {
					if status.Conditions[i].Type == string(gatewayv1.ListenerConditionConflicted) {
						conflicted = &status.Conditions[i]
					}
				}
				require.NotNil(t, conflicted, "listener %s must report a Conflicted condition", status.Name)
				assert.Equal(t, tc.wantStatus, conflicted.Status, "listener %s", status.Name)
				assert.Equal(t, string(tc.wantReason), conflicted.Reason, "listener %s", status.Name)
			}
		})
	}
}
