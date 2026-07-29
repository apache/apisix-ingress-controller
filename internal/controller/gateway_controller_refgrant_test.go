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

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	"sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/apache/apisix-ingress-controller/internal/provider"
)

// gatewayWithRefs builds a Gateway whose HTTPS listener references a server
// certificate Secret and a frontendValidation CA ConfigMap, both in refNamespace.
func gatewayWithRefs(refNamespace string) *gatewayv1.Gateway {
	return &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "gw"},
		Spec: gatewayv1.GatewaySpec{
			Listeners: []gatewayv1.Listener{
				{
					Name:     "https",
					Port:     443,
					Protocol: gatewayv1.HTTPSProtocolType,
					TLS: &gatewayv1.ListenerTLSConfig{
						Mode: ptr.To(gatewayv1.TLSModeTerminate),
						CertificateRefs: []gatewayv1.SecretObjectReference{{
							Kind:      ptr.To(gatewayv1.Kind(KindSecret)),
							Name:      "cert",
							Namespace: ptr.To(gatewayv1.Namespace(refNamespace)),
						}},
					},
				},
			},
			TLS: &gatewayv1.GatewayTLSConfig{
				Frontend: &gatewayv1.FrontendTLSConfig{
					Default: gatewayv1.TLSConfig{
						Validation: &gatewayv1.FrontendTLSValidation{
							CACertificateRefs: []gatewayv1.ObjectReference{{
								Group:     "",
								Kind:      KindConfigMap,
								Name:      "ca",
								Namespace: ptr.To(gatewayv1.Namespace(refNamespace)),
							}},
						},
					},
				},
			},
		},
	}
}

// referenceGrant permits a Gateway in "default" to reference Secrets and
// ConfigMaps in grantNamespace.
func referenceGrant(grantNamespace string) *v1beta1.ReferenceGrant {
	return &v1beta1.ReferenceGrant{
		ObjectMeta: metav1.ObjectMeta{Namespace: grantNamespace, Name: "grant"},
		Spec: v1beta1.ReferenceGrantSpec{
			From: []v1beta1.ReferenceGrantFrom{{
				Group:     gatewayv1.GroupName,
				Kind:      KindGateway,
				Namespace: "default",
			}},
			To: []v1beta1.ReferenceGrantTo{
				{Group: "", Kind: KindSecret},
				{Group: "", Kind: KindConfigMap},
			},
		},
	}
}

// processListenerConfig must not load a cross-namespace certificate Secret or CA
// ConfigMap into the data plane unless a ReferenceGrant permits it: otherwise the
// listener would report RefNotPermitted in status yet still be programmed with a
// cert/CA the target namespace never shared.
func TestProcessListenerConfig_CrossNamespaceReferenceGrant(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, gatewayv1.Install(scheme))
	require.NoError(t, v1beta1.Install(scheme))

	certSecret := func(ns string) *corev1.Secret {
		return &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: "cert"}}
	}
	caConfigMap := func(ns string) *corev1.ConfigMap {
		return &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: "ca"}}
	}
	certNN := k8stypes.NamespacedName{Namespace: "other", Name: "cert"}
	caNN := k8stypes.NamespacedName{Namespace: "other", Name: "ca"}

	// The gating only prunes cross-namespace refs; keep ReferenceGrant enabled so the
	// grant path is exercised too.
	SetEnableReferenceGrant(true)
	defer SetEnableReferenceGrant(false)

	run := func(objs ...client.Object) *provider.TranslateContext {
		cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
		r := &GatewayReconciler{Client: cli, Log: logr.Discard()}
		tctx := provider.NewDefaultTranslateContext(context.Background())
		r.processListenerConfig(tctx, gatewayWithRefs(objs[0].GetNamespace()))
		return tctx
	}

	t.Run("cross-namespace refs without a grant are pruned", func(t *testing.T) {
		tctx := run(certSecret("other"), caConfigMap("other"))
		assert.NotContains(t, tctx.Secrets, certNN, "unpermitted cross-ns cert must not reach the data plane")
		assert.NotContains(t, tctx.ConfigMaps, caNN, "unpermitted cross-ns CA must not reach the data plane")
	})

	t.Run("cross-namespace refs with a grant are loaded", func(t *testing.T) {
		tctx := run(certSecret("other"), caConfigMap("other"), referenceGrant("other"))
		assert.Contains(t, tctx.Secrets, certNN, "permitted cross-ns cert must be programmed")
		assert.Contains(t, tctx.ConfigMaps, caNN, "permitted cross-ns CA must be programmed")
	})

	t.Run("same-namespace refs need no grant", func(t *testing.T) {
		tctx := run(certSecret("default"), caConfigMap("default"))
		assert.Contains(t, tctx.Secrets, k8stypes.NamespacedName{Namespace: "default", Name: "cert"})
		assert.Contains(t, tctx.ConfigMaps, k8stypes.NamespacedName{Namespace: "default", Name: "ca"})
	})
}
