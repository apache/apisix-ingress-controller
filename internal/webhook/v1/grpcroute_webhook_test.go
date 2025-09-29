// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/internal/controller/config"
)

func buildGRPCRouteValidator(t *testing.T, objects ...runtime.Object) *GRPCRouteCustomValidator {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, gatewayv1.Install(scheme))

	managed := []runtime.Object{
		&gatewayv1.GatewayClass{
			ObjectMeta: metav1.ObjectMeta{Name: "apisix-gateway-class"},
			Spec: gatewayv1.GatewayClassSpec{
				ControllerName: gatewayv1.GatewayController(config.ControllerConfig.ControllerName),
			},
		},
		&gatewayv1.Gateway{
			ObjectMeta: metav1.ObjectMeta{Name: "test-gateway", Namespace: "default"},
			Spec: gatewayv1.GatewaySpec{
				GatewayClassName: gatewayv1.ObjectName("apisix-gateway-class"),
			},
		},
	}
	allObjects := append(managed, objects...)
	builder := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(allObjects...)

	return NewGRPCRouteCustomValidator(builder.Build())
}

func TestGRPCRouteCustomValidator_WarnsForMissingService(t *testing.T) {
	route := &gatewayv1.GRPCRoute{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
		Spec: gatewayv1.GRPCRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{{
					Name: gatewayv1.ObjectName("test-gateway"),
				}},
			},
			Rules: []gatewayv1.GRPCRouteRule{{
				BackendRefs: []gatewayv1.GRPCBackendRef{{
					BackendRef: gatewayv1.BackendRef{
						BackendObjectReference: gatewayv1.BackendObjectReference{
							Name: gatewayv1.ObjectName("missing"),
						},
					},
				}},
			}},
		},
	}

	validator := buildGRPCRouteValidator(t)
	warnings, err := validator.ValidateCreate(context.Background(), route)
	require.NoError(t, err)
	require.Len(t, warnings, 1)
	assert.Equal(t, warnings[0], "Referenced Service 'default/missing' not found")
}

func TestGRPCRouteCustomValidator_NoWarningsWhenServiceExists(t *testing.T) {
	service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "backend", Namespace: "default"}}
	validator := buildGRPCRouteValidator(t, service)

	route := &gatewayv1.GRPCRoute{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
		Spec: gatewayv1.GRPCRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{{
					Name: gatewayv1.ObjectName("test-gateway"),
				}},
			},
			Rules: []gatewayv1.GRPCRouteRule{{
				BackendRefs: []gatewayv1.GRPCBackendRef{{
					BackendRef: gatewayv1.BackendRef{
						BackendObjectReference: gatewayv1.BackendObjectReference{
							Name: gatewayv1.ObjectName("backend"),
						},
					},
				}},
			}},
		},
	}

	warnings, err := validator.ValidateCreate(context.Background(), route)
	require.NoError(t, err)
	assert.Empty(t, warnings)
}
