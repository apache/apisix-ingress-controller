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
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const frontendCACert = `-----BEGIN CERTIFICATE-----
MIIBQzCB6qADAgECAgEBMAoGCCqGSM49BAMCMBIxEDAOBgNVBAMTB3Rlc3QtY2Ew
HhcNNzAwMTAxMDAwMDAwWhcNMzgwMTE5MDMxNDA4WjASMRAwDgYDVQQDEwd0ZXN0
LWNhMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEJo4AsM30ZHN+mYeHjqwceGBz
V2bMz1+OyNXuaPYVrSF7HShZhanOYNHb6QLNhjGxMsBDQHVLolPjyTQJp9R5GqMx
MC8wDgYDVR0PAQH/BAQDAgIEMB0GA1UdDgQWBBRzjh0YVmnpN/cFJziO0aYySuti
4DAKBggqhkjOPQQDAgNIADBFAiEA7fEGiQA7wX0LrrkRH4KplAPOgVV5Kvm/1dv1
3TLq9ssCIHKkv2dhydRvv36KC1WsRDcrl7W+7YmEnCS9PZfb8agM
-----END CERTIFICATE-----`

func newFrontendConditions() (resolvedRefs, programmed, accepted metav1.Condition) {
	mk := func(t gatewayv1.ListenerConditionType, r gatewayv1.ListenerConditionReason) metav1.Condition {
		return metav1.Condition{Type: string(t), Status: metav1.ConditionTrue, Reason: string(r)}
	}
	return mk(gatewayv1.ListenerConditionResolvedRefs, gatewayv1.ListenerReasonResolvedRefs),
		mk(gatewayv1.ListenerConditionProgrammed, gatewayv1.ListenerReasonProgrammed),
		mk(gatewayv1.ListenerConditionAccepted, gatewayv1.ListenerReasonAccepted)
}

// TestValidateListenerFrontendValidation checks the listener conditions produced
// for Gateway API v1.6 frontend client-cert CA validation.
func TestValidateListenerFrontendValidation(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, gatewayv1.Install(scheme))

	gateway := &gatewayv1.Gateway{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "gw"}}
	ref := func(kind, name string) gatewayv1.ObjectReference {
		return gatewayv1.ObjectReference{Group: "", Kind: gatewayv1.Kind(kind), Name: gatewayv1.ObjectName(name)}
	}

	t.Run("missing CA ConfigMap: ResolvedRefs and Accepted both False", func(t *testing.T) {
		cli := fake.NewClientBuilder().WithScheme(scheme).Build()
		resolvedRefs, programmed, accepted := newFrontendConditions()
		validateListenerFrontendValidation(context.Background(), cli, gateway,
			&gatewayv1.FrontendTLSValidation{CACertificateRefs: []gatewayv1.ObjectReference{ref("ConfigMap", "missing")}},
			&resolvedRefs, &programmed, &accepted)

		assert.Equal(t, metav1.ConditionFalse, resolvedRefs.Status)
		assert.Equal(t, string(gatewayv1.ListenerReasonInvalidCACertificateRef), resolvedRefs.Reason)
		assert.Equal(t, metav1.ConditionFalse, accepted.Status)
		assert.Equal(t, string(gatewayv1.ListenerReasonNoValidCACertificate), accepted.Reason)
	})

	t.Run("unsupported Kind: InvalidCACertificateKind", func(t *testing.T) {
		cli := fake.NewClientBuilder().WithScheme(scheme).Build()
		resolvedRefs, programmed, accepted := newFrontendConditions()
		validateListenerFrontendValidation(context.Background(), cli, gateway,
			&gatewayv1.FrontendTLSValidation{CACertificateRefs: []gatewayv1.ObjectReference{ref("Pod", "x")}},
			&resolvedRefs, &programmed, &accepted)

		assert.Equal(t, string(gatewayv1.ListenerReasonInvalidCACertificateKind), resolvedRefs.Reason)
		assert.Equal(t, metav1.ConditionFalse, accepted.Status)
		assert.Equal(t, string(gatewayv1.ListenerReasonNoValidCACertificate), accepted.Reason)
	})

	t.Run("AllowInsecureFallback mode is not programmable", func(t *testing.T) {
		cli := fake.NewClientBuilder().WithScheme(scheme).Build()
		resolvedRefs, programmed, accepted := newFrontendConditions()
		validateListenerFrontendValidation(context.Background(), cli, gateway,
			&gatewayv1.FrontendTLSValidation{
				Mode:              gatewayv1.AllowInsecureFallback,
				CACertificateRefs: []gatewayv1.ObjectReference{ref("ConfigMap", "ca")},
			},
			&resolvedRefs, &programmed, &accepted)

		assert.Equal(t, metav1.ConditionFalse, programmed.Status)
		assert.Contains(t, programmed.Message, "AllowInsecureFallback")
	})

	t.Run("one valid ref keeps Accepted True while ResolvedRefs stays False", func(t *testing.T) {
		validCM := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "ca"},
			Data:       map[string]string{corev1.ServiceAccountRootCAKey: frontendCACert},
		}
		cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(validCM).Build()
		resolvedRefs, programmed, accepted := newFrontendConditions()
		validateListenerFrontendValidation(context.Background(), cli, gateway,
			&gatewayv1.FrontendTLSValidation{CACertificateRefs: []gatewayv1.ObjectReference{
				ref("ConfigMap", "ca"),
				ref("ConfigMap", "missing"),
			}},
			&resolvedRefs, &programmed, &accepted)

		assert.Equal(t, metav1.ConditionFalse, resolvedRefs.Status, "an invalid ref still fails ResolvedRefs")
		assert.Equal(t, metav1.ConditionTrue, accepted.Status, "Accepted stays True while at least one CA is valid")
	})
}
