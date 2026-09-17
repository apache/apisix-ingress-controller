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
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/funcr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	internaltypes "github.com/apache/apisix-ingress-controller/internal/types"
)

const (
	extensionRefTestNamespace = "default"
	extensionRefTestBackend   = "backend"
	extensionRefTestPort      = int32(8080)
)

func newExtensionRefTranslateContext() *provider.TranslateContext {
	tctx := provider.NewDefaultTranslateContext(context.Background())
	key := types.NamespacedName{Namespace: extensionRefTestNamespace, Name: extensionRefTestBackend}
	tctx.Services[key] = &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: key.Name, Namespace: key.Namespace},
		Spec: corev1.ServiceSpec{Ports: []corev1.ServicePort{{
			Name: "http",
			Port: extensionRefTestPort,
		}}},
	}
	tctx.EndpointSlices[key] = []discoveryv1.EndpointSlice{{
		ObjectMeta: metav1.ObjectMeta{Name: "backend-1", Namespace: key.Namespace},
		Ports: []discoveryv1.EndpointPort{{
			Name: ptr.To("http"),
			Port: ptr.To(extensionRefTestPort),
		}},
		Endpoints: []discoveryv1.Endpoint{{
			Addresses:  []string{"10.0.0.1"},
			Conditions: discoveryv1.EndpointConditions{Ready: ptr.To(true)},
		}},
	}}
	return tctx
}

func extensionRefTestBackendRef() gatewayv1.BackendRef {
	return gatewayv1.BackendRef{BackendObjectReference: gatewayv1.BackendObjectReference{
		Name: gatewayv1.ObjectName(extensionRefTestBackend),
		Port: ptr.To(extensionRefTestPort),
	}}
}

func assertExtensionRefResponse(t *testing.T, plugins map[string]any) {
	t.Helper()
	fault, ok := plugins["fault-injection"].(map[string]any)
	require.True(t, ok)
	abort, ok := fault["abort"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 500, abort["http_status"])
}

func newExtensionRefTestLogger() (logr.Logger, *strings.Builder) {
	var logged strings.Builder
	logger := funcr.New(func(prefix, args string) {
		logged.WriteString(prefix)
		logged.WriteString(args)
	}, funcr.Options{Verbosity: 10})
	return logger, &logged
}

func assertExtensionRefDiagnostic(t *testing.T, logged string, routeKind string) {
	t.Helper()
	assert.Contains(t, logged, "failed to fill plugins from "+routeKind+" filters")
	assert.Contains(t, logged, `"namespace"="default"`)
	assert.Contains(t, logged, `"name"="route"`)
	assert.Contains(t, logged, `"ruleIndex"=0`)
}

func TestTranslateHTTPRouteUnresolvedExtensionRefIsScopedToRule(t *testing.T) {
	tests := []struct {
		name         string
		ref          gatewayv1.LocalObjectReference
		pluginConfig *v1alpha1.PluginConfig
	}{
		{
			name: "unsupported group",
			ref: gatewayv1.LocalObjectReference{
				Group: "example.com",
				Kind:  internaltypes.KindPluginConfig,
				Name:  "filter",
			},
			pluginConfig: &v1alpha1.PluginConfig{ObjectMeta: metav1.ObjectMeta{
				Namespace: extensionRefTestNamespace,
				Name:      "filter",
			}},
		},
		{
			name: "unsupported kind",
			ref: gatewayv1.LocalObjectReference{
				Group: gatewayv1.Group(v1alpha1.GroupVersion.Group),
				Kind:  "OtherFilter",
				Name:  "filter",
			},
		},
		{
			name: "missing PluginConfig",
			ref: gatewayv1.LocalObjectReference{
				Group: gatewayv1.Group(v1alpha1.GroupVersion.Group),
				Kind:  internaltypes.KindPluginConfig,
				Name:  "missing",
			},
		},
		{
			name: "PluginConfig cannot be rendered",
			ref: gatewayv1.LocalObjectReference{
				Group: gatewayv1.Group(v1alpha1.GroupVersion.Group),
				Kind:  internaltypes.KindPluginConfig,
				Name:  "filter",
			},
			pluginConfig: &v1alpha1.PluginConfig{
				ObjectMeta: metav1.ObjectMeta{Namespace: extensionRefTestNamespace, Name: "filter"},
				Spec: v1alpha1.PluginConfigSpec{Plugins: []v1alpha1.Plugin{{
					Name:   "ip-restriction",
					Config: apiextensionsv1.JSON{Raw: []byte(`["10.0.0.0/8"]`)},
				}}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tctx := newExtensionRefTranslateContext()
			logger, logged := newExtensionRefTestLogger()
			if tt.pluginConfig != nil {
				tctx.PluginConfigs[types.NamespacedName{
					Namespace: tt.pluginConfig.Namespace,
					Name:      tt.pluginConfig.Name,
				}] = tt.pluginConfig
			}

			route := &gatewayv1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{Name: "route", Namespace: extensionRefTestNamespace},
				Spec: gatewayv1.HTTPRouteSpec{Rules: []gatewayv1.HTTPRouteRule{
					{
						Filters: []gatewayv1.HTTPRouteFilter{{
							Type:         gatewayv1.HTTPRouteFilterExtensionRef,
							ExtensionRef: &tt.ref,
						}},
						BackendRefs: []gatewayv1.HTTPBackendRef{{BackendRef: extensionRefTestBackendRef()}},
					},
					{BackendRefs: []gatewayv1.HTTPBackendRef{{BackendRef: extensionRefTestBackendRef()}}},
				}},
			}

			result, err := NewTranslator(logger, "").TranslateHTTPRoute(tctx, route)
			require.NoError(t, err)
			require.Len(t, result.Services, 2)
			assertExtensionRefResponse(t, result.Services[0].Plugins)
			_, unaffectedRuleHasFault := result.Services[1].Plugins["fault-injection"]
			assert.False(t, unaffectedRuleHasFault)
			assertExtensionRefDiagnostic(t, logged.String(), "HTTPRoute")
			assert.NotContains(t, logged.String(), "10.0.0.0/8")
		})
	}
}

func TestTranslateGRPCRouteUnresolvedExtensionRefIsScopedToRule(t *testing.T) {
	tctx := newExtensionRefTranslateContext()
	logger, logged := newExtensionRefTestLogger()
	ref := gatewayv1.LocalObjectReference{
		Group: "example.com",
		Kind:  internaltypes.KindPluginConfig,
		Name:  "filter",
	}
	tctx.PluginConfigs[types.NamespacedName{Namespace: extensionRefTestNamespace, Name: "filter"}] =
		&v1alpha1.PluginConfig{ObjectMeta: metav1.ObjectMeta{Namespace: extensionRefTestNamespace, Name: "filter"}}

	route := &gatewayv1.GRPCRoute{
		ObjectMeta: metav1.ObjectMeta{Name: "route", Namespace: extensionRefTestNamespace},
		Spec: gatewayv1.GRPCRouteSpec{Rules: []gatewayv1.GRPCRouteRule{
			{
				Filters: []gatewayv1.GRPCRouteFilter{{
					Type:         gatewayv1.GRPCRouteFilterExtensionRef,
					ExtensionRef: &ref,
				}},
				BackendRefs: []gatewayv1.GRPCBackendRef{{BackendRef: extensionRefTestBackendRef()}},
			},
			{BackendRefs: []gatewayv1.GRPCBackendRef{{BackendRef: extensionRefTestBackendRef()}}},
		}},
	}

	result, err := NewTranslator(logger, "").TranslateGRPCRoute(tctx, route)
	require.NoError(t, err)
	require.Len(t, result.Services, 2)
	assertExtensionRefResponse(t, result.Services[0].Plugins)
	_, unaffectedRuleHasFault := result.Services[1].Plugins["fault-injection"]
	assert.False(t, unaffectedRuleHasFault)
	assertExtensionRefDiagnostic(t, logged.String(), "GRPCRoute")
}
