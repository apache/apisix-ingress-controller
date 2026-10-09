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
	"github.com/stretchr/testify/require"
	networkingv1 "k8s.io/api/networking/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/internal/utils"
)

func TestGatewayProxyPluginRenderingErrors(t *testing.T) {
	invalidConfigs := []struct {
		name      string
		configure func(*v1alpha1.GatewayProxy)
		errText   string
	}{
		{
			name: "plugin config",
			configure: func(gatewayProxy *v1alpha1.GatewayProxy) {
				gatewayProxy.Spec.Plugins = []v1alpha1.GatewayProxyPlugin{
					{
						Name:    "response-rewrite",
						Enabled: true,
						Config:  apiextensionsv1.JSON{Raw: []byte(`["not-an-object"]`)},
					},
				}
			},
			errText: `failed to unmarshal config of GatewayProxy plugin "response-rewrite"`,
		},
		{
			name: "plugin metadata",
			configure: func(gatewayProxy *v1alpha1.GatewayProxy) {
				gatewayProxy.Spec.PluginMetadata = map[string]apiextensionsv1.JSON{
					"key-auth": {Raw: []byte(`["not-an-object"]`)},
				}
			},
			errText: `failed to unmarshal GatewayProxy plugin metadata for "key-auth"`,
		},
	}

	callers := []struct {
		name      string
		translate func(*Translator, v1alpha1.GatewayProxy) (*TranslateResult, error)
	}{
		{
			name: "Gateway",
			translate: func(tr *Translator, gatewayProxy v1alpha1.GatewayProxy) (*TranslateResult, error) {
				tctx := provider.NewDefaultTranslateContext(context.Background())
				gateway := &gatewayv1.Gateway{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "gateway"}}
				tctx.GatewayProxies[utils.NamespacedNameKind(gateway)] = gatewayProxy
				return tr.TranslateGateway(tctx, gateway)
			},
		},
		{
			name: "IngressClass",
			translate: func(tr *Translator, gatewayProxy v1alpha1.GatewayProxy) (*TranslateResult, error) {
				tctx := provider.NewDefaultTranslateContext(context.Background())
				ingressClass := &networkingv1.IngressClass{ObjectMeta: metav1.ObjectMeta{Name: "apisix"}}
				tctx.GatewayProxies[utils.NamespacedNameKind(ingressClass)] = gatewayProxy
				return tr.TranslateIngressClass(tctx, ingressClass)
			},
		},
	}

	for _, caller := range callers {
		for _, invalidConfig := range invalidConfigs {
			t.Run(caller.name+"/"+invalidConfig.name, func(t *testing.T) {
				tr := &Translator{Log: logr.Discard()}
				gatewayProxy := v1alpha1.GatewayProxy{}
				invalidConfig.configure(&gatewayProxy)

				result, err := caller.translate(tr, gatewayProxy)
				require.ErrorContains(t, err, invalidConfig.errText)
				require.Nil(t, result)
			})
		}
	}
}
