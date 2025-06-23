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
	"cmp"
	"context"
	"fmt"
	"slices"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	"sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/controller/indexer"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/internal/utils"
)

func (r *HTTPRouteReconciler) processHTTPRoutePolicies(tctx *provider.TranslateContext, httpRoute *gatewayv1.HTTPRoute) error {
	// list HTTPRoutePolicies, which sectionName is not specified
	var (
		list v1alpha1.HTTPRoutePolicyList
		key  = indexer.GenIndexKeyWithGK(gatewayv1.GroupName, "HTTPRoute", httpRoute.GetNamespace(), httpRoute.GetName())
	)
	if err := r.List(context.Background(), &list, client.MatchingFields{indexer.PolicyTargetRefs: key}); err != nil {
		return err
	}

	if len(list.Items) == 0 {
		return nil
	}

	// set namespace in prentRef if it is not set
	var parentRefs = make([]gatewayv1.ParentReference, len(httpRoute.Spec.ParentRefs))
	for i := range httpRoute.Spec.ParentRefs {
		ref := httpRoute.Spec.ParentRefs[i]
		if ref.Namespace == nil || *ref.Namespace == "" {
			ref.Namespace = (*gatewayv1.Namespace)(&httpRoute.Namespace)
		}
		parentRefs[i] = ref
	}

	var conflicts = make(map[types.NamespacedName]v1alpha1.HTTPRoutePolicy)
	for _, rule := range httpRoute.Spec.Rules {
		var policies = findPoliciesWhichTargetRefTheRule(rule.Name, "HTTPRoute", list)
		if conflict := checkPoliciesConflict(policies); conflict {
			for _, policy := range policies {
				namespacedName := types.NamespacedName{Namespace: policy.GetNamespace(), Name: policy.GetName()}
				conflicts[namespacedName] = policy
			}
		}
	}

	for i := range list.Items {
		var (
			policy         = list.Items[i]
			namespacedName = types.NamespacedName{Namespace: policy.GetNamespace(), Name: policy.GetName()}
			condition      metav1.Condition
		)
		if _, conflict := conflicts[namespacedName]; conflict {
			condition.Status = metav1.ConditionFalse
			condition.Reason = string(v1alpha2.PolicyReasonConflicted)
			condition.Message = "HTTPRoutePolicy conflict with others target to the HTTPRoute"
		} else {
			tctx.HTTPRoutePolicies = append(tctx.HTTPRoutePolicies, policy)
		}

		if updated := setAncestorsForHTTPRoutePolicyStatus(parentRefs, &policy, condition); updated {
			tctx.StatusUpdaters = append(tctx.StatusUpdaters, status.Update{
				NamespacedName: utils.NamespacedName(&policy),
				Resource:       policy.DeepCopy(),
				Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
					t, ok := obj.(*v1alpha1.HTTPRoutePolicy)
					if !ok {
						err := fmt.Errorf("unsupported object type %T", obj)
						panic(err)
					}
					t.Status = policy.Status
					return t
				}),
			})
		}
	}

	return nil
}

func (r *HTTPRouteReconciler) updateHTTPRoutePolicyStatusOnDeleting(ctx context.Context, nn types.NamespacedName) error {
	var (
		list v1alpha1.HTTPRoutePolicyList
		key  = indexer.GenIndexKeyWithGK(gatewayv1.GroupName, "HTTPRoute", nn.Namespace, nn.Name)
	)
	if err := r.List(ctx, &list, client.MatchingFields{indexer.PolicyTargetRefs: key}); err != nil {
		return err
	}
	var (
		httpRoutes = make(map[types.NamespacedName]gatewayv1.HTTPRoute)
	)
	for _, policy := range list.Items {
		// collect all parentRefs for the HTTPRoutePolicy
		var parentRefs []gatewayv1.ParentReference
		for _, ref := range policy.Spec.TargetRefs {
			var namespacedName = types.NamespacedName{Namespace: policy.GetNamespace(), Name: string(ref.Name)}
			httpRoute, ok := httpRoutes[namespacedName]
			if !ok {
				if err := r.Get(ctx, namespacedName, &httpRoute); err != nil {
					continue
				}
				httpRoutes[namespacedName] = httpRoute
			}
			parentRefs = append(parentRefs, httpRoute.Spec.ParentRefs...)
		}
		// delete AncestorRef which is not exist in the all parentRefs for each policy
		updateDeleteAncestors(r.Updater, policy, parentRefs)
	}

	return nil
}

func (r *IngressReconciler) processHTTPRoutePolicies(tctx *provider.TranslateContext, ingress *networkingv1.Ingress) error {
	var (
		list v1alpha1.HTTPRoutePolicyList
		key  = indexer.GenIndexKeyWithGK(networkingv1.GroupName, "Ingress", ingress.GetNamespace(), ingress.GetName())
	)
	if err := r.List(context.Background(), &list, client.MatchingFields{indexer.PolicyTargetRefs: key}); err != nil {
		return err
	}

	if len(list.Items) == 0 {
		return nil
	}

	var (
		condition metav1.Condition
	)
	if conflict := checkPoliciesConflict(list.Items); conflict {
		condition.Status = metav1.ConditionFalse
		condition.Reason = string(v1alpha2.PolicyReasonConflicted)
		condition.Message = "HTTPRoutePolicy conflict with others target to the Ingress"
	} else {
		tctx.HTTPRoutePolicies = list.Items
	}

	for i := range list.Items {
		policy := list.Items[i]
		if updated := setAncestorsForHTTPRoutePolicyStatus(tctx.RouteParentRefs, &policy, condition); updated {
			tctx.StatusUpdaters = append(tctx.StatusUpdaters, status.Update{
				NamespacedName: utils.NamespacedName(&policy),
				Resource:       policy.DeepCopy(),
				Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
					t, ok := obj.(*v1alpha1.HTTPRoutePolicy)
					if !ok {
						err := fmt.Errorf("unsupported object type %T", obj)
						panic(err)
					}
					t.Status = policy.Status
					return t
				}),
			})
		}
	}

	return nil
}

func (r *IngressReconciler) updateHTTPRoutePolicyStatusOnDeleting(ctx context.Context, nn types.NamespacedName) error {
	var (
		list v1alpha1.HTTPRoutePolicyList
		key  = indexer.GenIndexKeyWithGK(networkingv1.GroupName, "Ingress", nn.Namespace, nn.Name)
	)
	if err := r.List(ctx, &list, client.MatchingFields{indexer.PolicyTargetRefs: key}); err != nil {
		return err
	}
	var (
		ingress2ParentRef = make(map[types.NamespacedName]gatewayv1.ParentReference)
	)
	for _, policy := range list.Items {
		// collect all parentRefs for the HTTPRoutePolicy
		var parentRefs []gatewayv1.ParentReference
		for _, ref := range policy.Spec.TargetRefs {
			var namespacedName = types.NamespacedName{Namespace: policy.GetNamespace(), Name: string(ref.Name)}
			parentRef, ok := ingress2ParentRef[namespacedName]
			if !ok {
				var ingress networkingv1.Ingress
				if err := r.Get(ctx, namespacedName, &ingress); err != nil {
					continue
				}
				ingressClass, err := r.getIngressClass(ctx, &ingress)
				if err != nil {
					continue
				}
				parentRef = gatewayv1.ParentReference{
					Group: ptr.To(gatewayv1.Group(ingressClass.GroupVersionKind().Group)),
					Kind:  ptr.To(gatewayv1.Kind(KindIngressClass)),
					Name:  gatewayv1.ObjectName(ingressClass.Name),
				}
				ingress2ParentRef[namespacedName] = parentRef
			}
			parentRefs = append(parentRefs, parentRef)
		}
		// delete AncestorRef which is not exist in the all parentRefs
		updateDeleteAncestors(r.Updater, policy, parentRefs)
	}

	return nil
}

func setAncestorsForHTTPRoutePolicyStatus(parentRefs []gatewayv1.ParentReference, policy *v1alpha1.HTTPRoutePolicy, condition metav1.Condition) bool {
	return SetAncestors(&policy.Status, parentRefs, metav1.Condition{
		Type:               cmp.Or(condition.Type, string(v1alpha2.PolicyConditionAccepted)),
		Status:             cmp.Or(condition.Status, metav1.ConditionTrue),
		ObservedGeneration: policy.GetGeneration(),
		LastTransitionTime: metav1.Now(),
		Reason:             cmp.Or(condition.Reason, string(v1alpha2.PolicyReasonAccepted)),
		Message:            condition.Message,
	})
}

// checkPoliciesConflict determines if there is a conflict among the given HTTPRoutePolicy objects based on their priority values.
// It returns true if any policy has a different priority than the first policy in the list, otherwise false.
// An empty or single-element policies slice is considered non-conflicting.
// The function assumes all policies have a valid Spec.Priority field for comparison.
func checkPoliciesConflict(policies []v1alpha1.HTTPRoutePolicy) bool {
	if len(policies) == 0 {
		return false
	}
	priority := policies[0].Spec.Priority
	for _, policy := range policies {
		if !ptr.Equal(policy.Spec.Priority, priority) {
			return true
		}
	}
	return false
}

// findPoliciesWhichTargetRefTheRule filters HTTPRoutePolicy objects whose TargetRefs match the given ruleName and kind.
// A match occurs if the TargetRef's Kind equals the provided kind and its SectionName is nil, empty, or equal to ruleName.
func findPoliciesWhichTargetRefTheRule(ruleName *gatewayv1.SectionName, kind string, list v1alpha1.HTTPRoutePolicyList) (policies []v1alpha1.HTTPRoutePolicy) {
	for _, policy := range list.Items {
		for _, ref := range policy.Spec.TargetRefs {
			if string(ref.Kind) == kind && (ref.SectionName == nil || *ref.SectionName == "" || ptr.Equal(ref.SectionName, ruleName)) {
				policies = append(policies, policy)
				break
			}
		}
	}
	return
}

// updateDeleteAncestors removes ancestor references from HTTPRoutePolicy statuses that are no longer present in the provided parentRefs.
func updateDeleteAncestors(updater status.Updater, policy v1alpha1.HTTPRoutePolicy, parentRefs []gatewayv1.ParentReference) {
	length := len(policy.Status.Ancestors)
	policy.Status.Ancestors = slices.DeleteFunc(policy.Status.Ancestors, func(ancestor v1alpha2.PolicyAncestorStatus) bool {
		return !slices.ContainsFunc(parentRefs, func(ref gatewayv1.ParentReference) bool {
			return parentRefValueEqual(ancestor.AncestorRef, ref)
		})
	})
	if length != len(policy.Status.Ancestors) {
		updater.Update(status.Update{
			NamespacedName: utils.NamespacedName(&policy),
			Resource:       policy.DeepCopy(),
			Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
				t, ok := obj.(*v1alpha1.HTTPRoutePolicy)
				if !ok {
					err := fmt.Errorf("unsupported object type %T", obj)
					panic(err)
				}
				t.Status = policy.Status
				return t
			}),
		})
	}
}
