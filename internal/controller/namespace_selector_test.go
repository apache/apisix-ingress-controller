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
	k8stypes "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"

	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
)

const (
	watchedNamespace   = "watched"
	unwatchedNamespace = "unwatched"
)

func setNamespaceSelector(t *testing.T, entries ...string) {
	t.Helper()
	require.NoError(t, SetNamespaceSelector(entries))
	t.Cleanup(func() { namespaceSelector = nil })
}

func selectorNamespaces() []client.Object {
	return []client.Object{
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
			Name:   watchedNamespace,
			Labels: map[string]string{"team": "a"},
		}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
			Name:   unwatchedNamespace,
			Labels: map[string]string{"team": "b"},
		}},
	}
}

func TestSetNamespaceSelector(t *testing.T) {
	t.Cleanup(func() { namespaceSelector = nil })

	require.Error(t, SetNamespaceSelector([]string{"team in a"}))

	require.NoError(t, SetNamespaceSelector([]string{""}))
	assert.False(t, namespaceSelectorEnabled(), "[\"\"] disables the selector as in 1.x")
	assert.True(t, namespaceLabelsMatch(nil))

	require.NoError(t, SetNamespaceSelector([]string{"team=a", "team=b", "env=prod"}))
	assert.True(t, namespaceSelectorEnabled())
	assert.True(t, namespaceLabelsMatch(map[string]string{"team": "a", "env": "prod"}))
	assert.True(t, namespaceLabelsMatch(map[string]string{"team": "b", "env": "prod"}))
	assert.False(t, namespaceLabelsMatch(map[string]string{"team": "a"}), "entries on different keys are ANDed")
	assert.False(t, namespaceLabelsMatch(map[string]string{"team": "c", "env": "prod"}))
	assert.False(t, namespaceLabelsMatch(nil))
}

func TestIsWatchedNamespace(t *testing.T) {
	cli := fake.NewClientBuilder().WithScheme(retractPluginConfigScheme(t)).
		WithObjects(selectorNamespaces()...).Build()
	ctx := context.Background()

	watched, err := IsWatchedNamespace(ctx, cli, unwatchedNamespace)
	require.NoError(t, err)
	assert.True(t, watched, "every namespace is watched without a selector")

	setNamespaceSelector(t, "team=a")

	for ns, want := range map[string]bool{
		watchedNamespace:   true,
		unwatchedNamespace: false,
		"missing":          false,
		"":                 true,
	} {
		watched, err := IsWatchedNamespace(ctx, cli, ns)
		require.NoError(t, err, ns)
		assert.Equal(t, want, watched, ns)
	}
}

func TestFindMatchingIngressClassByObject_NamespaceSelector(t *testing.T) {
	cli := fake.NewClientBuilder().WithScheme(retractPluginConfigScheme(t)).
		WithObjects(append(selectorNamespaces(), retractIngressClass())...).Build()
	setNamespaceSelector(t, "team=a")

	route := func(ns string) *apiv2.ApisixRoute {
		return &apiv2.ApisixRoute{
			ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: "route"},
			Spec:       apiv2.ApisixRouteSpec{IngressClassName: "apisix"},
		}
	}

	ic, err := FindMatchingIngressClass(context.Background(), cli, logr.Discard(), route(watchedNamespace))
	require.NoError(t, err)
	assert.Equal(t, "apisix", ic.Name)

	_, err = FindMatchingIngressClass(context.Background(), cli, logr.Discard(), route(unwatchedNamespace))
	require.ErrorIs(t, err, ErrNamespaceNotWatched)
	assert.True(t, isIngressClassSelectionAbsent(err))
	assert.False(t, MatchesIngressClass(cli, logr.Discard(), route(unwatchedNamespace)))
}

func TestNamespaceSelectorChangedPredicate(t *testing.T) {
	setNamespaceSelector(t, "team=a")
	pred := namespaceSelectorChangedPredicate()

	ns := func(labels map[string]string) *corev1.Namespace {
		return &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "ns", Labels: labels}}
	}
	watching := map[string]string{"team": "a"}

	assert.True(t, pred.Update(event.UpdateEvent{ObjectOld: ns(nil), ObjectNew: ns(watching)}))
	assert.True(t, pred.Update(event.UpdateEvent{ObjectOld: ns(watching), ObjectNew: ns(nil)}))
	assert.False(t, pred.Update(event.UpdateEvent{
		ObjectOld: ns(watching),
		ObjectNew: ns(map[string]string{"team": "a", "other": "x"}),
	}), "a label change that keeps the match result must not requeue")
	assert.False(t, pred.Create(event.CreateEvent{Object: ns(watching)}))
	assert.False(t, pred.Delete(event.DeleteEvent{Object: ns(watching)}))
}

// An ApisixRoute in a namespace that stops matching the selector must be
// retracted, just like one whose IngressClass is no longer ours.
func TestApisixRouteReconcile_RetractsOutsideWatchedNamespace(t *testing.T) {
	scheme := retractPluginConfigScheme(t)
	route := &apiv2.ApisixRoute{
		ObjectMeta: metav1.ObjectMeta{Namespace: unwatchedNamespace, Name: "route"},
		Spec:       apiv2.ApisixRouteSpec{IngressClassName: "apisix"},
	}
	cli := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(append(selectorNamespaces(), retractIngressClass(), route)...).
		WithStatusSubresource(route).
		Build()
	setNamespaceSelector(t, "team=a")

	prov := &recordingProvider{}
	updater := &recordingUpdater{}
	r := &ApisixRouteReconciler{
		Client:   cli,
		Scheme:   scheme,
		Log:      logr.Discard(),
		Provider: prov,
		Updater:  updater,
		Readier:  newRetractReadier(t, cli),
	}

	key := k8stypes.NamespacedName{Namespace: unwatchedNamespace, Name: "route"}
	result, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: key})

	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
	assert.Equal(t, []k8stypes.NamespacedName{key}, prov.deleted)
	assert.Zero(t, prov.updated)
	assert.Empty(t, updater.updates, "the status of an unwatched object belongs to another controller")
}

func TestApisixTlsReconcile_RetractsOutsideWatchedNamespace(t *testing.T) {
	scheme := retractPluginConfigScheme(t)
	tls := &apiv2.ApisixTls{
		ObjectMeta: metav1.ObjectMeta{Namespace: unwatchedNamespace, Name: "tls"},
		Spec:       apiv2.ApisixTlsSpec{IngressClassName: "apisix"},
	}
	cli := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(append(selectorNamespaces(), retractIngressClass(), tls)...).
		WithStatusSubresource(tls).
		Build()
	setNamespaceSelector(t, "team=a")

	prov := &recordingProvider{}
	r := &ApisixTlsReconciler{
		Client:   cli,
		Scheme:   scheme,
		Log:      logr.Discard(),
		Provider: prov,
		Updater:  &recordingUpdater{},
		Readier:  newRetractReadier(t, cli),
	}

	key := k8stypes.NamespacedName{Namespace: unwatchedNamespace, Name: "tls"}
	_, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: key})

	require.NoError(t, err)
	assert.Equal(t, []k8stypes.NamespacedName{key}, prov.deleted)
	assert.Zero(t, prov.updated)
}
