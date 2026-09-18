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
	apiMeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	internaltypes "github.com/apache/apisix-ingress-controller/internal/types"
)

func TestHTTPRouteExtensionRefResolutionCondition(t *testing.T) {
	testRouteExtensionRefResolutionCondition(t, func(t *testing.T, cli client.Client, ref gatewayv1.LocalObjectReference) (*provider.TranslateContext, error) {
		tctx := provider.NewDefaultTranslateContext(context.Background())
		r := &HTTPRouteReconciler{Client: cli}
		route := &gatewayv1.HTTPRoute{
			ObjectMeta: metav1.ObjectMeta{Name: "route", Namespace: "default"},
			Spec: gatewayv1.HTTPRouteSpec{Rules: []gatewayv1.HTTPRouteRule{{
				Filters: []gatewayv1.HTTPRouteFilter{{
					Type:         gatewayv1.HTTPRouteFilterExtensionRef,
					ExtensionRef: &ref,
				}},
			}}},
		}
		return tctx, r.processHTTPRoute(tctx, route)
	})
}

func TestGRPCRouteExtensionRefResolutionCondition(t *testing.T) {
	testRouteExtensionRefResolutionCondition(t, func(t *testing.T, cli client.Client, ref gatewayv1.LocalObjectReference) (*provider.TranslateContext, error) {
		tctx := provider.NewDefaultTranslateContext(context.Background())
		r := &GRPCRouteReconciler{Client: cli}
		route := &gatewayv1.GRPCRoute{
			ObjectMeta: metav1.ObjectMeta{Name: "route", Namespace: "default"},
			Spec: gatewayv1.GRPCRouteSpec{Rules: []gatewayv1.GRPCRouteRule{{
				Filters: []gatewayv1.GRPCRouteFilter{{
					Type:         gatewayv1.GRPCRouteFilterExtensionRef,
					ExtensionRef: &ref,
				}},
			}}},
		}
		return tctx, r.processGRPCRoute(tctx, route)
	})
}

func testRouteExtensionRefResolutionCondition(
	t *testing.T,
	process func(*testing.T, client.Client, gatewayv1.LocalObjectReference) (*provider.TranslateContext, error),
) {
	t.Helper()

	tests := []struct {
		name         string
		ref          gatewayv1.LocalObjectReference
		objects      []client.Object
		wantReason   gatewayv1.RouteConditionReason
		wantResolved bool
		wantLoaded   bool
	}{
		{
			name: "supported reference",
			ref: gatewayv1.LocalObjectReference{
				Group: gatewayv1.Group(v1alpha1.GroupVersion.Group),
				Kind:  internaltypes.KindPluginConfig,
				Name:  "filter",
			},
			objects: []client.Object{&v1alpha1.PluginConfig{ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "filter",
			}}},
			wantReason:   gatewayv1.RouteReasonResolvedRefs,
			wantResolved: true,
			wantLoaded:   true,
		},
		{
			name: "unsupported group",
			ref: gatewayv1.LocalObjectReference{
				Group: "example.com",
				Kind:  internaltypes.KindPluginConfig,
				Name:  "filter",
			},
			objects: []client.Object{&v1alpha1.PluginConfig{ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "filter",
			}}},
			wantReason: gatewayv1.RouteReasonInvalidKind,
		},
		{
			name: "unsupported kind",
			ref: gatewayv1.LocalObjectReference{
				Group: gatewayv1.Group(v1alpha1.GroupVersion.Group),
				Kind:  "OtherFilter",
				Name:  "filter",
			},
			wantReason: gatewayv1.RouteReasonInvalidKind,
		},
		{
			name: "missing PluginConfig",
			ref: gatewayv1.LocalObjectReference{
				Group: gatewayv1.Group(v1alpha1.GroupVersion.Group),
				Kind:  internaltypes.KindPluginConfig,
				Name:  "missing",
			},
			wantReason: gatewayv1.RouteReasonBackendNotFound,
		},
		{
			name: "missing plugin Secret",
			ref: gatewayv1.LocalObjectReference{
				Group: gatewayv1.Group(v1alpha1.GroupVersion.Group),
				Kind:  internaltypes.KindPluginConfig,
				Name:  "filter",
			},
			objects: []client.Object{&v1alpha1.PluginConfig{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "filter"},
				Spec: v1alpha1.PluginConfigSpec{Plugins: []v1alpha1.Plugin{{
					Name:      "openid-connect",
					SecretRef: &corev1.LocalObjectReference{Name: "missing"},
				}}},
			}},
			wantReason: gatewayv1.RouteReasonBackendNotFound,
			wantLoaded: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			require.NoError(t, corev1.AddToScheme(scheme))
			require.NoError(t, v1alpha1.AddToScheme(scheme))
			cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.objects...).Build()

			tctx, err := process(t, cli, tt.ref)
			if tt.wantResolved {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}

			status := gatewayv1.RouteParentStatus{}
			SetRouteConditionResolvedRefs(&status, 1, err)
			condition := apiMeta.FindStatusCondition(status.Conditions, string(gatewayv1.RouteConditionResolvedRefs))
			require.NotNil(t, condition)
			assert.Equal(t, tt.wantReason, gatewayv1.RouteConditionReason(condition.Reason))
			assert.Equal(t, tt.wantResolved, condition.Status == metav1.ConditionTrue)

			_, loaded := tctx.PluginConfigs[k8stypes.NamespacedName{Namespace: "default", Name: string(tt.ref.Name)}]
			assert.Equal(t, tt.wantLoaded, loaded)
		})
	}
}
