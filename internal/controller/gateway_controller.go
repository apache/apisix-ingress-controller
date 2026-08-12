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
	"errors"
	"fmt"
	"net"
	"reflect"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
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
	"github.com/apache/apisix-ingress-controller/internal/controller/indexer"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	internaltypes "github.com/apache/apisix-ingress-controller/internal/types"
	"github.com/apache/apisix-ingress-controller/internal/utils"
	pkgutils "github.com/apache/apisix-ingress-controller/pkg/utils"
)

// GatewayReconciler reconciles a Gateway object.
type GatewayReconciler struct { //nolint:revive
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger

	Provider provider.Provider

	Updater status.Updater
}

// SetupWithManager sets up the controller with the Manager.
func (r *GatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	bdr := ctrl.NewControllerManagedBy(mgr).
		For(
			&gatewayv1.Gateway{},
			builder.WithPredicates(
				predicate.And(
					predicate.NewPredicateFuncs(r.checkGatewayClass),
					predicate.GenerationChangedPredicate{},
				),
			),
		).
		Watches(
			&gatewayv1.GatewayClass{},
			handler.EnqueueRequestsFromMapFunc(r.listGatewayForGatewayClass),
			builder.WithPredicates(
				predicate.NewPredicateFuncs(r.matchesGatewayClass),
			),
		).
		Watches(
			&gatewayv1.HTTPRoute{},
			handler.EnqueueRequestsFromMapFunc(r.listGatewaysForStatusParentRefs),
		).
		Watches(
			&gatewayv1.GRPCRoute{},
			handler.EnqueueRequestsFromMapFunc(r.listGatewaysForStatusParentRefs),
		).
		Watches(
			&v1alpha1.GatewayProxy{},
			handler.EnqueueRequestsFromMapFunc(r.listGatewaysForGatewayProxy),
		).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.listGatewaysForSecret),
		).
		Watches(
			&corev1.ConfigMap{},
			handler.EnqueueRequestsFromMapFunc(r.listGatewaysForConfigMap),
		)

	if GetEnableReferenceGrant() {
		bdr.Watches(&gatewayv1.ReferenceGrant{},
			handler.EnqueueRequestsFromMapFunc(r.listReferenceGrantsForGateway),
			builder.WithPredicates(referenceGrantPredicates(KindGateway)),
		)
	}
	hasTCPRoute, err := pkgutils.HasAPIResource(mgr, &gatewayv1.TCPRoute{})
	if err != nil {
		return err
	}
	if hasTCPRoute {
		bdr.Watches(
			&gatewayv1.TCPRoute{},
			handler.EnqueueRequestsFromMapFunc(r.listGatewaysForStatusParentRefs),
		)
	}
	hasTLSRoute, err := pkgutils.HasAPIResource(mgr, &gatewayv1.TLSRoute{})
	if err != nil {
		return err
	}
	if hasTLSRoute {
		bdr.Watches(
			&gatewayv1.TLSRoute{},
			handler.EnqueueRequestsFromMapFunc(r.listGatewaysForStatusParentRefs),
		)
	}
	hasUDPRoute, err := pkgutils.HasAPIResource(mgr, &gatewayv1.UDPRoute{})
	if err != nil {
		return err
	}
	if hasUDPRoute {
		bdr.Watches(
			&gatewayv1.UDPRoute{},
			handler.EnqueueRequestsFromMapFunc(r.listGatewaysForStatusParentRefs),
		)
	}

	return bdr.Complete(r)
}

func (r *GatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	gateway := new(gatewayv1.Gateway)
	if err := r.Get(ctx, req.NamespacedName, gateway); err != nil {
		if client.IgnoreNotFound(err) == nil {
			gateway.Namespace = req.Namespace
			gateway.Name = req.Name

			gateway.TypeMeta = metav1.TypeMeta{
				Kind:       KindGateway,
				APIVersion: gatewayv1.GroupVersion.String(),
			}

			if err := r.Provider.Delete(ctx, gateway); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	if !r.checkGatewayClass(gateway) {
		return ctrl.Result{}, nil
	}

	conditionProgrammedStatus, conditionProgrammedMsg := true, "Programmed"

	r.Log.Info("gateway has been accepted", "gateway", gateway.GetName())
	type conditionStatus struct {
		status bool
		msg    string
	}
	acceptStatus := conditionStatus{
		status: true,
		msg:    acceptedMessage("gateway"),
	}

	// create a translation context
	tctx := provider.NewDefaultTranslateContext(ctx)

	r.processListenerConfig(tctx, gateway)
	if err := r.processInfrastructure(tctx, gateway); err != nil {
		acceptStatus = conditionStatus{
			status: false,
			msg:    err.Error(),
		}
	}

	var addrs []gatewayv1.GatewayStatusAddress

	rk := utils.NamespacedNameKind(gateway)

	gatewayProxy, ok := tctx.GatewayProxies[rk]
	if !ok {
		acceptStatus = conditionStatus{
			status: false,
			msg:    "gateway proxy not found",
		}
	} else {
		statusAddresses, err := r.resolveStatusAddresses(ctx, &gatewayProxy)
		if err != nil {
			// fail the reconcile so a missing or invalid publish Service retries
			// with backoff, mirroring the Ingress status path
			r.Log.Error(err, "failed to resolve gateway status addresses", "gateway", req.NamespacedName)
			return ctrl.Result{}, err
		}
		for _, addr := range statusAddresses {
			addrType := gatewayv1.IPAddressType
			if net.ParseIP(addr) == nil {
				addrType = gatewayv1.HostnameAddressType
			}
			addrs = append(addrs,
				gatewayv1.GatewayStatusAddress{
					Type:  &addrType,
					Value: addr,
				},
			)
		}
	}

	// deduplicate in case statusAddress contains repeated values
	addrs = deduplicateGatewayStatusAddresses(addrs)

	listenerStatuses, err := getListenerStatus(ctx, r.Client, gateway)
	if err != nil {
		r.Log.Error(err, "failed to get listener status", "gateway", req.NamespacedName)
		return ctrl.Result{}, err
	}

	if err := r.Provider.Update(ctx, tctx, gateway); err != nil {
		acceptStatus = conditionStatus{
			status: false,
			msg:    err.Error(),
		}
	}

	accepted := SetGatewayConditionAccepted(gateway, acceptStatus.status, acceptStatus.msg)
	programmed := SetGatewayConditionProgrammed(gateway, conditionProgrammedStatus, conditionProgrammedMsg)
	addressesChanged := !reflect.DeepEqual(gateway.Status.Addresses, addrs)
	if accepted || programmed || addressesChanged || len(listenerStatuses) > 0 {
		if addressesChanged {
			gateway.Status.Addresses = addrs
		}
		if len(listenerStatuses) > 0 {
			gateway.Status.Listeners = listenerStatuses
		}

		r.Updater.Update(status.Update{
			NamespacedName: utils.NamespacedName(gateway),
			Resource:       &gatewayv1.Gateway{},
			Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
				t, ok := obj.(*gatewayv1.Gateway)
				if !ok {
					err := fmt.Errorf("unsupported object type %T", obj)
					panic(err)
				}
				tCopy := t.DeepCopy()
				tCopy.Status = gateway.Status
				return tCopy
			}),
		})

		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

// resolveStatusAddresses returns the addresses to publish in
// Gateway.status.addresses: the statically configured statusAddress if set,
// otherwise the external addresses of the Service named by publishService.
// This mirrors the Ingress status path, so the same GatewayProxy yields the
// same addresses for both APIs.
func (r *GatewayReconciler) resolveStatusAddresses(
	ctx context.Context,
	gatewayProxy *v1alpha1.GatewayProxy,
) ([]string, error) {
	if len(gatewayProxy.Spec.StatusAddress) > 0 {
		return utils.Filter(gatewayProxy.Spec.StatusAddress, func(addr string) bool {
			return addr != ""
		}), nil
	}

	if gatewayProxy.Spec.PublishService == "" {
		return nil, nil
	}

	// a bare name is resolved against the GatewayProxy's namespace
	svc, err := resolvePublishService(ctx, r.Client, gatewayProxy.Spec.PublishService, gatewayProxy.GetNamespace())
	if err != nil {
		return nil, err
	}
	if svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
		r.Log.Info("publish service is not a LoadBalancer; no address to publish",
			"service", gatewayProxy.Spec.PublishService, "type", svc.Spec.Type)
		return nil, nil
	}
	return serviceLoadBalancerAddresses(svc), nil
}

func (r *GatewayReconciler) matchesGatewayClass(obj client.Object) bool {
	gateway, ok := obj.(*gatewayv1.GatewayClass)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to Gateway")
		return false
	}
	return matchesController(string(gateway.Spec.ControllerName))
}

/*
	func (r *GatewayReconciler) matchesGatewayForControlPlaneConfig(obj client.Object) bool {
		gateway, ok := obj.(*gatewayv1.Gateway)
		if !ok {
			r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to Gateway")
			return false
		}
		cfg := config.GetControlPlaneConfigByGatewatName(gateway.GetName())
		ok = true
		if cfg == nil {
			ok = false
		}
		return ok
	}
*/

func (r *GatewayReconciler) listGatewayForGatewayClass(ctx context.Context, gatewayClass client.Object) []reconcile.Request {
	gatewayList := &gatewayv1.GatewayList{}
	if err := r.List(context.Background(), gatewayList); err != nil {
		r.Log.Error(err, "failed to list gateways for gateway class",
			"gatewayclass", gatewayClass.GetName(),
		)
		return nil
	}

	/*
		gateways := []gatewayv1.Gateway{}
		for _, gateway := range gatewayList.Items {
			if cp := config.GetControlPlaneConfigByGatewatName(gateway.GetName()); cp != nil {
				gateways = append(gateways, gateway)
			}
		}
	*/
	return reconcileGatewaysMatchGatewayClass(gatewayClass, gatewayList.Items)
}

func (r *GatewayReconciler) checkGatewayClass(obj client.Object) bool {
	gateway := obj.(*gatewayv1.Gateway)
	gatewayClass := &gatewayv1.GatewayClass{}
	if err := r.Get(context.Background(), client.ObjectKey{Name: string(gateway.Spec.GatewayClassName)}, gatewayClass); err != nil {
		r.Log.Error(err, "failed to get gateway class", "gateway", gateway.GetName(), "gatewayclass", gateway.Spec.GatewayClassName)
		return false
	}

	return matchesController(string(gatewayClass.Spec.ControllerName))
}

func (r *GatewayReconciler) listGatewaysForGatewayProxy(ctx context.Context, obj client.Object) []reconcile.Request {
	gatewayProxy, ok := obj.(*v1alpha1.GatewayProxy)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to GatewayProxy")
		return nil
	}
	namespace := gatewayProxy.GetNamespace()
	name := gatewayProxy.GetName()

	gatewayList := &gatewayv1.GatewayList{}
	if err := r.List(ctx, gatewayList, client.MatchingFields{
		indexer.ParametersRef: indexer.GenIndexKey(namespace, name),
	}); err != nil {
		r.Log.Error(err, "failed to list gateways for gateway proxy", "gatewayproxy", gatewayProxy.GetName())
		return nil
	}

	recs := make([]reconcile.Request, 0, len(gatewayList.Items))
	for _, gateway := range gatewayList.Items {
		if !r.checkGatewayClass(&gateway) {
			continue
		}
		recs = append(recs, reconcile.Request{
			NamespacedName: client.ObjectKey{
				Namespace: gateway.GetNamespace(),
				Name:      gateway.GetName(),
			},
		})
	}
	return recs
}

func (r *GatewayReconciler) listGatewaysForStatusParentRefs(ctx context.Context, obj client.Object) []reconcile.Request {
	route := internaltypes.NewRouteAdapter(obj)
	reqs := []reconcile.Request{}
	for _, routeParentStatus := range route.GetParentStatuses() {
		gatewayNamespace := route.GetNamespace()
		parentRef := routeParentStatus.ParentRef
		if parentRef.Group != nil && *parentRef.Group != gatewayv1.GroupName {
			continue
		}
		if parentRef.Kind != nil && *parentRef.Kind != internaltypes.KindGateway {
			continue
		}
		if parentRef.Namespace != nil {
			gatewayNamespace = string(*parentRef.Namespace)
		}

		gateway := new(gatewayv1.Gateway)
		if err := r.Get(ctx, client.ObjectKey{
			Namespace: gatewayNamespace,
			Name:      string(parentRef.Name),
		}, gateway); err != nil {
			continue
		}

		if !r.checkGatewayClass(gateway) {
			continue
		}

		reqs = append(reqs, reconcile.Request{
			NamespacedName: client.ObjectKey{
				Namespace: gatewayNamespace,
				Name:      string(parentRef.Name),
			},
		})
	}
	return reqs
}

func (r *GatewayReconciler) listGatewaysForSecret(ctx context.Context, obj client.Object) (requests []reconcile.Request) {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		r.Log.Error(
			errors.New("unexpected object type"),
			"Secret watch predicate received unexpected object type",
			"expected", FullTypeName(new(corev1.Secret)), "found", FullTypeName(obj),
		)
		return nil
	}
	var gatewayList gatewayv1.GatewayList
	if err := r.List(ctx, &gatewayList, client.MatchingFields{
		indexer.SecretIndexRef: indexer.GenIndexKey(secret.GetNamespace(), secret.GetName()),
	}); err != nil {
		r.Log.Error(err, "failed to list gateways")
		return nil
	}
	for _, gateway := range gatewayList.Items {
		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: gateway.GetNamespace(),
				Name:      gateway.GetName(),
			},
		})
	}
	return requests
}

func (r *GatewayReconciler) listGatewaysForConfigMap(ctx context.Context, obj client.Object) (requests []reconcile.Request) {
	configMap, ok := obj.(*corev1.ConfigMap)
	if !ok {
		r.Log.Error(
			errors.New("unexpected object type"),
			"ConfigMap watch predicate received unexpected object type",
			"expected", FullTypeName(new(corev1.ConfigMap)), "found", FullTypeName(obj),
		)
		return nil
	}
	var gatewayList gatewayv1.GatewayList
	if err := r.List(ctx, &gatewayList, client.MatchingFields{
		indexer.ConfigMapIndexRef: indexer.GenIndexKey(configMap.GetNamespace(), configMap.GetName()),
	}); err != nil {
		r.Log.Error(err, "failed to list gateways for configmap")
		return nil
	}
	for _, gateway := range gatewayList.Items {
		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: gateway.GetNamespace(),
				Name:      gateway.GetName(),
			},
		})
	}
	return requests
}

func (r *GatewayReconciler) listReferenceGrantsForGateway(ctx context.Context, obj client.Object) (requests []reconcile.Request) {
	grant, ok := obj.(*gatewayv1.ReferenceGrant)
	if !ok {
		r.Log.Error(
			errors.New("unexpected object type"),
			"ReferenceGrant watch predicate received unexpected object type",
			"expected", FullTypeName(new(gatewayv1.ReferenceGrant)), "found", FullTypeName(obj),
		)
		return nil
	}

	var gatewayList gatewayv1.GatewayList
	if err := r.List(ctx, &gatewayList); err != nil {
		r.Log.Error(err, "failed to list gateways in watch predicate", "ReferenceGrant", grant.GetName())
		return nil
	}

	for _, gateway := range gatewayList.Items {
		gw := gatewayv1.ReferenceGrantFrom{
			Group:     gatewayv1.GroupName,
			Kind:      KindGateway,
			Namespace: gatewayv1.Namespace(gateway.GetNamespace()),
		}
		for _, from := range grant.Spec.From {
			if from == gw {
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Namespace: gateway.GetNamespace(),
						Name:      gateway.GetName(),
					},
				})
			}
		}
	}
	return requests
}

func (r *GatewayReconciler) processInfrastructure(tctx *provider.TranslateContext, gateway *gatewayv1.Gateway) error {
	return ProcessGatewayProxy(r.Client, r.Log, tctx, gateway, utils.NamespacedNameKind(gateway))
}

func (r *GatewayReconciler) processListenerConfig(tctx *provider.TranslateContext, gateway *gatewayv1.Gateway) {
	listeners := gateway.Spec.Listeners
	for _, listener := range listeners {
		if listener.TLS == nil {
			continue
		}
		for _, ref := range listener.TLS.CertificateRefs {
			ns := gateway.GetNamespace()
			if ref.Namespace != nil {
				ns = string(*ref.Namespace)
			}
			if ref.Kind != nil && *ref.Kind == KindSecret {
				// Declared per ref: a listener may carry several certificateRefs, and
				// each tctx.Secrets entry must point to its own Secret rather than all
				// aliasing one shared variable that ends up holding the last one loaded.
				secret := corev1.Secret{}
				// A cross-namespace certificateRef must be authorized by a ReferenceGrant,
				// or the data plane would program a certificate the target namespace never
				// permitted. The listener status already reports RefNotPermitted for this.
				if !checkReferenceGrant(context.Background(), r.Client,
					gatewayv1.ReferenceGrantFrom{
						Group:     gatewayv1.GroupName,
						Kind:      KindGateway,
						Namespace: gatewayv1.Namespace(gateway.Namespace),
					},
					gatewayv1.ObjectReference{
						Group:     corev1.GroupName,
						Kind:      KindSecret,
						Name:      ref.Name,
						Namespace: ref.Namespace,
					},
				) {
					r.Log.V(1).Info("skipping cross-namespace certificateRef not permitted by any ReferenceGrant",
						"listener", listener.Name, "secret", client.ObjectKey{Namespace: ns, Name: string(ref.Name)})
					continue
				}
				if err := r.Get(context.Background(), client.ObjectKey{
					Namespace: ns,
					Name:      string(ref.Name),
				}, &secret); err != nil {
					r.Log.Error(err, "failed to get secret", "namespace", ns, "name", ref.Name)
					SetGatewayListenerConditionProgrammed(gateway, string(listener.Name), false, err.Error())
					SetGatewayListenerConditionResolvedRefs(gateway, string(listener.Name), false, err.Error())
					break
				}
				r.Log.Info("Setting secret for listener", "listener", listener.Name, "secret", secret.Name, " namespace", ns)
				tctx.Secrets[types.NamespacedName{Namespace: ns, Name: string(ref.Name)}] = &secret
			}
		}
		// frontendValidation references CA ConfigMaps or Secrets used for downstream mTLS.
		// In Gateway API v1.6 it is declared at the Gateway level (spec.tls.frontend);
		// resolve the config that applies to this HTTPS listener by its port.
		if validation := internaltypes.FrontendTLSValidationForListener(gateway, listener); validation != nil {
			for _, ref := range validation.CACertificateRefs {
				ns := gateway.GetNamespace()
				if ref.Namespace != nil {
					ns = string(*ref.Namespace)
				}
				nn := types.NamespacedName{Namespace: ns, Name: string(ref.Name)}
				kind := KindConfigMap
				if ref.Kind != "" {
					kind = string(ref.Kind)
				}
				// A cross-namespace CA ref must be authorized by a ReferenceGrant, or the
				// data plane would enable downstream mTLS with a CA the target namespace
				// never permitted. The listener status already reports RefNotPermitted.
				if !checkReferenceGrant(context.Background(), r.Client,
					gatewayv1.ReferenceGrantFrom{
						Group:     gatewayv1.GroupName,
						Kind:      KindGateway,
						Namespace: gatewayv1.Namespace(gateway.Namespace),
					},
					gatewayv1.ObjectReference{
						Group:     corev1.GroupName,
						Kind:      gatewayv1.Kind(kind),
						Name:      ref.Name,
						Namespace: ref.Namespace,
					},
				) {
					r.Log.V(1).Info("skipping cross-namespace caCertificateRef not permitted by any ReferenceGrant",
						"listener", listener.Name, "ref", nn)
					continue
				}
				switch kind {
				case KindConfigMap:
					configMap := corev1.ConfigMap{}
					if err := r.Get(context.Background(), nn, &configMap); err != nil {
						r.Log.Error(err, "failed to get CA configmap", "namespace", ns, "name", ref.Name)
						SetGatewayListenerConditionProgrammed(gateway, string(listener.Name), false, err.Error())
						SetGatewayListenerConditionResolvedRefs(gateway, string(listener.Name), false, err.Error())
						continue
					}
					r.Log.Info("Setting CA configmap for listener", "listener", listener.Name, "configmap", configMap.Name, "namespace", ns)
					tctx.ConfigMaps[nn] = &configMap
				case KindSecret:
					caSecret := corev1.Secret{}
					if err := r.Get(context.Background(), nn, &caSecret); err != nil {
						r.Log.Error(err, "failed to get CA secret", "namespace", ns, "name", ref.Name)
						SetGatewayListenerConditionProgrammed(gateway, string(listener.Name), false, err.Error())
						SetGatewayListenerConditionResolvedRefs(gateway, string(listener.Name), false, err.Error())
						continue
					}
					r.Log.Info("Setting CA secret for listener", "listener", listener.Name, "secret", caSecret.Name, "namespace", ns)
					tctx.Secrets[nn] = &caSecret
				}
			}
		}
	}
}
