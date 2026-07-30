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
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	internaltypes "github.com/apache/apisix-ingress-controller/internal/types"
)

func TestBuildPluginConfig_NonObjectConfigIsRejected(t *testing.T) {
	translator := NewTranslator(logr.Discard(), "")

	for _, raw := range []string{`["10.0.0.0/8"]`, `"whitelist"`, `42`} {
		plugin := apiv2.ApisixRoutePlugin{
			Name:   "ip-restriction",
			Enable: true,
			Config: apiextensionsv1.JSON{Raw: []byte(raw)},
		}
		config, err := translator.buildPluginConfig(plugin, "default", nil)
		assert.Error(t, err, "config %s must be rejected", raw)
		assert.ErrorContains(t, err, "ip-restriction")
		assert.Nil(t, config)
	}
}

func TestBuildPluginConfig_ValidConfigWithSecretRef(t *testing.T) {
	translator := NewTranslator(logr.Discard(), "")

	plugin := apiv2.ApisixRoutePlugin{
		Name:      "ip-restriction",
		Enable:    true,
		Config:    apiextensionsv1.JSON{Raw: []byte(`{"whitelist":["10.0.0.0/8"]}`)},
		SecretRef: "cred",
	}
	secrets := map[types.NamespacedName]*corev1.Secret{
		{Namespace: "default", Name: "cred"}: {
			Data: map[string][]byte{"message": []byte("denied")},
		},
	}
	config, err := translator.buildPluginConfig(plugin, "default", secrets)
	assert.NoError(t, err)
	assert.Equal(t, []any{"10.0.0.0/8"}, config["whitelist"])
	assert.Equal(t, "denied", config["message"])
}

func TestBuildPlugins_MalformedRoutePluginFailsTranslation(t *testing.T) {
	translator := NewTranslator(logr.Discard(), "")
	tctx := provider.NewDefaultTranslateContext(context.Background())

	ar := &apiv2.ApisixRoute{
		ObjectMeta: metav1.ObjectMeta{Name: "test-route", Namespace: "default"},
	}
	rule := apiv2.ApisixRouteHTTP{
		Name: "rule1",
		Plugins: []apiv2.ApisixRoutePlugin{{
			Name:   "ip-restriction",
			Enable: true,
			Config: apiextensionsv1.JSON{Raw: []byte(`["10.0.0.0/8"]`)},
		}},
	}

	plugins, err := translator.buildPlugins(tctx, ar, rule)
	assert.Error(t, err)
	assert.Nil(t, plugins)
}

func TestBuildPlugins_MalformedReferencedPluginConfigFailsTranslation(t *testing.T) {
	translator := NewTranslator(logr.Discard(), "")
	tctx := provider.NewDefaultTranslateContext(context.Background())
	tctx.ApisixPluginConfigs[types.NamespacedName{Namespace: "default", Name: "pc"}] = &apiv2.ApisixPluginConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "pc", Namespace: "default"},
		Spec: apiv2.ApisixPluginConfigSpec{
			Plugins: []apiv2.ApisixRoutePlugin{{
				Name:   "ip-restriction",
				Enable: true,
				Config: apiextensionsv1.JSON{Raw: []byte(`["10.0.0.0/8"]`)},
			}},
		},
	}

	ar := &apiv2.ApisixRoute{
		ObjectMeta: metav1.ObjectMeta{Name: "test-route", Namespace: "default"},
	}
	rule := apiv2.ApisixRouteHTTP{
		Name:             "rule1",
		PluginConfigName: "pc",
	}

	plugins, err := translator.buildPlugins(tctx, ar, rule)
	assert.Error(t, err)
	assert.Nil(t, plugins)
}

func TestTranslateStreamRule_MalformedPluginConfigFailsTranslation(t *testing.T) {
	translator := NewTranslator(logr.Discard(), "")
	tctx := provider.NewDefaultTranslateContext(context.Background())

	ar := &apiv2.ApisixRoute{
		ObjectMeta: metav1.ObjectMeta{Name: "test-route", Namespace: "default"},
	}
	part := apiv2.ApisixRouteStream{
		Name:     "stream1",
		Protocol: "TCP",
		Plugins: []apiv2.ApisixRoutePlugin{{
			Name:   "ip-restriction",
			Enable: true,
			Config: apiextensionsv1.JSON{Raw: []byte(`["10.0.0.0/8"]`)},
		}},
	}

	svc, err := translator.translateStreamRule(tctx, ar, part)
	assert.Error(t, err)
	assert.Nil(t, svc)
}

func TestTranslateApisixConsumer_MalformedPluginConfigFailsTranslation(t *testing.T) {
	translator := NewTranslator(logr.Discard(), "")
	tctx := provider.NewDefaultTranslateContext(context.Background())

	ac := &apiv2.ApisixConsumer{
		ObjectMeta: metav1.ObjectMeta{Name: "test-consumer", Namespace: "default"},
		Spec: apiv2.ApisixConsumerSpec{
			Plugins: []apiv2.ApisixRoutePlugin{{
				Name:   "ip-restriction",
				Enable: true,
				Config: apiextensionsv1.JSON{Raw: []byte(`["10.0.0.0/8"]`)},
			}},
		},
	}

	result, err := translator.TranslateApisixConsumer(tctx, ac)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestTranslateApisixGlobalRule_MalformedPluginConfigFailsTranslation(t *testing.T) {
	translator := NewTranslator(logr.Discard(), "")
	tctx := provider.NewDefaultTranslateContext(context.Background())

	obj := &apiv2.ApisixGlobalRule{
		ObjectMeta: metav1.ObjectMeta{Name: "test-global-rule", Namespace: "default"},
		Spec: apiv2.ApisixGlobalRuleSpec{
			Plugins: []apiv2.ApisixRoutePlugin{{
				Name:   "ip-restriction",
				Enable: true,
				Config: apiextensionsv1.JSON{Raw: []byte(`["10.0.0.0/8"]`)},
			}},
		},
	}

	result, err := translator.TranslateApisixGlobalRule(tctx, obj)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestLoadPluginConfigPluginsForIngress_MalformedPluginConfigFailsTranslation(t *testing.T) {
	translator := NewTranslator(logr.Discard(), "")
	tctx := provider.NewDefaultTranslateContext(context.Background())
	tctx.ApisixPluginConfigs[types.NamespacedName{Namespace: "default", Name: "pc"}] = &apiv2.ApisixPluginConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "pc", Namespace: "default"},
		Spec: apiv2.ApisixPluginConfigSpec{
			Plugins: []apiv2.ApisixRoutePlugin{{
				Name:   "ip-restriction",
				Enable: true,
				Config: apiextensionsv1.JSON{Raw: []byte(`["10.0.0.0/8"]`)},
			}},
		},
	}

	plugins, err := translator.loadPluginConfigPluginsForIngress(tctx, "default", "pc")
	assert.Error(t, err)
	assert.Nil(t, plugins)
}

func TestFillPluginFromExtensionRef_MalformedPluginConfigFailsTranslation(t *testing.T) {
	translator := NewTranslator(logr.Discard(), "")
	tctx := provider.NewDefaultTranslateContext(context.Background())
	tctx.PluginConfigs[types.NamespacedName{Namespace: "default", Name: "pc"}] = &v1alpha1.PluginConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "pc", Namespace: "default"},
		Spec: v1alpha1.PluginConfigSpec{
			Plugins: []v1alpha1.Plugin{{
				Name:   "ip-restriction",
				Config: apiextensionsv1.JSON{Raw: []byte(`["10.0.0.0/8"]`)},
			}},
		},
	}

	plugins := make(adctypes.Plugins)
	ref := &gatewayv1.LocalObjectReference{
		Kind: gatewayv1.Kind(internaltypes.KindPluginConfig),
		Name: "pc",
	}
	err := translator.fillPluginFromExtensionRef(plugins, "default", ref, tctx)
	assert.Error(t, err)
	assert.Empty(t, plugins)
}
