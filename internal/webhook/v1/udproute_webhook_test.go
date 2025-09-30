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
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/apache/apisix-ingress-controller/internal/controller/config"
)

func buildUDPRouteValidator(t *testing.T, objects ...runtime.Object) *UDPRouteCustomValidator {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, gatewayv1.Install(scheme))
	require.NoError(t, gatewayv1alpha2.Install(scheme))

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

	return NewUDPRouteCustomValidator(builder.Build())
}

func TestUDPRouteCustomValidator_WarnsForMissingReferences(t *testing.T) {
	route := &gatewayv1alpha2.UDPRoute{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
		Spec: gatewayv1alpha2.UDPRouteSpec{
			CommonRouteSpec: gatewayv1alpha2.CommonRouteSpec{
				ParentRefs: []gatewayv1alpha2.ParentReference{{
					Name: gatewayv1alpha2.ObjectName("test-gateway"),
				}},
			},
			Rules: []gatewayv1alpha2.UDPRouteRule{{
				BackendRefs: []gatewayv1alpha2.BackendRef{
					{
						BackendObjectReference: gatewayv1alpha2.BackendObjectReference{
							Name: gatewayv1alpha2.ObjectName("missing-svc"),
						},
					},
				},
			}},
		},
	}

	validator := buildUDPRouteValidator(t)
	warnings, err := validator.ValidateCreate(context.Background(), route)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"Referenced Service 'default/missing-svc' not found",
	}, warnings)
}

func TestUDPRouteCustomValidator_NoWarningsWhenResourcesExist(t *testing.T) {
	objs := []runtime.Object{
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "backend", Namespace: "default"}},
	}

	validator := buildUDPRouteValidator(t, objs...)

	route := &gatewayv1alpha2.UDPRoute{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
		Spec: gatewayv1alpha2.UDPRouteSpec{
			CommonRouteSpec: gatewayv1alpha2.CommonRouteSpec{
				ParentRefs: []gatewayv1alpha2.ParentReference{{
					Name: gatewayv1alpha2.ObjectName("test-gateway"),
				}},
			},
			Rules: []gatewayv1alpha2.UDPRouteRule{{
				BackendRefs: []gatewayv1alpha2.BackendRef{
					{
						BackendObjectReference: gatewayv1alpha2.BackendObjectReference{
							Name: gatewayv1alpha2.ObjectName("backend"),
						},
					},
				},
			}},
		},
	}

	warnings, err := validator.ValidateCreate(context.Background(), route)
	require.NoError(t, err)
	assert.Empty(t, warnings)
}

func TestUDPRouteCustomValidator_ValidateUpdate(t *testing.T) {
	objs := []runtime.Object{
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "backend", Namespace: "default"}},
	}

	validator := buildUDPRouteValidator(t, objs...)

	oldRoute := &gatewayv1alpha2.UDPRoute{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
		Spec: gatewayv1alpha2.UDPRouteSpec{
			CommonRouteSpec: gatewayv1alpha2.CommonRouteSpec{
				ParentRefs: []gatewayv1alpha2.ParentReference{{
					Name: gatewayv1alpha2.ObjectName("test-gateway"),
				}},
			},
			Rules: []gatewayv1alpha2.UDPRouteRule{{
				BackendRefs: []gatewayv1alpha2.BackendRef{
					{
						BackendObjectReference: gatewayv1alpha2.BackendObjectReference{
							Name: gatewayv1alpha2.ObjectName("backend"),
						},
					},
				},
			}},
		},
	}

	newRoute := &gatewayv1alpha2.UDPRoute{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
		Spec: gatewayv1alpha2.UDPRouteSpec{
			CommonRouteSpec: gatewayv1alpha2.CommonRouteSpec{
				ParentRefs: []gatewayv1alpha2.ParentReference{{
					Name: gatewayv1alpha2.ObjectName("test-gateway"),
				}},
			},
			Rules: []gatewayv1alpha2.UDPRouteRule{{
				BackendRefs: []gatewayv1alpha2.BackendRef{
					{
						BackendObjectReference: gatewayv1alpha2.BackendObjectReference{
							Name: gatewayv1alpha2.ObjectName("backend"),
						},
					},
				},
			}},
		},
	}

	warnings, err := validator.ValidateUpdate(context.Background(), oldRoute, newRoute)
	require.NoError(t, err)
	assert.Empty(t, warnings)
}

func TestUDPRouteCustomValidator_ValidateDelete(t *testing.T) {
	validator := buildUDPRouteValidator(t)

	route := &gatewayv1alpha2.UDPRoute{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
		Spec: gatewayv1alpha2.UDPRouteSpec{
			CommonRouteSpec: gatewayv1alpha2.CommonRouteSpec{
				ParentRefs: []gatewayv1alpha2.ParentReference{{
					Name: gatewayv1alpha2.ObjectName("test-gateway"),
				}},
			},
		},
	}

	warnings, err := validator.ValidateDelete(context.Background(), route)
	require.NoError(t, err)
	assert.Empty(t, warnings)
}

func TestUDPRouteCustomValidator_CrossNamespaceBackendRefs(t *testing.T) {
	otherNamespace := gatewayv1alpha2.Namespace("other")
	objs := []runtime.Object{
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "backend", Namespace: "other"}},
	}

	validator := buildUDPRouteValidator(t, objs...)

	route := &gatewayv1alpha2.UDPRoute{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
		Spec: gatewayv1alpha2.UDPRouteSpec{
			CommonRouteSpec: gatewayv1alpha2.CommonRouteSpec{
				ParentRefs: []gatewayv1alpha2.ParentReference{{
					Name: gatewayv1alpha2.ObjectName("test-gateway"),
				}},
			},
			Rules: []gatewayv1alpha2.UDPRouteRule{{
				BackendRefs: []gatewayv1alpha2.BackendRef{
					{
						BackendObjectReference: gatewayv1alpha2.BackendObjectReference{
							Name:      gatewayv1alpha2.ObjectName("backend"),
							Namespace: &otherNamespace,
						},
					},
				},
			}},
		},
	}

	warnings, err := validator.ValidateCreate(context.Background(), route)
	require.NoError(t, err)
	// Cross-namespace Service references should have no warnings since the Service exists
	assert.Empty(t, warnings)
}

func TestUDPRouteCustomValidator_MultipleBackendRefs(t *testing.T) {
	objs := []runtime.Object{
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "backend-1", Namespace: "default"}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "backend-2", Namespace: "default"}},
	}

	validator := buildUDPRouteValidator(t, objs...)

	route := &gatewayv1alpha2.UDPRoute{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
		Spec: gatewayv1alpha2.UDPRouteSpec{
			CommonRouteSpec: gatewayv1alpha2.CommonRouteSpec{
				ParentRefs: []gatewayv1alpha2.ParentReference{{
					Name: gatewayv1alpha2.ObjectName("test-gateway"),
				}},
			},
			Rules: []gatewayv1alpha2.UDPRouteRule{{
				BackendRefs: []gatewayv1alpha2.BackendRef{
					{
						BackendObjectReference: gatewayv1alpha2.BackendObjectReference{
							Name: gatewayv1alpha2.ObjectName("backend-1"),
						},
					},
					{
						BackendObjectReference: gatewayv1alpha2.BackendObjectReference{
							Name: gatewayv1alpha2.ObjectName("backend-2"),
						},
					},
					{
						BackendObjectReference: gatewayv1alpha2.BackendObjectReference{
							Name: gatewayv1alpha2.ObjectName("missing-backend"),
						},
					},
				},
			}},
		},
	}

	warnings, err := validator.ValidateCreate(context.Background(), route)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"Referenced Service 'default/missing-backend' not found",
	}, warnings)
}
