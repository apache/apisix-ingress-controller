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
	gatewaynetworkingk8siov1 "sigs.k8s.io/gateway-api/apis/v1"
)

func buildGRPCRouteValidator(t *testing.T, objects ...runtime.Object) *GRPCRouteCustomValidator {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, gatewaynetworkingk8siov1.Install(scheme))

	builder := fake.NewClientBuilder().WithScheme(scheme)
	if len(objects) > 0 {
		builder = builder.WithRuntimeObjects(objects...)
	}

	return NewGRPCRouteCustomValidator(builder.Build())
}

func TestGRPCRouteCustomValidator_WarnsForMissingService(t *testing.T) {
	route := &gatewaynetworkingk8siov1.GRPCRoute{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
		Spec: gatewaynetworkingk8siov1.GRPCRouteSpec{
			Rules: []gatewaynetworkingk8siov1.GRPCRouteRule{{
				BackendRefs: []gatewaynetworkingk8siov1.GRPCBackendRef{{
					BackendRef: gatewaynetworkingk8siov1.BackendRef{
						BackendObjectReference: gatewaynetworkingk8siov1.BackendObjectReference{
							Name: gatewaynetworkingk8siov1.ObjectName("missing"),
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

	route := &gatewaynetworkingk8siov1.GRPCRoute{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
		Spec: gatewaynetworkingk8siov1.GRPCRouteSpec{
			Rules: []gatewaynetworkingk8siov1.GRPCRouteRule{{
				BackendRefs: []gatewaynetworkingk8siov1.GRPCBackendRef{{
					BackendRef: gatewaynetworkingk8siov1.BackendRef{
						BackendObjectReference: gatewaynetworkingk8siov1.BackendObjectReference{
							Name: gatewaynetworkingk8siov1.ObjectName("backend"),
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
