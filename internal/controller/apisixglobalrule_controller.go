// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"context"
	"errors"
	"fmt"

	"github.com/api7/gopkg/pkg/log"
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
	ingressClass, err := r.getIngressClass(&globalRule)
	if err != nil {
		log.Error(err, "failed to get IngressClass")
		return ctrl.Result{}, err
	}

	// process IngressClass parameters if they reference GatewayProxy
	if err := r.processIngressClassParameters(ctx, tctx, &globalRule, ingressClass); err != nil {
		log.Error(err, "failed to process IngressClass parameters", "ingressClass", ingressClass.Name)
		return ctrl.Result{}, err
	}

	// Sync the global rule to APISIX
	if err := r.Provider.Update(ctx, tctx, &globalRule); err != nil {
		log.Error(err, "failed to sync global rule to provider")
		// Update status with failure condition
		r.updateStatus(&globalRule, metav1.Condition{
			Type:               string(gatewayv1.RouteConditionAccepted),
			Status:             metav1.ConditionFalse,
			ObservedGeneration: globalRule.Generation,
			LastTransitionTime: metav1.Now(),
			Reason:             string(apiv2.ReasonSyncFailed),
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
				predicate.NewPredicateFuncs(r.matchesIngressController),
			),
		).
		Watches(&v1alpha1.GatewayProxy{},
			handler.EnqueueRequestsFromMapFunc(r.listGlobalRulesForGatewayProxy),
		).
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

// matchesIngressController check if the ingress class is controlled by us
func (r *ApisixGlobalRuleReconciler) matchesIngressController(obj client.Object) bool {
	ingressClass, ok := obj.(*networkingv1.IngressClass)
	if !ok {
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

	var requests []reconcile.Request

	// List all global rules and filter based on ingress class
	globalRuleList := &apiv2.ApisixGlobalRuleList{}
	if err := r.List(ctx, globalRuleList); err != nil {
		r.Log.Error(err, "failed to list global rules")
		return nil
	}

	isDefaultClass := IsDefaultIngressClass(ingressClass)
	for _, globalRule := range globalRuleList.Items {
		if (isDefaultClass && globalRule.Spec.IngressClassName == "") ||
			globalRule.Spec.IngressClassName == ingressClass.Name {
			requests = append(requests, reconcile.Request{
				NamespacedName: client.ObjectKey{
					Namespace: globalRule.Namespace,
					Name:      globalRule.Name,
				},
			})
		}
	}

	return requests
}

// listGlobalRulesForGatewayProxy list all global rules that use a specific gateway proxy
func (r *ApisixGlobalRuleReconciler) listGlobalRulesForGatewayProxy(ctx context.Context, obj client.Object) []reconcile.Request {
	gatewayProxy, ok := obj.(*v1alpha1.GatewayProxy)
	if !ok {
		return nil
	}

	// Find all ingress classes that reference this gateway proxy
	ingressClassList := &networkingv1.IngressClassList{}
	if err := r.List(ctx, ingressClassList, client.MatchingFields{
		indexer.IngressClassParametersRef: indexer.GenIndexKey(gatewayProxy.GetNamespace(), gatewayProxy.GetName()),
	}); err != nil {
		r.Log.Error(err, "failed to list ingress classes for gateway proxy", "gatewayproxy", gatewayProxy.GetName())
		return nil
	}

	var requests []reconcile.Request
	for _, ingressClass := range ingressClassList.Items {
		requests = append(requests, r.listGlobalRulesForIngressClass(ctx, &ingressClass)...)
	}

	// Remove duplicates
	uniqueRequests := make(map[string]reconcile.Request)
	for _, request := range requests {
		uniqueRequests[request.String()] = request
	}

	distinctRequests := make([]reconcile.Request, 0, len(uniqueRequests))
	for _, request := range uniqueRequests {
		distinctRequests = append(distinctRequests, request)
	}

	return distinctRequests
}

// getIngressClass get the ingress class for the global rule
func (r *ApisixGlobalRuleReconciler) getIngressClass(globalRule *apiv2.ApisixGlobalRule) (*networkingv1.IngressClass, error) {
	if globalRule.Spec.IngressClassName == "" {
		// Check for default ingress class
		ingressClassList := &networkingv1.IngressClassList{}
		if err := r.List(context.Background(), ingressClassList, client.MatchingFields{
			indexer.IngressClass: config.GetControllerName(),
		}); err != nil {
			r.Log.Error(err, "failed to list ingress classes")
			return nil, err
		}

		// Find the ingress class that is marked as default
		for _, ic := range ingressClassList.Items {
			if IsDefaultIngressClass(&ic) && matchesController(ic.Spec.Controller) {
				return &ic, nil
			}
		}
		log.Debugw("no default ingress class found")
		return nil, errors.New("no default ingress class found")
	}

	// Check if the specified ingress class is controlled by us
	var ingressClass networkingv1.IngressClass
	if err := r.Get(context.Background(), client.ObjectKey{Name: globalRule.Spec.IngressClassName}, &ingressClass); err != nil {
		return nil, err
	}

	if matchesController(ingressClass.Spec.Controller) {
		return &ingressClass, nil
	}

	return nil, errors.New("ingress class is not controlled by us")
}

// processIngressClassParameters processes the IngressClass parameters that reference GatewayProxy
func (r *ApisixGlobalRuleReconciler) processIngressClassParameters(ctx context.Context, tctx *provider.TranslateContext, globalRule *apiv2.ApisixGlobalRule, ingressClass *networkingv1.IngressClass) error {
	if ingressClass == nil || ingressClass.Spec.Parameters == nil {
		return nil
	}

	ingressClassKind := utils.NamespacedNameKind(ingressClass)
	globalRuleKind := utils.NamespacedNameKind(globalRule)

	parameters := ingressClass.Spec.Parameters
	// check if the parameters reference GatewayProxy
	if parameters.APIGroup != nil && *parameters.APIGroup == v1alpha1.GroupVersion.Group && parameters.Kind == KindGatewayProxy {
		ns := globalRule.GetNamespace()
		if parameters.Namespace != nil {
			ns = *parameters.Namespace
		}

		gatewayProxy := &v1alpha1.GatewayProxy{}
		if err := r.Get(ctx, client.ObjectKey{
			Namespace: ns,
			Name:      parameters.Name,
		}, gatewayProxy); err != nil {
			r.Log.Error(err, "failed to get GatewayProxy", "namespace", ns, "name", parameters.Name)
			return err
		}

		r.Log.Info("found GatewayProxy for IngressClass", "ingressClass", ingressClass.Name, "gatewayproxy", gatewayProxy.Name)
		tctx.GatewayProxies[ingressClassKind] = *gatewayProxy
		tctx.ResourceParentRefs[globalRuleKind] = append(tctx.ResourceParentRefs[globalRuleKind], ingressClassKind)

		// check if the provider field references a secret
		if gatewayProxy.Spec.Provider != nil && gatewayProxy.Spec.Provider.Type == v1alpha1.ProviderTypeControlPlane {
			if gatewayProxy.Spec.Provider.ControlPlane != nil &&
				gatewayProxy.Spec.Provider.ControlPlane.Auth.Type == v1alpha1.AuthTypeAdminKey &&
				gatewayProxy.Spec.Provider.ControlPlane.Auth.AdminKey != nil &&
				gatewayProxy.Spec.Provider.ControlPlane.Auth.AdminKey.ValueFrom != nil &&
				gatewayProxy.Spec.Provider.ControlPlane.Auth.AdminKey.ValueFrom.SecretKeyRef != nil {

				secretRef := gatewayProxy.Spec.Provider.ControlPlane.Auth.AdminKey.ValueFrom.SecretKeyRef
				secret := &corev1.Secret{}
				if err := r.Get(ctx, client.ObjectKey{
					Namespace: ns,
					Name:      secretRef.Name,
				}, secret); err != nil {
					r.Log.Error(err, "failed to get secret for GatewayProxy provider",
						"namespace", ns,
						"name", secretRef.Name)
					return err
				}

				r.Log.Info("found secret for GatewayProxy provider",
					"ingressClass", ingressClass.Name,
					"gatewayproxy", gatewayProxy.Name,
					"secret", secretRef.Name)

				tctx.Secrets[types.NamespacedName{
					Namespace: ns,
					Name:      secretRef.Name,
				}] = secret
			}
		}
	}

	return nil
}

// updateStatus updates the ApisixGlobalRule status with the given condition
func (r *ApisixGlobalRuleReconciler) updateStatus(globalRule *apiv2.ApisixGlobalRule, condition metav1.Condition) {
	r.Updater.Update(status.Update{
		NamespacedName: NamespacedName(globalRule),
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
