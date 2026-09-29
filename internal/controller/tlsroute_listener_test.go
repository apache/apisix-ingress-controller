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
	"net/http"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	"github.com/apache/apisix-ingress-controller/internal/manager/readiness"
	"github.com/apache/apisix-ingress-controller/internal/provider"
)

type tlsRouteRecordingProvider struct {
	tctx *provider.TranslateContext
}

func (p *tlsRouteRecordingProvider) Register(string, *http.ServeMux) {}

func (p *tlsRouteRecordingProvider) Update(_ context.Context, tctx *provider.TranslateContext, _ client.Object) error {
	p.tctx = tctx
	return nil
}

func (p *tlsRouteRecordingProvider) Delete(context.Context, client.Object) error { return nil }

func (p *tlsRouteRecordingProvider) Start(context.Context) error { return nil }

func (p *tlsRouteRecordingProvider) NeedLeaderElection() bool { return true }

type discardStatusUpdater struct{}

func (discardStatusUpdater) Update(status.Update) {}

func TestTLSRouteReconcilePropagatesExplicitListener(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, gatewayv1.Install(scheme))

	hostname := gatewayv1.Hostname("api6.com")
	sectionName := gatewayv1.SectionName("tls-main")
	port := gatewayv1.PortNumber(9110)
	gatewayClass := newParentRefGatewayClass()
	gateway := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "gateway"},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: gatewayv1.ObjectName(gatewayClass.Name),
			Listeners: []gatewayv1.Listener{{
				Name:     sectionName,
				Protocol: gatewayv1.TLSProtocolType,
				Port:     port,
				Hostname: &hostname,
			}},
		},
	}
	route := &gatewayv1.TLSRoute{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "route"},
		Spec: gatewayv1.TLSRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{{
					Name:        gatewayv1.ObjectName(gateway.Name),
					SectionName: &sectionName,
				}},
			},
			Hostnames: []gatewayv1.Hostname{hostname},
		},
	}
	cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(gatewayClass, gateway, route).Build()
	provider := &tlsRouteRecordingProvider{}
	readier := readiness.NewReadinessManager(cli, logr.Discard())
	require.NoError(t, readier.Start(context.Background()))
	reconciler := &TLSRouteReconciler{
		Client:   cli,
		Scheme:   scheme,
		Log:      logr.Discard(),
		Provider: provider,
		Updater:  discardStatusUpdater{},
		Readier:  readier,
	}

	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: k8stypes.NamespacedName{Namespace: route.Namespace, Name: route.Name},
	})
	require.NoError(t, err)
	require.NotNil(t, provider.tctx)
	require.Len(t, provider.tctx.Listeners, 1)
	assert.Equal(t, sectionName, provider.tctx.Listeners[0].Name)
	assert.Equal(t, port, provider.tctx.Listeners[0].Port)
	assert.True(t, provider.tctx.HasExplicitListenerMatch)
}
