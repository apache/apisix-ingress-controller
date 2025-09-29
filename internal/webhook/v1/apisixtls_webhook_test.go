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

func buildApisixTlsValidator(t *testing.T, objects ...runtime.Object) *ApisixTlsCustomValidator {
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

	return NewApisixTlsCustomValidator(builder.Build())
}

func newApisixTls() *apisixv2.ApisixTls {
	return &apisixv2.ApisixTls{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "default",
		},
		Spec: apisixv2.ApisixTlsSpec{
			IngressClassName: "apisix",
			Hosts:            []apisixv2.HostType{"example.com"},
			Secret: apisixv2.ApisixSecret{
				Name:      "server-cert",
				Namespace: "default",
			},
		},
	}
}

func TestApisixTlsValidator_MissingServerSecret(t *testing.T) {
	tls := newApisixTls()
	validator := buildApisixTlsValidator(t)

	warnings, err := validator.ValidateCreate(context.Background(), tls)
	require.NoError(t, err)
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], "Referenced Secret 'default/server-cert' not found")
}

func TestApisixTlsValidator_MissingClientSecret(t *testing.T) {
	tls := newApisixTls()
	tls.Spec.Client = &apisixv2.ApisixMutualTlsClientConfig{
		CASecret: apisixv2.ApisixSecret{
			Name:      "mtls-ca",
			Namespace: "mtls",
		},
	}

	serverSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "server-cert",
			Namespace: "default",
		},
	}

	validator := buildApisixTlsValidator(t, serverSecret)

	warnings, err := validator.ValidateCreate(context.Background(), tls)
	require.NoError(t, err)
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], "Referenced Secret 'mtls/mtls-ca' not found")
}

func TestApisixTlsValidator_NoWarningsWhenSecretsExist(t *testing.T) {
	tls := newApisixTls()
	tls.Spec.Client = &apisixv2.ApisixMutualTlsClientConfig{
		CASecret: apisixv2.ApisixSecret{
			Name:      "mtls-ca",
			Namespace: "mtls",
		},
	}

	objects := []runtime.Object{
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "server-cert", Namespace: "default"}},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "mtls-ca", Namespace: "mtls"}},
	}

	validator := buildApisixTlsValidator(t, objects...)

	warnings, err := validator.ValidateCreate(context.Background(), tls)
	require.NoError(t, err)
	require.Empty(t, warnings)
}
