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

func newGatewayProxy(tlsVerify *bool, caBundle string) *v1alpha1.GatewayProxy {
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
					CaBundle:  caBundle,
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

func TestTranslateGatewayProxyToConfigCaBundle(t *testing.T) {
	t.Run("carries the CA bundle into the config", func(t *testing.T) {
		tr := &Translator{Log: logr.Discard()}
		tctx := provider.NewDefaultTranslateContext(context.Background())

		cfg, err := tr.TranslateGatewayProxyToConfig(tctx, newGatewayProxy(ptr.To(true), testCACert), false)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.True(t, cfg.TlsVerify)
		assert.Equal(t, testCACert, cfg.CaBundle)
	})

	t.Run("leaves the CA bundle empty when unset", func(t *testing.T) {
		tr := &Translator{Log: logr.Discard()}
		tctx := provider.NewDefaultTranslateContext(context.Background())

		cfg, err := tr.TranslateGatewayProxyToConfig(tctx, newGatewayProxy(ptr.To(true), ""), false)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Empty(t, cfg.CaBundle)
	})

	t.Run("rejects a CA bundle that is not PEM encoded", func(t *testing.T) {
		tr := &Translator{Log: logr.Discard()}
		tctx := provider.NewDefaultTranslateContext(context.Background())

		cfg, err := tr.TranslateGatewayProxyToConfig(tctx, newGatewayProxy(ptr.To(true), "not-a-certificate"), false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid caBundle")
		assert.Nil(t, cfg)
	})

	t.Run("still carries the CA bundle when verification is off", func(t *testing.T) {
		tr := &Translator{Log: logr.Discard()}
		tctx := provider.NewDefaultTranslateContext(context.Background())

		cfg, err := tr.TranslateGatewayProxyToConfig(tctx, newGatewayProxy(ptr.To(false), testCACert), false)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.False(t, cfg.TlsVerify)
		assert.Equal(t, testCACert, cfg.CaBundle)
	})
}
