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

	"github.com/api7/gopkg/pkg/log"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	"sigs.k8s.io/gateway-api/apis/v1alpha2"
	"sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/controller/indexer"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/internal/utils"
)

// HTTPRouteReconciler reconciles a GatewayClass object.
type HTTPRouteReconciler struct { //nolint:revive
	client.Client
	Scheme *runtime.Scheme

	Log logr.Logger

	Provider provider.Provider

	genericEvent chan event.GenericEvent

	Updater status.Updater
}

// SetupWithManager sets up the controller with the Manager.
func (r *HTTPRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.genericEvent = make(chan event.GenericEvent, 100)

	bdr := ctrl.NewControllerManagedBy(mgr).
		For(&gatewayv1.HTTPRoute{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Watches(&discoveryv1.EndpointSlice{},
			handler.EnqueueRequestsFromMapFunc(r.listHTTPRoutesByServiceBef),
		).
		Watches(&v1alpha1.PluginConfig{},
			handler.EnqueueRequestsFromMapFunc(r.listHTTPRoutesByExtensionRef),
		).
		Watches(&gatewayv1.Gateway{},
			handler.EnqueueRequestsFromMapFunc(r.listHTTPRoutesForGateway),
			builder.WithPredicates(
				predicate.Funcs{
					GenericFunc: func(e event.GenericEvent) bool {
						return false
					},
					DeleteFunc: func(e event.DeleteEvent) bool {
						return false
					},
					CreateFunc: func(e event.CreateEvent) bool {
						return true
					},
					UpdateFunc: func(e event.UpdateEvent) bool {
						return true
					},
				},
			),
		).
		Watches(&v1alpha1.BackendTrafficPolicy{},
			handler.EnqueueRequestsFromMapFunc(r.listHTTPRoutesForBackendTrafficPolicy),
			builder.WithPredicates(
				BackendTrafficPolicyPredicateFunc(r.genericEvent),
			),
		).
		Watches(&v1alpha1.HTTPRoutePolicy{},
			handler.EnqueueRequestsFromMapFunc(r.listHTTPRouteByHTTPRoutePolicy),
			builder.WithPredicates(httpRoutePolicyPredicateFuncs(r.genericEvent)),
		).
		Watches(&v1alpha1.GatewayProxy{},
			handler.EnqueueRequestsFromMapFunc(r.listHTTPRoutesForGatewayProxy),
		).
		WatchesRawSource(
			source.Channel(
				r.genericEvent,
				handler.EnqueueRequestsFromMapFunc(r.listHTTPRouteForGenericEvent),
			),
		)

	if GetEnableReferenceGrant() {
		bdr.Watches(&v1beta1.ReferenceGrant{},
			handler.EnqueueRequestsFromMapFunc(r.listHTTPRoutesForReferenceGrant),
			builder.WithPredicates(referenceGrantPredicates(KindHTTPRoute)),
		)
	}

	return bdr.Complete(r)
}

func (r *HTTPRouteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	hr := new(gatewayv1.HTTPRoute)
	if err := r.Get(ctx, req.NamespacedName, hr); err != nil {
		if client.IgnoreNotFound(err) == nil {
			if err := r.updateHTTPRoutePolicyStatusOnDeleting(ctx, req.NamespacedName); err != nil {
				return ctrl.Result{}, err
			}
			hr.Namespace = req.Namespace
			hr.Name = req.Name

			hr.TypeMeta = metav1.TypeMeta{
				Kind:       KindHTTPRoute,
				APIVersion: gatewayv1.GroupVersion.String(),
			}

			if err := r.Provider.Delete(ctx, hr); err != nil {
				r.Log.Error(err, "failed to delete httproute", "httproute", hr)
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	type ResourceStatus struct {
		status bool
		msg    string
	}

	// Only keep acceptStatus since we're using error objects directly now
	acceptStatus := ResourceStatus{
		status: true,
		msg:    "Route is accepted",
	}

	gateways, err := ParseRouteParentRefs(ctx, r.Client, hr, hr.Spec.ParentRefs)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(gateways) == 0 {
		return ctrl.Result{}, nil
	}

	tctx := provider.NewDefaultTranslateContext(ctx)

	tctx.RouteParentRefs = hr.Spec.ParentRefs
	rk := utils.NamespacedNameKind(hr)
	for _, gateway := range gateways {
		if err := ProcessGatewayProxy(r.Client, tctx, gateway.Gateway, rk); err != nil {
			acceptStatus.status = false
			acceptStatus.msg = err.Error()
		}
	}

	var backendRefErr error
	if err := r.processHTTPRoute(tctx, hr); err != nil {
		// When encountering a backend reference error, it should not affect the acceptance status
		if IsSomeReasonError(err, gatewayv1.RouteReasonInvalidKind) {
			backendRefErr = err
		} else {
			acceptStatus.status = false
			acceptStatus.msg = err.Error()
		}
	}

	if err := r.processHTTPRoutePolicies(tctx, hr); err != nil {
		acceptStatus.status = false
		acceptStatus.msg = err.Error()
	}

	// Store the backend reference error for later use.
	// If the backend reference error is because of an invalid kind, use this error first
	if err := r.processHTTPRouteBackendRefs(tctx, req.NamespacedName); err != nil && backendRefErr == nil {
		backendRefErr = err
	}

	ProcessBackendTrafficPolicy(r.Client, r.Log, tctx)

	filteredHTTPRoute, err := filterHostnames(gateways, hr.DeepCopy())
	if err != nil {
		acceptStatus.status = false
		acceptStatus.msg = err.Error()
	}

	// TODO: diff the old and new status
	hr.Status.Parents = make([]gatewayv1.RouteParentStatus, 0, len(gateways))
	for _, gateway := range gateways {
		parentStatus := gatewayv1.RouteParentStatus{}
		SetRouteParentRef(&parentStatus, gateway.Gateway.Name, gateway.Gateway.Namespace)
		for _, condition := range gateway.Conditions {
			parentStatus.Conditions = MergeCondition(parentStatus.Conditions, condition)
		}
		SetRouteConditionAccepted(&parentStatus, hr.GetGeneration(), acceptStatus.status, acceptStatus.msg)
		SetRouteConditionResolvedRefs(&parentStatus, hr.GetGeneration(), backendRefErr)

		hr.Status.Parents = append(hr.Status.Parents, parentStatus)
	}

	r.Updater.Update(status.Update{
		NamespacedName: utils.NamespacedName(hr),
		Resource:       &gatewayv1.HTTPRoute{},
		Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
			h, ok := obj.(*gatewayv1.HTTPRoute)
			if !ok {
				err := fmt.Errorf("unsupported object type %T", obj)
				panic(err)
			}
			hCopy := h.DeepCopy()
			hCopy.Status = hr.Status
			return hCopy
		}),
	})
	UpdateStatus(r.Updater, r.Log, tctx)

	if isRouteAccepted(gateways) && err == nil {
		routeToUpdate := hr
		if filteredHTTPRoute != nil {
			log.Debugw("filteredHTTPRoute", zap.Any("filteredHTTPRoute", filteredHTTPRoute))
			routeToUpdate = filteredHTTPRoute
		}
		if err := r.Provider.Update(ctx, tctx, routeToUpdate); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *HTTPRouteReconciler) listHTTPRoutesByServiceBef(ctx context.Context, obj client.Object) []reconcile.Request {
	endpointSlice, ok := obj.(*discoveryv1.EndpointSlice)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to EndpointSlice")
		return nil
	}
	namespace := endpointSlice.GetNamespace()
	serviceName := endpointSlice.Labels[discoveryv1.LabelServiceName]

	hrList := &gatewayv1.HTTPRouteList{}
	if err := r.List(ctx, hrList, client.MatchingFields{
		indexer.ServiceIndexRef: indexer.GenIndexKey(namespace, serviceName),
	}); err != nil {
		r.Log.Error(err, "failed to list httproutes by service", "service", serviceName)
		return nil
	}
	requests := make([]reconcile.Request, 0, len(hrList.Items))
	for _, hr := range hrList.Items {
		requests = append(requests, reconcile.Request{
			NamespacedName: client.ObjectKey{
				Namespace: hr.Namespace,
				Name:      hr.Name,
			},
		})
	}
	return requests
}

func (r *HTTPRouteReconciler) listHTTPRoutesByExtensionRef(ctx context.Context, obj client.Object) []reconcile.Request {
	pluginconfig, ok := obj.(*v1alpha1.PluginConfig)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to EndpointSlice")
		return nil
	}
	namespace := pluginconfig.GetNamespace()
	name := pluginconfig.GetName()

	hrList := &gatewayv1.HTTPRouteList{}
	if err := r.List(ctx, hrList, client.MatchingFields{
		indexer.ExtensionRef: indexer.GenIndexKey(namespace, name),
	}); err != nil {
		r.Log.Error(err, "failed to list httproutes by extension reference", "extension", name)
		return nil
	}
	requests := make([]reconcile.Request, 0, len(hrList.Items))
	for _, hr := range hrList.Items {
		requests = append(requests, reconcile.Request{
			NamespacedName: client.ObjectKey{
				Namespace: hr.Namespace,
				Name:      hr.Name,
			},
		})
	}
	return requests
}

func (r *HTTPRouteReconciler) listHTTPRoutesForBackendTrafficPolicy(ctx context.Context, obj client.Object) []reconcile.Request {
	policy, ok := obj.(*v1alpha1.BackendTrafficPolicy)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to BackendTrafficPolicy")
		return nil
	}

	httprouteList := []gatewayv1.HTTPRoute{}
	for _, targetRef := range policy.Spec.TargetRefs {
		service := &corev1.Service{}
		if err := r.Get(ctx, client.ObjectKey{
			Namespace: policy.Namespace,
			Name:      string(targetRef.Name),
		}, service); err != nil {
			if client.IgnoreNotFound(err) != nil {
				r.Log.Error(err, "failed to get service", "namespace", policy.Namespace, "name", targetRef.Name)
			}
			continue
		}
		hrList := &gatewayv1.HTTPRouteList{}
		if err := r.List(ctx, hrList, client.MatchingFields{
			indexer.ServiceIndexRef: indexer.GenIndexKey(policy.Namespace, string(targetRef.Name)),
		}); err != nil {
			r.Log.Error(err, "failed to list httproutes by service reference", "service", targetRef.Name)
			return nil
		}
		httprouteList = append(httprouteList, hrList.Items...)
	}
	var namespacedNameMap = make(map[types.NamespacedName]struct{})
	requests := make([]reconcile.Request, 0, len(httprouteList))
	for _, hr := range httprouteList {
		key := types.NamespacedName{
			Namespace: hr.Namespace,
			Name:      hr.Name,
		}
		if _, ok := namespacedNameMap[key]; !ok {
			namespacedNameMap[key] = struct{}{}
			requests = append(requests, reconcile.Request{
				NamespacedName: client.ObjectKey{
					Namespace: hr.Namespace,
					Name:      hr.Name,
				},
			})
		}
	}
	return requests
}

func (r *HTTPRouteReconciler) listHTTPRoutesForGateway(ctx context.Context, obj client.Object) []reconcile.Request {
	gateway, ok := obj.(*gatewayv1.Gateway)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to Gateway")
	}
	hrList := &gatewayv1.HTTPRouteList{}
	if err := r.List(ctx, hrList, client.MatchingFields{
		indexer.ParentRefs: indexer.GenIndexKey(gateway.Namespace, gateway.Name),
	}); err != nil {
		r.Log.Error(err, "failed to list httproutes by gateway", "gateway", gateway.Name)
		return nil
	}
	requests := make([]reconcile.Request, 0, len(hrList.Items))
	for _, hr := range hrList.Items {
		requests = append(requests, reconcile.Request{
			NamespacedName: client.ObjectKey{
				Namespace: hr.Namespace,
				Name:      hr.Name,
			},
		})
	}
	return requests
}

func (r *HTTPRouteReconciler) listHTTPRouteByHTTPRoutePolicy(ctx context.Context, obj client.Object) (requests []reconcile.Request) {
	httpRoutePolicy, ok := obj.(*v1alpha1.HTTPRoutePolicy)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to HTTPRoutePolicy")
		return nil
	}

	var keys = make(map[types.NamespacedName]struct{})
	for _, ref := range httpRoutePolicy.Spec.TargetRefs {
		if ref.Kind != "HTTPRoute" {
			continue
		}
		key := types.NamespacedName{
			Namespace: obj.GetNamespace(),
			Name:      string(ref.Name),
		}
		if _, ok := keys[key]; ok {
			continue
		}

		var httpRoute gatewayv1.HTTPRoute
		if err := r.Get(ctx, key, &httpRoute); err != nil {
			r.Log.Error(err, "failed to get HTTPRoute by HTTPRoutePolicy targetRef", "namespace", key.Namespace, "name", key.Name)
			continue
		}
		if ref.SectionName != nil {
			var matchRuleName bool
			for _, rule := range httpRoute.Spec.Rules {
				if rule.Name != nil && *rule.Name == *ref.SectionName {
					matchRuleName = true
					break
				}
			}
			if !matchRuleName {
				r.Log.Error(errors.Errorf("failed to get HTTPRoute rule by HTTPRoutePolicy targetRef"), "namespace", key.Namespace, "name", key.Name, "sectionName", *ref.SectionName)
				continue
			}
		}
		keys[key] = struct{}{}
		requests = append(requests, reconcile.Request{NamespacedName: key})
	}

	return requests
}

func (r *HTTPRouteReconciler) listHTTPRouteForGenericEvent(ctx context.Context, obj client.Object) (requests []reconcile.Request) {
	switch obj.(type) {
	case *v1alpha1.BackendTrafficPolicy:
		return r.listHTTPRoutesForBackendTrafficPolicy(ctx, obj)
	case *v1alpha1.HTTPRoutePolicy:
		return r.listHTTPRouteByHTTPRoutePolicy(ctx, obj)
	default:
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to BackendTrafficPolicy or HTTPRoutePolicy")
		return nil
	}
}

func (r *HTTPRouteReconciler) processHTTPRouteBackendRefs(tctx *provider.TranslateContext, hrNN types.NamespacedName) error {
	var terr error
	for _, backend := range tctx.BackendRefs {
		targetNN := types.NamespacedName{
			Namespace: hrNN.Namespace,
			Name:      string(backend.Name),
		}
		if backend.Namespace != nil {
			targetNN.Namespace = string(*backend.Namespace)
		}

		if backend.Kind != nil && *backend.Kind != "Service" {
			terr = newInvalidKindError(*backend.Kind)
			continue
		}

		if backend.Port == nil {
			terr = fmt.Errorf("port is required")
			continue
		}

		var service corev1.Service
		if err := r.Get(tctx, targetNN, &service); err != nil {
			terr = err
			if client.IgnoreNotFound(err) == nil {
				terr = ReasonError{
					Reason:  string(gatewayv1.RouteReasonBackendNotFound),
					Message: fmt.Sprintf("Service %s not found", targetNN),
				}
			}
			continue
		}

		// if cross namespaces between HTTPRoute and referenced Service, check ReferenceGrant
		if hrNN.Namespace != targetNN.Namespace {
			if permitted := checkReferenceGrant(tctx,
				r.Client,
				v1beta1.ReferenceGrantFrom{
					Group:     gatewayv1.GroupName,
					Kind:      KindHTTPRoute,
					Namespace: v1beta1.Namespace(hrNN.Namespace),
				},
				gatewayv1.ObjectReference{
					Group:     corev1.GroupName,
					Kind:      KindService,
					Name:      gatewayv1.ObjectName(targetNN.Name),
					Namespace: (*gatewayv1.Namespace)(&targetNN.Namespace),
				},
			); !permitted {
				terr = ReasonError{
					Reason:  string(v1beta1.RouteReasonRefNotPermitted),
					Message: fmt.Sprintf("%s is in a different namespace than the HTTPRoute %s and no ReferenceGrant allowing reference is configured", targetNN, hrNN),
				}
				continue
			}
		}

		if service.Spec.Type == corev1.ServiceTypeExternalName {
			tctx.Services[targetNN] = &service
			return nil
		}

		portExists := false
		for _, port := range service.Spec.Ports {
			if port.Port == int32(*backend.Port) {
				portExists = true
				break
			}
		}
		if !portExists {
			terr = fmt.Errorf("port %d not found in service %s", *backend.Port, targetNN.Name)
			continue
		}
		tctx.Services[targetNN] = &service

		endpointSliceList := new(discoveryv1.EndpointSliceList)
		if err := r.List(tctx, endpointSliceList,
			client.InNamespace(targetNN.Namespace),
			client.MatchingLabels{
				discoveryv1.LabelServiceName: targetNN.Name,
			},
		); err != nil {
			r.Log.Error(err, "failed to list endpoint slices", "Service", targetNN)
			terr = err
			continue
		}

		tctx.EndpointSlices[targetNN] = endpointSliceList.Items
	}
	return terr
}

func (r *HTTPRouteReconciler) processHTTPRoute(tctx *provider.TranslateContext, httpRoute *gatewayv1.HTTPRoute) error {
	var terror error
	for _, rule := range httpRoute.Spec.Rules {
		for _, filter := range rule.Filters {
			if filter.Type != gatewayv1.HTTPRouteFilterExtensionRef || filter.ExtensionRef == nil {
				continue
			}
			if filter.ExtensionRef.Kind == "PluginConfig" {
				pluginconfig := new(v1alpha1.PluginConfig)
				if err := r.Get(context.Background(), client.ObjectKey{
					Namespace: httpRoute.GetNamespace(),
					Name:      string(filter.ExtensionRef.Name),
				}, pluginconfig); err != nil {
					terror = err
					continue
				}
				tctx.PluginConfigs[types.NamespacedName{
					Namespace: httpRoute.GetNamespace(),
					Name:      string(filter.ExtensionRef.Name),
				}] = pluginconfig
			}
		}
		for _, backend := range rule.BackendRefs {
			if backend.Kind != nil && *backend.Kind != "Service" {
				terror = newInvalidKindError(*backend.Kind)
				continue
			}
			tctx.BackendRefs = append(tctx.BackendRefs, gatewayv1.BackendRef{
				BackendObjectReference: gatewayv1.BackendObjectReference{
					Name:      backend.Name,
					Namespace: cmp.Or(backend.Namespace, (*gatewayv1.Namespace)(&httpRoute.Namespace)),
					Port:      backend.Port,
				},
			})
		}
	}

	return terror
}

func httpRoutePolicyPredicateFuncs(channel chan event.GenericEvent) predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldPolicy, ok0 := e.ObjectOld.(*v1alpha1.HTTPRoutePolicy)
			newPolicy, ok1 := e.ObjectNew.(*v1alpha1.HTTPRoutePolicy)
			if !ok0 || !ok1 {
				return false
			}
			discardsRefs := slices.DeleteFunc(oldPolicy.Spec.TargetRefs, func(oldRef v1alpha2.LocalPolicyTargetReferenceWithSectionName) bool {
				return slices.ContainsFunc(newPolicy.Spec.TargetRefs, func(newRef v1alpha2.LocalPolicyTargetReferenceWithSectionName) bool {
					return oldRef.LocalPolicyTargetReference == newRef.LocalPolicyTargetReference && ptr.Equal(oldRef.SectionName, newRef.SectionName)
				})
			})
			if len(discardsRefs) > 0 {
				dump := oldPolicy.DeepCopy()
				dump.Spec.TargetRefs = discardsRefs
				channel <- event.GenericEvent{Object: dump}
			}
			return true
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
}

// listHTTPRoutesForGatewayProxy list all HTTPRoute resources that are affected by a given GatewayProxy
func (r *HTTPRouteReconciler) listHTTPRoutesForGatewayProxy(ctx context.Context, obj client.Object) []reconcile.Request {
	gatewayProxy, ok := obj.(*v1alpha1.GatewayProxy)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to GatewayProxy")
		return nil
	}

	namespace := gatewayProxy.GetNamespace()
	name := gatewayProxy.GetName()

	// find all gateways that reference this gateway proxy
	gatewayList := &gatewayv1.GatewayList{}
	if err := r.List(ctx, gatewayList, client.MatchingFields{
		indexer.ParametersRef: indexer.GenIndexKey(namespace, name),
	}); err != nil {
		r.Log.Error(err, "failed to list gateways for gateway proxy", "gatewayproxy", gatewayProxy.GetName())
		return nil
	}

	var requests []reconcile.Request

	// for each gateway, find all HTTPRoute resources that reference it
	for _, gateway := range gatewayList.Items {
		httpRouteList := &gatewayv1.HTTPRouteList{}
		if err := r.List(ctx, httpRouteList, client.MatchingFields{
			indexer.ParentRefs: indexer.GenIndexKey(gateway.Namespace, gateway.Name),
		}); err != nil {
			r.Log.Error(err, "failed to list httproutes for gateway", "gateway", gateway.Name)
			continue
		}

		for _, httpRoute := range httpRouteList.Items {
			requests = append(requests, reconcile.Request{
				NamespacedName: client.ObjectKey{
					Namespace: httpRoute.Namespace,
					Name:      httpRoute.Name,
				},
			})
		}
	}

	return requests
}

func (r *HTTPRouteReconciler) listHTTPRoutesForReferenceGrant(ctx context.Context, obj client.Object) (requests []reconcile.Request) {
	grant, ok := obj.(*v1beta1.ReferenceGrant)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to ReferenceGrant")
		return nil
	}

	var httpRouteList gatewayv1.HTTPRouteList
	if err := r.List(ctx, &httpRouteList); err != nil {
		r.Log.Error(err, "failed to list httproutes for reference ReferenceGrant", "ReferenceGrant", types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()})
		return nil
	}

	for _, httpRoute := range httpRouteList.Items {
		hr := v1beta1.ReferenceGrantFrom{
			Group:     gatewayv1.GroupName,
			Kind:      KindHTTPRoute,
			Namespace: v1beta1.Namespace(httpRoute.GetNamespace()),
		}
		for _, from := range grant.Spec.From {
			if from == hr {
				requests = append(requests, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: httpRoute.GetNamespace(),
						Name:      httpRoute.GetName(),
					},
				})
			}
		}
	}
	return requests
}
