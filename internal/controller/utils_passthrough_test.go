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
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// A TLS Passthrough listener cannot be programmed by APISIX, so it must report
// Accepted=False/UnsupportedValue with no attached routes, instead of silently
// claiming to be healthy while nothing is served.
func TestGetListenerStatus_TLSPassthroughUnsupported(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, gatewayv1.Install(scheme))
	cli := fake.NewClientBuilder().WithScheme(scheme).Build()

	passthrough := gatewayv1.TLSModePassthrough
	gateway := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "gw"},
		Spec: gatewayv1.GatewaySpec{
			Listeners: []gatewayv1.Listener{
				{
					Name:     "tls-passthrough",
					Port:     443,
					Protocol: gatewayv1.TLSProtocolType,
					Hostname: ptr.To(gatewayv1.Hostname("passthrough.example.com")),
					TLS:      &gatewayv1.ListenerTLSConfig{Mode: &passthrough},
				},
			},
		},
	}

	statuses, err := getListenerStatus(context.Background(), cli, gateway)
	require.NoError(t, err)
	require.Len(t, statuses, 1)

	accepted := findListenerCondition(statuses[0], gatewayv1.ListenerConditionAccepted)
	require.NotNil(t, accepted)
	assert.Equal(t, metav1.ConditionFalse, accepted.Status)
	assert.Equal(t, string(gatewayv1.ListenerReasonUnsupportedValue), accepted.Reason)

	programmed := findListenerCondition(statuses[0], gatewayv1.ListenerConditionProgrammed)
	require.NotNil(t, programmed)
	assert.Equal(t, metav1.ConditionFalse, programmed.Status)

	assert.Equal(t, int32(0), statuses[0].AttachedRoutes)
	assert.Empty(t, statuses[0].SupportedKinds)
}

func findListenerCondition(status gatewayv1.ListenerStatus, t gatewayv1.ListenerConditionType) *metav1.Condition {
	for i := range status.Conditions {
		if status.Conditions[i].Type == string(t) {
			return &status.Conditions[i]
		}
	}
	return nil
}
