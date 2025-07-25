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

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/controller/config"
	"github.com/apache/apisix-ingress-controller/internal/controller/indexer"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	"github.com/apache/apisix-ingress-controller/internal/manager/readiness"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/internal/utils"
)

// ApisixTlsReconciler reconciles a ApisixTls object
type ApisixTlsReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Provider provider.Provider
	Updater  status.Updater
	Readier  readiness.ReadinessManager
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApisixTlsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv2.ApisixTls{},
			builder.WithPredicates(
				predicate.NewPredicateFuncs(r.checkIngressClass),
			),
		).
		WithEventFilter(
			predicate.Or(
				predicate.GenerationChangedPredicate{},
				predicate.AnnotationChangedPredicate{},
				predicate.NewPredicateFuncs(TypePredicate[*corev1.Secret]()),
			),
		).
		Watches(
			&networkingv1.IngressClass{},
			handler.EnqueueRequestsFromMapFunc(r.listApisixTlsForIngressClass),
			builder.WithPredicates(
				predicate.NewPredicateFuncs(matchesIngressController),
			),
		).
		Watches(&v1alpha1.GatewayProxy{},
			handler.EnqueueRequestsFromMapFunc(r.listApisixTlsForGatewayProxy),
		).
		Watches(&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.listApisixTlsForSecret),
		).
		Complete(r)
}

// Reconcile implements the reconciliation logic for ApisixTls
func (r *ApisixTlsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	defer r.Readier.Done(&apiv2.ApisixTls{}, req.NamespacedName)
	var tls apiv2.ApisixTls
	if err := r.Get(ctx, req.NamespacedName, &tls); err != nil {
		if client.IgnoreNotFound(err) == nil {
			// Create a minimal object for deletion
			tls.Namespace = req.Namespace
			tls.Name = req.Name
			tls.TypeMeta = metav1.TypeMeta{
				Kind:       KindApisixTls,
				APIVersion: apiv2.GroupVersion.String(),
			}
			// Delete from provider
			if err := r.Provider.Delete(ctx, &tls); err != nil {
				r.Log.Error(err, "failed to delete TLS from provider")
				return ctrl.Result{}, err
			}
			r.Log.Info("deleted apisix tls", "tls", tls.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	r.Log.Info("reconciling apisix tls", "tls", tls.Name)

	// create a translate context
	tctx := provider.NewDefaultTranslateContext(ctx)

	// get the ingress class
	ingressClass, err := GetIngressClass(tctx, r.Client, r.Log, tls.Spec.IngressClassName)
	if err != nil {
		r.Log.Error(err, "failed to get IngressClass")
		r.updateStatus(&tls, metav1.Condition{
			Type:               string(apiv2.ConditionTypeAccepted),
			Status:             metav1.ConditionFalse,
			ObservedGeneration: tls.Generation,
			LastTransitionTime: metav1.Now(),
			Reason:             string(apiv2.ConditionReasonInvalidSpec),
			Message:            err.Error(),
		})
		return ctrl.Result{}, err
	}

	// process IngressClass parameters if they reference GatewayProxy
	if err := ProcessIngressClassParameters(tctx, r.Client, r.Log, &tls, ingressClass); err != nil {
		r.Log.Error(err, "failed to process IngressClass parameters", "ingressClass", ingressClass.Name)
		r.updateStatus(&tls, metav1.Condition{
			Type:               string(apiv2.ConditionTypeAccepted),
			Status:             metav1.ConditionFalse,
			ObservedGeneration: tls.Generation,
			LastTransitionTime: metav1.Now(),
			Reason:             string(apiv2.ConditionReasonInvalidSpec),
			Message:            err.Error(),
		})
		return ctrl.Result{}, err
	}

	// process ApisixTls validation
	if err := r.processApisixTls(ctx, tctx, &tls); err != nil {
		r.Log.Error(err, "failed to process ApisixTls")
		r.updateStatus(&tls, metav1.Condition{
			Type:               string(apiv2.ConditionTypeAccepted),
			Status:             metav1.ConditionFalse,
			ObservedGeneration: tls.Generation,
			LastTransitionTime: metav1.Now(),
			Reason:             string(apiv2.ConditionReasonInvalidSpec),
			Message:            err.Error(),
		})
		return ctrl.Result{}, err
	}

	if err := r.Provider.Update(ctx, tctx, &tls); err != nil {
		r.Log.Error(err, "failed to sync apisix tls to provider")
		// Update status with failure condition
		r.updateStatus(&tls, metav1.Condition{
			Type:               string(apiv2.ConditionTypeAccepted),
			Status:             metav1.ConditionFalse,
			ObservedGeneration: tls.Generation,
			LastTransitionTime: metav1.Now(),
			Reason:             string(apiv2.ConditionReasonSyncFailed),
			Message:            err.Error(),
		})
		return ctrl.Result{}, err
	}

	// Update status with success condition
	r.updateStatus(&tls, metav1.Condition{
		Type:               string(apiv2.ConditionTypeAccepted),
		Status:             metav1.ConditionTrue,
		ObservedGeneration: tls.Generation,
		LastTransitionTime: metav1.Now(),
		Reason:             string(apiv2.ConditionReasonAccepted),
		Message:            "The apisix tls has been accepted and synced to APISIX",
	})

	return ctrl.Result{}, nil
}

func (r *ApisixTlsReconciler) processApisixTls(ctx context.Context, tc *provider.TranslateContext, tls *apiv2.ApisixTls) error {
	// Validate the main TLS secret
	if err := r.validateSecret(ctx, tc, tls.Spec.Secret); err != nil {
		return fmt.Errorf("invalid apisix tls secret: %w", err)
	}

	// Validate the client CA secret if mutual TLS is configured
	if tls.Spec.Client != nil {
		if err := r.validateSecret(ctx, tc, tls.Spec.Client.CASecret); err != nil {
			return fmt.Errorf("invalid client CA secret: %w", err)
		}
	}

	return nil
}

func (r *ApisixTlsReconciler) validateSecret(ctx context.Context, tc *provider.TranslateContext, secretRef apiv2.ApisixSecret) error {
	secretKey := types.NamespacedName{
		Namespace: secretRef.Namespace,
		Name:      secretRef.Name,
	}

	var secret corev1.Secret
	if err := r.Get(ctx, secretKey, &secret); err != nil {
		return fmt.Errorf("failed to get secret %s: %w", secretKey.String(), err)
	}

	tc.Secrets[secretKey] = &secret
	return nil
}

// updateStatus updates the ApisixTls status with the given condition
func (r *ApisixTlsReconciler) updateStatus(tls *apiv2.ApisixTls, condition metav1.Condition) {
	r.Updater.Update(status.Update{
		NamespacedName: utils.NamespacedName(tls),
		Resource:       &apiv2.ApisixTls{},
		Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
			tlsCopy, ok := obj.(*apiv2.ApisixTls)
			if !ok {
				err := fmt.Errorf("unsupported object type %T", obj)
				panic(err)
			}
			tlsResult := tlsCopy.DeepCopy()
			tlsResult.Status.Conditions = []metav1.Condition{condition}
			return tlsResult
		}),
	})
}

// checkIngressClass checks if the ApisixTls uses the ingress class that we control
func (r *ApisixTlsReconciler) checkIngressClass(obj client.Object) bool {
	tls, ok := obj.(*apiv2.ApisixTls)
	if !ok {
		return false
	}

	return r.matchesIngressClass(tls.Spec.IngressClassName)
}

// matchesIngressClass checks if the given ingress class name matches our controlled classes
func (r *ApisixTlsReconciler) matchesIngressClass(ingressClassName string) bool {
	if ingressClassName == "" {
		// Check for default ingress class
		ingressClassList := &networkingv1.IngressClassList{}
		if err := r.List(context.Background(), ingressClassList, client.MatchingFields{
			indexer.IngressClass: config.GetControllerName(),
		}); err != nil {
			r.Log.Error(err, "failed to list ingress classes")
			return false
		}

		// Find the ingress class that is marked as default
		for _, ic := range ingressClassList.Items {
			if IsDefaultIngressClass(&ic) && matchesController(ic.Spec.Controller) {
				return true
			}
		}
		return false
	}

	// Check if the specified ingress class is controlled by us
	var ingressClass networkingv1.IngressClass
	if err := r.Get(context.Background(), client.ObjectKey{Name: ingressClassName}, &ingressClass); err != nil {
		r.Log.Error(err, "failed to get ingress class", "ingressClass", ingressClassName)
		return false
	}

	return matchesController(ingressClass.Spec.Controller)
}

func (r *ApisixTlsReconciler) listApisixTlsForSecret(ctx context.Context, obj client.Object) []reconcile.Request {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return nil
	}

	return ListRequests(
		ctx,
		r.Client,
		r.Log,
		&apiv2.ApisixConsumerList{},
		client.MatchingFields{
			indexer.SecretIndexRef: indexer.GenIndexKey(secret.GetNamespace(), secret.GetName()),
		},
	)
}

// listApisixTlsForIngressClass list all TLS that use a specific ingress class
func (r *ApisixTlsReconciler) listApisixTlsForIngressClass(ctx context.Context, obj client.Object) []reconcile.Request {
	ingressClass, ok := obj.(*networkingv1.IngressClass)
	if !ok {
		return nil
	}

	return ListMatchingRequests(
		ctx,
		r.Client,
		r.Log,
		&apiv2.ApisixTlsList{},
		func(obj client.Object) bool {
			atls, ok := obj.(*apiv2.ApisixTls)
			if !ok {
				r.Log.Error(fmt.Errorf("expected ApisixTls, got %T", obj), "failed to match object type")
				return false
			}
			return (IsDefaultIngressClass(ingressClass) && atls.Spec.IngressClassName == "") || atls.Spec.IngressClassName == ingressClass.Name
		},
	)
}

// listApisixTlsForGatewayProxy list all TLS that use a specific gateway proxy
func (r *ApisixTlsReconciler) listApisixTlsForGatewayProxy(ctx context.Context, obj client.Object) []reconcile.Request {
	return listIngressClassRequestsForGatewayProxy(ctx, r.Client, obj, r.Log, r.listApisixTlsForIngressClass)
}
