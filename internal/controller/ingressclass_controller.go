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
	"fmt"

	"github.com/api7/gopkg/pkg/log"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/controller/indexer"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/internal/utils"
)

// IngressClassReconciler reconciles a IngressClass object.
type IngressClassReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger

	Provider provider.Provider
}

// SetupWithManager sets up the controller with the Manager.
func (r *IngressClassReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(
			&networkingv1.IngressClass{},
			builder.WithPredicates(
				predicate.NewPredicateFuncs(r.matchesController),
			),
		).
		WithEventFilter(
			predicate.Or(
				predicate.GenerationChangedPredicate{},
				predicate.NewPredicateFuncs(func(obj client.Object) bool {
					_, ok := obj.(*corev1.Secret)
					return ok
				}),
			),
		).
		Watches(
			&v1alpha1.GatewayProxy{},
			handler.EnqueueRequestsFromMapFunc(r.listIngressClassesForGatewayProxy),
		).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.listIngressClassesForSecret),
		).
		Complete(r)
}

func (r *IngressClassReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ingressClass := new(networkingv1.IngressClass)
	if err := r.Get(ctx, req.NamespacedName, ingressClass); err != nil {
		if client.IgnoreNotFound(err) == nil {
			ingressClass.Name = req.Name

			ingressClass.TypeMeta = metav1.TypeMeta{
				Kind:       KindIngressClass,
				APIVersion: networkingv1.SchemeGroupVersion.String(),
			}

			if err := r.Provider.Delete(ctx, ingressClass); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Create a translate context
	tctx := provider.NewDefaultTranslateContext(ctx)

	if err := r.processInfrastructure(tctx, ingressClass); err != nil {
		r.Log.Error(err, "failed to process infrastructure for ingressclass", "ingressclass", ingressClass.GetName())
		return ctrl.Result{}, err
	}

	if err := r.Provider.Update(ctx, tctx, ingressClass); err != nil {
		r.Log.Error(err, "failed to update ingressclass", "ingressclass", ingressClass.GetName())
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *IngressClassReconciler) matchesController(obj client.Object) bool {
	ingressClass, ok := obj.(*networkingv1.IngressClass)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to IngressClass")
		return false
	}
	return matchesController(ingressClass.Spec.Controller)
}

func (r *IngressClassReconciler) listIngressClassesForGatewayProxy(ctx context.Context, obj client.Object) []reconcile.Request {
	gatewayProxy, ok := obj.(*v1alpha1.GatewayProxy)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to GatewayProxy")
		return nil
	}
	namespace := gatewayProxy.GetNamespace()
	name := gatewayProxy.GetName()

	ingressClassList := &networkingv1.IngressClassList{}
	if err := r.List(ctx, ingressClassList, client.MatchingFields{
		indexer.IngressClassParametersRef: indexer.GenIndexKey(namespace, name),
	}); err != nil {
		r.Log.Error(err, "failed to list ingress classes for gateway proxy", "gatewayproxy", gatewayProxy.GetName())
		return nil
	}

	recs := make([]reconcile.Request, 0, len(ingressClassList.Items))
	for _, ingressClass := range ingressClassList.Items {
		if !r.matchesController(&ingressClass) {
			continue
		}
		recs = append(recs, reconcile.Request{
			NamespacedName: client.ObjectKey{
				Name: ingressClass.GetName(),
			},
		})
	}
	return recs
}

func (r *IngressClassReconciler) listIngressClassesForSecret(ctx context.Context, obj client.Object) []reconcile.Request {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to Secret")
		return nil
	}

	// 1. list gateway proxies by secret
	gatewayProxyList := &v1alpha1.GatewayProxyList{}
	if err := r.List(ctx, gatewayProxyList, client.MatchingFields{
		indexer.SecretIndexRef: indexer.GenIndexKey(secret.GetNamespace(), secret.GetName()),
	}); err != nil {
		r.Log.Error(err, "failed to list gateway proxies by secret", "secret", secret.GetName())
		return nil
	}

	// 2. list ingress classes by gateway proxies
	requests := make([]reconcile.Request, 0)
	for _, gatewayProxy := range gatewayProxyList.Items {
		requests = append(requests, r.listIngressClassesForGatewayProxy(ctx, &gatewayProxy)...)
	}

	return distinctRequests(requests)
}

func (r *IngressClassReconciler) processInfrastructure(tctx *provider.TranslateContext, ingressClass *networkingv1.IngressClass) error {
	if ingressClass.Spec.Parameters == nil {
		return nil
	}

	if ingressClass.Spec.Parameters.APIGroup == nil ||
		*ingressClass.Spec.Parameters.APIGroup != v1alpha1.GroupVersion.Group ||
		ingressClass.Spec.Parameters.Kind != KindGatewayProxy {
		return nil
	}

	namespace := ingressClass.Namespace
	if ingressClass.Spec.Parameters.Namespace != nil {
		namespace = *ingressClass.Spec.Parameters.Namespace
	}

	gatewayProxy := new(v1alpha1.GatewayProxy)
	if err := r.Get(context.Background(), client.ObjectKey{
		Namespace: namespace,
		Name:      ingressClass.Spec.Parameters.Name,
	}, gatewayProxy); err != nil {
		return fmt.Errorf("failed to get gateway proxy: %w", err)
	}

	rk := utils.NamespacedNameKind(ingressClass)

	tctx.GatewayProxies[rk] = *gatewayProxy
	tctx.ResourceParentRefs[rk] = append(tctx.ResourceParentRefs[rk], rk)

	// Load secrets if needed
	if gatewayProxy.Spec.Provider != nil && gatewayProxy.Spec.Provider.ControlPlane != nil {
		auth := gatewayProxy.Spec.Provider.ControlPlane.Auth
		if auth.Type == v1alpha1.AuthTypeAdminKey && auth.AdminKey != nil && auth.AdminKey.ValueFrom != nil {
			if auth.AdminKey.ValueFrom.SecretKeyRef != nil {
				secretRef := auth.AdminKey.ValueFrom.SecretKeyRef
				secret := &corev1.Secret{}
				if err := r.Get(context.Background(), client.ObjectKey{
					Namespace: namespace,
					Name:      secretRef.Name,
				}, secret); err != nil {
					log.Error(err, "failed to get secret for gateway proxy", "namespace", namespace, "name", secretRef.Name)
					return err
				}
				tctx.Secrets[client.ObjectKey{
					Namespace: namespace,
					Name:      secretRef.Name,
				}] = secret
			}
		}
	}

	_, ok := tctx.GatewayProxies[rk]
	if !ok {
		return fmt.Errorf("no gateway proxy found for ingress class")
	}

	return nil
}
