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
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestRouteParentResolverGatewayDeletion(t *testing.T) {
	for _, tc := range []struct {
		name     string
		route    client.Object
		protocol gatewayv1.ProtocolType
	}{
		{"HTTPRoute", &gatewayv1.HTTPRoute{}, gatewayv1.HTTPProtocolType},
		{"GRPCRoute", &gatewayv1.GRPCRoute{}, gatewayv1.HTTPProtocolType},
		{"TCPRoute", &gatewayv1.TCPRoute{}, gatewayv1.TCPProtocolType},
		{"UDPRoute", &gatewayv1.UDPRoute{}, gatewayv1.UDPProtocolType},
		{"TLSRoute", &gatewayv1.TLSRoute{}, gatewayv1.TLSProtocolType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, knownSibling := range []bool{false, true} {
				name := "never-resolved sibling"
				if knownSibling {
					name = "previously-resolved sibling"
				}
				t.Run(name, func(t *testing.T) {
					ctx := context.Background()
					route := tc.route.DeepCopyObject().(client.Object)
					route.SetNamespace("default")
					route.SetName("route")
					route.SetUID("route-uid")
					classA := newParentRefGatewayClass()
					classB := classA.DeepCopy()
					classB.Name = "sibling-class"
					listener := gatewayv1.Listener{Name: "listener", Port: 9000, Protocol: tc.protocol}
					if tc.protocol == gatewayv1.TLSProtocolType {
						listener.TLS = &gatewayv1.ListenerTLSConfig{Mode: ptr.To(gatewayv1.TLSModePassthrough)}
					}
					a := &gatewayv1.Gateway{
						ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "a", UID: "a-uid"},
						Spec:       gatewayv1.GatewaySpec{GatewayClassName: gatewayv1.ObjectName(classA.Name), Listeners: []gatewayv1.Listener{listener}},
					}
					b := a.DeepCopy()
					b.Name, b.UID = "b", "b-uid"
					b.Spec.GatewayClassName = gatewayv1.ObjectName(classB.Name)
					objects := []client.Object{classA, a, b}
					if knownSibling {
						objects = append(objects, classB)
					}
					cli := fake.NewClientBuilder().WithScheme(parentRefTestScheme(t)).WithObjects(objects...).Build()
					refs := []gatewayv1.ParentReference{{Name: "a"}, {Name: "b"}}
					resolver := &routeParentResolver{}
					parents, err := resolver.Resolve(ctx, cli, logr.Discard(), route, refs)
					require.NoError(t, err)
					expected := 1
					if knownSibling {
						expected = 2
					}
					require.Len(t, parents, expected)
					require.True(t, isRouteAccepted(parents))
					if knownSibling {
						require.NoError(t, cli.Delete(ctx, classB))
					}
					require.NoError(t, cli.Delete(ctx, a))
					parents, err = resolver.Resolve(ctx, cli, logr.Discard(), route, refs)
					require.NoError(t, err)
					if knownSibling {
						require.Len(t, parents, 1)
						require.Equal(t, "b", parents[0].Gateway.Name)
						require.True(t, isRouteAccepted(parents))
						// Listener changes still apply while its class is absent.
						b.Spec.Listeners[0].AllowedRoutes = &gatewayv1.AllowedRoutes{
							Kinds: []gatewayv1.RouteGroupKind{{Kind: "UnsupportedRoute"}},
						}
						require.NoError(t, cli.Update(ctx, b))
						parents, err = resolver.Resolve(ctx, cli, logr.Discard(), route, refs)
						require.NoError(t, err)
						require.False(t, isRouteAccepted(parents))
						require.NoError(t, cli.Delete(ctx, b))
						parents, err = resolver.Resolve(ctx, cli, logr.Discard(), route, refs)
						require.NoError(t, err)
					}
					require.Empty(t, parents)
					require.Empty(t, resolver.parents)
				})
			}
		})
	}
}

func TestRouteParentResolverDoesNotReuseOwnership(t *testing.T) {
	for _, change := range []string{"gateway recreated", "route recreated", "class changed", "route deleted", "parent removed", "class reassigned"} {
		t.Run(change, func(t *testing.T) {
			ctx := context.Background()
			gc := newParentRefGatewayClass()
			gw := &gatewayv1.Gateway{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "gw", UID: "gateway-uid"},
				Spec:       gatewayv1.GatewaySpec{GatewayClassName: gatewayv1.ObjectName(gc.Name), Listeners: []gatewayv1.Listener{{Name: "http", Port: 80, Protocol: gatewayv1.HTTPProtocolType}}},
			}
			route := &gatewayv1.HTTPRoute{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "route", UID: "route-uid"}}
			cli := fake.NewClientBuilder().WithScheme(parentRefTestScheme(t)).WithObjects(gc, gw).Build()
			resolver := &routeParentResolver{}
			refs := []gatewayv1.ParentReference{{Name: "gw"}}
			parents, err := resolver.Resolve(ctx, cli, logr.Discard(), route, refs)
			require.NoError(t, err)
			require.Len(t, parents, 1)
			require.NoError(t, cli.Delete(ctx, gc))
			switch change {
			case "gateway recreated":
				require.NoError(t, cli.Delete(ctx, gw))
				gw.ResourceVersion = ""
				gw.UID = "new-gateway-uid"
				require.NoError(t, cli.Create(ctx, gw))
			case "route recreated":
				route.UID = "new-route-uid"
			case "class changed":
				gw.Spec.GatewayClassName = "different-class"
				require.NoError(t, cli.Update(ctx, gw))
			case "route deleted":
				resolver.Forget(client.ObjectKeyFromObject(route))
			case "parent removed":
				_, err = resolver.Resolve(ctx, cli, logr.Discard(), route, nil)
				require.NoError(t, err)
			case "class reassigned":
				gc.ResourceVersion = ""
				gc.Spec.ControllerName = "other.example/controller"
				require.NoError(t, cli.Create(ctx, gc))
			}
			parents, err = resolver.Resolve(ctx, cli, logr.Discard(), route, refs)
			require.NoError(t, err)
			require.Empty(t, parents)
		})
	}
}

func TestRouteParentResolverPreservesOwnershipOnReadError(t *testing.T) {
	ctx := context.Background()
	gc := newParentRefGatewayClass()
	gw := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "gw", UID: "gateway-uid"},
		Spec:       gatewayv1.GatewaySpec{GatewayClassName: gatewayv1.ObjectName(gc.Name), Listeners: []gatewayv1.Listener{{Name: "http", Port: 80, Protocol: gatewayv1.HTTPProtocolType}}},
	}
	route := &gatewayv1.HTTPRoute{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "route"}}
	fail := false
	readErr := errors.New("temporary API failure")
	cli := fake.NewClientBuilder().WithScheme(parentRefTestScheme(t)).WithObjects(gc, gw).
		WithInterceptorFuncs(interceptor.Funcs{Get: func(ctx context.Context, cli client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			if _, ok := obj.(*gatewayv1.GatewayClass); ok && fail {
				return readErr
			}
			return cli.Get(ctx, key, obj, opts...)
		}}).Build()
	resolver := &routeParentResolver{}
	refs := []gatewayv1.ParentReference{{Name: "gw"}}
	_, err := resolver.Resolve(ctx, cli, logr.Discard(), route, refs)
	require.NoError(t, err)
	fail = true
	_, err = resolver.Resolve(ctx, cli, logr.Discard(), route, refs)
	require.ErrorIs(t, err, readErr)
	fail = false
	require.NoError(t, cli.Delete(ctx, gc))
	parents, err := resolver.Resolve(ctx, cli, logr.Discard(), route, refs)
	require.NoError(t, err)
	require.Len(t, parents, 1)
}
