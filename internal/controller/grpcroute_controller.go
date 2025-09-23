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
	"cmp"
	"context"
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	"sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/controller/indexer"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	"github.com/apache/apisix-ingress-controller/internal/manager/readiness"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/internal/types"
	"github.com/apache/apisix-ingress-controller/internal/utils"
)

// GRPCRouteReconciler reconciles a GatewayClass object.
type GRPCRouteReconciler struct { //nolint:revive
	client.Client
	Scheme *runtime.Scheme

	Log logr.Logger

	Provider provider.Provider

	genericEvent chan event.GenericEvent

	Updater status.Updater
	Readier readiness.ReadinessManager
}

// SetupWithManager sets up the controller with the Manager.
func (r *GRPCRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.genericEvent = make(chan event.GenericEvent, 100)

	bdr := ctrl.NewControllerManagedBy(mgr).
		For(&gatewayv1.GRPCRoute{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Watches(&discoveryv1.EndpointSlice{},
			handler.EnqueueRequestsFromMapFunc(r.listGRPCRoutesByServiceRef),
		).
		Watches(&v1alpha1.PluginConfig{},
			handler.EnqueueRequestsFromMapFunc(r.listGRPCRoutesByExtensionRef),
		).
		Watches(&gatewayv1.Gateway{},
			handler.EnqueueRequestsFromMapFunc(r.listGRPCRoutesForGateway),
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
			handler.EnqueueRequestsFromMapFunc(r.listGRPCRoutesForBackendTrafficPolicy),
			builder.WithPredicates(
				BackendTrafficPolicyPredicateFunc(r.genericEvent),
			),
		).
		Watches(&v1alpha1.GatewayProxy{},
			handler.EnqueueRequestsFromMapFunc(r.listGRPCRoutesForGatewayProxy),
		).
		WatchesRawSource(
			source.Channel(
				r.genericEvent,
				handler.EnqueueRequestsFromMapFunc(r.listGRPCRouteForGenericEvent),
			),
		)

	if GetEnableReferenceGrant() {
		bdr.Watches(&v1beta1.ReferenceGrant{},
			handler.EnqueueRequestsFromMapFunc(r.listGRPCRoutesForReferenceGrant),
			builder.WithPredicates(referenceGrantPredicates(KindGRPCRoute)),
		)
	}

	return bdr.Complete(r)
}

func (r *GRPCRouteReconciler) listGRPCRoutesByExtensionRef(ctx context.Context, obj client.Object) []reconcile.Request {
	pluginconfig, ok := obj.(*v1alpha1.PluginConfig)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to PluginConfig")
		return nil
	}
	namespace := pluginconfig.GetNamespace()
	name := pluginconfig.GetName()

	grList := &gatewayv1.GRPCRouteList{}
	if err := r.List(ctx, grList, client.MatchingFields{
		indexer.ExtensionRef: indexer.GenIndexKey(namespace, name),
	}); err != nil {
		r.Log.Error(err, "failed to list grpcroutes by extension reference", "extension", name)
		return nil
	}
	requests := make([]reconcile.Request, 0, len(grList.Items))
	for _, gr := range grList.Items {
		requests = append(requests, reconcile.Request{
			NamespacedName: client.ObjectKey{
				Namespace: gr.Namespace,
				Name:      gr.Name,
			},
		})
	}
	return requests
}

func (r *GRPCRouteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	defer r.Readier.Done(&gatewayv1.GRPCRoute{}, req.NamespacedName)
	gr := new(gatewayv1.GRPCRoute)
	if err := r.Get(ctx, req.NamespacedName, gr); err != nil {
		if client.IgnoreNotFound(err) == nil {
			gr.Namespace = req.Namespace
			gr.Name = req.Name

			gr.TypeMeta = metav1.TypeMeta{
				Kind:       KindGRPCRoute,
				APIVersion: gatewayv1.GroupVersion.String(),
			}

			if err := r.Provider.Delete(ctx, gr); err != nil {
				r.Log.Error(err, "failed to delete grpcroute", "grpcroute", gr)
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

	gateways, err := ParseRouteParentRefs(ctx, r.Client, gr, gr.Spec.ParentRefs)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(gateways) == 0 {
		return ctrl.Result{}, nil
	}

	tctx := provider.NewDefaultTranslateContext(ctx)

	tctx.RouteParentRefs = gr.Spec.ParentRefs
	rk := utils.NamespacedNameKind(gr)
	for _, gateway := range gateways {
		if err := ProcessGatewayProxy(r.Client, r.Log, tctx, gateway.Gateway, rk); err != nil {
			acceptStatus.status = false
			acceptStatus.msg = err.Error()
		}
		if gateway.Listener != nil {
			tctx.Listeners = append(tctx.Listeners, *gateway.Listener)
		}
	}

	var backendRefErr error
	if err := r.processGRPCRoute(tctx, gr); err != nil {
		// When encountering a backend reference error, it should not affect the acceptance status
		if types.IsSomeReasonError(err, gatewayv1.RouteReasonInvalidKind) {
			backendRefErr = err
		} else {
			acceptStatus.status = false
			acceptStatus.msg = err.Error()
		}
	}

	// Store the backend reference error for later use.
	// If the backend reference error is because of an invalid kind, use this error first
	if err := r.processGRPCRouteBackendRefs(tctx, req.NamespacedName); err != nil && backendRefErr == nil {
		backendRefErr = err
	}

	ProcessBackendTrafficPolicy(r.Client, r.Log, tctx)

	// TODO: diff the old and new status
	gr.Status.Parents = make([]gatewayv1.RouteParentStatus, 0, len(gateways))
	for _, gateway := range gateways {
		parentStatus := gatewayv1.RouteParentStatus{}
		SetRouteParentRef(&parentStatus, gateway.Gateway.Name, gateway.Gateway.Namespace)
		for _, condition := range gateway.Conditions {
			parentStatus.Conditions = MergeCondition(parentStatus.Conditions, condition)
		}
		SetRouteConditionAccepted(&parentStatus, gr.GetGeneration(), acceptStatus.status, acceptStatus.msg)
		SetRouteConditionResolvedRefs(&parentStatus, gr.GetGeneration(), backendRefErr)

		gr.Status.Parents = append(gr.Status.Parents, parentStatus)
	}

	r.Updater.Update(status.Update{
		NamespacedName: utils.NamespacedName(gr),
		Resource:       &gatewayv1.GRPCRoute{},
		Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
			h, ok := obj.(*gatewayv1.GRPCRoute)
			if !ok {
				err := fmt.Errorf("unsupported object type %T", obj)
				panic(err)
			}
			hCopy := h.DeepCopy()
			hCopy.Status = gr.Status
			return hCopy
		}),
	})
	UpdateStatus(r.Updater, r.Log, tctx)

	if isRouteAccepted(gateways) && err == nil {
		routeToUpdate := gr
		if err := r.Provider.Update(ctx, tctx, routeToUpdate); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *GRPCRouteReconciler) listGRPCRoutesByServiceRef(ctx context.Context, obj client.Object) []reconcile.Request {
	endpointSlice, ok := obj.(*discoveryv1.EndpointSlice)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to EndpointSlice")
		return nil
	}
	namespace := endpointSlice.GetNamespace()
	serviceName := endpointSlice.Labels[discoveryv1.LabelServiceName]

	gList := &gatewayv1.GRPCRouteList{}
	if err := r.List(ctx, gList, client.MatchingFields{
		indexer.ServiceIndexRef: indexer.GenIndexKey(namespace, serviceName),
	}); err != nil {
		r.Log.Error(err, "failed to list grpcroutes by service", "service", serviceName)
		return nil
	}
	requests := make([]reconcile.Request, 0, len(gList.Items))
	for _, gr := range gList.Items {
		requests = append(requests, reconcile.Request{
			NamespacedName: client.ObjectKey{
				Namespace: gr.Namespace,
				Name:      gr.Name,
			},
		})
	}
	return requests
}

func (r *GRPCRouteReconciler) listGRPCRoutesForBackendTrafficPolicy(ctx context.Context, obj client.Object) []reconcile.Request {
	policy, ok := obj.(*v1alpha1.BackendTrafficPolicy)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to BackendTrafficPolicy")
		return nil
	}

	grpcRouteList := []gatewayv1.GRPCRoute{}
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
		grList := &gatewayv1.GRPCRouteList{}
		if err := r.List(ctx, grList, client.MatchingFields{
			indexer.ServiceIndexRef: indexer.GenIndexKey(policy.Namespace, string(targetRef.Name)),
		}); err != nil {
			r.Log.Error(err, "failed to list grpcroutes by service reference", "service", targetRef.Name)
			return nil
		}
		grpcRouteList = append(grpcRouteList, grList.Items...)
	}
	var namespacedNameMap = make(map[k8stypes.NamespacedName]struct{})
	requests := make([]reconcile.Request, 0, len(grpcRouteList))
	for _, gr := range grpcRouteList {
		key := k8stypes.NamespacedName{
			Namespace: gr.Namespace,
			Name:      gr.Name,
		}
		if _, ok := namespacedNameMap[key]; !ok {
			namespacedNameMap[key] = struct{}{}
			requests = append(requests, reconcile.Request{
				NamespacedName: client.ObjectKey{
					Namespace: gr.Namespace,
					Name:      gr.Name,
				},
			})
		}
	}
	return requests
}

func (r *GRPCRouteReconciler) listGRPCRoutesForGateway(ctx context.Context, obj client.Object) []reconcile.Request {
	gateway, ok := obj.(*gatewayv1.Gateway)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to Gateway")
	}
	grList := &gatewayv1.GRPCRouteList{}
	if err := r.List(ctx, grList, client.MatchingFields{
		indexer.ParentRefs: indexer.GenIndexKey(gateway.Namespace, gateway.Name),
	}); err != nil {
		r.Log.Error(err, "failed to list grpcroutes by gateway", "gateway", gateway.Name)
		return nil
	}
	requests := make([]reconcile.Request, 0, len(grList.Items))
	for _, gr := range grList.Items {
		requests = append(requests, reconcile.Request{
			NamespacedName: client.ObjectKey{
				Namespace: gr.Namespace,
				Name:      gr.Name,
			},
		})
	}
	return requests
}

func (r *GRPCRouteReconciler) listGRPCRouteForGenericEvent(ctx context.Context, obj client.Object) (requests []reconcile.Request) {
	switch obj.(type) {
	case *v1alpha1.BackendTrafficPolicy:
		return r.listGRPCRoutesForBackendTrafficPolicy(ctx, obj)
	default:
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to BackendTrafficPolicy")
		return nil
	}
}

func (r *GRPCRouteReconciler) processGRPCRouteBackendRefs(tctx *provider.TranslateContext, grNN k8stypes.NamespacedName) error {
	var terr error
	for _, backend := range tctx.BackendRefs {
		targetNN := k8stypes.NamespacedName{
			Namespace: grNN.Namespace,
			Name:      string(backend.Name),
		}
		if backend.Namespace != nil {
			targetNN.Namespace = string(*backend.Namespace)
		}

		if backend.Kind != nil && *backend.Kind != "Service" {
			terr = types.NewInvalidKindError(*backend.Kind)
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
				terr = types.ReasonError{
					Reason:  string(gatewayv1.RouteReasonBackendNotFound),
					Message: fmt.Sprintf("Service %s not found", targetNN),
				}
			}
			continue
		}

		// if cross namespaces between GRPCRoute and referenced Service, check ReferenceGrant
		if grNN.Namespace != targetNN.Namespace {
			if permitted := checkReferenceGrant(tctx,
				r.Client,
				v1beta1.ReferenceGrantFrom{
					Group:     gatewayv1.GroupName,
					Kind:      KindGRPCRoute,
					Namespace: v1beta1.Namespace(grNN.Namespace),
				},
				gatewayv1.ObjectReference{
					Group:     corev1.GroupName,
					Kind:      KindService,
					Name:      gatewayv1.ObjectName(targetNN.Name),
					Namespace: (*gatewayv1.Namespace)(&targetNN.Namespace),
				},
			); !permitted {
				terr = types.ReasonError{
					Reason:  string(v1beta1.RouteReasonRefNotPermitted),
					Message: fmt.Sprintf("%s is in a different namespace than the GRPCRoute %s and no ReferenceGrant allowing reference is configured", targetNN, grNN),
				}
				continue
			}
		}

		if service.Spec.Type == corev1.ServiceTypeExternalName {
			tctx.Services[targetNN] = &service
			continue
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

func (r *GRPCRouteReconciler) processGRPCRoute(tctx *provider.TranslateContext, grpcroute *gatewayv1.GRPCRoute) error {
	var terror error
	for _, rule := range grpcroute.Spec.Rules {
		for _, filter := range rule.Filters {
			if filter.Type != gatewayv1.GRPCRouteFilterExtensionRef || filter.ExtensionRef == nil {
				continue
			}
			if filter.ExtensionRef.Kind == "PluginConfig" {
				pluginconfig := new(v1alpha1.PluginConfig)
				if err := r.Get(context.Background(), client.ObjectKey{
					Namespace: grpcroute.GetNamespace(),
					Name:      string(filter.ExtensionRef.Name),
				}, pluginconfig); err != nil {
					terror = err
					continue
				}
				tctx.PluginConfigs[k8stypes.NamespacedName{
					Namespace: grpcroute.GetNamespace(),
					Name:      string(filter.ExtensionRef.Name),
				}] = pluginconfig
			}
		}
		for _, backend := range rule.BackendRefs {
			if backend.Kind != nil && *backend.Kind != "Service" {
				terror = types.NewInvalidKindError(*backend.Kind)
				continue
			}
			tctx.BackendRefs = append(tctx.BackendRefs, gatewayv1.BackendRef{
				BackendObjectReference: gatewayv1.BackendObjectReference{
					Name:      backend.Name,
					Namespace: cmp.Or(backend.Namespace, (*gatewayv1.Namespace)(&grpcroute.Namespace)),
					Port:      backend.Port,
				},
			})
		}
	}

	return terror
}

// listGRPCRoutesForGatewayProxy list all GRPCRoute resources that are affected by a given GatewayProxy
func (r *GRPCRouteReconciler) listGRPCRoutesForGatewayProxy(ctx context.Context, obj client.Object) []reconcile.Request {
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

	// for each gateway, find all GRPCRoute resources that reference it
	for _, gateway := range gatewayList.Items {
		grpcRouteList := &gatewayv1.GRPCRouteList{}
		if err := r.List(ctx, grpcRouteList, client.MatchingFields{
			indexer.ParentRefs: indexer.GenIndexKey(gateway.Namespace, gateway.Name),
		}); err != nil {
			r.Log.Error(err, "failed to list grpcroutes for gateway", "gateway", gateway.Name)
			continue
		}

		for _, grpcRoute := range grpcRouteList.Items {
			requests = append(requests, reconcile.Request{
				NamespacedName: client.ObjectKey{
					Namespace: grpcRoute.Namespace,
					Name:      grpcRoute.Name,
				},
			})
		}
	}

	return requests
}

func (r *GRPCRouteReconciler) listGRPCRoutesForReferenceGrant(ctx context.Context, obj client.Object) (requests []reconcile.Request) {
	grant, ok := obj.(*v1beta1.ReferenceGrant)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to ReferenceGrant")
		return nil
	}

	var grpcRouteList gatewayv1.GRPCRouteList
	if err := r.List(ctx, &grpcRouteList); err != nil {
		r.Log.Error(err, "failed to list grpcroutes for reference ReferenceGrant", "ReferenceGrant", k8stypes.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()})
		return nil
	}

	for _, grpcRoute := range grpcRouteList.Items {
		gr := v1beta1.ReferenceGrantFrom{
			Group:     gatewayv1.GroupName,
			Kind:      KindGRPCRoute,
			Namespace: v1beta1.Namespace(grpcRoute.GetNamespace()),
		}
		for _, from := range grant.Spec.From {
			if from == gr {
				requests = append(requests, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: grpcRoute.GetNamespace(),
						Name:      grpcRoute.GetName(),
					},
				})
			}
		}
	}
	return requests
}
