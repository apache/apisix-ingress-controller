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

package translator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/provider"
)

func TestTranslateHTTPRouteServerPortVars(t *testing.T) {
	sectionName := gatewayv1.SectionName("http-main")
	pathMatchType := gatewayv1.PathMatchPathPrefix
	pathValue := "/"

	tests := []struct {
		name       string
		parentRefs []gatewayv1.ParentReference
		listeners  []gatewayv1.Listener
		expected   adctypes.Vars
	}{
		{
			name: "no injection for single listener without sectionName",
			parentRefs: []gatewayv1.ParentReference{
				{Name: "gw"},
			},
			listeners: []gatewayv1.Listener{
				{Name: "http-main", Protocol: gatewayv1.HTTPProtocolType, Port: gatewayv1.PortNumber(9080)},
			},
			expected: nil,
		},
		{
			name: "injection for single listener with explicit sectionName",
			parentRefs: []gatewayv1.ParentReference{
				{Name: "gw", SectionName: &sectionName},
			},
			listeners: []gatewayv1.Listener{
				{Name: "http-main", Protocol: gatewayv1.HTTPProtocolType, Port: gatewayv1.PortNumber(9080)},
			},
			expected: adctypes.Vars{
				{
					{StrVal: "server_port"},
					{StrVal: "=="},
					{StrVal: "9080"},
				},
			},
		},
		{
			name: "injection for multiple listener ports",
			parentRefs: []gatewayv1.ParentReference{
				{Name: "gw"},
			},
			listeners: []gatewayv1.Listener{
				{Name: "http-main", Protocol: gatewayv1.HTTPProtocolType, Port: gatewayv1.PortNumber(9081)},
				{Name: "http-alt", Protocol: gatewayv1.HTTPProtocolType, Port: gatewayv1.PortNumber(9080)},
			},
			expected: adctypes.Vars{
				{
					{StrVal: "server_port"},
					{StrVal: "in"},
					{SliceVal: []adctypes.StringOrSlice{
						{StrVal: "9080"},
						{StrVal: "9081"},
					}},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tctx := provider.NewDefaultTranslateContext(context.Background())
			tctx.RouteParentRefs = tt.parentRefs
			tctx.Listeners = tt.listeners

			httpRoute := &gatewayv1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "route",
					Namespace: "default",
				},
				Spec: gatewayv1.HTTPRouteSpec{
					Rules: []gatewayv1.HTTPRouteRule{
						{
							Matches: []gatewayv1.HTTPRouteMatch{
								{
									Path: &gatewayv1.HTTPPathMatch{
										Type:  &pathMatchType,
										Value: &pathValue,
									},
								},
							},
						},
					},
				},
			}

			got, err := (&Translator{}).TranslateHTTPRoute(tctx, httpRoute)
			assert.NoError(t, err)
			if assert.Len(t, got.Services, 1) && assert.Len(t, got.Services[0].Routes, 1) {
				assert.Equal(t, tt.expected, got.Services[0].Routes[0].Vars)
			}
		})
	}
}
