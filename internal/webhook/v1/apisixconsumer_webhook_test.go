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
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	apisixv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/controller/config"
)

func buildApisixConsumerValidator(t *testing.T, objects ...runtime.Object) *ApisixConsumerCustomValidator {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, networkingv1.AddToScheme(scheme))
	require.NoError(t, apisixv2.AddToScheme(scheme))

	managed := []runtime.Object{
		&networkingv1.IngressClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: "apisix",
				Annotations: map[string]string{
					"ingressclass.kubernetes.io/is-default-class": "true",
				},
			},
			Spec: networkingv1.IngressClassSpec{
				Controller: config.ControllerConfig.ControllerName,
			},
		},
	}
	allObjects := append(managed, objects...)
	builder := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(allObjects...)

	return NewApisixConsumerCustomValidator(builder.Build())
}

func TestApisixConsumerValidator_MissingBasicAuthSecret(t *testing.T) {
	consumer := &apisixv2.ApisixConsumer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "default",
		},
		Spec: apisixv2.ApisixConsumerSpec{
			IngressClassName: "apisix",
			AuthParameter: apisixv2.ApisixConsumerAuthParameter{
				BasicAuth: &apisixv2.ApisixConsumerBasicAuth{
					SecretRef: &corev1.LocalObjectReference{Name: "basic-auth"},
				},
			},
		},
	}

	validator := buildApisixConsumerValidator(t)

	warnings, err := validator.ValidateCreate(context.Background(), consumer)
	require.NoError(t, err)
	require.Equal(t, 1, len(warnings))
	require.Equal(t, "Referenced Secret 'default/basic-auth' not found", warnings[0])
}

func TestApisixConsumerValidator_MultipleSecretWarnings(t *testing.T) {
	consumer := &apisixv2.ApisixConsumer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "default",
		},
		Spec: apisixv2.ApisixConsumerSpec{
			IngressClassName: "apisix",
			AuthParameter: apisixv2.ApisixConsumerAuthParameter{
				BasicAuth: &apisixv2.ApisixConsumerBasicAuth{
					SecretRef: &corev1.LocalObjectReference{Name: "basic-auth"},
				},
				JwtAuth: &apisixv2.ApisixConsumerJwtAuth{
					SecretRef: &corev1.LocalObjectReference{Name: "jwt-auth"},
				},
				HMACAuth: &apisixv2.ApisixConsumerHMACAuth{
					SecretRef: &corev1.LocalObjectReference{Name: "hmac-auth"},
				},
			},
		},
	}

	basicAuthSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "basic-auth",
			Namespace: "default",
		},
	}

	validator := buildApisixConsumerValidator(t, basicAuthSecret)

	warnings, err := validator.ValidateCreate(context.Background(), consumer)
	require.NoError(t, err)
	require.Len(t, warnings, 2)
	require.ElementsMatch(t, []string{
		"Referenced Secret 'default/jwt-auth' not found",
		"Referenced Secret 'default/hmac-auth' not found",
	}, warnings)
}

func TestApisixConsumerValidator_NoWarningsWhenSecretsExist(t *testing.T) {
	consumer := &apisixv2.ApisixConsumer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "default",
		},
		Spec: apisixv2.ApisixConsumerSpec{
			IngressClassName: "apisix",
			AuthParameter: apisixv2.ApisixConsumerAuthParameter{
				KeyAuth: &apisixv2.ApisixConsumerKeyAuth{
					SecretRef: &corev1.LocalObjectReference{Name: "key-auth"},
				},
				WolfRBAC: &apisixv2.ApisixConsumerWolfRBAC{
					SecretRef: &corev1.LocalObjectReference{Name: "wolf-rbac"},
				},
			},
		},
	}

	secrets := []runtime.Object{
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "key-auth", Namespace: "default"}},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "wolf-rbac", Namespace: "default"}},
	}

	validator := buildApisixConsumerValidator(t, secrets...)

	warnings, err := validator.ValidateCreate(context.Background(), consumer)
	require.NoError(t, err)
	require.Empty(t, warnings)
}
