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

func buildGatewayValidator(t *testing.T, objects ...runtime.Object) *GatewayCustomValidator {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, gatewayv1.Install(scheme))

	builder := fake.NewClientBuilder().WithScheme(scheme)
	if len(objects) > 0 {
		builder = builder.WithRuntimeObjects(objects...)
	}

	return NewGatewayCustomValidator(builder.Build())
}

func TestGatewayCustomValidator_WarnsWhenTLSSecretMissing(t *testing.T) {
	className := gatewayv1.ObjectName("apisix")
	gatewayClass := &gatewayv1.GatewayClass{
		ObjectMeta: metav1.ObjectMeta{Name: string(className)},
		Spec: gatewayv1.GatewayClassSpec{
			ControllerName: gatewayv1.GatewayController(config.ControllerConfig.ControllerName),
		},
	}
	validator := buildGatewayValidator(t, gatewayClass)

	gateway := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Name: "example", Namespace: "default"},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: className,
			Listeners: []gatewayv1.Listener{{
				Name:     "https",
				Port:     443,
				Protocol: gatewayv1.HTTPSProtocolType,
				TLS: &gatewayv1.GatewayTLSConfig{
					CertificateRefs: []gatewayv1.SecretObjectReference{{
						Name: "missing-cert",
					}},
				},
			}},
		},
	}

	warnings, err := validator.ValidateCreate(context.Background(), gateway)
	require.NoError(t, err)
	require.Len(t, warnings, 1)
	assert.Equal(t, warnings[0], "Referenced Secret 'default/missing-cert' not found")
}

func TestGatewayCustomValidator_NoWarningsWhenSecretExists(t *testing.T) {
	className := gatewayv1.ObjectName("apisix")
	gatewayClass := &gatewayv1.GatewayClass{
		ObjectMeta: metav1.ObjectMeta{Name: string(className)},
		Spec: gatewayv1.GatewayClassSpec{
			ControllerName: gatewayv1.GatewayController(config.ControllerConfig.ControllerName),
		},
	}
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "tls-cert", Namespace: "default"}}
	validator := buildGatewayValidator(t, gatewayClass, secret)

	gateway := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Name: "example", Namespace: "default"},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: className,
			Listeners: []gatewayv1.Listener{{
				Name:     "https",
				Port:     443,
				Protocol: gatewayv1.HTTPSProtocolType,
				TLS: &gatewayv1.GatewayTLSConfig{
					CertificateRefs: []gatewayv1.SecretObjectReference{{
						Name: "tls-cert",
					}},
				},
			}},
		},
	}

	warnings, err := validator.ValidateCreate(context.Background(), gateway)
	require.NoError(t, err)
	assert.Empty(t, warnings)
}
