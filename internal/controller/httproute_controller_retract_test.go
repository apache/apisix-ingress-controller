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
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/controller/config"
	"github.com/apache/apisix-ingress-controller/internal/manager/readiness"
)

const (
	retractGatewayNamespace = "infra"
	retractRouteNamespace   = "tenant"
	retractRouteName        = "route"
)

// newHTTPRouteRetractFixture builds a Gateway of our class in retractGatewayNamespace
// and an HTTPRoute in retractRouteNamespace that names it as its parent. from
// controls the listener's allowedRoutes, which is what revoking cross-namespace
// access changes.
func newHTTPRouteRetractFixture(
	t *testing.T,
	from gatewayv1.FromNamespaces,
) (*HTTPRouteReconciler, *recordingProvider, *recordingUpdater) {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, gatewayv1.Install(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	gatewayClass := &gatewayv1.GatewayClass{
		ObjectMeta: metav1.ObjectMeta{Name: "apisix"},
		Spec: gatewayv1.GatewayClassSpec{
			ControllerName: gatewayv1.GatewayController(config.ControllerConfig.ControllerName),
		},
	}
	gateway := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Namespace: retractGatewayNamespace, Name: "gw"},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: "apisix",
			Listeners: []gatewayv1.Listener{{
				Name:     "http",
				Protocol: gatewayv1.HTTPProtocolType,
				Port:     80,
				AllowedRoutes: &gatewayv1.AllowedRoutes{
					Namespaces: &gatewayv1.RouteNamespaces{From: &from},
				},
			}},
		},
	}
	route := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{Namespace: retractRouteNamespace, Name: retractRouteName},
		Spec: gatewayv1.HTTPRouteSpec{
			// A route with hostnames of its own is what makes filterHostnames fail
			// once no listener matches, which is how a route-wide reason used to
			// overwrite the parent's own.
			Hostnames: []gatewayv1.Hostname{"tenant.example"},
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{{
					Name:      gatewayv1.ObjectName(gateway.Name),
					Namespace: (*gatewayv1.Namespace)(&gateway.Namespace),
				}},
			},
		},
	}

	cli := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects([]client.Object{gatewayClass, gateway, route}...).
		WithStatusSubresource(route).
		Build()

	readier := readiness.NewReadinessManager(cli, logr.Discard())
	require.NoError(t, readier.Start(context.Background()))

	prov := &recordingProvider{}
	updater := &recordingUpdater{}
	return &HTTPRouteReconciler{
		Client:   cli,
		Scheme:   scheme,
		Log:      logr.Discard(),
		Provider: prov,
		Updater:  updater,
		Readier:  readier,
	}, prov, updater
}

func reconcileRetractHTTPRoute(t *testing.T, r *HTTPRouteReconciler) (ctrl.Result, error) {
	t.Helper()
	return r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: k8stypes.NamespacedName{Namespace: retractRouteNamespace, Name: retractRouteName},
	})
}

var retractRouteKey = k8stypes.NamespacedName{Namespace: retractRouteNamespace, Name: retractRouteName}

// Narrowing a listener's allowedRoutes leaves the HTTPRoute in place but stops it
// being accepted. The configuration an earlier reconcile published must be
// retracted, otherwise the data plane keeps serving a route the Gateway no longer
// admits and only deleting the HTTPRoute clears it.
func TestHTTPRouteReconcile_RetractsWhenListenerStopsAllowingRoute(t *testing.T) {
	r, prov, updater := newHTTPRouteRetractFixture(t, gatewayv1.NamespacesFromSame)

	result, err := reconcileRetractHTTPRoute(t, r)

	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
	assert.Equal(t, []k8stypes.NamespacedName{retractRouteKey}, prov.deleted)
	assert.Zero(t, prov.updated, "a route that is not accepted must not be published")

	// The reason must say why this parent rejected the route. The route-wide
	// status derived from filterHostnames would otherwise report
	// NoMatchingListenerHostname, which is a symptom of nothing having matched
	// rather than the cause.
	require.Len(t, updater.updates, 1)
	mutated, ok := updater.updates[0].Mutator.Mutate(&gatewayv1.HTTPRoute{}).(*gatewayv1.HTTPRoute)
	require.True(t, ok)
	require.Len(t, mutated.Status.Parents, 1)
	accepted := meta.FindStatusCondition(mutated.Status.Parents[0].Conditions, string(gatewayv1.RouteConditionAccepted))
	require.NotNil(t, accepted)
	assert.Equal(t, metav1.ConditionFalse, accepted.Status)
	assert.Equal(t, string(gatewayv1.RouteReasonNotAllowedByListeners), accepted.Reason)
}

// An accepted route must still be published and must not be retracted.
func TestHTTPRouteReconcile_PublishesAcceptedRoute(t *testing.T) {
	r, prov, _ := newHTTPRouteRetractFixture(t, gatewayv1.NamespacesFromAll)

	_, err := reconcileRetractHTTPRoute(t, r)

	require.NoError(t, err)
	assert.Empty(t, prov.deleted, "an accepted route must not be retracted")
	assert.Equal(t, 1, prov.updated)
}

// A provider failure while retracting must surface so the reconcile is retried.
func TestHTTPRouteReconcile_RetractErrorIsReturned(t *testing.T) {
	r, prov, _ := newHTTPRouteRetractFixture(t, gatewayv1.NamespacesFromSame)
	prov.deleteErr = errors.New("provider unavailable")

	_, err := reconcileRetractHTTPRoute(t, r)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider unavailable")
}
