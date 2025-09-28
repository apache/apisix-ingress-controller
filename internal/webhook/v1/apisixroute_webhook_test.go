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

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	apisixv2 "github.com/apache/apisix-ingress-controller/api/v2"
)

func buildApisixRouteValidator(t *testing.T, objects ...runtime.Object) *ApisixRouteCustomValidator {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, apisixv2.AddToScheme(scheme))

	builder := fake.NewClientBuilder().WithScheme(scheme)
	if len(objects) > 0 {
		builder = builder.WithRuntimeObjects(objects...)
	}

	return NewApisixRouteCustomValidator(builder.Build())
}

func TestApisixRouteValidator_MissingHTTPService(t *testing.T) {
	route := &apisixv2.ApisixRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "default",
		},
		Spec: apisixv2.ApisixRouteSpec{
			HTTP: []apisixv2.ApisixRouteHTTP{{
				Name: "rule",
				Backends: []apisixv2.ApisixRouteHTTPBackend{{
					ServiceName: "backend",
				}},
			}},
		},
	}

	validator := buildApisixRouteValidator(t)

	warnings, err := validator.ValidateCreate(context.Background(), route)
	require.NoError(t, err)
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], "Referenced Service 'default/backend' not found")
}

func TestApisixRouteValidator_MissingPluginSecret(t *testing.T) {
	route := &apisixv2.ApisixRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "default",
		},
		Spec: apisixv2.ApisixRouteSpec{
			HTTP: []apisixv2.ApisixRouteHTTP{{
				Name: "rule",
				Backends: []apisixv2.ApisixRouteHTTPBackend{{
					ServiceName: "backend",
				}},
				Plugins: []apisixv2.ApisixRoutePlugin{{
					Name:      "jwt-auth",
					Enable:    true,
					SecretRef: "jwt-secret",
				}},
			}},
		},
	}

	backendSvc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "backend", Namespace: "default"}}

	validator := buildApisixRouteValidator(t, backendSvc)

	warnings, err := validator.ValidateCreate(context.Background(), route)
	require.NoError(t, err)
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], "Referenced Secret 'default/jwt-secret' not found")
}

func TestApisixRouteValidator_MissingStreamService(t *testing.T) {
	route := &apisixv2.ApisixRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "default",
		},
		Spec: apisixv2.ApisixRouteSpec{
			Stream: []apisixv2.ApisixRouteStream{{
				Name:     "stream",
				Protocol: "TCP",
				Backend: apisixv2.ApisixRouteStreamBackend{
					ServiceName: "stream-svc",
				},
			}},
		},
	}

	validator := buildApisixRouteValidator(t)

	warnings, err := validator.ValidateCreate(context.Background(), route)
	require.NoError(t, err)
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], "Referenced Service 'default/stream-svc' not found")
}

func TestApisixRouteValidator_NoWarnings(t *testing.T) {
	route := &apisixv2.ApisixRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "default",
		},
		Spec: apisixv2.ApisixRouteSpec{
			HTTP: []apisixv2.ApisixRouteHTTP{{
				Name: "rule",
				Backends: []apisixv2.ApisixRouteHTTPBackend{{
					ServiceName: "backend",
				}},
				Plugins: []apisixv2.ApisixRoutePlugin{{
					Name:      "jwt-auth",
					Enable:    true,
					SecretRef: "jwt-secret",
				}},
			}},
		},
	}

	objs := []runtime.Object{
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "backend", Namespace: "default"}},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "jwt-secret", Namespace: "default"}},
	}

	validator := buildApisixRouteValidator(t, objs...)

	warnings, err := validator.ValidateCreate(context.Background(), route)
	require.NoError(t, err)
	require.Empty(t, warnings)
}
