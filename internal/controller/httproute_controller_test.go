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
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/internal/manager/readiness"
)

type noopReadier struct {
	readiness.ReadinessManager
}

func (noopReadier) Done(client.Object, k8stypes.NamespacedName) {}

// TestHTTPRouteReconcile_EmptyGateways covers the two ways a route resolves to no
// Gateway of this controller. Only a resolved parent naming another controller
// proves the route is not ours; an unresolvable parent must leave the data plane
// alone, since a missing cluster-scoped GatewayClass empties the list for every
// route under it at once.
func TestHTTPRouteReconcile_EmptyGateways(t *testing.T) {
	scheme := parentRefTestScheme(t)

	route := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "r"},
		Spec: gatewayv1.HTTPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{{Name: "gw"}},
			},
		},
	}
	gw := func(class string) *gatewayv1.Gateway {
		return &gatewayv1.Gateway{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "gw"},
			Spec: gatewayv1.GatewaySpec{
				GatewayClassName: gatewayv1.ObjectName(class),
				Listeners: []gatewayv1.Listener{
					{Name: "http", Port: 80, Protocol: gatewayv1.HTTPProtocolType},
				},
			},
		}
	}

	for _, tc := range []struct {
		name       string
		objects    []client.Object
		wantDelete bool
	}{
		{
			name: "gatewayclass of another controller",
			objects: []client.Object{route, gw("other"), &gatewayv1.GatewayClass{
				ObjectMeta: metav1.ObjectMeta{Name: "other"},
				Spec:       gatewayv1.GatewayClassSpec{ControllerName: "example.com/other-controller"},
			}},
			wantDelete: true,
		},
		{
			name:       "gatewayclass missing",
			objects:    []client.Object{route, gw("gone")},
			wantDelete: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tc.objects...).Build()
			prov := &recordingProvider{}
			r := &HTTPRouteReconciler{
				Client:   cli,
				Scheme:   scheme,
				Log:      logr.Discard(),
				Provider: prov,
				Readier:  noopReadier{},
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_, err := r.Reconcile(ctx, ctrl.Request{
				NamespacedName: k8stypes.NamespacedName{Namespace: "default", Name: "r"},
			})
			assert.NoError(t, err)
			assert.Equal(t, tc.wantDelete, len(prov.deleted) == 1,
				"unexpected data plane delete, deletes=%d", len(prov.deleted))
		})
	}
}
