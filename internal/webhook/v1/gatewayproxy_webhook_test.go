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

	v1alpha1 "github.com/apache/apisix-ingress-controller/api/v1alpha1"
)

func buildGatewayProxyValidator(t *testing.T, objects ...runtime.Object) *GatewayProxyCustomValidator {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	builder := fake.NewClientBuilder().WithScheme(scheme)
	if len(objects) > 0 {
		builder = builder.WithRuntimeObjects(objects...)
	}

	return NewGatewayProxyCustomValidator(builder.Build())
}

func newGatewayProxy() *v1alpha1.GatewayProxy {
	return &v1alpha1.GatewayProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "default",
		},
		Spec: v1alpha1.GatewayProxySpec{
			Provider: &v1alpha1.GatewayProxyProvider{
				Type: v1alpha1.ProviderTypeControlPlane,
				ControlPlane: &v1alpha1.ControlPlaneProvider{
					Service: &v1alpha1.ProviderService{Name: "control-plane", Port: 9180},
					Auth: v1alpha1.ControlPlaneAuth{
						Type: v1alpha1.AuthTypeAdminKey,
						AdminKey: &v1alpha1.AdminKeyAuth{
							ValueFrom: &v1alpha1.AdminKeyValueFrom{
								SecretKeyRef: &v1alpha1.SecretKeySelector{
									Name: "admin-key",
									Key:  "token",
								},
							},
						},
					},
				},
			},
		},
	}
}

func newGatewayProxyWithEndpoints(name string, endpoints []string) *v1alpha1.GatewayProxy {
	gp := newGatewayProxy()
	gp.Name = name
	gp.Spec.Provider.ControlPlane.Service = nil
	gp.Spec.Provider.ControlPlane.Endpoints = endpoints
	return gp
}

func setInlineAdminKey(gp *v1alpha1.GatewayProxy, value string) {
	if gp == nil || gp.Spec.Provider == nil || gp.Spec.Provider.ControlPlane == nil {
		return
	}
	if gp.Spec.Provider.ControlPlane.Auth.AdminKey == nil {
		gp.Spec.Provider.ControlPlane.Auth.AdminKey = &v1alpha1.AdminKeyAuth{}
	}
	gp.Spec.Provider.ControlPlane.Auth.AdminKey.Value = value
	gp.Spec.Provider.ControlPlane.Auth.AdminKey.ValueFrom = nil
}

func setSecretAdminKey(gp *v1alpha1.GatewayProxy, name, key string) {
	if gp == nil || gp.Spec.Provider == nil || gp.Spec.Provider.ControlPlane == nil {
		return
	}
	if gp.Spec.Provider.ControlPlane.Auth.AdminKey == nil {
		gp.Spec.Provider.ControlPlane.Auth.AdminKey = &v1alpha1.AdminKeyAuth{}
	}
	gp.Spec.Provider.ControlPlane.Auth.AdminKey.Value = ""
	gp.Spec.Provider.ControlPlane.Auth.AdminKey.ValueFrom = &v1alpha1.AdminKeyValueFrom{
		SecretKeyRef: &v1alpha1.SecretKeySelector{
			Name: name,
			Key:  key,
		},
	}
}

func TestGatewayProxyValidator_MissingService(t *testing.T) {
	gp := newGatewayProxy()
	gp.Spec.Provider.ControlPlane.Auth.AdminKey = nil
	validator := buildGatewayProxyValidator(t)

	warnings, err := validator.ValidateCreate(context.Background(), gp)
	require.NoError(t, err)
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], "Service 'default/control-plane' not found")
}

func TestGatewayProxyValidator_MissingAdminSecret(t *testing.T) {
	gp := newGatewayProxy()
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "control-plane",
			Namespace: "default",
		},
	}
	validator := buildGatewayProxyValidator(t, service)

	warnings, err := validator.ValidateCreate(context.Background(), gp)
	require.NoError(t, err)
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], "Secret 'default/admin-key' not found")
}

func TestGatewayProxyValidator_MissingAdminSecretKey(t *testing.T) {
	gp := newGatewayProxy()
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "admin-key",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"wrong": []byte("value"),
		},
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "control-plane",
			Namespace: "default",
		},
	}

	validator := buildGatewayProxyValidator(t, secret, service)

	warnings, err := validator.ValidateCreate(context.Background(), gp)
	require.NoError(t, err)
	require.Len(t, warnings, 1)
	require.Contains(t, warnings[0], "Secret key 'token' not found")
}

func TestGatewayProxyValidator_NoWarnings(t *testing.T) {
	gp := newGatewayProxy()
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "admin-key",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"token": []byte("value"),
		},
	}
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "control-plane",
			Namespace: "default",
		},
	}

	validator := buildGatewayProxyValidator(t, secret, service)

	warnings, err := validator.ValidateCreate(context.Background(), gp)
	require.NoError(t, err)
	require.Empty(t, warnings)
}

func TestGatewayProxyValidator_DetectsServiceConflict(t *testing.T) {
	existing := newGatewayProxy()
	existing.Name = "existing"

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "control-plane",
			Namespace: "default",
		},
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "admin-key",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"token": []byte("value"),
		},
	}

	validator := buildGatewayProxyValidator(t, existing, service, secret)

	candidate := newGatewayProxy()
	candidate.Name = "candidate"

	warnings, err := validator.ValidateCreate(context.Background(), candidate)
	require.Error(t, err)
	require.Len(t, warnings, 0)
	require.Contains(t, err.Error(), "gateway group conflict")
	require.Contains(t, err.Error(), "Service default/control-plane port 9180")
	require.Contains(t, err.Error(), "AdminKey secret default/admin-key key token")
}

func TestGatewayProxyValidator_DetectsEndpointConflict(t *testing.T) {
	existing := newGatewayProxyWithEndpoints("existing", []string{"https://127.0.0.1:9443", "https://10.0.0.1:9443"})
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "admin-key",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"token": []byte("value"),
		},
	}
	validator := buildGatewayProxyValidator(t, existing, secret)

	candidate := newGatewayProxyWithEndpoints("candidate", []string{"https://10.0.0.1:9443", "https://127.0.0.1:9443"})

	warnings, err := validator.ValidateCreate(context.Background(), candidate)
	require.Error(t, err)
	require.Len(t, warnings, 0)
	require.Contains(t, err.Error(), "gateway group conflict")
	require.Contains(t, err.Error(), "endpoints [https://10.0.0.1:9443, https://127.0.0.1:9443]")
	require.Contains(t, err.Error(), "AdminKey secret default/admin-key key token")
}

func TestGatewayProxyValidator_AllowsDistinctGatewayGroups(t *testing.T) {
	existing := newGatewayProxyWithEndpoints("existing", []string{"https://127.0.0.1:9443"})
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "admin-key",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"token": []byte("value"),
		},
	}
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "control-plane",
			Namespace: "default",
		},
	}
	validator := buildGatewayProxyValidator(t, existing, secret, service)

	candidate := newGatewayProxy()
	candidate.Name = "candidate"
	candidate.Spec.Provider.ControlPlane.Service = &v1alpha1.ProviderService{
		Name: "control-plane",
		Port: 9180,
	}

	warnings, err := validator.ValidateCreate(context.Background(), candidate)
	require.NoError(t, err)
	require.Empty(t, warnings)
}

func TestGatewayProxyValidator_AllowsServiceConflictWithDifferentAdminSecret(t *testing.T) {
	existing := newGatewayProxy()
	existing.Name = "existing"

	candidate := newGatewayProxy()
	candidate.Name = "candidate"
	setSecretAdminKey(candidate, "admin-key-alt", "token")

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "control-plane",
			Namespace: "default",
		},
	}
	existingSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "admin-key",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"token": []byte("value"),
		},
	}
	altSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "admin-key-alt",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"token": []byte("value"),
		},
	}

	validator := buildGatewayProxyValidator(t, existing, service, existingSecret, altSecret)

	warnings, err := validator.ValidateCreate(context.Background(), candidate)
	require.NoError(t, err)
	require.Empty(t, warnings)
}

func TestGatewayProxyValidator_DetectsInlineAdminKeyConflict(t *testing.T) {
	existing := newGatewayProxyWithEndpoints("existing", []string{"https://127.0.0.1:9443", "https://10.0.0.1:9443"})
	setInlineAdminKey(existing, "inline-cred")

	candidate := newGatewayProxyWithEndpoints("candidate", []string{"https://10.0.0.1:9443"})
	setInlineAdminKey(candidate, "inline-cred")

	validator := buildGatewayProxyValidator(t, existing)

	warnings, err := validator.ValidateCreate(context.Background(), candidate)
	require.Error(t, err)
	require.Len(t, warnings, 0)
	require.Contains(t, err.Error(), "gateway group conflict")
	require.Contains(t, err.Error(), "control plane endpoints [https://10.0.0.1:9443]")
	require.Contains(t, err.Error(), "inline AdminKey value")
}

func TestGatewayProxyValidator_AllowsEndpointOverlapWithDifferentAdminKey(t *testing.T) {
	existing := newGatewayProxyWithEndpoints("existing", []string{"https://127.0.0.1:9443", "https://10.0.0.1:9443"})

	candidate := newGatewayProxyWithEndpoints("candidate", []string{"https://10.0.0.1:9443", "https://192.168.0.1:9443"})
	setSecretAdminKey(candidate, "admin-key-alt", "token")

	existingSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "admin-key",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"token": []byte("value"),
		},
	}
	altSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "admin-key-alt",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"token": []byte("value"),
		},
	}

	validator := buildGatewayProxyValidator(t, existing, existingSecret, altSecret)

	warnings, err := validator.ValidateCreate(context.Background(), candidate)
	require.NoError(t, err)
	require.Empty(t, warnings)
}
