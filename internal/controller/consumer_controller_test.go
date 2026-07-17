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
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	"sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/provider"
)

const (
	consumerNS = "team-a"
	secretNS   = "team-b"
)

func buildConsumerReconciler(t *testing.T, objs ...runtime.Object) *ConsumerReconciler {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))
	require.NoError(t, gatewayv1.Install(scheme))
	require.NoError(t, v1beta1.Install(scheme))

	cli := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(objs...).Build()
	return &ConsumerReconciler{Client: cli, Log: logr.Discard()}
}

func crossNamespaceConsumer() *v1alpha1.Consumer {
	target := secretNS
	return &v1alpha1.Consumer{
		ObjectMeta: metav1.ObjectMeta{Name: "attacker", Namespace: consumerNS},
		Spec: v1alpha1.ConsumerSpec{
			Credentials: []v1alpha1.Credential{{
				Name:      "cred",
				Type:      "key-auth",
				SecretRef: &v1alpha1.SecretReference{Name: "victim-secret", Namespace: &target},
			}},
		},
	}
}

func victimSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "victim-secret", Namespace: secretNS},
		Data:       map[string][]byte{"key": []byte("victim-key")},
	}
}

func secretGrant() *v1beta1.ReferenceGrant {
	return &v1beta1.ReferenceGrant{
		ObjectMeta: metav1.ObjectMeta{Name: "allow-consumer", Namespace: secretNS},
		Spec: v1beta1.ReferenceGrantSpec{
			From: []v1beta1.ReferenceGrantFrom{{
				Group:     v1beta1.Group(v1alpha1.GroupVersion.Group),
				Kind:      "Consumer",
				Namespace: consumerNS,
			}},
			To: []v1beta1.ReferenceGrantTo{{
				Group: "",
				Kind:  "Secret",
			}},
		},
	}
}

// Without a ReferenceGrant the cross-namespace secret must not be bound.
func TestProcessSpec_CrossNamespaceSecretRef_DeniedWithoutGrant(t *testing.T) {
	SetEnableReferenceGrant(true)
	defer SetEnableReferenceGrant(false)

	r := buildConsumerReconciler(t, victimSecret())
	consumer := crossNamespaceConsumer()
	tctx := provider.NewDefaultTranslateContext(context.Background())

	err := r.processSpec(context.Background(), tctx, consumer)
	require.Error(t, err)
	require.Empty(t, tctx.Secrets, "foreign secret must not be loaded without a ReferenceGrant")
}

// A matching ReferenceGrant permits the cross-namespace secret.
func TestProcessSpec_CrossNamespaceSecretRef_AllowedWithGrant(t *testing.T) {
	SetEnableReferenceGrant(true)
	defer SetEnableReferenceGrant(false)

	r := buildConsumerReconciler(t, victimSecret(), secretGrant())
	consumer := crossNamespaceConsumer()
	tctx := provider.NewDefaultTranslateContext(context.Background())

	err := r.processSpec(context.Background(), tctx, consumer)
	require.NoError(t, err)
	require.Contains(t, tctx.Secrets, types.NamespacedName{Namespace: secretNS, Name: "victim-secret"})
}

// Same-namespace SecretRef needs no grant.
func TestProcessSpec_SameNamespaceSecretRef_Allowed(t *testing.T) {
	SetEnableReferenceGrant(true)
	defer SetEnableReferenceGrant(false)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "local-secret", Namespace: consumerNS},
		Data:       map[string][]byte{"key": []byte("local-key")},
	}
	r := buildConsumerReconciler(t, secret)
	consumer := &v1alpha1.Consumer{
		ObjectMeta: metav1.ObjectMeta{Name: "local", Namespace: consumerNS},
		Spec: v1alpha1.ConsumerSpec{
			Credentials: []v1alpha1.Credential{{
				Name:      "cred",
				Type:      "key-auth",
				SecretRef: &v1alpha1.SecretReference{Name: "local-secret"},
			}},
		},
	}
	tctx := provider.NewDefaultTranslateContext(context.Background())

	err := r.processSpec(context.Background(), tctx, consumer)
	require.NoError(t, err)
	require.Contains(t, tctx.Secrets, types.NamespacedName{Namespace: consumerNS, Name: "local-secret"})
}
