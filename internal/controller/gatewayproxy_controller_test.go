// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package controller

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/controller/indexer"
)

func gatewayProxyControllerScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, v1alpha1.AddToScheme(scheme))
	require.NoError(t, networkingv1.AddToScheme(scheme))
	return scheme
}

func newGatewayProxyControllerClient(t *testing.T, objects ...client.Object) client.Client {
	t.Helper()
	return fake.NewClientBuilder().
		WithScheme(gatewayProxyControllerScheme(t)).
		WithObjects(objects...).
		WithIndex(&networkingv1.IngressClass{}, indexer.IngressClassParametersRef,
			indexer.IngressClassParametersRefIndexFunc).
		Build()
}

func TestGatewayProxyReconcileDeletesInactiveConfiguration(t *testing.T) {
	key := types.NamespacedName{Namespace: "default", Name: "gp-a"}
	validProvider := &v1alpha1.GatewayProxyProvider{
		Type:         v1alpha1.ProviderTypeControlPlane,
		ControlPlane: &v1alpha1.ControlPlaneProvider{},
	}

	tests := []struct {
		name    string
		objects []client.Object
	}{
		{name: "GatewayProxy deleted"},
		{
			name: "provider cleared",
			objects: []client.Object{&v1alpha1.GatewayProxy{
				ObjectMeta: metav1.ObjectMeta{Namespace: key.Namespace, Name: key.Name},
				Spec: v1alpha1.GatewayProxySpec{
					Provider: &v1alpha1.GatewayProxyProvider{},
				},
			}},
		},
		{
			name: "no referrers",
			objects: []client.Object{&v1alpha1.GatewayProxy{
				ObjectMeta: metav1.ObjectMeta{Namespace: key.Namespace, Name: key.Name},
				Spec:       v1alpha1.GatewayProxySpec{Provider: validProvider},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &recordingProvider{}
			reconciler := &GatewayProxyController{
				Client:            newGatewayProxyControllerClient(t, tt.objects...),
				Log:               logr.Discard(),
				Provider:          provider,
				disableGatewayAPI: true,
			}

			result, err := reconciler.Reconcile(context.Background(), ctrl.Request{NamespacedName: key})

			require.NoError(t, err)
			assert.Equal(t, ctrl.Result{}, result)
			assert.Equal(t, []types.NamespacedName{key}, provider.deleted)
			assert.Zero(t, provider.updated)
		})
	}
}
