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
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/controller/config"
	"github.com/apache/apisix-ingress-controller/internal/controller/indexer"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/internal/utils"
)

// ApisixGlobalRuleReconciler reconciles a ApisixGlobalRule object
type ApisixGlobalRuleReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Provider provider.Provider
	Updater  status.Updater
}

// Reconcile implements the reconciliation logic for ApisixGlobalRule
func (r *ApisixGlobalRuleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var globalRule apiv2.ApisixGlobalRule
	if err := r.Get(ctx, req.NamespacedName, &globalRule); err != nil {
		if client.IgnoreNotFound(err) == nil {
			// Create a minimal object for deletion
			globalRule.Namespace = req.Namespace
			globalRule.Name = req.Name
			globalRule.TypeMeta = metav1.TypeMeta{
				Kind:       KindApisixGlobalRule,
				APIVersion: apiv2.GroupVersion.String(),
			}
			// Delete from provider
			if err := r.Provider.Delete(ctx, &globalRule); err != nil {
				r.Log.Error(err, "failed to delete global rule from provider")
				return ctrl.Result{}, err
			}
			r.Log.Info("deleted global rule", "globalrule", globalRule.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	r.Log.Info("reconciling global rule", "globalrule", globalRule.Name)

	// create a translate context
	tctx := provider.NewDefaultTranslateContext(ctx)

	// get the ingress class
	ingressClass, err := GetIngressClass(tctx, r.Client, r.Log, globalRule.Spec.IngressClassName)
	if err != nil {
		r.Log.Error(err, "failed to get IngressClass")
		return ctrl.Result{}, err
	}

	// process IngressClass parameters if they reference GatewayProxy
	if err := ProcessIngressClassParameters(tctx, r.Client, r.Log, &globalRule, ingressClass); err != nil {
		r.Log.Error(err, "failed to process IngressClass parameters", "ingressClass", ingressClass.Name)
		return ctrl.Result{}, err
	}

	// Sync the global rule to APISIX
	if err := r.Provider.Update(ctx, tctx, &globalRule); err != nil {
		r.Log.Error(err, "failed to sync global rule to provider")
		// Update status with failure condition
		r.updateStatus(&globalRule, metav1.Condition{
			Type:               string(apiv2.ConditionTypeAccepted),
			Status:             metav1.ConditionFalse,
			ObservedGeneration: globalRule.Generation,
			LastTransitionTime: metav1.Now(),
			Reason:             string(apiv2.ConditionReasonSyncFailed),
			Message:            err.Error(),
		})
		return ctrl.Result{}, err
	}

	// Update status with success condition
	r.updateStatus(&globalRule, metav1.Condition{
		Type:               string(gatewayv1.RouteConditionAccepted),
		Status:             metav1.ConditionTrue,
		ObservedGeneration: globalRule.Generation,
		LastTransitionTime: metav1.Now(),
		Reason:             string(gatewayv1.RouteReasonAccepted),
		Message:            "The global rule has been accepted and synced to APISIX",
	})

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApisixGlobalRuleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv2.ApisixGlobalRule{},
			builder.WithPredicates(
				predicate.NewPredicateFuncs(r.checkIngressClass),
			),
		).
		WithEventFilter(
			predicate.Or(
				predicate.GenerationChangedPredicate{},
				predicate.AnnotationChangedPredicate{},
			),
		).
		Watches(
			&networkingv1.IngressClass{},
			handler.EnqueueRequestsFromMapFunc(r.listGlobalRulesForIngressClass),
			builder.WithPredicates(
				predicate.NewPredicateFuncs(matchesIngressController),
			),
		).
		Watches(&v1alpha1.GatewayProxy{},
			handler.EnqueueRequestsFromMapFunc(r.listGlobalRulesForGatewayProxy),
		).
		Named("apisixglobalrule").
		Complete(r)
}

// checkIngressClass checks if the ApisixGlobalRule uses the ingress class that we control
func (r *ApisixGlobalRuleReconciler) checkIngressClass(obj client.Object) bool {
	globalRule, ok := obj.(*apiv2.ApisixGlobalRule)
	if !ok {
		return false
	}

	return r.matchesIngressClass(globalRule.Spec.IngressClassName)
}

// matchesIngressClass checks if the given ingress class name matches our controlled classes
func (r *ApisixGlobalRuleReconciler) matchesIngressClass(ingressClassName string) bool {
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

// listGlobalRulesForIngressClass list all global rules that use a specific ingress class
func (r *ApisixGlobalRuleReconciler) listGlobalRulesForIngressClass(ctx context.Context, obj client.Object) []reconcile.Request {
	ingressClass, ok := obj.(*networkingv1.IngressClass)
	if !ok {
		return nil
	}

	return ListMatchingRequests(
		ctx,
		r.Client,
		r.Log,
		&apiv2.ApisixGlobalRuleList{},
		func(obj client.Object) bool {
			agr, ok := obj.(*apiv2.ApisixGlobalRule)
			if !ok {
				r.Log.Error(fmt.Errorf("expected ApisixGlobalRule, got %T", obj), "failed to match object type")
				return false
			}
			return (IsDefaultIngressClass(ingressClass) && agr.Spec.IngressClassName == "") || agr.Spec.IngressClassName == ingressClass.Name
		},
	)
}

func (r *ApisixGlobalRuleReconciler) listGlobalRulesForGatewayProxy(ctx context.Context, obj client.Object) []reconcile.Request {
	return listIngressClassRequestsForGatewayProxy(ctx, r.Client, obj, r.Log, r.listGlobalRulesForIngressClass)
}

// updateStatus updates the ApisixGlobalRule status with the given condition
func (r *ApisixGlobalRuleReconciler) updateStatus(globalRule *apiv2.ApisixGlobalRule, condition metav1.Condition) {
	r.Updater.Update(status.Update{
		NamespacedName: utils.NamespacedName(globalRule),
		Resource:       &apiv2.ApisixGlobalRule{},
		Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
			gr, ok := obj.(*apiv2.ApisixGlobalRule)
			if !ok {
				err := fmt.Errorf("unsupported object type %T", obj)
				panic(err)
			}
			grCopy := gr.DeepCopy()
			grCopy.Status.Conditions = []metav1.Condition{condition}
			return grCopy
		}),
	})
}
