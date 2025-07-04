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

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/controller/indexer"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/internal/utils"
)

// GatewayProxyController reconciles a GatewayProxy object.
type GatewayProxyController struct {
	client.Client

	Scheme   *runtime.Scheme
	Log      logr.Logger
	Provider provider.Provider
}

func (r *GatewayProxyController) SetupWithManager(mrg ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mrg).
		For(&v1alpha1.GatewayProxy{}).
		WithEventFilter(
			predicate.Or(
				predicate.GenerationChangedPredicate{},
				predicate.NewPredicateFuncs(TypePredicate[*corev1.Secret]()),
			),
		).
		Watches(&corev1.Service{},
			handler.EnqueueRequestsFromMapFunc(r.listGatewayProxiesForProviderService),
		).
		Watches(&discoveryv1.EndpointSlice{},
			handler.EnqueueRequestsFromMapFunc(r.listGatewayProxiesForProviderEndpointSlice),
		).
		Watches(&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.listGatewayProxiesForSecret),
		).
		Complete(r)
}

func (r *GatewayProxyController) Reconcile(ctx context.Context, req ctrl.Request) (reconcile.Result, error) {
	var tctx = provider.NewDefaultTranslateContext(ctx)

	var gp v1alpha1.GatewayProxy
	if err := r.Get(ctx, req.NamespacedName, &gp); err != nil {
		if client.IgnoreNotFound(err) == nil {
			gp.Namespace = req.Namespace
			gp.Name = req.Name
			err = r.Provider.Update(ctx, tctx, &gp)
		}
		return ctrl.Result{}, err
	}

	// if there is no provider, update with empty translate context
	if gp.Spec.Provider == nil || gp.Spec.Provider.ControlPlane == nil {
		return reconcile.Result{}, r.Provider.Update(ctx, tctx, &gp)
	}

	// process endpoints for provider service
	providerService := gp.Spec.Provider.ControlPlane.Service
	if providerService == nil {
		tctx.EndpointSlices[req.NamespacedName] = nil
	} else {
		if err := addProviderEndpointsToTranslateContext(tctx, r.Client, types.NamespacedName{
			Namespace: gp.Namespace,
			Name:      providerService.Name,
		}); err != nil {
			return reconcile.Result{}, err
		}
	}

	// process secret for provider auth
	auth := gp.Spec.Provider.ControlPlane.Auth
	if auth.AdminKey != nil && auth.AdminKey.ValueFrom != nil && auth.AdminKey.ValueFrom.SecretKeyRef != nil {
		var (
			secret   corev1.Secret
			secretNN = types.NamespacedName{
				Namespace: gp.GetNamespace(),
				Name:      auth.AdminKey.ValueFrom.SecretKeyRef.Name,
			}
		)
		if err := r.Get(ctx, secretNN, &secret); err != nil {
			r.Log.Error(err, "failed to get secret", "secret", secretNN)
			return reconcile.Result{}, err
		}
		tctx.Secrets[secretNN] = &secret
	}

	// list Gateways that reference the GatewayProxy
	var (
		gatewayList      gatewayv1.GatewayList
		ingressClassList networkingv1.IngressClassList
		indexKey         = indexer.GenIndexKey(gp.GetNamespace(), gp.GetName())
	)
	if err := r.List(ctx, &gatewayList, client.MatchingFields{indexer.ParametersRef: indexKey}); err != nil {
		r.Log.Error(err, "failed to list GatewayList")
		return ctrl.Result{}, nil
	}

	// list IngressClasses that reference the GatewayProxy
	if err := r.List(ctx, &ingressClassList, client.MatchingFields{indexer.IngressClassParametersRef: indexKey}); err != nil {
		r.Log.Error(err, "failed to list IngressClassList")
		return reconcile.Result{}, err
	}

	// append referrers to translate context
	for _, item := range gatewayList.Items {
		tctx.GatewayProxyReferrers[req.NamespacedName] = append(tctx.GatewayProxyReferrers[req.NamespacedName], utils.NamespacedNameKind(&item))
	}
	for _, item := range ingressClassList.Items {
		tctx.GatewayProxyReferrers[req.NamespacedName] = append(tctx.GatewayProxyReferrers[req.NamespacedName], utils.NamespacedNameKind(&item))
	}

	if err := r.Provider.Update(ctx, tctx, &gp); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *GatewayProxyController) listGatewayProxiesForProviderService(ctx context.Context, obj client.Object) (requests []reconcile.Request) {
	service, ok := obj.(*corev1.Service)
	if !ok {
		r.Log.Error(errors.New("unexpected object type"), "failed to convert object to Service")
		return nil
	}

	return ListRequests(ctx, r.Client, r.Log, &v1alpha1.GatewayProxyList{}, client.MatchingFields{
		indexer.ServiceIndexRef: indexer.GenIndexKey(service.GetNamespace(), service.GetName()),
	})
}

func (r *GatewayProxyController) listGatewayProxiesForProviderEndpointSlice(ctx context.Context, obj client.Object) (requests []reconcile.Request) {
	endpointSlice, ok := obj.(*discoveryv1.EndpointSlice)
	if !ok {
		r.Log.Error(errors.New("unexpected object type"), "failed to convert object to EndpointSlice")
		return nil
	}

	return ListRequests(ctx, r.Client, r.Log, &v1alpha1.GatewayProxyList{}, client.MatchingFields{
		indexer.ServiceIndexRef: indexer.GenIndexKey(endpointSlice.GetNamespace(), endpointSlice.Labels[discoveryv1.LabelServiceName]),
	})
}

func (r *GatewayProxyController) listGatewayProxiesForSecret(ctx context.Context, object client.Object) []reconcile.Request {
	secret, ok := object.(*corev1.Secret)
	if !ok {
		r.Log.Error(errors.New("unexpected object type"), "failed to convert object to Secret")
		return nil
	}
	return ListRequests(ctx, r.Client, r.Log, &v1alpha1.GatewayProxyList{}, client.MatchingFields{
		indexer.SecretIndexRef: indexer.GenIndexKey(secret.GetNamespace(), secret.GetName()),
	})
}
