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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/controller/indexer"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	"github.com/apache/apisix-ingress-controller/internal/manager/readiness"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/internal/types"
	"github.com/apache/apisix-ingress-controller/internal/utils"
)

// ApisixGlobalRuleReconciler reconciles a ApisixGlobalRule object
type ApisixGlobalRuleReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Provider provider.Provider
	Updater  status.Updater

	Readier readiness.ReadinessManager
}

// Reconcile implements the reconciliation logic for ApisixGlobalRule
func (r *ApisixGlobalRuleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	defer r.Readier.Done(&apiv2.ApisixGlobalRule{}, req.NamespacedName)
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

	r.Log.V(1).Info("reconciling ApisixGlobalRule", "object", globalRule)

	// create a translate context
	tctx := provider.NewDefaultTranslateContext(ctx)

	// get the ingress class
	ingressClass, err := FindMatchingIngressClass(tctx, r.Client, r.Log, &globalRule)
	if err != nil {
		r.Log.V(1).Info("no matching IngressClass available",
			"ingressClassName", globalRule.Spec.IngressClassName,
			"error", err.Error())
		if err := r.Provider.Delete(ctx, &globalRule); err != nil {
			r.Log.Error(err, "failed to delete global rule from provider")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// process IngressClass parameters if they reference GatewayProxy
	if err := ProcessIngressClassParameters(tctx, r.Client, r.Log, &globalRule, ingressClass); err != nil {
		r.Log.Error(err, "failed to process IngressClass parameters", "ingressClass", ingressClass.Name)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Validate plugins and their secrets
	if err := r.validatePlugins(tctx, &globalRule, globalRule.Spec.Plugins); err != nil {
		r.Log.Error(err, "failed to validate plugins")
		// Update status with failure condition
		r.updateStatus(&globalRule, metav1.Condition{
			Type:               string(apiv2.ConditionTypeAccepted),
			Status:             metav1.ConditionFalse,
			ObservedGeneration: globalRule.Generation,
			LastTransitionTime: metav1.Now(),
			Reason:             string(apiv2.ConditionReasonInvalidSpec),
			Message:            err.Error(),
		})
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
				MatchesIngressClassPredicate(r.Client, r.Log),
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
			handler.EnqueueRequestsFromMapFunc(r.listGlobalRulesForIngressClass),
			builder.WithPredicates(
				predicate.NewPredicateFuncs(matchesIngressController),
			),
		).
		Watches(&v1alpha1.GatewayProxy{},
			handler.EnqueueRequestsFromMapFunc(r.listGlobalRulesForGatewayProxy),
		).
		Watches(&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.listGlobalRulesForSecret),
		).
		Named("apisixglobalrule").
		Complete(r)
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

func (r *ApisixGlobalRuleReconciler) listGlobalRulesForSecret(ctx context.Context, obj client.Object) []reconcile.Request {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return nil
	}

	return ListRequests(
		ctx,
		r.Client,
		r.Log,
		&apiv2.ApisixGlobalRuleList{},
		client.MatchingFields{
			indexer.SecretIndexRef: indexer.GenIndexKey(secret.GetNamespace(), secret.GetName()),
		},
	)
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

// validatePlugins validates plugins and their secret references
func (r *ApisixGlobalRuleReconciler) validatePlugins(tctx *provider.TranslateContext, in *apiv2.ApisixGlobalRule, plugins []apiv2.ApisixRoutePlugin) error {
	// check secret
	for _, plugin := range plugins {
		if !plugin.Enable {
			continue
		}
		// check secret
		if err := r.validateSecrets(tctx, in, plugin.SecretRef); err != nil {
			return err
		}
	}
	return nil
}

// validateSecrets validates that the secret exists and adds it to the translate context
func (r *ApisixGlobalRuleReconciler) validateSecrets(tctx *provider.TranslateContext, in *apiv2.ApisixGlobalRule, secretRef string) error {
	if secretRef == "" {
		return nil
	}
	var (
		secret = corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretRef,
				Namespace: in.Namespace,
			},
		}
		secretNN = utils.NamespacedName(&secret)
	)
	if err := r.Get(tctx, secretNN, &secret); err != nil {
		return types.ReasonError{
			Reason:  string(apiv2.ConditionReasonInvalidSpec),
			Message: fmt.Sprintf("failed to get Secret: %s", secretNN),
		}
	}

	tctx.Secrets[utils.NamespacedName(&secret)] = &secret
	return nil
}
