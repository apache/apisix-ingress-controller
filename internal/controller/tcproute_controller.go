// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	"sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/controller/indexer"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	"github.com/apache/apisix-ingress-controller/internal/manager/readiness"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/internal/types"
	"github.com/apache/apisix-ingress-controller/internal/utils"
)

// TCPRouteReconciler reconciles a TCPRoute object.
type TCPRouteReconciler struct { //nolint:revive
	client.Client
	Scheme *runtime.Scheme

	Log logr.Logger

	Provider provider.Provider

	Updater status.Updater
	Readier readiness.ReadinessManager
}

// SetupWithManager sets up the controller with the Manager.
func (r *TCPRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {

	bdr := ctrl.NewControllerManagedBy(mgr).
		For(&gatewayv1alpha2.TCPRoute{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Watches(&discoveryv1.EndpointSlice{},
			handler.EnqueueRequestsFromMapFunc(r.listTCPRoutesByServiceRef),
		).
		Watches(&gatewayv1.Gateway{},
			handler.EnqueueRequestsFromMapFunc(r.listTCPRoutesForGateway),
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
			handler.EnqueueRequestsFromMapFunc(r.listTCPRoutesForBackendTrafficPolicy),
		).
		Watches(&v1alpha1.GatewayProxy{},
			handler.EnqueueRequestsFromMapFunc(r.listTCPRoutesForGatewayProxy),
		)

	if GetEnableReferenceGrant() {
		bdr.Watches(&v1beta1.ReferenceGrant{},
			handler.EnqueueRequestsFromMapFunc(r.listTCPRoutesForReferenceGrant),
			builder.WithPredicates(referenceGrantPredicates(KindTCPRoute)),
		)
	}

	return bdr.Complete(r)
}

func (r *TCPRouteReconciler) listTCPRoutesForBackendTrafficPolicy(ctx context.Context, obj client.Object) []reconcile.Request {
	policy, ok := obj.(*v1alpha1.BackendTrafficPolicy)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to BackendTrafficPolicy")
		return nil
	}

	tcprouteList := []gatewayv1alpha2.TCPRoute{}
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
		tcprList := &gatewayv1alpha2.TCPRouteList{}
		if err := r.List(ctx, tcprList, client.MatchingFields{
			indexer.ServiceIndexRef: indexer.GenIndexKey(policy.Namespace, string(targetRef.Name)),
		}); err != nil {
			r.Log.Error(err, "failed to list tcproutes by service reference", "service", targetRef.Name)
			return nil
		}
		tcprouteList = append(tcprouteList, tcprList.Items...)
	}
	var namespacedNameMap = make(map[k8stypes.NamespacedName]struct{})
	requests := make([]reconcile.Request, 0, len(tcprouteList))
	for _, tr := range tcprouteList {
		key := k8stypes.NamespacedName{
			Namespace: tr.Namespace,
			Name:      tr.Name,
		}
		if _, ok := namespacedNameMap[key]; !ok {
			namespacedNameMap[key] = struct{}{}
			requests = append(requests, reconcile.Request{
				NamespacedName: client.ObjectKey{
					Namespace: tr.Namespace,
					Name:      tr.Name,
				},
			})
		}
	}
	return requests
}

func (r *TCPRouteReconciler) listTCPRoutesForGateway(ctx context.Context, obj client.Object) []reconcile.Request {
	gateway, ok := obj.(*gatewayv1.Gateway)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to Gateway")
	}
	tcprList := &gatewayv1alpha2.TCPRouteList{}
	if err := r.List(ctx, tcprList, client.MatchingFields{
		indexer.ParentRefs: indexer.GenIndexKey(gateway.Namespace, gateway.Name),
	}); err != nil {
		r.Log.Error(err, "failed to list tcproutes by gateway", "gateway", gateway.Name)
		return nil
	}

	requests := make([]reconcile.Request, 0, len(tcprList.Items))
	for _, tcr := range tcprList.Items {
		requests = append(requests, reconcile.Request{
			NamespacedName: client.ObjectKey{
				Namespace: tcr.Namespace,
				Name:      tcr.Name,
			},
		})
	}
	return requests
}

// listTCPRoutesForGatewayProxy list all TCPRoute resources that are affected by a given GatewayProxy
func (r *TCPRouteReconciler) listTCPRoutesForGatewayProxy(ctx context.Context, obj client.Object) []reconcile.Request {
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

	// for each gateway, find all TCPRoute resources that reference it
	for _, gateway := range gatewayList.Items {
		tcpRouteList := &gatewayv1alpha2.TCPRouteList{}
		if err := r.List(ctx, tcpRouteList, client.MatchingFields{
			indexer.ParentRefs: indexer.GenIndexKey(gateway.Namespace, gateway.Name),
		}); err != nil {
			r.Log.Error(err, "failed to list tcproutes for gateway", "gateway", gateway.Name)
			continue
		}

		for _, tcpRoute := range tcpRouteList.Items {
			requests = append(requests, reconcile.Request{
				NamespacedName: client.ObjectKey{
					Namespace: tcpRoute.Namespace,
					Name:      tcpRoute.Name,
				},
			})
		}
	}

	return requests
}

func (r *TCPRouteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	defer r.Readier.Done(&gatewayv1alpha2.TCPRoute{}, req.NamespacedName)
	tr := new(gatewayv1alpha2.TCPRoute)
	if err := r.Get(ctx, req.NamespacedName, tr); err != nil {
		if client.IgnoreNotFound(err) == nil {
			tr.Namespace = req.Namespace
			tr.Name = req.Name

			tr.TypeMeta = metav1.TypeMeta{
				Kind:       KindTCPRoute,
				APIVersion: gatewayv1alpha2.GroupVersion.String(),
			}

			if err := r.Provider.Delete(ctx, tr); err != nil {
				r.Log.Error(err, "failed to delete tcproute", "tcproute", tr)
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

	acceptStatus := ResourceStatus{
		status: true,
		msg:    "Route is accepted",
	}

	gateways, err := ParseRouteParentRefs(ctx, r.Client, tr, tr.Spec.ParentRefs)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(gateways) == 0 {
		return ctrl.Result{}, nil
	}

	tctx := provider.NewDefaultTranslateContext(ctx)

	tctx.RouteParentRefs = tr.Spec.ParentRefs
	rk := utils.NamespacedNameKind(tr)
	for _, gateway := range gateways {
		if err := ProcessGatewayProxy(r.Client, r.Log, tctx, gateway.Gateway, rk); err != nil {
			acceptStatus.status = false
			acceptStatus.msg = err.Error()
		}
	}

	var backendRefErr error
	if err := r.processTCPRoute(tctx, tr); err != nil {
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
	if err := r.processTCPRouteBackendRefs(tctx, req.NamespacedName); err != nil && backendRefErr == nil {
		backendRefErr = err
	}

	ProcessBackendTrafficPolicy(r.Client, r.Log, tctx)
	tr.Status.Parents = make([]gatewayv1.RouteParentStatus, 0, len(gateways))
	for _, gateway := range gateways {
		parentStatus := gatewayv1.RouteParentStatus{}
		SetRouteParentRef(&parentStatus, gateway.Gateway.Name, gateway.Gateway.Namespace)
		for _, condition := range gateway.Conditions {
			parentStatus.Conditions = MergeCondition(parentStatus.Conditions, condition)
		}
		SetRouteConditionAccepted(&parentStatus, tr.GetGeneration(), acceptStatus.status, acceptStatus.msg)
		SetRouteConditionResolvedRefs(&parentStatus, tr.GetGeneration(), backendRefErr)

		tr.Status.Parents = append(tr.Status.Parents, parentStatus)
	}

	r.Updater.Update(status.Update{
		NamespacedName: utils.NamespacedName(tr),
		Resource:       &gatewayv1alpha2.TCPRoute{},
		Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
			t, ok := obj.(*gatewayv1alpha2.TCPRoute)
			if !ok {
				err := fmt.Errorf("unsupported object type %T", obj)
				panic(err)
			}
			tCopy := t.DeepCopy()
			tCopy.Status = tr.Status
			return tCopy
		}),
	})
	UpdateStatus(r.Updater, r.Log, tctx)
	if isRouteAccepted(gateways) {
		routeToUpdate := tr
		if err := r.Provider.Update(ctx, tctx, routeToUpdate); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *TCPRouteReconciler) processTCPRoute(tctx *provider.TranslateContext, tcpRoute *gatewayv1alpha2.TCPRoute) error {
	var terror error
	for _, rule := range tcpRoute.Spec.Rules {
		for _, backend := range rule.BackendRefs {
			if backend.Kind != nil && *backend.Kind != KindService {
				terror = types.NewInvalidKindError(*backend.Kind)
				continue
			}
			tctx.BackendRefs = append(tctx.BackendRefs, gatewayv1.BackendRef{
				BackendObjectReference: gatewayv1.BackendObjectReference{
					Name:      backend.Name,
					Namespace: cmp.Or(backend.Namespace, (*gatewayv1.Namespace)(&tcpRoute.Namespace)),
					Port:      backend.Port,
				},
			})
		}
	}

	return terror
}

func (r *TCPRouteReconciler) processTCPRouteBackendRefs(tctx *provider.TranslateContext, trNN k8stypes.NamespacedName) error {
	var terr error
	for _, backend := range tctx.BackendRefs {
		targetNN := k8stypes.NamespacedName{
			Namespace: trNN.Namespace,
			Name:      string(backend.Name),
		}
		if backend.Namespace != nil {
			targetNN.Namespace = string(*backend.Namespace)
		}

		if backend.Kind != nil && *backend.Kind != KindService {
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

		// if cross namespaces between TCPRoute and referenced Service, check ReferenceGrant
		if trNN.Namespace != targetNN.Namespace {
			if permitted := checkReferenceGrant(tctx,
				r.Client,
				v1beta1.ReferenceGrantFrom{
					Group:     gatewayv1.GroupName,
					Kind:      KindTCPRoute,
					Namespace: v1beta1.Namespace(trNN.Namespace),
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
					Message: fmt.Sprintf("%s is in a different namespace than the TCPRoute %s and no ReferenceGrant allowing reference is configured", targetNN, trNN),
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

func (r *TCPRouteReconciler) listTCPRoutesForReferenceGrant(ctx context.Context, obj client.Object) (requests []reconcile.Request) {
	grant, ok := obj.(*v1beta1.ReferenceGrant)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to ReferenceGrant")
		return nil
	}

	var tcpRouteList gatewayv1alpha2.TCPRouteList
	if err := r.List(ctx, &tcpRouteList); err != nil {
		r.Log.Error(err, "failed to list tcproutes for reference ReferenceGrant", "ReferenceGrant", k8stypes.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()})
		return nil
	}

	for _, tcpRoute := range tcpRouteList.Items {
		tr := v1beta1.ReferenceGrantFrom{
			Group:     gatewayv1.GroupName,
			Kind:      KindTCPRoute,
			Namespace: v1beta1.Namespace(tcpRoute.GetNamespace()),
		}
		for _, from := range grant.Spec.From {
			if from == tr {
				requests = append(requests, reconcile.Request{
					NamespacedName: client.ObjectKey{
						Namespace: tcpRoute.GetNamespace(),
						Name:      tcpRoute.GetName(),
					},
				})
			}
		}
	}
	return requests
}

func (r *TCPRouteReconciler) listTCPRoutesByServiceRef(ctx context.Context, obj client.Object) []reconcile.Request {
	endpointSlice, ok := obj.(*discoveryv1.EndpointSlice)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to EndpointSlice")
		return nil
	}
	namespace := endpointSlice.GetNamespace()
	serviceName := endpointSlice.Labels[discoveryv1.LabelServiceName]

	trList := &gatewayv1alpha2.TCPRouteList{}
	if err := r.List(ctx, trList, client.MatchingFields{
		indexer.ServiceIndexRef: indexer.GenIndexKey(namespace, serviceName),
	}); err != nil {
		r.Log.Error(err, "failed to list tcproutes by service", "service", serviceName)
		return nil
	}
	requests := make([]reconcile.Request, 0, len(trList.Items))
	for _, tr := range trList.Items {
		requests = append(requests, reconcile.Request{
			NamespacedName: client.ObjectKey{
				Namespace: tr.Namespace,
				Name:      tr.Name,
			},
		})
	}
	return requests
}
