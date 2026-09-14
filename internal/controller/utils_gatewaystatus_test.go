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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/internal/controller/indexer"
)

// A listener whose protocol the implementation does not serve must say so with
// UnsupportedProtocol and advertise no route kinds, rather than be accepted and
// then quietly serve nothing.
func TestGetListenerStatus_UnsupportedProtocol(t *testing.T) {
	scheme := parentRefTestScheme(t)

	for _, tc := range []struct {
		name       string
		listener   gatewayv1.Listener
		wantStatus metav1.ConditionStatus
		wantReason gatewayv1.ListenerConditionReason
		wantKinds  int
	}{
		{
			name: "unknown protocol",
			listener: gatewayv1.Listener{
				Name: "invalid", Port: 1111, Protocol: gatewayv1.ProtocolType("INVALID"),
			},
			wantStatus: metav1.ConditionFalse,
			wantReason: gatewayv1.ListenerReasonUnsupportedProtocol,
			wantKinds:  0,
		},
		{
			name: "known protocol",
			listener: gatewayv1.Listener{
				Name: "http", Port: 80, Protocol: gatewayv1.HTTPProtocolType,
			},
			wantStatus: metav1.ConditionTrue,
			wantReason: gatewayv1.ListenerReasonAccepted,
			wantKinds:  2,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gw := &gatewayv1.Gateway{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "gw"},
				Spec: gatewayv1.GatewaySpec{
					GatewayClassName: "apisix",
					Listeners:        []gatewayv1.Listener{tc.listener},
				},
			}
			cli := fake.NewClientBuilder().WithScheme(scheme).
				WithObjects(newParentRefGatewayClass(), gw).
				// An HTTP listener counts both route kinds it can serve, so both
				// indexes have to exist even though no route does.
				WithIndex(&gatewayv1.HTTPRoute{}, indexer.ParentRefs,
					func(client.Object) []string { return nil }).
				WithIndex(&gatewayv1.GRPCRoute{}, indexer.ParentRefs,
					func(client.Object) []string { return nil }).
				Build()

			statuses, err := getListenerStatus(context.Background(), cli, gw)
			require.NoError(t, err)
			require.Len(t, statuses, 1)

			var accepted *metav1.Condition
			for i := range statuses[0].Conditions {
				if statuses[0].Conditions[i].Type == string(gatewayv1.ListenerConditionAccepted) {
					accepted = &statuses[0].Conditions[i]
				}
			}
			require.NotNil(t, accepted, "listener must report an Accepted condition")
			assert.Equal(t, tc.wantStatus, accepted.Status)
			assert.Equal(t, string(tc.wantReason), accepted.Reason)
			assert.Len(t, statuses[0].SupportedKinds, tc.wantKinds)
			assert.Zero(t, statuses[0].AttachedRoutes)
		})
	}
}

// The Gateway's own Accepted condition follows its listeners: one bad listener
// is enough for ListenersNotValid, and the status separates "some listeners
// work" from "none do".
func TestGatewayAcceptanceFromListeners(t *testing.T) {
	listener := func(name string, accepted bool) gatewayv1.ListenerStatus {
		status := metav1.ConditionTrue
		if !accepted {
			status = metav1.ConditionFalse
		}
		return gatewayv1.ListenerStatus{
			Name: gatewayv1.SectionName(name),
			Conditions: []metav1.Condition{{
				Type:   string(gatewayv1.ListenerConditionAccepted),
				Status: status,
			}},
		}
	}

	for _, tc := range []struct {
		name        string
		listeners   []gatewayv1.ListenerStatus
		wantInvalid bool
		wantStatus  bool
	}{
		{
			name:        "all listeners accepted",
			listeners:   []gatewayv1.ListenerStatus{listener("http", true), listener("https", true)},
			wantInvalid: false,
		},
		{
			name:        "one of two listeners rejected",
			listeners:   []gatewayv1.ListenerStatus{listener("http", true), listener("invalid", false)},
			wantInvalid: true,
			wantStatus:  true,
		},
		{
			name:        "every listener rejected",
			listeners:   []gatewayv1.ListenerStatus{listener("invalid", false)},
			wantInvalid: true,
			wantStatus:  false,
		},
		{
			name:        "no listeners",
			listeners:   nil,
			wantInvalid: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, invalid := gatewayAcceptanceFromListeners(tc.listeners)
			assert.Equal(t, tc.wantInvalid, invalid)
			if invalid {
				assert.Equal(t, tc.wantStatus, status)
			}
		})
	}
}

// Accepted=False used to report Reason=Accepted, which says nothing about what
// went wrong.
func TestSetGatewayConditionAccepted_Reason(t *testing.T) {
	gw := &gatewayv1.Gateway{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "gw"}}

	SetGatewayConditionAccepted(gw, false, gatewayv1.GatewayReasonInvalidParameters, "parametersRef is not resolvable")

	require.Len(t, gw.Status.Conditions, 1)
	assert.Equal(t, string(gatewayv1.GatewayConditionAccepted), gw.Status.Conditions[0].Type)
	assert.Equal(t, metav1.ConditionFalse, gw.Status.Conditions[0].Status)
	assert.Equal(t, string(gatewayv1.GatewayReasonInvalidParameters), gw.Status.Conditions[0].Reason)
}
