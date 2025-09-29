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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	apisixv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/controller/config"
	"github.com/apache/apisix-ingress-controller/internal/controller/indexer"
)

func buildIngressValidator(t *testing.T, objects ...runtime.Object) *IngressCustomValidator {
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
	builder := fake.NewClientBuilder().
		WithScheme(scheme).
		WithIndex(&networkingv1.IngressClass{}, indexer.IngressClass, indexer.IngressClassIndexFunc).
		WithRuntimeObjects(allObjects...)

	return NewIngressCustomValidator(builder.Build())
}

func TestIngressCustomValidator_ValidateCreate_UnsupportedAnnotations(t *testing.T) {
	validator := buildIngressValidator(t)
	obj := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ingress",
			Namespace: "default",
			Annotations: map[string]string{
				"k8s.apisix.apache.org/use-regex":        "true",
				"k8s.apisix.apache.org/enable-websocket": "true",
			},
		},
	}

	warnings, err := validator.ValidateCreate(context.TODO(), obj)
	assert.NoError(t, err)
	assert.Len(t, warnings, 2)

	// Check that warnings contain the expected unsupported annotations
	warningsStr := strings.Join(warnings, " ")
	assert.Contains(t, warningsStr, "k8s.apisix.apache.org/use-regex")
	assert.Contains(t, warningsStr, "k8s.apisix.apache.org/enable-websocket")
}

func TestIngressCustomValidator_ValidateCreate_SupportedAnnotations(t *testing.T) {
	validator := buildIngressValidator(t)
	obj := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ingress",
			Namespace: "default",
			Annotations: map[string]string{
				"ingressclass.kubernetes.io/is-default-class": "true",
			},
		},
	}

	warnings, err := validator.ValidateCreate(context.TODO(), obj)
	assert.NoError(t, err)
	assert.Empty(t, warnings)
}

func TestIngressCustomValidator_ValidateUpdate_UnsupportedAnnotations(t *testing.T) {
	validator := buildIngressValidator(t)
	oldObj := &networkingv1.Ingress{}
	obj := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ingress",
			Namespace: "default",
			Annotations: map[string]string{
				"k8s.apisix.apache.org/enable-cors":       "true",
				"k8s.apisix.apache.org/cors-allow-origin": "*",
			},
		},
	}

	warnings, err := validator.ValidateUpdate(context.TODO(), oldObj, obj)
	assert.NoError(t, err)
	assert.Len(t, warnings, 2)

	// Check that warnings contain the expected unsupported annotations
	warningsStr := strings.Join(warnings, " ")
	assert.Contains(t, warningsStr, "k8s.apisix.apache.org/enable-cors")
	assert.Contains(t, warningsStr, "k8s.apisix.apache.org/cors-allow-origin")
}

func TestIngressCustomValidator_ValidateDelete_NoWarnings(t *testing.T) {
	validator := buildIngressValidator(t)
	obj := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ingress",
			Namespace: "default",
			Annotations: map[string]string{
				"k8s.apisix.apache.org/use-regex": "true",
			},
		},
	}

	warnings, err := validator.ValidateDelete(context.TODO(), obj)
	assert.NoError(t, err)
	assert.Empty(t, warnings)
}

func TestIngressCustomValidator_ValidateCreate_NoAnnotations(t *testing.T) {
	validator := buildIngressValidator(t)
	obj := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ingress",
			Namespace: "default",
		},
	}

	warnings, err := validator.ValidateCreate(context.TODO(), obj)
	assert.NoError(t, err)
	assert.Empty(t, warnings)
}

func TestIngressCustomValidator_WarnsForMissingServiceAndSecret(t *testing.T) {
	validator := buildIngressValidator(t)
	obj := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "test-ingress", Namespace: "default"},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{{
				IngressRuleValue: networkingv1.IngressRuleValue{HTTP: &networkingv1.HTTPIngressRuleValue{
					Paths: []networkingv1.HTTPIngressPath{{
						Backend: networkingv1.IngressBackend{
							Service: &networkingv1.IngressServiceBackend{Name: "default-svc"},
						},
					}},
				}},
			}},
			TLS: []networkingv1.IngressTLS{{SecretName: "missing-cert"}},
		},
	}

	warnings, err := validator.ValidateCreate(context.Background(), obj)
	require.NoError(t, err)
	require.Len(t, warnings, 2)
	require.Contains(t, warnings, "Referenced Service 'default/default-svc' not found")
	require.Contains(t, warnings, "Referenced Secret 'default/missing-cert' not found")
}

func TestIngressCustomValidator_NoWarningsWhenReferencesExist(t *testing.T) {
	service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "default-svc", Namespace: "default"}}
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "tls-cert", Namespace: "default"}}
	validator := buildIngressValidator(t, service, secret)

	obj := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "test-ingress", Namespace: "default"},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{{
				IngressRuleValue: networkingv1.IngressRuleValue{HTTP: &networkingv1.HTTPIngressRuleValue{
					Paths: []networkingv1.HTTPIngressPath{{
						Backend: networkingv1.IngressBackend{
							Service: &networkingv1.IngressServiceBackend{Name: "default-svc"},
						},
					}},
				}},
			}},
			TLS: []networkingv1.IngressTLS{{SecretName: "tls-cert"}},
		},
	}

	warnings, err := validator.ValidateCreate(context.Background(), obj)
	require.NoError(t, err)
	assert.Empty(t, warnings)
}
