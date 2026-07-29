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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/internal/provider"
)

const testCACert = `-----BEGIN CERTIFICATE-----
MIIBQzCB6qADAgECAgEBMAoGCCqGSM49BAMCMBIxEDAOBgNVBAMTB3Rlc3QtY2Ew
HhcNNzAwMTAxMDAwMDAwWhcNMzgwMTE5MDMxNDA4WjASMRAwDgYDVQQDEwd0ZXN0
LWNhMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEJo4AsM30ZHN+mYeHjqwceGBz
V2bMz1+OyNXuaPYVrSF7HShZhanOYNHb6QLNhjGxMsBDQHVLolPjyTQJp9R5GqMx
MC8wDgYDVR0PAQH/BAQDAgIEMB0GA1UdDgQWBBRzjh0YVmnpN/cFJziO0aYySuti
4DAKBggqhkjOPQQDAgNIADBFAiEA7fEGiQA7wX0LrrkRH4KplAPOgVV5Kvm/1dv1
3TLq9ssCIHKkv2dhydRvv36KC1WsRDcrl7W+7YmEnCS9PZfb8agM
-----END CERTIFICATE-----`

func newTLSGateway(frontendValidation *gatewayv1.FrontendTLSValidation) *gatewayv1.Gateway {
	return &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "gw",
		},
		Spec: gatewayv1.GatewaySpec{
			// In Gateway API v1.6 frontendValidation is declared at the Gateway level.
			TLS: &gatewayv1.GatewayTLSConfig{
				Frontend: &gatewayv1.FrontendTLSConfig{
					Default: gatewayv1.TLSConfig{
						Validation: frontendValidation,
					},
				},
			},
			Listeners: []gatewayv1.Listener{
				{
					Name:     "https",
					Protocol: gatewayv1.HTTPSProtocolType,
					Hostname: ptr.To(gatewayv1.Hostname("example.com")),
					TLS: &gatewayv1.ListenerTLSConfig{
						Mode: ptr.To(gatewayv1.TLSModeTerminate),
						CertificateRefs: []gatewayv1.SecretObjectReference{
							{
								Kind: ptr.To(gatewayv1.Kind("Secret")),
								Name: gatewayv1.ObjectName("server-cert"),
							},
						},
					},
				},
			},
		},
	}
}

// TestTranslateGateway_InsecureFallbackIsContained pins the blast radius of an
// unsupported frontendValidation mode: it must cost only the listener that asked
// for it, not the whole Gateway. A per-port override is the cheapest way for a
// user to trip this, and returning an error here would abort TranslateGateway and
// stop every SSL object, global rule and plugin metadata for the Gateway.
func TestTranslateGateway_InsecureFallbackIsContained(t *testing.T) {
	tr := &Translator{Log: logr.Discard()}
	gateway := newTLSGateway(nil)
	gateway.Spec.TLS.Frontend.PerPort = []gatewayv1.TLSPortConfig{
		{
			Port: gatewayv1.PortNumber(8443),
			TLS: gatewayv1.TLSConfig{
				Validation: &gatewayv1.FrontendTLSValidation{
					Mode: gatewayv1.AllowInsecureFallback,
					CACertificateRefs: []gatewayv1.ObjectReference{
						{Group: "", Kind: "ConfigMap", Name: "ca-cm"},
					},
				},
			},
		},
	}
	// A second HTTPS listener on the overridden port; the first one stays on 443.
	unsupported := *gateway.Spec.Listeners[0].DeepCopy()
	unsupported.Name = "https-8443"
	unsupported.Port = gatewayv1.PortNumber(8443)
	unsupported.Hostname = ptr.To(gatewayv1.Hostname("fallback.example.com"))
	gateway.Spec.Listeners = append(gateway.Spec.Listeners, unsupported)

	result, err := tr.TranslateGateway(newTranslateContextWithTLS(), gateway)
	require.NoError(t, err, "one unsupported listener must not fail the Gateway")
	require.NotNil(t, result)
	// Only the listener on 443 is programmed; the 8443 one contributes nothing.
	require.Len(t, result.SSL, 1)
	assert.Contains(t, result.SSL[0].Snis, "example.com")
	assert.NotContains(t, result.SSL[0].Snis, "fallback.example.com")
}

func newTranslateContextWithTLS() *provider.TranslateContext {
	tctx := provider.NewDefaultTranslateContext(context.Background())
	tctx.Secrets[types.NamespacedName{Namespace: "default", Name: "server-cert"}] = &corev1.Secret{
		Data: map[string][]byte{
			"cert": []byte("server-cert-data"),
			"key":  []byte("server-key-data"),
		},
	}
	tctx.ConfigMaps[types.NamespacedName{Namespace: "default", Name: "ca-cm"}] = &corev1.ConfigMap{
		Data: map[string]string{
			corev1.ServiceAccountRootCAKey: testCACert,
		},
	}
	tctx.Secrets[types.NamespacedName{Namespace: "default", Name: "ca-secret"}] = &corev1.Secret{
		Data: map[string][]byte{
			corev1.ServiceAccountRootCAKey: []byte(testCACert),
		},
	}
	return tctx
}

func TestTranslateSecret_Passthrough(t *testing.T) {
	// A TLS passthrough listener does not terminate TLS, so it carries no
	// certificateRefs; translating it must not error, otherwise the whole Gateway
	// is rejected with Accepted=False.
	tr := &Translator{Log: logr.Discard()}
	gateway := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "gw"},
		Spec: gatewayv1.GatewaySpec{
			Listeners: []gatewayv1.Listener{
				{
					Name:     "tls-passthrough",
					Protocol: gatewayv1.TLSProtocolType,
					Port:     443,
					TLS: &gatewayv1.ListenerTLSConfig{
						Mode: ptr.To(gatewayv1.TLSModePassthrough),
					},
				},
			},
		},
	}

	sslObjs, err := tr.translateSecret(newTranslateContextWithTLS(), gateway.Spec.Listeners[0], gateway)
	require.NoError(t, err)
	assert.Empty(t, sslObjs, "passthrough listener should not produce SSL objects")
}

func TestTranslateSecret_FrontendValidation(t *testing.T) {
	t.Run("with frontendValidation sets downstream mTLS client CA", func(t *testing.T) {
		tr := &Translator{Log: logr.Discard()}
		gateway := newTLSGateway(&gatewayv1.FrontendTLSValidation{
			CACertificateRefs: []gatewayv1.ObjectReference{
				{
					Group: "",
					Kind:  "ConfigMap",
					Name:  "ca-cm",
				},
			},
		})
		tctx := newTranslateContextWithTLS()

		sslObjs, err := tr.translateSecret(tctx, gateway.Spec.Listeners[0], gateway)
		require.NoError(t, err)
		require.Len(t, sslObjs, 1)
		require.NotNil(t, sslObjs[0].Client, "client mTLS config should be set")
		assert.Equal(t, testCACert, sslObjs[0].Client.CA)
		assert.Equal(t, []string{"example.com"}, sslObjs[0].Snis)
	})

	t.Run("with Secret CA ref sets downstream mTLS client CA", func(t *testing.T) {
		tr := &Translator{Log: logr.Discard()}
		gateway := newTLSGateway(&gatewayv1.FrontendTLSValidation{
			CACertificateRefs: []gatewayv1.ObjectReference{
				{Group: "", Kind: "Secret", Name: "ca-secret"},
			},
		})
		tctx := newTranslateContextWithTLS()

		sslObjs, err := tr.translateSecret(tctx, gateway.Spec.Listeners[0], gateway)
		require.NoError(t, err)
		require.Len(t, sslObjs, 1)
		require.NotNil(t, sslObjs[0].Client, "client mTLS config should be set")
		assert.Equal(t, testCACert, sslObjs[0].Client.CA)
	})

	t.Run("missing CA Secret returns error", func(t *testing.T) {
		tr := &Translator{Log: logr.Discard()}
		gateway := newTLSGateway(&gatewayv1.FrontendTLSValidation{
			CACertificateRefs: []gatewayv1.ObjectReference{
				{Kind: "Secret", Name: "missing"},
			},
		})
		tctx := newTranslateContextWithTLS()

		_, err := tr.translateSecret(tctx, gateway.Spec.Listeners[0], gateway)
		require.Error(t, err)
	})

	t.Run("without frontendValidation leaves client nil", func(t *testing.T) {
		tr := &Translator{Log: logr.Discard()}
		gateway := newTLSGateway(nil)
		tctx := newTranslateContextWithTLS()

		sslObjs, err := tr.translateSecret(tctx, gateway.Spec.Listeners[0], gateway)
		require.NoError(t, err)
		require.Len(t, sslObjs, 1)
		assert.Nil(t, sslObjs[0].Client)
	})

	t.Run("AllowInsecureFallback emits no SSL for that listener", func(t *testing.T) {
		// The mode cannot be expressed on APISIX, so the listener is reported
		// Accepted=False/UnsupportedValue and programmed with nothing. Serving TLS
		// without the requested client validation would be less safe than serving
		// nothing, and failing here would take down the whole Gateway translation.
		tr := &Translator{Log: logr.Discard()}
		gateway := newTLSGateway(&gatewayv1.FrontendTLSValidation{
			Mode: gatewayv1.AllowInsecureFallback,
			CACertificateRefs: []gatewayv1.ObjectReference{
				{Group: "", Kind: "ConfigMap", Name: "ca-cm"},
			},
		})
		tctx := newTranslateContextWithTLS()

		sslObjs, err := tr.translateSecret(tctx, gateway.Spec.Listeners[0], gateway)
		require.NoError(t, err)
		assert.Empty(t, sslObjs)
	})

	t.Run("frontendValidation is ignored on a non-HTTPS listener", func(t *testing.T) {
		// spec.tls.frontend applies only to HTTPS listeners in Gateway API v1.6, so a
		// TLS (Terminate) listener must not get downstream mTLS attached.
		tr := &Translator{Log: logr.Discard()}
		gateway := newTLSGateway(&gatewayv1.FrontendTLSValidation{
			CACertificateRefs: []gatewayv1.ObjectReference{
				{Group: "", Kind: "ConfigMap", Name: "ca-cm"},
			},
		})
		gateway.Spec.Listeners[0].Protocol = gatewayv1.TLSProtocolType
		tctx := newTranslateContextWithTLS()

		sslObjs, err := tr.translateSecret(tctx, gateway.Spec.Listeners[0], gateway)
		require.NoError(t, err)
		require.Len(t, sslObjs, 1)
		assert.Nil(t, sslObjs[0].Client, "client mTLS must not be set for a non-HTTPS listener")
	})

	t.Run("missing CA ConfigMap returns error", func(t *testing.T) {
		tr := &Translator{Log: logr.Discard()}
		gateway := newTLSGateway(&gatewayv1.FrontendTLSValidation{
			CACertificateRefs: []gatewayv1.ObjectReference{
				{Kind: "ConfigMap", Name: "missing"},
			},
		})
		tctx := newTranslateContextWithTLS()

		_, err := tr.translateSecret(tctx, gateway.Spec.Listeners[0], gateway)
		require.Error(t, err)
	})

	t.Run("unsupported CA ref kind returns error", func(t *testing.T) {
		tr := &Translator{Log: logr.Discard()}
		gateway := newTLSGateway(&gatewayv1.FrontendTLSValidation{
			CACertificateRefs: []gatewayv1.ObjectReference{
				{Kind: "Pod", Name: "ca-cm"},
			},
		})
		tctx := newTranslateContextWithTLS()

		_, err := tr.translateSecret(tctx, gateway.Spec.Listeners[0], gateway)
		require.Error(t, err)
	})

	t.Run("unsupported CA ref group returns error", func(t *testing.T) {
		tr := &Translator{Log: logr.Discard()}
		gateway := newTLSGateway(&gatewayv1.FrontendTLSValidation{
			CACertificateRefs: []gatewayv1.ObjectReference{
				{Group: "example.com", Kind: "ConfigMap", Name: "ca-cm"},
			},
		})
		tctx := newTranslateContextWithTLS()

		_, err := tr.translateSecret(tctx, gateway.Spec.Listeners[0], gateway)
		require.Error(t, err)
	})

	t.Run("malformed CA data returns error", func(t *testing.T) {
		tr := &Translator{Log: logr.Discard()}
		gateway := newTLSGateway(&gatewayv1.FrontendTLSValidation{
			CACertificateRefs: []gatewayv1.ObjectReference{
				{Kind: "ConfigMap", Name: "ca-cm"},
			},
		})
		tctx := newTranslateContextWithTLS()
		tctx.ConfigMaps[types.NamespacedName{Namespace: "default", Name: "ca-cm"}] = &corev1.ConfigMap{
			Data: map[string]string{corev1.ServiceAccountRootCAKey: "   not a pem cert   "},
		}

		_, err := tr.translateSecret(tctx, gateway.Spec.Listeners[0], gateway)
		require.Error(t, err)
	})
}
