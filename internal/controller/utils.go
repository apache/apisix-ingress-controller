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
	"encoding/pem"
	"errors"
	"fmt"
	"path"
	"reflect"
	"slices"
	"strings"

	"github.com/api7/gopkg/pkg/log"
	"github.com/samber/lo"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	"sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/controller/config"
	"github.com/apache/apisix-ingress-controller/internal/provider"
)

const (
	KindGateway      = "Gateway"
	KindHTTPRoute    = "HTTPRoute"
	KindGatewayClass = "GatewayClass"
	KindIngress      = "Ingress"
	KindIngressClass = "IngressClass"
	KindGatewayProxy = "GatewayProxy"
	KindSecret       = "Secret"
)

const defaultIngressClassAnnotation = "ingressclass.kubernetes.io/is-default-class"

var (
	ErrNoMatchingListenerHostname = errors.New("no matching hostnames in listener")
)

// IsDefaultIngressClass returns whether an IngressClass is the default IngressClass.
func IsDefaultIngressClass(obj client.Object) bool {
	if ingressClass, ok := obj.(*networkingv1.IngressClass); ok {
		return ingressClass.Annotations[defaultIngressClassAnnotation] == "true"
	}
	return false
}

func acceptedMessage(kind string) string {
	return fmt.Sprintf("the %s has been accepted by the apisix-ingress-controller", kind)
}

func MergeCondition(conditions []metav1.Condition, newCondition metav1.Condition) []metav1.Condition {
	if newCondition.LastTransitionTime.IsZero() {
		newCondition.LastTransitionTime = metav1.Now()
	}
	newConditions := []metav1.Condition{}
	for _, condition := range conditions {
		if condition.Type != newCondition.Type {
			newConditions = append(newConditions, condition)
		}
	}
	newConditions = append(newConditions, newCondition)
	return newConditions
}

func setGatewayCondition(gw *gatewayv1.Gateway, newCondition metav1.Condition) {
	gw.Status.Conditions = MergeCondition(gw.Status.Conditions, newCondition)
}

func setListenerCondition(gw *gatewayv1.Gateway, listenerName string, newCondition metav1.Condition) {
	for i, listener := range gw.Status.Listeners {
		if listener.Name == gatewayv1.SectionName(listenerName) {
			gw.Status.Listeners[i].Conditions = MergeCondition(listener.Conditions, newCondition)
			return
		}
	}
}

func reconcileGatewaysMatchGatewayClass(gatewayClass client.Object, gateways []gatewayv1.Gateway) (recs []reconcile.Request) {
	for _, gateway := range gateways {
		if string(gateway.Spec.GatewayClassName) == gatewayClass.GetName() {
			recs = append(recs, reconcile.Request{
				NamespacedName: client.ObjectKey{
					Name:      gateway.GetName(),
					Namespace: gateway.GetNamespace(),
				},
			})
		}
	}
	return
}

func IsConditionPresentAndEqual(conditions []metav1.Condition, condition metav1.Condition) bool {
	for _, cond := range conditions {
		if cond.Type == condition.Type &&
			cond.Reason == condition.Reason &&
			cond.Status == condition.Status &&
			cond.ObservedGeneration == condition.ObservedGeneration {
			return true
		}
	}
	return false
}

func SetGatewayConditionAccepted(gw *gatewayv1.Gateway, status bool, message string) (ok bool) {
	condition := metav1.Condition{
		Type:               string(gatewayv1.GatewayConditionAccepted),
		Status:             ConditionStatus(status),
		Reason:             string(gatewayv1.GatewayReasonAccepted),
		ObservedGeneration: gw.GetGeneration(),
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}

	if !IsConditionPresentAndEqual(gw.Status.Conditions, condition) {
		setGatewayCondition(gw, condition)
		ok = true
	}
	return
}

func SetGatewayListenerConditionAccepted(gw *gatewayv1.Gateway, listenerName string, status bool, message string) (ok bool) {
	conditionStatus := metav1.ConditionTrue
	if !status {
		conditionStatus = metav1.ConditionFalse
	}

	condition := metav1.Condition{
		Type:               string(gatewayv1.ListenerConditionAccepted),
		Status:             conditionStatus,
		Reason:             string(gatewayv1.ListenerConditionAccepted),
		ObservedGeneration: gw.GetGeneration(),
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}

	if !IsConditionPresentAndEqual(gw.Status.Conditions, condition) {
		setListenerCondition(gw, listenerName, condition)
		ok = true
	}
	return
}

func SetGatewayListenerConditionProgrammed(gw *gatewayv1.Gateway, listenerName string, status bool, message string) (ok bool) {
	condition := metav1.Condition{
		Type:               string(gatewayv1.ListenerConditionProgrammed),
		Status:             ConditionStatus(status),
		Reason:             string(gatewayv1.ListenerReasonProgrammed),
		ObservedGeneration: gw.GetGeneration(),
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}

	if !IsConditionPresentAndEqual(gw.Status.Conditions, condition) {
		setListenerCondition(gw, listenerName, condition)
		ok = true
	}
	return
}

func SetGatewayListenerConditionResolvedRefs(gw *gatewayv1.Gateway, listenerName string, status bool, message string) (ok bool) {
	condition := metav1.Condition{
		Type:               string(gatewayv1.ListenerConditionResolvedRefs),
		Status:             ConditionStatus(status),
		Reason:             string(gatewayv1.ListenerReasonResolvedRefs),
		ObservedGeneration: gw.GetGeneration(),
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}

	if !IsConditionPresentAndEqual(gw.Status.Conditions, condition) {
		setListenerCondition(gw, listenerName, condition)
		ok = true
	}
	return
}

func SetGatewayConditionProgrammed(gw *gatewayv1.Gateway, status bool, message string) (ok bool) {
	condition := metav1.Condition{
		Type:               string(gatewayv1.GatewayConditionProgrammed),
		Status:             ConditionStatus(status),
		Reason:             string(gatewayv1.GatewayReasonProgrammed),
		ObservedGeneration: gw.GetGeneration(),
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}

	if !IsConditionPresentAndEqual(gw.Status.Conditions, condition) {
		setGatewayCondition(gw, condition)
		ok = true
	}
	return
}

func ConditionStatus(status bool) metav1.ConditionStatus {
	if status {
		return metav1.ConditionTrue
	}
	return metav1.ConditionFalse
}

func SetRouteConditionAccepted(routeParentStatus *gatewayv1.RouteParentStatus, generation int64, status bool, message string) {
	condition := metav1.Condition{
		Type:               string(gatewayv1.RouteConditionAccepted),
		Status:             ConditionStatus(status),
		Reason:             string(gatewayv1.RouteReasonAccepted),
		ObservedGeneration: generation,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}
	if message == ErrNoMatchingListenerHostname.Error() {
		condition.Reason = string(gatewayv1.RouteReasonNoMatchingListenerHostname)
	}

	if !IsConditionPresentAndEqual(routeParentStatus.Conditions, condition) && !slices.ContainsFunc(routeParentStatus.Conditions, func(item metav1.Condition) bool {
		return item.Type == condition.Type && item.Status == metav1.ConditionFalse && condition.Status == metav1.ConditionTrue
	}) {
		routeParentStatus.Conditions = MergeCondition(routeParentStatus.Conditions, condition)
	}
}

// SetRouteConditionResolvedRefs sets the ResolvedRefs condition with proper reason based on error type
func SetRouteConditionResolvedRefs(routeParentStatus *gatewayv1.RouteParentStatus, generation int64, err error) {
	var (
		reason  string
		status  = metav1.ConditionTrue
		message = "backendRefs are resolved"
	)

	if err != nil {
		status = metav1.ConditionFalse
		message = err.Error()
		reason = string(gatewayv1.RouteReasonResolvedRefs)

		if IsInvalidKindError(err) {
			reason = string(gatewayv1.RouteReasonInvalidKind)
		} else if IsBackendNotFoundError(err) {
			reason = string(gatewayv1.RouteReasonBackendNotFound)
		}
	} else {
		reason = string(gatewayv1.RouteReasonResolvedRefs)
	}

	condition := metav1.Condition{
		Type:               string(gatewayv1.RouteConditionResolvedRefs),
		Status:             status,
		Reason:             reason,
		ObservedGeneration: generation,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}

	if !IsConditionPresentAndEqual(routeParentStatus.Conditions, condition) {
		routeParentStatus.Conditions = MergeCondition(routeParentStatus.Conditions, condition)
	}
}

func SetRouteParentRef(routeParentStatus *gatewayv1.RouteParentStatus, gatewayName string, namespace string) {
	kind := gatewayv1.Kind(KindGateway)
	group := gatewayv1.Group(gatewayv1.GroupName)
	ns := gatewayv1.Namespace(namespace)
	routeParentStatus.ParentRef = gatewayv1.ParentReference{
		Kind:      &kind,
		Group:     &group,
		Name:      gatewayv1.ObjectName(gatewayName),
		Namespace: &ns,
	}
	routeParentStatus.ControllerName = gatewayv1.GatewayController(config.ControllerConfig.ControllerName)
}

func ParseRouteParentRefs(
	ctx context.Context, mgrc client.Client, route client.Object, parentRefs []gatewayv1.ParentReference,
) ([]RouteParentRefContext, error) {
	gateways := make([]RouteParentRefContext, 0)
	for _, parentRef := range parentRefs {
		namespace := route.GetNamespace()
		if parentRef.Namespace != nil {
			namespace = string(*parentRef.Namespace)
		}
		name := string(parentRef.Name)

		if parentRef.Kind != nil && *parentRef.Kind != KindGateway {
			continue
		}

		gateway := gatewayv1.Gateway{}
		if err := mgrc.Get(ctx, client.ObjectKey{
			Namespace: namespace,
			Name:      name,
		}, &gateway); err != nil {
			if client.IgnoreNotFound(err) == nil {
				continue
			}
			return nil, fmt.Errorf("failed to retrieve gateway for route: %w", err)
		}

		gatewayClass := gatewayv1.GatewayClass{}
		if err := mgrc.Get(ctx, client.ObjectKey{
			Name: string(gateway.Spec.GatewayClassName),
		}, &gatewayClass); err != nil {
			if client.IgnoreNotFound(err) == nil {
				continue
			}
			return nil, fmt.Errorf("failed to retrieve gatewayclass for gateway: %w", err)
		}

		if string(gatewayClass.Spec.ControllerName) != config.ControllerConfig.ControllerName {
			continue
		}

		matched := false
		reason := gatewayv1.RouteReasonNoMatchingParent
		var listenerName string

		for _, listener := range gateway.Spec.Listeners {
			if parentRef.SectionName != nil {
				if *parentRef.SectionName != "" && *parentRef.SectionName != listener.Name {
					continue
				}
			}

			if parentRef.Port != nil {
				if *parentRef.Port != listener.Port {
					continue
				}
			}

			if !routeMatchesListenerType(route, listener) {
				continue
			}

			if !routeHostnamesIntersectsWithListenerHostname(route, listener) {
				reason = gatewayv1.RouteReasonNoMatchingListenerHostname
				continue
			}

			listenerName = string(listener.Name)
			ok, err := routeMatchesListenerAllowedRoutes(ctx, mgrc, route, listener.AllowedRoutes, gateway.Namespace, parentRef.Namespace)
			if err != nil {
				log.Warnf("failed matching listener %s to a route %s for gateway %s: %v",
					listener.Name, route.GetName(), gateway.Name, err,
				)
			}
			if !ok {
				reason = gatewayv1.RouteReasonNotAllowedByListeners
				continue
			}

			// TODO: check if the listener status is programmed

			matched = true
			break
		}

		if matched {
			gateways = append(gateways, RouteParentRefContext{
				Gateway:      &gateway,
				ListenerName: listenerName,
				Conditions: []metav1.Condition{{
					Type:               string(gatewayv1.RouteConditionAccepted),
					Status:             metav1.ConditionTrue,
					Reason:             string(gatewayv1.RouteReasonAccepted),
					ObservedGeneration: route.GetGeneration(),
				}},
			})
		} else {
			gateways = append(gateways, RouteParentRefContext{
				Gateway:      &gateway,
				ListenerName: listenerName,
				Conditions: []metav1.Condition{{
					Type:               string(gatewayv1.RouteConditionAccepted),
					Status:             metav1.ConditionFalse,
					Reason:             string(reason),
					ObservedGeneration: route.GetGeneration(),
				}},
			})
		}
	}

	return gateways, nil
}

func checkRouteAcceptedByListener(
	ctx context.Context,
	mgrc client.Client,
	route client.Object,
	gateway gatewayv1.Gateway,
	listener gatewayv1.Listener,
	parentRef gatewayv1.ParentReference,
) (bool, gatewayv1.RouteConditionReason, error) {
	if parentRef.SectionName != nil {
		if *parentRef.SectionName != "" && *parentRef.SectionName != listener.Name {
			return false, gatewayv1.RouteReasonNoMatchingParent, nil
		}
	}
	if parentRef.Port != nil {
		if *parentRef.Port != listener.Port {
			return false, gatewayv1.RouteReasonNoMatchingParent, nil
		}
	}
	if !routeMatchesListenerType(route, listener) {
		return false, gatewayv1.RouteReasonNoMatchingParent, nil
	}
	if !routeHostnamesIntersectsWithListenerHostname(route, listener) {
		return false, gatewayv1.RouteReasonNoMatchingListenerHostname, nil
	}
	if ok, err := routeMatchesListenerAllowedRoutes(ctx, mgrc, route, listener.AllowedRoutes, gateway.Namespace, parentRef.Namespace); err != nil {
		return false, gatewayv1.RouteReasonNotAllowedByListeners, fmt.Errorf("failed matching listener %s to a route %s for gateway %s: %w",
			listener.Name, route.GetName(), gateway.Name, err,
		)
	} else if !ok {
		return false, gatewayv1.RouteReasonNotAllowedByListeners, nil
	}
	return true, gatewayv1.RouteReasonAccepted, nil
}

func routeHostnamesIntersectsWithListenerHostname(route client.Object, listener gatewayv1.Listener) bool {
	switch r := route.(type) {
	case *gatewayv1.HTTPRoute:
		return listenerHostnameIntersectWithRouteHostnames(listener, r.Spec.Hostnames)
	default:
		return false
	}
}

func listenerHostnameIntersectWithRouteHostnames(listener gatewayv1.Listener, hostnames []gatewayv1.Hostname) bool {
	if len(hostnames) == 0 {
		return true
	}

	// if the listener has no hostname, all hostnames automatically intersect
	if listener.Hostname == nil || *listener.Hostname == "" {
		return true
	}

	// iterate over all the hostnames and check that at least one intersect with the listener hostname
	for _, hostname := range hostnames {
		if HostnamesIntersect(string(*listener.Hostname), string(hostname)) {
			return true
		}
	}

	return false
}

func HostnamesIntersect(a, b string) bool {
	return HostnamesMatch(a, b) || HostnamesMatch(b, a)
}

// HostnamesMatch checks that the hostnameB matches the hostnameA. HostnameA is treated as mask
// to be checked against the hostnameB.
func HostnamesMatch(hostnameA, hostnameB string) bool {
	// the hostnames are in the form of "foo.bar.com"; split them
	// in a slice of substrings
	hostnameALabels := strings.Split(hostnameA, ".")
	hostnameBLabels := strings.Split(hostnameB, ".")

	var a, b int
	var wildcard bool

	// iterate over the parts of both the hostnames
	for a, b = 0, 0; a < len(hostnameALabels) && b < len(hostnameBLabels); a, b = a+1, b+1 {
		var matchFound bool

		// if the current part of B is a wildcard, we need to find the first
		// A part that matches with the following B part
		if wildcard {
			for ; b < len(hostnameBLabels); b++ {
				if hostnameALabels[a] == hostnameBLabels[b] {
					matchFound = true
					break
				}
			}
		}

		// if no match was found, the hostnames don't match
		if wildcard && !matchFound {
			return false
		}

		// check if at least on of the current parts are a wildcard; if so, continue
		if hostnameALabels[a] == "*" {
			wildcard = true
			continue
		}
		// reset the wildcard  variables
		wildcard = false

		// if the current a part is different from the b part, the hostnames are incompatible
		if hostnameALabels[a] != hostnameBLabels[b] {
			return false
		}
	}
	return len(hostnameBLabels)-b == len(hostnameALabels)-a
}

func routeMatchesListenerAllowedRoutes(
	ctx context.Context,
	mgrc client.Client,
	route client.Object,
	allowedRoutes *gatewayv1.AllowedRoutes,
	gatewayNamespace string,
	parentRefNamespace *gatewayv1.Namespace,
) (bool, error) {
	if allowedRoutes == nil {
		return true, nil
	}

	if !isRouteKindAllowed(route, allowedRoutes.Kinds) {
		return false, fmt.Errorf("route %s/%s is not allowed in the kind", route.GetNamespace(), route.GetName())
	}

	if !isRouteNamespaceAllowed(ctx, route, mgrc, gatewayNamespace, parentRefNamespace, allowedRoutes.Namespaces) {
		return false, fmt.Errorf("route %s/%s is not allowed in the namespace", route.GetNamespace(), route.GetName())
	}

	return true, nil
}

func isRouteKindAllowed(route client.Object, kinds []gatewayv1.RouteGroupKind) (ok bool) {
	ok = true
	if len(kinds) > 0 {
		_, ok = lo.Find(kinds, func(rgk gatewayv1.RouteGroupKind) bool {
			gvk := route.GetObjectKind().GroupVersionKind()
			return (rgk.Group != nil && string(*rgk.Group) == gvk.Group) && string(rgk.Kind) == gvk.Kind
		})
	}
	return
}

func isRouteNamespaceAllowed(
	ctx context.Context,
	route client.Object,
	mgrc client.Client,
	gatewayNamespace string,
	parentRefNamespace *gatewayv1.Namespace,
	routeNamespaces *gatewayv1.RouteNamespaces,
) bool {
	if routeNamespaces == nil || routeNamespaces.From == nil {
		return true
	}

	switch *routeNamespaces.From {
	case gatewayv1.NamespacesFromAll:
		return true

	case gatewayv1.NamespacesFromSame:
		if parentRefNamespace == nil {
			return gatewayNamespace == route.GetNamespace()
		}
		return route.GetNamespace() == string(*parentRefNamespace)

	case gatewayv1.NamespacesFromSelector:
		namespace := corev1.Namespace{}
		if err := mgrc.Get(ctx, client.ObjectKey{Name: route.GetNamespace()}, &namespace); err != nil {
			return false
		}

		s, err := metav1.LabelSelectorAsSelector(routeNamespaces.Selector)
		if err != nil {
			return false
		}
		return s.Matches(labels.Set(namespace.Labels))
	default:
		return true
	}
}

func routeMatchesListenerType(route client.Object, listener gatewayv1.Listener) bool {
	switch route.(type) {
	case *gatewayv1.HTTPRoute:
		if listener.Protocol != gatewayv1.HTTPProtocolType && listener.Protocol != gatewayv1.HTTPSProtocolType {
			return false
		}

		if listener.Protocol == gatewayv1.HTTPSProtocolType {
			if listener.TLS == nil {
				return false
			}

			if listener.TLS.Mode != nil && *listener.TLS.Mode != gatewayv1.TLSModeTerminate {
				return false
			}
		}
	default:
		return false
	}
	return true
}

func getAttachedRoutesForListener(ctx context.Context, mgrc client.Client, gateway gatewayv1.Gateway, listener gatewayv1.Listener) (int32, error) {
	httpRouteList := gatewayv1.HTTPRouteList{}
	if err := mgrc.List(ctx, &httpRouteList); err != nil {
		return 0, err
	}
	var attachedRoutes int32
	for _, route := range httpRouteList.Items {
		route := route
		acceptedByGateway := lo.ContainsBy(route.Status.Parents, func(parentStatus gatewayv1.RouteParentStatus) bool {
			parentRef := parentStatus.ParentRef
			if parentRef.Group != nil && *parentRef.Group != gatewayv1.GroupName {
				return false
			}
			if parentRef.Kind != nil && *parentRef.Kind != KindGateway {
				return false
			}
			gatewayNamespace := route.Namespace
			if parentRef.Namespace != nil {
				gatewayNamespace = string(*parentRef.Namespace)
			}
			return gateway.Namespace == gatewayNamespace && gateway.Name == string(parentRef.Name)
		})
		if !acceptedByGateway {
			continue
		}

		for _, parentRef := range route.Spec.ParentRefs {
			ok, _, err := checkRouteAcceptedByListener(
				ctx,
				mgrc,
				&route,
				gateway,
				listener,
				parentRef,
			)
			if err != nil {
				return 0, err
			}
			if ok {
				attachedRoutes++
			}
		}
	}
	return attachedRoutes, nil
}

func getListenerStatus(
	ctx context.Context,
	mrgc client.Client,
	gateway *gatewayv1.Gateway,
	grants []v1beta1.ReferenceGrant,
) ([]gatewayv1.ListenerStatus, error) {
	statuses := make(map[gatewayv1.SectionName]gatewayv1.ListenerStatus, len(gateway.Spec.Listeners))

	for i, listener := range gateway.Spec.Listeners {
		attachedRoutes, err := getAttachedRoutesForListener(ctx, mrgc, *gateway, listener)
		if err != nil {
			return nil, err
		}
		var (
			now                 = metav1.Now()
			conditionProgrammed = metav1.Condition{
				Type:               string(gatewayv1.ListenerConditionProgrammed),
				Status:             metav1.ConditionTrue,
				ObservedGeneration: gateway.GetGeneration(),
				LastTransitionTime: now,
				Reason:             string(gatewayv1.ListenerReasonProgrammed),
			}
			conditionAccepted = metav1.Condition{
				Type:               string(gatewayv1.ListenerConditionAccepted),
				Status:             metav1.ConditionTrue,
				ObservedGeneration: gateway.GetGeneration(),
				LastTransitionTime: now,
				Reason:             string(gatewayv1.ListenerReasonAccepted),
			}
			conditionConflicted = metav1.Condition{
				Type:               string(gatewayv1.ListenerConditionConflicted),
				Status:             metav1.ConditionTrue,
				ObservedGeneration: gateway.GetGeneration(),
				LastTransitionTime: now,
				Reason:             string(gatewayv1.ListenerReasonNoConflicts),
			}
			conditionResolvedRefs = metav1.Condition{
				Type:               string(gatewayv1.ListenerConditionResolvedRefs),
				Status:             metav1.ConditionTrue,
				ObservedGeneration: gateway.GetGeneration(),
				LastTransitionTime: now,
				Reason:             string(gatewayv1.ListenerReasonResolvedRefs),
			}

			supportedKinds = []gatewayv1.RouteGroupKind{}
		)

		if listener.AllowedRoutes == nil || listener.AllowedRoutes.Kinds == nil {
			supportedKinds = []gatewayv1.RouteGroupKind{
				{
					Kind: KindHTTPRoute,
				},
			}
		} else {
			for _, kind := range listener.AllowedRoutes.Kinds {
				if kind.Group != nil && *kind.Group != gatewayv1.GroupName {
					conditionResolvedRefs.Status = metav1.ConditionFalse
					conditionResolvedRefs.Reason = string(gatewayv1.ListenerReasonInvalidRouteKinds)
					continue
				}
				switch kind.Kind {
				case KindHTTPRoute:
					supportedKinds = append(supportedKinds, kind)
				default:
					conditionResolvedRefs.Status = metav1.ConditionFalse
					conditionResolvedRefs.Reason = string(gatewayv1.ListenerReasonInvalidRouteKinds)
				}
			}
		}

		if listener.TLS != nil {
			// TODO: support TLS
			var (
				secret corev1.Secret
			)
			for _, ref := range listener.TLS.CertificateRefs {
				if ref.Group != nil && *ref.Group != corev1.GroupName {
					conditionResolvedRefs.Status = metav1.ConditionFalse
					conditionResolvedRefs.Reason = string(gatewayv1.ListenerReasonInvalidCertificateRef)
					conditionResolvedRefs.Message = fmt.Sprintf(`Invalid Group, expect "", got "%s"`, *ref.Group)
					conditionProgrammed.Status = metav1.ConditionFalse
					conditionProgrammed.Reason = string(gatewayv1.ListenerReasonInvalid)
					break
				}
				if ref.Kind != nil && *ref.Kind != KindSecret {
					conditionResolvedRefs.Status = metav1.ConditionFalse
					conditionResolvedRefs.Reason = string(gatewayv1.ListenerReasonInvalidCertificateRef)
					conditionResolvedRefs.Message = fmt.Sprintf(`Invalid Kind, expect "Secret", got "%s"`, *ref.Kind)
					conditionProgrammed.Status = metav1.ConditionFalse
					conditionProgrammed.Reason = string(gatewayv1.ListenerReasonInvalid)
					break
				}
				if ok := checkReferenceGrantBetweenGatewayAndSecret(gateway.Namespace, ref, grants); !ok {
					conditionResolvedRefs.Status = metav1.ConditionFalse
					conditionResolvedRefs.Reason = string(gatewayv1.ListenerReasonRefNotPermitted)
					conditionResolvedRefs.Message = "certificateRefs cross namespaces is not permitted"
					conditionProgrammed.Status = metav1.ConditionFalse
					conditionProgrammed.Reason = string(gatewayv1.ListenerReasonInvalid)
					break
				}
				ns := gateway.Namespace
				if ref.Namespace != nil {
					ns = string(*ref.Namespace)
				}
				if err := mrgc.Get(ctx, client.ObjectKey{Namespace: ns, Name: string(ref.Name)}, &secret); err != nil {
					conditionResolvedRefs.Status = metav1.ConditionFalse
					conditionResolvedRefs.Reason = string(gatewayv1.ListenerReasonInvalidCertificateRef)
					conditionResolvedRefs.Message = err.Error()
					conditionProgrammed.Status = metav1.ConditionFalse
					conditionProgrammed.Reason = string(gatewayv1.ListenerReasonInvalid)
					break
				}
				if cause, ok := isTLSSecretValid(&secret); !ok {
					conditionResolvedRefs.Status = metav1.ConditionFalse
					conditionResolvedRefs.Reason = string(gatewayv1.ListenerReasonInvalidCertificateRef)
					conditionResolvedRefs.Message = fmt.Sprintf("Malformed Secret referenced: %s", cause)
					conditionProgrammed.Status = metav1.ConditionFalse
					conditionProgrammed.Reason = string(gatewayv1.ListenerReasonInvalid)
					break
				}
			}
		}

		status := gatewayv1.ListenerStatus{
			Name: listener.Name,
			Conditions: []metav1.Condition{
				conditionProgrammed,
				conditionAccepted,
				conditionConflicted,
				conditionResolvedRefs,
			},
			SupportedKinds: supportedKinds,
			AttachedRoutes: attachedRoutes,
		}

		changed := false
		if len(gateway.Status.Listeners) > i {
			if gateway.Status.Listeners[i].AttachedRoutes != attachedRoutes {
				changed = true
			}
			for _, condition := range status.Conditions {
				if !IsConditionPresentAndEqual(gateway.Status.Listeners[i].Conditions, condition) {
					changed = true
					break
				}
			}
		} else {
			changed = true
		}

		if changed {
			statuses[listener.Name] = status
		} else {
			statuses[listener.Name] = gateway.Status.Listeners[i]
		}
	}

	// check for conflicts

	statusArray := []gatewayv1.ListenerStatus{}
	for _, status := range statuses {
		statusArray = append(statusArray, status)
	}

	return statusArray, nil
}

// SplitMetaNamespaceKey returns the namespace and name that
// MetaNamespaceKeyFunc encoded into key.
func SplitMetaNamespaceKey(key string) (namespace, name string, err error) {
	parts := strings.Split(key, "/")
	switch len(parts) {
	case 1:
		// name only, no namespace
		return "", parts[0], nil
	case 2:
		// namespace and name
		return parts[0], parts[1], nil
	}

	return "", "", fmt.Errorf("unexpected key format: %q", key)
}

func ProcessGatewayProxy(r client.Client, tctx *provider.TranslateContext, gateway *gatewayv1.Gateway, rk provider.ResourceKind) error {
	if gateway == nil {
		return nil
	}
	infra := gateway.Spec.Infrastructure
	if infra == nil || infra.ParametersRef == nil {
		return nil
	}

	gatewayKind := provider.ResourceKind{
		Kind:      gateway.Kind,
		Namespace: gateway.Namespace,
		Name:      gateway.Name,
	}

	ns := gateway.GetNamespace()
	paramRef := infra.ParametersRef
	if string(paramRef.Group) == v1alpha1.GroupVersion.Group && string(paramRef.Kind) == KindGatewayProxy {
		gatewayProxy := &v1alpha1.GatewayProxy{}
		if err := r.Get(context.Background(), client.ObjectKey{
			Namespace: ns,
			Name:      paramRef.Name,
		}, gatewayProxy); err != nil {
			log.Errorw("failed to get GatewayProxy", zap.String("namespace", ns), zap.String("name", paramRef.Name), zap.Error(err))
			return err
		} else {
			log.Infow("found GatewayProxy for Gateway", zap.String("namespace", gateway.Namespace), zap.String("name", gateway.Name))
			tctx.GatewayProxies[gatewayKind] = *gatewayProxy
			tctx.ResourceParentRefs[rk] = append(tctx.ResourceParentRefs[rk], gatewayKind)

			// Process provider secrets if provider exists
			if gatewayProxy.Spec.Provider != nil && gatewayProxy.Spec.Provider.Type == v1alpha1.ProviderTypeControlPlane {
				if gatewayProxy.Spec.Provider.ControlPlane != nil &&
					gatewayProxy.Spec.Provider.ControlPlane.Auth.Type == v1alpha1.AuthTypeAdminKey &&
					gatewayProxy.Spec.Provider.ControlPlane.Auth.AdminKey != nil &&
					gatewayProxy.Spec.Provider.ControlPlane.Auth.AdminKey.ValueFrom != nil &&
					gatewayProxy.Spec.Provider.ControlPlane.Auth.AdminKey.ValueFrom.SecretKeyRef != nil {

					secretRef := gatewayProxy.Spec.Provider.ControlPlane.Auth.AdminKey.ValueFrom.SecretKeyRef
					secret := &corev1.Secret{}
					if err := r.Get(context.Background(), client.ObjectKey{
						Namespace: ns,
						Name:      secretRef.Name,
					}, secret); err != nil {
						log.Error(err, "failed to get secret for GatewayProxy provider",
							"namespace", ns,
							"name", secretRef.Name)
						return err
					}

					log.Info("found secret for GatewayProxy provider",
						"gateway", gateway.Name,
						"gatewayproxy", gatewayProxy.Name,
						"secret", secretRef.Name)

					tctx.Secrets[types.NamespacedName{
						Namespace: ns,
						Name:      secretRef.Name,
					}] = secret
				}
			}
		}
	}

	_, ok := tctx.GatewayProxies[gatewayKind]
	if !ok {
		return fmt.Errorf("no gateway proxy found for gateway: %s", gateway.Name)
	}

	return nil
}

// FullTypeName returns the fully qualified name of the type of the given value.
func FullTypeName(a any) string {
	typeOf := reflect.TypeOf(a)
	pkgPath := typeOf.PkgPath()
	name := typeOf.String()
	if typeOf.Kind() == reflect.Ptr {
		pkgPath = typeOf.Elem().PkgPath()
	}
	return path.Join(path.Dir(pkgPath), name)
}

// InvalidKindError represents an error when backend reference kind is not supported
type InvalidKindError struct {
	Kind string
}

// Error implements the error interface
func (e *InvalidKindError) Error() string {
	return fmt.Sprintf("%s %s", string(gatewayv1.RouteReasonInvalidKind), e.Kind)
}

// NewInvalidKindError creates a new InvalidKindError
func NewInvalidKindError(kind string) *InvalidKindError {
	return &InvalidKindError{Kind: kind}
}

// IsInvalidKindError checks if the error is an InvalidKindError
func IsInvalidKindError(err error) bool {
	_, ok := err.(*InvalidKindError)
	return ok
}

// BackendNotFoundError represents an error when a backend service is not found
type BackendNotFoundError struct {
	Name      string
	Namespace string
}

// Error implements the error interface
func (e *BackendNotFoundError) Error() string {
	return fmt.Sprintf("Service %s/%s not found", e.Namespace, e.Name)
}

// NewBackendNotFoundError creates a new BackendNotFoundError
func NewBackendNotFoundError(namespace, name string) *BackendNotFoundError {
	return &BackendNotFoundError{
		Name:      name,
		Namespace: namespace,
	}
}

// IsBackendNotFoundError checks if the error is a BackendNotFoundError
func IsBackendNotFoundError(err error) bool {
	_, ok := err.(*BackendNotFoundError)
	return ok
}

// filterHostnames accepts a list of gateways and an HTTPRoute, and returns a copy of the HTTPRoute with only the hostnames that match the listener hostnames of the gateways.
// If the HTTPRoute hostnames do not intersect with the listener hostnames of the gateways, it returns an ErrNoMatchingListenerHostname error.
func filterHostnames(gateways []RouteParentRefContext, httpRoute *gatewayv1.HTTPRoute) (*gatewayv1.HTTPRoute, error) {
	filteredHostnames := make([]gatewayv1.Hostname, 0)

	// If the HTTPRoute does not specify hostnames, we use the union of the listener hostnames of all supported gateways
	// If any supported listener does not specify a hostname, the HTTPRoute hostnames remain empty to match any hostname
	if len(httpRoute.Spec.Hostnames) == 0 {
		hostnames, matchAnyHost := getUnionOfGatewayHostnames(gateways)
		if matchAnyHost {
			return httpRoute, nil
		}
		filteredHostnames = hostnames
	} else {
		// If the HTTPRoute specifies hostnames, we need to find the intersection with the gateway listener hostnames
		for _, hostname := range httpRoute.Spec.Hostnames {
			if hostnameMatching := getMinimumHostnameIntersection(gateways, hostname); hostnameMatching != "" {
				filteredHostnames = append(filteredHostnames, hostnameMatching)
			}
		}
		if len(filteredHostnames) == 0 {
			return httpRoute, ErrNoMatchingListenerHostname
		}
	}

	log.Debugw("filtered hostnames", zap.Any("httpRouteHostnames", httpRoute.Spec.Hostnames), zap.Any("hostnames", filteredHostnames))
	httpRoute.Spec.Hostnames = filteredHostnames
	return httpRoute, nil
}

// getUnionOfGatewayHostnames returns the union of the hostnames specified in all supported gateways
// The second return value indicates whether any listener can match any hostname
func getUnionOfGatewayHostnames(gateways []RouteParentRefContext) ([]gatewayv1.Hostname, bool) {
	hostnames := make([]gatewayv1.Hostname, 0)

	for _, gateway := range gateways {
		if gateway.ListenerName != "" {
			// If a listener name is specified, only check that listener
			for _, listener := range gateway.Gateway.Spec.Listeners {
				if string(listener.Name) == gateway.ListenerName {
					// If a listener does not specify a hostname, it can match any hostname
					if listener.Hostname == nil {
						return nil, true
					}
					hostnames = append(hostnames, *listener.Hostname)
					break
				}
			}
		} else {
			// Otherwise, check all listeners
			for _, listener := range gateway.Gateway.Spec.Listeners {
				// Only consider listeners that can effectively configure hostnames (HTTP, HTTPS, or TLS)
				if isListenerHostnameEffective(listener) {
					if listener.Hostname == nil {
						return nil, true
					}
					hostnames = append(hostnames, *listener.Hostname)
				}
			}
		}
	}

	return hostnames, false
}

// getMinimumHostnameIntersection returns the smallest intersection hostname
// - If the listener hostname is empty, return the HTTPRoute hostname
// - If the listener hostname is a wildcard of the HTTPRoute hostname, return the HTTPRoute hostname
// - If the HTTPRoute hostname is a wildcard of the listener hostname, return the listener hostname
// - If the HTTPRoute hostname and listener hostname are the same, return it
// - If none of the above, return an empty string
func getMinimumHostnameIntersection(gateways []RouteParentRefContext, hostname gatewayv1.Hostname) gatewayv1.Hostname {
	for _, gateway := range gateways {
		for _, listener := range gateway.Gateway.Spec.Listeners {
			// If a listener name is specified, only check that listener
			// If the listener name is not specified, check all listeners
			if gateway.ListenerName == "" || gateway.ListenerName == string(listener.Name) {
				if listener.Hostname == nil || *listener.Hostname == "" {
					return hostname
				}
				if HostnamesMatch(string(*listener.Hostname), string(hostname)) {
					return hostname
				}
				if HostnamesMatch(string(hostname), string(*listener.Hostname)) {
					return *listener.Hostname
				}
			}
		}
	}

	return ""
}

// isListenerHostnameEffective checks if a listener can specify a hostname to match the hostname in the request
// Basically, check if the listener uses HTTP, HTTPS, or TLS protocol
func isListenerHostnameEffective(listener gatewayv1.Listener) bool {
	return listener.Protocol == gatewayv1.HTTPProtocolType ||
		listener.Protocol == gatewayv1.HTTPSProtocolType ||
		listener.Protocol == gatewayv1.TLSProtocolType
}

func isRouteAccepted(gateways []RouteParentRefContext) bool {
	for _, gateway := range gateways {
		for _, condition := range gateway.Conditions {
			if condition.Type == string(gatewayv1.RouteConditionAccepted) && condition.Status == metav1.ConditionTrue {
				return true
			}
		}
	}
	return false
}

func isTLSSecretValid(secret *corev1.Secret) (string, bool) {
	var ok bool
	var crt, key []byte
	if crt, ok = secret.Data["tls.crt"]; !ok {
		return "Missing tls.crt", false
	}
	if key, ok = secret.Data["tls.key"]; !ok {
		return "Missing tls.key", false
	}
	if p, _ := pem.Decode(crt); p == nil {
		return "Malformed PEM tls.crt", false
	}
	if p, _ := pem.Decode(key); p == nil {
		return "Malformed PEM tls.key", false
	}
	return "", true
}

func checkReferenceGrantBetweenGatewayAndSecret(gwNamespace string, certRef gatewayv1.SecretObjectReference, grants []v1beta1.ReferenceGrant) bool {
	// if not cross namespaces
	if certRef.Namespace == nil || string(*certRef.Namespace) == gwNamespace {
		return true
	}

	for _, grant := range grants {
		if grant.Namespace == string(*certRef.Namespace) {
			for _, from := range grant.Spec.From {
				gw := v1beta1.ReferenceGrantFrom{
					Group:     gatewayv1.GroupName,
					Kind:      KindGateway,
					Namespace: v1beta1.Namespace(gwNamespace),
				}
				if from == gw {
					for _, to := range grant.Spec.To {
						if to.Group == corev1.GroupName && to.Kind == KindSecret && (to.Name == nil || *to.Name == certRef.Name) {
							return true
						}
					}
				}
			}
		}
	}
	return false
}
