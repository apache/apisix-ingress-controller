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

func newGRPCRouteFixture(t *testing.T) (*GRPCRouteReconciler, *recordingProvider, *recordingUpdater) {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, gatewayv1.Install(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	from := gatewayv1.NamespacesFromAll
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
	route := &gatewayv1.GRPCRoute{
		ObjectMeta: metav1.ObjectMeta{Namespace: retractRouteNamespace, Name: retractRouteName},
		Spec: gatewayv1.GRPCRouteSpec{
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
	return &GRPCRouteReconciler{
		Client:   cli,
		Scheme:   scheme,
		Log:      logr.Discard(),
		Provider: prov,
		Updater:  updater,
		Readier:  readier,
	}, prov, updater
}

func grpcAcceptedConditionFromUpdater(t *testing.T, updater *recordingUpdater) *metav1.Condition {
	t.Helper()
	require.NotEmpty(t, updater.updates)
	mutated, ok := updater.updates[0].Mutator.Mutate(&gatewayv1.GRPCRoute{}).(*gatewayv1.GRPCRoute)
	require.True(t, ok)
	require.Len(t, mutated.Status.Parents, 1)
	accepted := meta.FindStatusCondition(mutated.Status.Parents[0].Conditions, string(gatewayv1.RouteConditionAccepted))
	require.NotNil(t, accepted)
	return accepted
}

// A translation / Provider.Update failure must be visible on the route status
// instead of Accepted=True plus a silent requeue.
func TestGRPCRouteReconcile_UpdateErrorIsVisibleOnStatus(t *testing.T) {
	r, prov, updater := newGRPCRouteFixture(t)
	prov.updateErr = errors.New("translation failed")

	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: k8stypes.NamespacedName{Namespace: retractRouteNamespace, Name: retractRouteName},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "translation failed")
	assert.Equal(t, 1, prov.updated)
	assert.Empty(t, prov.deleted)

	accepted := grpcAcceptedConditionFromUpdater(t, updater)
	assert.Equal(t, metav1.ConditionFalse, accepted.Status)
	assert.Contains(t, accepted.Message, "translation failed")
}
