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

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/provider"
)

func newGatewayProxy(tlsVerify *bool, caCert string) *v1alpha1.GatewayProxy {
	// an empty string stands for the field being unset
	var caCertRef *v1alpha1.ControlPlaneCaCert
	if caCert != "" {
		caCertRef = &v1alpha1.ControlPlaneCaCert{Value: caCert}
	}
	return &v1alpha1.GatewayProxy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "gp",
		},
		Spec: v1alpha1.GatewayProxySpec{
			Provider: &v1alpha1.GatewayProxyProvider{
				Type: v1alpha1.ProviderTypeControlPlane,
				ControlPlane: &v1alpha1.ControlPlaneProvider{
					Endpoints: []string{"https://cp.example.com:9180"},
					TlsVerify: tlsVerify,
					CaCert:    caCertRef,
					Auth: v1alpha1.ControlPlaneAuth{
						Type: v1alpha1.AuthTypeAdminKey,
						AdminKey: &v1alpha1.AdminKeyAuth{
							Value: "admin-key",
						},
					},
				},
			},
		},
	}
}

func TestTranslateGatewayProxyToConfigCaCert(t *testing.T) {
	t.Run("carries the CA certificate into the config", func(t *testing.T) {
		tr := &Translator{Log: logr.Discard()}
		tctx := provider.NewDefaultTranslateContext(context.Background())

		cfg, err := tr.TranslateGatewayProxyToConfig(tctx, newGatewayProxy(ptr.To(true), testCACert), false)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.True(t, cfg.TlsVerify)
		assert.Equal(t, testCACert, cfg.CaCert)
	})

	t.Run("leaves the CA certificate empty when unset", func(t *testing.T) {
		tr := &Translator{Log: logr.Discard()}
		tctx := provider.NewDefaultTranslateContext(context.Background())

		cfg, err := tr.TranslateGatewayProxyToConfig(tctx, newGatewayProxy(ptr.To(true), ""), false)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Empty(t, cfg.CaCert)
	})

	// every certificate is parsed: x509.CertPool silently skips the blocks it
	// cannot decode, which would let a broken one through to the ADC server.
	for name, caCert := range map[string]string{
		"not PEM at all":                  "not-a-certificate",
		"a header with no certificate":    "-----BEGIN CERTIFICATE-----",
		"an unparseable body":             "-----BEGIN CERTIFICATE-----\nAAAA\n-----END CERTIFICATE-----",
		"a key rather than a certificate": "-----BEGIN RSA PRIVATE KEY-----\nAAAA\n-----END RSA PRIVATE KEY-----",
		"one good and one broken certificate": testCACert +
			"\n-----BEGIN CERTIFICATE-----\nAAAA\n-----END CERTIFICATE-----",
	} {
		t.Run("rejects a CA certificate that is "+name, func(t *testing.T) {
			tr := &Translator{Log: logr.Discard()}
			tctx := provider.NewDefaultTranslateContext(context.Background())

			cfg, err := tr.TranslateGatewayProxyToConfig(tctx, newGatewayProxy(ptr.To(true), caCert), false)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid caCert")
			assert.Nil(t, cfg)
		})
	}

	t.Run("accepts a bundle of several certificates", func(t *testing.T) {
		tr := &Translator{Log: logr.Discard()}
		tctx := provider.NewDefaultTranslateContext(context.Background())

		bundle := testCACert + "\n" + testCACert
		cfg, err := tr.TranslateGatewayProxyToConfig(tctx, newGatewayProxy(ptr.To(true), bundle), false)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, bundle, cfg.CaCert)
	})

	t.Run("still carries the CA certificate when verification is off", func(t *testing.T) {
		tr := &Translator{Log: logr.Discard()}
		tctx := provider.NewDefaultTranslateContext(context.Background())

		cfg, err := tr.TranslateGatewayProxyToConfig(tctx, newGatewayProxy(ptr.To(false), testCACert), false)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.False(t, cfg.TlsVerify)
		assert.Equal(t, testCACert, cfg.CaCert)
	})
}
