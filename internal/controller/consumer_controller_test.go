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
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/controller/indexer"
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
	require.NoError(t, gatewayv1.Install(scheme))

	cli := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(objs...).
		WithIndex(&v1alpha1.Consumer{}, indexer.SecretIndexRef, indexer.ConsumerSecretIndexFunc).
		WithIndex(&v1alpha1.Consumer{}, indexer.KeyAuthKey, indexer.ConsumerKeyAuthKeyIndexFunc).
		WithIndex(&corev1.Secret{}, indexer.KeyAuthKey, indexer.SecretKeyAuthKeyIndexFunc).
		Build()
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

func secretGrant() *gatewayv1.ReferenceGrant {
	return &gatewayv1.ReferenceGrant{
		ObjectMeta: metav1.ObjectMeta{Name: "allow-consumer", Namespace: secretNS},
		Spec: gatewayv1.ReferenceGrantSpec{
			From: []gatewayv1.ReferenceGrantFrom{{
				Group:     gatewayv1.Group(v1alpha1.GroupVersion.Group),
				Kind:      "Consumer",
				Namespace: consumerNS,
			}},
			To: []gatewayv1.ReferenceGrantTo{{
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

const sharedKey = "shared-key"

func keyAuthConsumer(name, gateway string, created time.Time, credentials ...v1alpha1.Credential) *v1alpha1.Consumer {
	return &v1alpha1.Consumer{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         consumerNS,
			CreationTimestamp: metav1.NewTime(created.Truncate(time.Second)),
		},
		Spec: v1alpha1.ConsumerSpec{
			GatewayRef:  v1alpha1.GatewayRef{Name: gateway},
			Credentials: credentials,
		},
	}
}

func inlineKeyAuth(name, key string) v1alpha1.Credential {
	return v1alpha1.Credential{
		Name:   name,
		Type:   "key-auth",
		Config: apiextensionsv1.JSON{Raw: []byte(`{"key":"` + key + `"}`)},
	}
}

func secretKeyAuth(name, secret string, namespace *string) v1alpha1.Credential {
	return v1alpha1.Credential{
		Name:      name,
		Type:      "key-auth",
		SecretRef: &v1alpha1.SecretReference{Name: secret, Namespace: namespace},
	}
}

func keyAuthSecret(name, namespace string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Data:       map[string][]byte{"key": []byte(sharedKey)},
	}
}

func skipDuplicates(t *testing.T, r *ConsumerReconciler, consumer *v1alpha1.Consumer) (*v1alpha1.Consumer, int) {
	t.Helper()
	tctx := provider.NewDefaultTranslateContext(context.Background())
	require.NoError(t, r.processSpec(context.Background(), tctx, consumer))
	published, skipped, err := r.skipDuplicateKeyAuthCredentials(context.Background(), tctx, consumer)
	require.NoError(t, err)
	return published, skipped
}

// A Secret created after its Consumer must not publish a key an older Consumer already uses.
func TestSkipDuplicateKeyAuth_SecretCreatedLater(t *testing.T) {
	now := time.Now()
	victim := keyAuthConsumer("victim", "gw", now.Add(-time.Hour), inlineKeyAuth("cred", sharedKey))
	attacker := keyAuthConsumer("attacker", "gw", now,
		secretKeyAuth("dup", "late-secret", nil),
		inlineKeyAuth("own", "attacker-key"),
	)
	r := buildConsumerReconciler(t, victim, attacker, keyAuthSecret("late-secret", consumerNS))

	published, skipped := skipDuplicates(t, r, attacker)
	require.Equal(t, 1, skipped)
	require.Len(t, published.Spec.Credentials, 1)
	require.Equal(t, "own", published.Spec.Credentials[0].Name)
	require.Len(t, attacker.Spec.Credentials, 2, "the cached Consumer must stay untouched")
}

// The older Consumer keeps its key.
func TestSkipDuplicateKeyAuth_OlderConsumerKeepsKey(t *testing.T) {
	now := time.Now()
	victim := keyAuthConsumer("victim", "gw", now.Add(-time.Hour), inlineKeyAuth("cred", sharedKey))
	attacker := keyAuthConsumer("attacker", "gw", now, secretKeyAuth("dup", "late-secret", nil))
	r := buildConsumerReconciler(t, victim, attacker, keyAuthSecret("late-secret", consumerNS))

	published, skipped := skipDuplicates(t, r, victim)
	require.Zero(t, skipped)
	require.Same(t, victim, published)
}

// Equal creation times fall back to namespace/name, so exactly one side wins.
func TestSkipDuplicateKeyAuth_TieBreak(t *testing.T) {
	now := time.Now()
	a := keyAuthConsumer("a", "gw", now, inlineKeyAuth("cred", sharedKey))
	b := keyAuthConsumer("b", "gw", now, inlineKeyAuth("cred", sharedKey))
	r := buildConsumerReconciler(t, a, b)

	_, skippedA := skipDuplicates(t, r, a)
	_, skippedB := skipDuplicates(t, r, b)
	require.Zero(t, skippedA)
	require.Equal(t, 1, skippedB)
}

// Consumers on different Gateways may share a key.
func TestSkipDuplicateKeyAuth_OtherGatewayIgnored(t *testing.T) {
	now := time.Now()
	other := keyAuthConsumer("other", "other-gw", now.Add(-time.Hour), inlineKeyAuth("cred", sharedKey))
	consumer := keyAuthConsumer("consumer", "gw", now, inlineKeyAuth("cred", sharedKey))
	r := buildConsumerReconciler(t, other, consumer)

	_, skipped := skipDuplicates(t, r, consumer)
	require.Zero(t, skipped)
}

// A cross-namespace Secret that no ReferenceGrant permits is never published, so it holds no key.
func TestSkipDuplicateKeyAuth_UngrantedSecretOwnsNoKey(t *testing.T) {
	SetEnableReferenceGrant(true)
	defer SetEnableReferenceGrant(false)

	now := time.Now()
	target := secretNS
	ungranted := keyAuthConsumer("ungranted", "gw", now.Add(-time.Hour), secretKeyAuth("cred", "foreign", &target))
	consumer := keyAuthConsumer("consumer", "gw", now, inlineKeyAuth("cred", sharedKey))
	r := buildConsumerReconciler(t, ungranted, consumer, keyAuthSecret("foreign", secretNS))

	_, skipped := skipDuplicates(t, r, consumer)
	require.Zero(t, skipped)
}

// Changes that free a key requeue the Consumer that was skipped for it.
func TestKeyAuthKeyOwnersRequeued(t *testing.T) {
	now := time.Now()
	victim := keyAuthConsumer("victim", "gw", now.Add(-time.Hour), inlineKeyAuth("cred", sharedKey))
	attacker := keyAuthConsumer("attacker", "gw", now, secretKeyAuth("dup", "late-secret", nil))
	secret := keyAuthSecret("late-secret", consumerNS)
	r := buildConsumerReconciler(t, victim, attacker, secret)

	attackerReq := reconcile.Request{NamespacedName: types.NamespacedName{Namespace: consumerNS, Name: "attacker"}}
	victimReq := reconcile.Request{NamespacedName: types.NamespacedName{Namespace: consumerNS, Name: "victim"}}

	require.Contains(t, r.listConsumersSharingKeyAuthKeys(context.Background(), victim), attackerReq)
	require.Contains(t, r.listConsumersForSecret(context.Background(), secret), victimReq)
}
