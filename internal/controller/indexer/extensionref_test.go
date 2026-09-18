// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package indexer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	internaltypes "github.com/apache/apisix-ingress-controller/internal/types"
)

func TestRouteExtensionRefIndexFuncUsesSupportedGroupAndKind(t *testing.T) {
	supported := gatewayv1.LocalObjectReference{
		Group: gatewayv1.Group(v1alpha1.GroupVersion.Group),
		Kind:  internaltypes.KindPluginConfig,
		Name:  "supported",
	}
	wrongGroup := gatewayv1.LocalObjectReference{
		Group: "example.com",
		Kind:  internaltypes.KindPluginConfig,
		Name:  "wrong-group",
	}
	wrongKind := gatewayv1.LocalObjectReference{
		Group: gatewayv1.Group(v1alpha1.GroupVersion.Group),
		Kind:  "OtherFilter",
		Name:  "wrong-kind",
	}

	httpRoute := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
		Spec: gatewayv1.HTTPRouteSpec{Rules: []gatewayv1.HTTPRouteRule{{
			Filters: []gatewayv1.HTTPRouteFilter{
				{Type: gatewayv1.HTTPRouteFilterExtensionRef, ExtensionRef: &supported},
				{Type: gatewayv1.HTTPRouteFilterExtensionRef, ExtensionRef: &wrongGroup},
				{Type: gatewayv1.HTTPRouteFilterExtensionRef, ExtensionRef: &wrongKind},
			},
		}}},
	}
	assert.Equal(t, []string{GenIndexKey("default", "supported")}, HTTPRouteExtensionIndexFunc(httpRoute))

	grpcRoute := &gatewayv1.GRPCRoute{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
		Spec: gatewayv1.GRPCRouteSpec{Rules: []gatewayv1.GRPCRouteRule{{
			Filters: []gatewayv1.GRPCRouteFilter{
				{Type: gatewayv1.GRPCRouteFilterExtensionRef, ExtensionRef: &supported},
				{Type: gatewayv1.GRPCRouteFilterExtensionRef, ExtensionRef: &wrongGroup},
				{Type: gatewayv1.GRPCRouteFilterExtensionRef, ExtensionRef: &wrongKind},
			},
		}}},
	}
	assert.Equal(t, []string{GenIndexKey("default", "supported")}, GRPCRouteExtensionIndexFunc(grpcRoute))
}
