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

	apisixv1alpha1 "github.com/apache/apisix-ingress-controller/api/v1alpha1"
)

func buildConsumerValidator(t *testing.T, objects ...runtime.Object) *ConsumerCustomValidator {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, apisixv1alpha1.AddToScheme(scheme))

	builder := fake.NewClientBuilder().WithScheme(scheme)
	if len(objects) > 0 {
		builder = builder.WithRuntimeObjects(objects...)
	}

	return NewConsumerCustomValidator(builder.Build())
}

func TestConsumerValidator_MissingSecretDefaultNamespace(t *testing.T) {
	consumer := &apisixv1alpha1.Consumer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "default",
		},
		Spec: apisixv1alpha1.ConsumerSpec{
			Credentials: []apisixv1alpha1.Credential{{
				Type: "jwt-auth",
				SecretRef: &apisixv1alpha1.SecretReference{
					Name: "jwt-secret",
				},
			}},
		},
	}

	validator := buildConsumerValidator(t)

	warnings, err := validator.ValidateCreate(context.Background(), consumer)
	require.NoError(t, err)
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], "Referenced Secret 'default/jwt-secret' not found")
}

func TestConsumerValidator_MissingSecretCustomNamespace(t *testing.T) {
	ns := "auth"
	consumer := &apisixv1alpha1.Consumer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "default",
		},
		Spec: apisixv1alpha1.ConsumerSpec{
			Credentials: []apisixv1alpha1.Credential{{
				Type: "jwt-auth",
				SecretRef: &apisixv1alpha1.SecretReference{
					Name:      "jwt-secret",
					Namespace: &ns,
				},
			}},
		},
	}

	validator := buildConsumerValidator(t)

	warnings, err := validator.ValidateCreate(context.Background(), consumer)
	require.NoError(t, err)
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], "Referenced Secret 'auth/jwt-secret' not found")
}

func TestConsumerValidator_NoWarnings(t *testing.T) {
	ns := "auth"
	consumer := &apisixv1alpha1.Consumer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "default",
		},
		Spec: apisixv1alpha1.ConsumerSpec{
			Credentials: []apisixv1alpha1.Credential{{
				Type: "jwt-auth",
				SecretRef: &apisixv1alpha1.SecretReference{
					Name:      "jwt-secret",
					Namespace: &ns,
				},
			}, {
				Type: "key-auth",
				SecretRef: &apisixv1alpha1.SecretReference{
					Name: "key-secret",
				},
			}},
		},
	}

	objs := []runtime.Object{
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "jwt-secret", Namespace: "auth"}},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "key-secret", Namespace: "default"}},
	}

	validator := buildConsumerValidator(t, objs...)

	warnings, err := validator.ValidateCreate(context.Background(), consumer)
	require.NoError(t, err)
	require.Empty(t, warnings)
}
