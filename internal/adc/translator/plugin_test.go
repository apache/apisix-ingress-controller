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
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	internaltypes "github.com/apache/apisix-ingress-controller/internal/types"
)

func TestRenderPluginConfig_NoSecretRef(t *testing.T) {
	config, err := renderPluginConfig(v1alpha1.Plugin{
		Name:   "response-rewrite",
		Config: apiextensionsv1.JSON{Raw: []byte(`{"body":"hello"}`)},
	}, "default", nil)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"body": "hello"}, config)
}

func TestRenderPluginConfig_NullConfigBecomesEmptyObject(t *testing.T) {
	config, err := renderPluginConfig(v1alpha1.Plugin{
		Name:   "prometheus",
		Config: apiextensionsv1.JSON{Raw: []byte(`null`)},
	}, "default", nil)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{}, config)
}

func TestRenderPluginConfig_SecretKeysAreNestedAndWin(t *testing.T) {
	secrets := map[types.NamespacedName]*corev1.Secret{
		{Namespace: "default", Name: "oidc-credentials"}: {
			Data: map[string][]byte{
				"client_id":      []byte("my-client"),
				"client_secret":  []byte("s3cr3t"),
				"session.secret": []byte("8f2a"),
			},
		},
	}
	config, err := renderPluginConfig(v1alpha1.Plugin{
		Name:      "openid-connect",
		Config:    apiextensionsv1.JSON{Raw: []byte(`{"client_id":"placeholder","scope":"openid profile"}`)},
		SecretRef: &corev1.LocalObjectReference{Name: "oidc-credentials"},
	}, "default", secrets)
	require.NoError(t, err)
	assert.Equal(t, "my-client", config["client_id"])
	assert.Equal(t, "s3cr3t", config["client_secret"])
	assert.Equal(t, "openid profile", config["scope"])
	assert.Equal(t, map[string]any{"secret": "8f2a"}, config["session"])
}

func TestRenderPluginConfig_MissingSecretIsAnError(t *testing.T) {
	config, err := renderPluginConfig(v1alpha1.Plugin{
		Name:      "openid-connect",
		SecretRef: &corev1.LocalObjectReference{Name: "oidc-credentials"},
	}, "default", nil)
	assert.ErrorContains(t, err, "default/oidc-credentials")
	assert.Nil(t, config)
}

func TestRenderPluginConfig_MalformedConfigIsAnError(t *testing.T) {
	config, err := renderPluginConfig(v1alpha1.Plugin{
		Name:   "ip-restriction",
		Config: apiextensionsv1.JSON{Raw: []byte(`["10.0.0.0/8"]`)},
	}, "default", nil)
	assert.ErrorContains(t, err, "ip-restriction")
	assert.Nil(t, config)
}

func TestFillPluginFromExtensionRef_ResolvesSecretRef(t *testing.T) {
	translator := NewTranslator(logr.Discard(), "")
	tctx := provider.NewDefaultTranslateContext(context.Background())
	tctx.PluginConfigs[types.NamespacedName{Namespace: "default", Name: "oidc"}] = &v1alpha1.PluginConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "oidc", Namespace: "default"},
		Spec: v1alpha1.PluginConfigSpec{
			Plugins: []v1alpha1.Plugin{{
				Name:      "openid-connect",
				Config:    apiextensionsv1.JSON{Raw: []byte(`{"scope":"openid profile"}`)},
				SecretRef: &corev1.LocalObjectReference{Name: "oidc-credentials"},
			}},
		},
	}
	tctx.Secrets[types.NamespacedName{Namespace: "default", Name: "oidc-credentials"}] = &corev1.Secret{
		Data: map[string][]byte{"client_secret": []byte("s3cr3t")},
	}

	plugins := adctypes.Plugins{}
	translator.fillPluginFromExtensionRef(plugins, "default", &gatewayv1.LocalObjectReference{
		Kind: internaltypes.KindPluginConfig,
		Name: "oidc",
	}, tctx)

	assert.Equal(t, map[string]any{
		"scope":         "openid profile",
		"client_secret": "s3cr3t",
	}, plugins["openid-connect"])
}

func TestFillPluginFromExtensionRef_SkipsPluginWithMissingSecret(t *testing.T) {
	translator := NewTranslator(logr.Discard(), "")
	tctx := provider.NewDefaultTranslateContext(context.Background())
	tctx.PluginConfigs[types.NamespacedName{Namespace: "default", Name: "oidc"}] = &v1alpha1.PluginConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "oidc", Namespace: "default"},
		Spec: v1alpha1.PluginConfigSpec{
			Plugins: []v1alpha1.Plugin{{
				Name:      "openid-connect",
				SecretRef: &corev1.LocalObjectReference{Name: "oidc-credentials"},
			}, {
				Name:   "response-rewrite",
				Config: apiextensionsv1.JSON{Raw: []byte(`{"body":"hello"}`)},
			}},
		},
	}

	plugins := adctypes.Plugins{}
	translator.fillPluginFromExtensionRef(plugins, "default", &gatewayv1.LocalObjectReference{
		Kind: internaltypes.KindPluginConfig,
		Name: "oidc",
	}, tctx)

	// A plugin whose Secret is missing must not be programmed with a partial
	// configuration, but it must not hide the other plugins either.
	assert.NotContains(t, plugins, "openid-connect")
	assert.Equal(t, map[string]any{"body": "hello"}, plugins["response-rewrite"])
}
