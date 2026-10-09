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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/provider"
)

// A secretRef inside an ApisixPluginConfig resolves in the plugin config's
// namespace, which is where the translator looks it up. Loading it from the
// route's namespace instead silently dropped the secret values whenever
// plugin_config_namespace pointed elsewhere.
func TestValidatePluginConfig_CrossNamespaceSecretRef(t *testing.T) {
	const (
		pcNamespace    = "ns-a"
		routeNamespace = "ns-b"
		secretName     = "my-secret"
		secretKey      = "credential"
	)

	pcSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: pcNamespace, Name: secretName},
		Data:       map[string][]byte{secretKey: []byte("from-pc-namespace")},
	}
	routeSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: routeNamespace, Name: secretName},
		Data:       map[string][]byte{secretKey: []byte("from-route-namespace")},
	}
	pc := &apiv2.ApisixPluginConfig{
		ObjectMeta: metav1.ObjectMeta{Namespace: pcNamespace, Name: "shared-config"},
		Spec: apiv2.ApisixPluginConfigSpec{
			Plugins: []apiv2.ApisixRoutePlugin{{Name: "echo", Enable: true, SecretRef: secretName}},
		},
	}
	route := &apiv2.ApisixRoute{
		ObjectMeta: metav1.ObjectMeta{Namespace: routeNamespace, Name: "cross-ns-route"},
	}
	rule := apiv2.ApisixRouteHTTP{
		Name:                  "cross-ns-rule",
		PluginConfigName:      pc.Name,
		PluginConfigNamespace: pcNamespace,
	}

	pcSecretKey := k8stypes.NamespacedName{Namespace: pcNamespace, Name: secretName}
	routeSecretKey := k8stypes.NamespacedName{Namespace: routeNamespace, Name: secretName}

	for name, tc := range map[string]struct {
		objects []client.Object
	}{
		"secret only in plugin config namespace": {objects: []client.Object{pc, pcSecret}},
		"same-named secret in both namespaces":   {objects: []client.Object{pc, pcSecret, routeSecret}},
	} {
		t.Run(name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			require.NoError(t, clientgoscheme.AddToScheme(scheme))
			require.NoError(t, apiv2.AddToScheme(scheme))
			r := &ApisixRouteReconciler{
				Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(tc.objects...).Build(),
				Scheme: scheme,
				Log:    logr.Discard(),
			}
			tctx := provider.NewDefaultTranslateContext(context.Background())

			require.NoError(t, r.validatePluginConfig(tctx, route, rule))

			require.Contains(t, tctx.Secrets, pcSecretKey)
			assert.Equal(t, "from-pc-namespace", string(tctx.Secrets[pcSecretKey].Data[secretKey]))
			assert.NotContains(t, tctx.Secrets, routeSecretKey)
		})
	}
}
