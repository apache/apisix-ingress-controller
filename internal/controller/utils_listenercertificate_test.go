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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/controller/indexer"
	"github.com/apache/apisix-ingress-controller/internal/id"
)

// TestGetListenerStatus_RejectedCertificate covers a certificate that passes every check
// the controller can make, yet is rejected by the data plane.
func TestGetListenerStatus_RejectedCertificate(t *testing.T) {
	scheme := parentRefTestScheme(t)
	require.NoError(t, corev1.AddToScheme(scheme))
	// Well-formed PEM framing is all the controller checks; the content is garbage the
	// data plane would reject.
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "cert"},
		Data: map[string][]byte{
			"tls.crt": []byte("-----BEGIN CERTIFICATE-----\nAAAA\n-----END CERTIFICATE-----\n"),
			"tls.key": []byte("-----BEGIN PRIVATE KEY-----\nAAAA\n-----END PRIVATE KEY-----\n"),
		},
	}
	httpsListener := func(name string) gatewayv1.Listener {
		return gatewayv1.Listener{
			Name:     gatewayv1.SectionName(name),
			Port:     443,
			Protocol: gatewayv1.HTTPSProtocolType,
			TLS: &gatewayv1.ListenerTLSConfig{
				CertificateRefs: []gatewayv1.SecretObjectReference{{Name: "cert"}},
			},
		}
	}
	gw := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "gw"},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: "apisix",
			Listeners:        []gatewayv1.Listener{httpsListener("rejected"), httpsListener("accepted")},
		},
	}
	cli := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(newParentRefGatewayClass(), gw, secret).
		WithIndex(&gatewayv1.HTTPRoute{}, indexer.ParentRefs, func(client.Object) []string { return nil }).
		WithIndex(&gatewayv1.GRPCRoute{}, indexer.ParentRefs, func(client.Object) []string { return nil }).
		Build()

	rejectedID := id.GenID(adctypes.ComposeGatewayListenerSSLName(KindGateway, "default", "gw", "rejected", 0))
	statuses, err := getListenerStatus(context.Background(), cli, gw, map[string]string{rejectedID: "invalid certificate"})
	require.NoError(t, err)
	require.Len(t, statuses, 2)

	conditionOf := func(status gatewayv1.ListenerStatus, conditionType gatewayv1.ListenerConditionType) metav1.Condition {
		for _, c := range status.Conditions {
			if c.Type == string(conditionType) {
				return c
			}
		}
		t.Fatalf("listener %s has no %s condition", status.Name, conditionType)
		return metav1.Condition{}
	}

	resolvedRefs := conditionOf(statuses[0], gatewayv1.ListenerConditionResolvedRefs)
	assert.Equal(t, metav1.ConditionFalse, resolvedRefs.Status)
	assert.Equal(t, string(gatewayv1.ListenerReasonInvalidCertificateRef), resolvedRefs.Reason)
	assert.Equal(t, "invalid certificate", resolvedRefs.Message)
	programmed := conditionOf(statuses[0], gatewayv1.ListenerConditionProgrammed)
	assert.Equal(t, metav1.ConditionFalse, programmed.Status)
	assert.Equal(t, string(gatewayv1.ListenerReasonInvalid), programmed.Reason)

	assert.Equal(t, metav1.ConditionTrue, conditionOf(statuses[1], gatewayv1.ListenerConditionResolvedRefs).Status)
	assert.Equal(t, metav1.ConditionTrue, conditionOf(statuses[1], gatewayv1.ListenerConditionProgrammed).Status)
}
