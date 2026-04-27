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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"

	apisixv1alpha1 "github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/controller/config"
	internaltypes "github.com/apache/apisix-ingress-controller/internal/types"
)

func withMockADCServer(t *testing.T, handler http.HandlerFunc) string {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Setenv("ADC_SERVER_URL", server.URL)
	t.Cleanup(server.Close)
	return server.URL
}

func managedIngressClassWithGatewayProxy(endpoint string) []runtime.Object {
	return managedIngressClassWithGatewayProxyMode(endpoint, "apisix-standalone")
}

func managedIngressClassWithGatewayProxyMode(endpoint, mode string) []runtime.Object {
	namespace := "default"

	return []runtime.Object{
		&networkingv1.IngressClass{
			ObjectMeta: metav1.ObjectMeta{Name: "apisix"},
			Spec: networkingv1.IngressClassSpec{
				Controller: config.ControllerConfig.ControllerName,
				Parameters: &networkingv1.IngressClassParametersReference{
					APIGroup:  ptr.To(apisixv1alpha1.GroupVersion.Group),
					Kind:      internaltypes.KindGatewayProxy,
					Name:      "gateway-proxy",
					Namespace: ptr.To(namespace),
				},
			},
		},
		&apisixv1alpha1.GatewayProxy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "gateway-proxy",
				Namespace: namespace,
			},
			Spec: apisixv1alpha1.GatewayProxySpec{
				Provider: &apisixv1alpha1.GatewayProxyProvider{
					Type: apisixv1alpha1.ProviderTypeControlPlane,
					ControlPlane: &apisixv1alpha1.ControlPlaneProvider{
						Mode:      mode,
						Endpoints: []string{endpoint},
						Auth: apisixv1alpha1.ControlPlaneAuth{
							Type: apisixv1alpha1.AuthTypeAdminKey,
							AdminKey: &apisixv1alpha1.AdminKeyAuth{
								Value: "token",
							},
						},
					},
				},
			},
		},
	}
}

func requireValidateRequest(t *testing.T, r *http.Request) {
	t.Helper()
	require.Equal(t, http.MethodPost, r.Method)
	require.Equal(t, "/apisix/admin/configs/validate", r.URL.Path)
	require.Equal(t, "token", r.Header.Get("X-API-KEY"))
}

func requireADCServerValidateRequest(t *testing.T, r *http.Request) {
	t.Helper()
	require.Equal(t, http.MethodPost, r.Method)
	require.Equal(t, "/validate", r.URL.Path)
}
