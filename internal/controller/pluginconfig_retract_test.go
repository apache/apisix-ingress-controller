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
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/adc/translator/annotations"
	"github.com/apache/apisix-ingress-controller/internal/controller/config"
	"github.com/apache/apisix-ingress-controller/internal/controller/indexer"
	"github.com/apache/apisix-ingress-controller/internal/manager/readiness"
)

const (
	retractPluginConfigNamespace = "default"
	retractPluginConfigName      = "shared"
)

func retractPluginConfigScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, apiv2.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))
	return scheme
}

func retractIngressClass() *networkingv1.IngressClass {
	return &networkingv1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{Name: "apisix"},
		Spec:       networkingv1.IngressClassSpec{Controller: config.GetControllerName()},
	}
}

func newRetractReadier(t *testing.T, cli client.Client) readiness.ReadinessManager {
	t.Helper()
	readier := readiness.NewReadinessManager(cli, logr.Discard())
	require.NoError(t, readier.Start(context.Background()))
	return readier
}

// failGetOn makes Get fail with a non-NotFound error for objects of type T, so a
// transient read failure can be told apart from an absent reference.
func failGetOn[T client.Object]() interceptor.Funcs {
	return interceptor.Funcs{
		Get: func(ctx context.Context, cli client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			if _, ok := obj.(T); ok {
				return k8serrors.NewInternalError(errors.New("boom"))
			}
			return cli.Get(ctx, key, obj, opts...)
		},
	}
}

func newApisixRoutePluginConfigFixture(
	t *testing.T,
	interceptorFuncs interceptor.Funcs,
	extraObjects ...client.Object,
) (*ApisixRouteReconciler, *recordingProvider) {
	t.Helper()

	scheme := retractPluginConfigScheme(t)
	route := &apiv2.ApisixRoute{
		ObjectMeta: metav1.ObjectMeta{Namespace: retractPluginConfigNamespace, Name: "route"},
		Spec: apiv2.ApisixRouteSpec{
			IngressClassName: "apisix",
			HTTP: []apiv2.ApisixRouteHTTP{{
				Name:             "rule",
				PluginConfigName: retractPluginConfigName,
				Match:            apiv2.ApisixRouteHTTPMatch{Hosts: []string{"repro.test"}, Paths: []string{"/*"}},
				Backends: []apiv2.ApisixRouteHTTPBackend{{
					ServiceName: "backend",
					ServicePort: intstr.FromInt32(80),
				}},
			}},
		},
	}

	objects := append([]client.Object{retractIngressClass(), route}, extraObjects...)
	cli := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(objects...).
		WithStatusSubresource(route).
		WithInterceptorFuncs(interceptorFuncs).
		Build()

	prov := &recordingProvider{}
	return &ApisixRouteReconciler{
		Client:   cli,
		Scheme:   scheme,
		Log:      logr.Discard(),
		Provider: prov,
		Updater:  &recordingUpdater{},
		Readier:  newRetractReadier(t, cli),
	}, prov
}

func newIngressPluginConfigFixture(
	t *testing.T,
	interceptorFuncs interceptor.Funcs,
	extraObjects ...client.Object,
) (*IngressReconciler, *recordingProvider) {
	t.Helper()

	scheme := retractPluginConfigScheme(t)
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   retractPluginConfigNamespace,
			Name:        "ing",
			Annotations: map[string]string{annotations.AnnotationsPluginConfigName: retractPluginConfigName},
		},
		Spec: networkingv1.IngressSpec{IngressClassName: ptrTo("apisix")},
	}

	objects := append([]client.Object{retractIngressClass(), ingress}, extraObjects...)
	cli := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(objects...).
		WithStatusSubresource(ingress).
		WithInterceptorFuncs(interceptorFuncs).
		// The reconcile lists HTTPRoutePolicies by target, which the fake client
		// only serves once the index exists. No policy is under test here.
		WithIndex(&v1alpha1.HTTPRoutePolicy{}, indexer.PolicyTargetRefs,
			func(client.Object) []string { return nil }).
		Build()

	prov := &recordingProvider{}
	return &IngressReconciler{
		Client:   cli,
		Scheme:   scheme,
		Log:      logr.Discard(),
		Provider: prov,
		Updater:  &recordingUpdater{},
		Readier:  newRetractReadier(t, cli),
	}, prov
}

func ptrTo[T any](v T) *T { return &v }

var (
	retractApisixRouteKey = k8stypes.NamespacedName{Namespace: retractPluginConfigNamespace, Name: "route"}
	retractIngressKey     = k8stypes.NamespacedName{Namespace: retractPluginConfigNamespace, Name: "ing"}
)

// Deleting a shared ApisixPluginConfig leaves the referencing ApisixRoute in place
// but untranslatable. Its published configuration must be retracted, otherwise the
// data plane keeps applying the deleted plugins while the status reports the spec
// as invalid, and only deleting the route itself clears it.
func TestApisixRouteReconcile_RetractsWhenPluginConfigIsMissing(t *testing.T) {
	r, prov := newApisixRoutePluginConfigFixture(t, interceptor.Funcs{})

	result, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: retractApisixRouteKey})

	// No error and no requeue: the reference does not come back on its own, so
	// retrying it forever with backoff only produces log noise. The
	// ApisixPluginConfig watch reconciles the route again when it returns.
	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
	assert.Equal(t, []k8stypes.NamespacedName{retractApisixRouteKey}, prov.deleted)
	assert.Zero(t, prov.updated)
}

// A read failure that is not NotFound is transient. Retracting on it would drop a
// working route because the API server hiccuped.
func TestApisixRouteReconcile_KeepsRouteWhenPluginConfigReadFails(t *testing.T) {
	r, prov := newApisixRoutePluginConfigFixture(t, failGetOn[*apiv2.ApisixPluginConfig]())

	_, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: retractApisixRouteKey})

	require.Error(t, err)
	assert.True(t, k8serrors.IsInternalError(err), "want the transient error to surface, got %v", err)
	assert.Empty(t, prov.deleted, "a transient read failure must not retract the route")
}

// The same applies to an Ingress that names the plugin config through its
// annotation.
func TestIngressReconcile_RetractsWhenPluginConfigIsMissing(t *testing.T) {
	r, prov := newIngressPluginConfigFixture(t, interceptor.Funcs{})

	result, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: retractIngressKey})

	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
	assert.Equal(t, []k8stypes.NamespacedName{retractIngressKey}, prov.deleted)
	assert.Zero(t, prov.updated)
}

func TestIngressReconcile_KeepsIngressWhenPluginConfigReadFails(t *testing.T) {
	r, prov := newIngressPluginConfigFixture(t, failGetOn[*apiv2.ApisixPluginConfig]())

	_, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: retractIngressKey})

	require.Error(t, err)
	assert.True(t, k8serrors.IsInternalError(err), "want the transient error to surface, got %v", err)
	assert.Empty(t, prov.deleted, "a transient read failure must not retract the Ingress")
}

// With the plugin config present the resources must still be published.
func TestReconcile_PublishesWhenPluginConfigExists(t *testing.T) {
	pc := &apiv2.ApisixPluginConfig{
		ObjectMeta: metav1.ObjectMeta{Namespace: retractPluginConfigNamespace, Name: retractPluginConfigName},
	}

	ar, arProv := newApisixRoutePluginConfigFixture(t, interceptor.Funcs{}, pc.DeepCopy())
	_, err := ar.Reconcile(context.Background(), ctrl.Request{NamespacedName: retractApisixRouteKey})
	require.NoError(t, err)
	assert.Empty(t, arProv.deleted)
	assert.Equal(t, 1, arProv.updated)

	ing, ingProv := newIngressPluginConfigFixture(t, interceptor.Funcs{}, pc.DeepCopy())
	_, err = ing.Reconcile(context.Background(), ctrl.Request{NamespacedName: retractIngressKey})
	require.NoError(t, err)
	assert.Empty(t, ingProv.deleted)
	assert.Equal(t, 1, ingProv.updated)
}
