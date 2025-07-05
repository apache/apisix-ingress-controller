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
	"errors"
	"fmt"
	"slices"

	"github.com/api7/gopkg/pkg/log"
	"github.com/go-logr/logr"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
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
	"github.com/apache/apisix-ingress-controller/internal/controller/indexer"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/internal/utils"
	pkgutils "github.com/apache/apisix-ingress-controller/pkg/utils"
)

// ApisixRouteReconciler reconciles a ApisixRoute object
type ApisixRouteReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Provider provider.Provider
	Updater  status.Updater
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApisixRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv2.ApisixRoute{}).
		WithEventFilter(
			predicate.Or(
				predicate.GenerationChangedPredicate{},
				predicate.AnnotationChangedPredicate{},
				predicate.NewPredicateFuncs(TypePredicate[*corev1.Secret]()),
			),
		).
		Watches(
			&networkingv1.IngressClass{},
			handler.EnqueueRequestsFromMapFunc(r.listApisixRouteForIngressClass),
			builder.WithPredicates(
				predicate.NewPredicateFuncs(matchesIngressController),
			),
		).
		Watches(&v1alpha1.GatewayProxy{},
			handler.EnqueueRequestsFromMapFunc(r.listApisixRouteForGatewayProxy),
		).
		Watches(&discoveryv1.EndpointSlice{},
			handler.EnqueueRequestsFromMapFunc(r.listApisixRoutesForService),
		).
		Watches(&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.listApisixRoutesForSecret),
		).
		Watches(&apiv2.ApisixUpstream{},
			handler.EnqueueRequestsFromMapFunc(r.listApisixRouteForApisixUpstream),
		).
		Watches(&apiv2.ApisixPluginConfig{},
			handler.EnqueueRequestsFromMapFunc(r.listApisixRoutesForPluginConfig),
		).
		Named("apisixroute").
		Complete(r)
}

func (r *ApisixRouteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var ar apiv2.ApisixRoute
	if err := r.Get(ctx, req.NamespacedName, &ar); err != nil {
		if client.IgnoreNotFound(err) == nil {
			ar.Namespace = req.Namespace
			ar.Name = req.Name
			ar.TypeMeta = metav1.TypeMeta{
				Kind:       KindApisixRoute,
				APIVersion: apiv2.GroupVersion.String(),
			}

			if err := r.Provider.Delete(ctx, &ar); err != nil {
				r.Log.Error(err, "failed to delete apisixroute", "apisixroute", ar)
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	var (
		tctx = provider.NewDefaultTranslateContext(ctx)
		ic   *networkingv1.IngressClass
		err  error
	)
	defer func() {
		r.updateStatus(&ar, err)
	}()

	if ic, err = GetIngressClass(tctx, r.Client, r.Log, ar.Spec.IngressClassName); err != nil {
		return ctrl.Result{}, err
	}
	if err = ProcessIngressClassParameters(tctx, r.Client, r.Log, &ar, ic); err != nil {
		return ctrl.Result{}, err
	}
	if err = r.processApisixRoute(ctx, tctx, &ar); err != nil {
		return ctrl.Result{}, err
	}
	if err = r.Provider.Update(ctx, tctx, &ar); err != nil {
		err = ReasonError{
			Reason:  string(apiv2.ConditionReasonSyncFailed),
			Message: err.Error(),
		}
		r.Log.Error(err, "failed to process", "apisixroute", ar)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ApisixRouteReconciler) processApisixRoute(ctx context.Context, tc *provider.TranslateContext, in *apiv2.ApisixRoute) error {
	var (
		rules = make(map[string]struct{})
	)
	for httpIndex, http := range in.Spec.HTTP {
		// check rule names
		if _, ok := rules[http.Name]; ok {
			return ReasonError{
				Reason:  string(apiv2.ConditionReasonInvalidSpec),
				Message: "duplicate route rule name",
			}
		}
		rules[http.Name] = struct{}{}

		// check secret
		for _, plugin := range http.Plugins {
			if !plugin.Enable {
				continue
			}
			// check secret
			if err := r.validateSecrets(ctx, tc, in, plugin.SecretRef); err != nil {
				return err
			}
		}
		// check plugin config reference
		if http.PluginConfigName != "" {
			if err := r.validatePluginConfig(ctx, tc, in, http); err != nil {
				return err
			}
		}

		// check vars
		if _, err := http.Match.NginxVars.ToVars(); err != nil {
			return ReasonError{
				Reason:  string(apiv2.ConditionReasonInvalidSpec),
				Message: fmt.Sprintf(".spec.http[%d].match.exprs: %s", httpIndex, err.Error()),
			}
		}

		// validate remote address
		if err := utils.ValidateRemoteAddrs(http.Match.RemoteAddrs); err != nil {
			return ReasonError{
				Reason:  string(apiv2.ConditionReasonInvalidSpec),
				Message: fmt.Sprintf(".spec.http[%d].match.remoteAddrs: %s", httpIndex, err.Error()),
			}
		}

		// process backend
		if err := r.validateBackends(ctx, tc, in, http); err != nil {
			return err
		}
		// process upstreams
		if err := r.validateUpstreams(ctx, tc, in, http); err != nil {
			return err
		}
	}

	return nil
}

func (r *ApisixRouteReconciler) validatePluginConfig(ctx context.Context, tc *provider.TranslateContext, in *apiv2.ApisixRoute, http apiv2.ApisixRouteHTTP) error {
	pcNamespace := in.Namespace
	if http.PluginConfigNamespace != "" {
		pcNamespace = http.PluginConfigNamespace
	}
	var (
		pc = apiv2.ApisixPluginConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      http.PluginConfigName,
				Namespace: pcNamespace,
			},
		}
		pcNN = utils.NamespacedName(&pc)
	)
	if err := r.Get(ctx, pcNN, &pc); err != nil {
		return ReasonError{
			Reason:  string(apiv2.ConditionReasonInvalidSpec),
			Message: fmt.Sprintf("failed to get ApisixPluginConfig: %s", pcNN),
		}
	}

	// Check if ApisixPluginConfig has IngressClassName and if it matches
	if in.Spec.IngressClassName != pc.Spec.IngressClassName && pc.Spec.IngressClassName != "" {
		var pcIC networkingv1.IngressClass
		if err := r.Get(ctx, client.ObjectKey{Name: pc.Spec.IngressClassName}, &pcIC); err != nil {
			return ReasonError{
				Reason:  string(apiv2.ConditionReasonInvalidSpec),
				Message: fmt.Sprintf("failed to get IngressClass %s for ApisixPluginConfig %s: %v", pc.Spec.IngressClassName, pcNN, err),
			}
		}
		if !matchesController(pcIC.Spec.Controller) {
			return ReasonError{
				Reason:  string(apiv2.ConditionReasonInvalidSpec),
				Message: fmt.Sprintf("ApisixPluginConfig %s references IngressClass %s with non-matching controller", pcNN, pc.Spec.IngressClassName),
			}
		}
	}

	tc.ApisixPluginConfigs[pcNN] = &pc

	// Also check secrets referenced by plugin config
	for _, plugin := range pc.Spec.Plugins {
		if !plugin.Enable {
			continue
		}
		if err := r.validateSecrets(ctx, tc, in, plugin.SecretRef); err != nil {
			return err
		}
	}
	return nil
}

func (r *ApisixRouteReconciler) validateSecrets(ctx context.Context, tc *provider.TranslateContext, in *apiv2.ApisixRoute, secretRef string) error {
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
	if err := r.Get(ctx, secretNN, &secret); err != nil {
		return ReasonError{
			Reason:  string(apiv2.ConditionReasonInvalidSpec),
			Message: fmt.Sprintf("failed to get Secret: %s", secretNN),
		}
	}

	tc.Secrets[utils.NamespacedName(&secret)] = &secret
	return nil
}

func (r *ApisixRouteReconciler) validateBackends(ctx context.Context, tc *provider.TranslateContext, in *apiv2.ApisixRoute, http apiv2.ApisixRouteHTTP) error {
	var backends = make(map[types.NamespacedName]struct{})
	for _, backend := range http.Backends {
		var (
			au        apiv2.ApisixUpstream
			service   corev1.Service
			serviceNN = types.NamespacedName{
				Namespace: in.GetNamespace(),
				Name:      backend.ServiceName,
			}
		)
		if _, ok := backends[serviceNN]; ok {
			return ReasonError{
				Reason:  string(apiv2.ConditionReasonInvalidSpec),
				Message: fmt.Sprintf("duplicate backend service: %s", serviceNN),
			}
		}
		backends[serviceNN] = struct{}{}

		if err := r.Get(ctx, serviceNN, &service); err != nil {
			if err = client.IgnoreNotFound(err); err == nil {
				r.Log.Error(errors.New("service not found"), "Service", serviceNN)
				continue
			}
			return err
		}

		// try to get apisixupstream with the same name as the backend service
		log.Debugw("try to get apisixupstream with the same name as the backend service", zap.Stringer("Service", serviceNN))
		if err := r.Get(ctx, serviceNN, &au); err != nil {
			log.Debugw("no ApisixUpstream with the same name as the backend service found", zap.Stringer("Service", serviceNN), zap.Error(err))
			if err = client.IgnoreNotFound(err); err != nil {
				return err
			}
		} else {
			tc.Upstreams[serviceNN] = &au
		}

		if service.Spec.Type == corev1.ServiceTypeExternalName {
			tc.Services[serviceNN] = &service
			continue
		}

		if backend.ResolveGranularity == "service" && service.Spec.ClusterIP == "" {
			r.Log.Error(errors.New("service has no ClusterIP"), "Service", serviceNN, "ResolveGranularity", backend.ResolveGranularity)
			continue
		}

		if !slices.ContainsFunc(service.Spec.Ports, func(port corev1.ServicePort) bool {
			return port.Port == int32(backend.ServicePort.IntValue())
		}) {
			r.Log.Error(errors.New("port not found in service"), "Service", serviceNN, "port", backend.ServicePort.String())
			continue
		}
		tc.Services[serviceNN] = &service

		var endpoints discoveryv1.EndpointSliceList
		if err := r.List(ctx, &endpoints,
			client.InNamespace(service.Namespace),
			client.MatchingLabels{
				discoveryv1.LabelServiceName: service.Name,
			},
		); err != nil {
			return ReasonError{
				Reason:  string(apiv2.ConditionReasonInvalidSpec),
				Message: fmt.Sprintf("failed to list endpoint slices: %v", err),
			}
		}

		// backend.subset specifies a subset of upstream nodes.
		// It specifies that the target pod's label should be a superset of the subset labels of the ApisixUpstream of the serviceName
		subsetLabels := r.getSubsetLabels(tc, serviceNN, backend)
		tc.EndpointSlices[serviceNN] = r.filterEndpointSlicesBySubsetLabels(ctx, endpoints.Items, subsetLabels)
	}

	return nil
}

func (r *ApisixRouteReconciler) validateUpstreams(ctx context.Context, tc *provider.TranslateContext, ar *apiv2.ApisixRoute, http apiv2.ApisixRouteHTTP) error {
	for _, upstream := range http.Upstreams {
		if upstream.Name == "" {
			continue
		}
		var (
			ups   apiv2.ApisixUpstream
			upsNN = types.NamespacedName{
				Namespace: ar.GetNamespace(),
				Name:      upstream.Name,
			}
		)
		if err := r.Get(ctx, upsNN, &ups); err != nil {
			r.Log.Error(err, "failed to get ApisixUpstream", "ApisixUpstream", upsNN)
			if client.IgnoreNotFound(err) == nil {
				continue
			}
			return err
		}
		tc.Upstreams[upsNN] = &ups

		for _, node := range ups.Spec.ExternalNodes {
			if node.Type == apiv2.ExternalTypeService {
				var (
					service   corev1.Service
					serviceNN = types.NamespacedName{Namespace: ups.GetNamespace(), Name: node.Name}
				)
				if err := r.Get(ctx, serviceNN, &service); err != nil {
					r.Log.Error(err, "failed to get service in ApisixUpstream", "ApisixUpstream", upsNN, "Service", serviceNN)
					if client.IgnoreNotFound(err) == nil {
						continue
					}
					return err
				}
				tc.Services[utils.NamespacedName(&service)] = &service
			}
		}

		if ups.Spec.TLSSecret != nil && ups.Spec.TLSSecret.Name != "" {
			var (
				secret   corev1.Secret
				secretNN = types.NamespacedName{Namespace: cmp.Or(ups.Spec.TLSSecret.Namespace, ar.GetNamespace()), Name: ups.Spec.TLSSecret.Name}
			)
			if err := r.Get(ctx, secretNN, &secret); err != nil {
				r.Log.Error(err, "failed to get secret in ApisixUpstream", "ApisixUpstream", upsNN, "Secret", secretNN)
				if client.IgnoreNotFound(err) != nil {
					return err
				}
			}
			tc.Secrets[secretNN] = &secret
		}
	}

	return nil
}

func (r *ApisixRouteReconciler) listApisixRoutesForService(ctx context.Context, obj client.Object) []reconcile.Request {
	endpointSlice, ok := obj.(*discoveryv1.EndpointSlice)
	if !ok {
		return nil
	}

	var (
		namespace   = endpointSlice.GetNamespace()
		serviceName = endpointSlice.Labels[discoveryv1.LabelServiceName]
		arList      apiv2.ApisixRouteList
	)
	if err := r.List(ctx, &arList, client.MatchingFields{
		indexer.ServiceIndexRef: indexer.GenIndexKey(namespace, serviceName),
	}); err != nil {
		r.Log.Error(err, "failed to list apisixroutes by service", "service", serviceName)
		return nil
	}
	requests := make([]reconcile.Request, 0, len(arList.Items))
	for _, ar := range arList.Items {
		requests = append(requests, reconcile.Request{NamespacedName: utils.NamespacedName(&ar)})
	}
	return pkgutils.DedupComparable(requests)
}

func (r *ApisixRouteReconciler) listApisixRoutesForSecret(ctx context.Context, obj client.Object) []reconcile.Request {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		return nil
	}

	var (
		arList      apiv2.ApisixRouteList
		pcList      apiv2.ApisixPluginConfigList
		allRequests = make([]reconcile.Request, 0)
	)

	// First, find ApisixRoutes that directly reference this secret
	if err := r.List(ctx, &arList, client.MatchingFields{
		indexer.SecretIndexRef: indexer.GenIndexKey(secret.GetNamespace(), secret.GetName()),
	}); err != nil {
		r.Log.Error(err, "failed to list apisixroutes by secret", "secret", secret.Name)
		return nil
	}
	for _, ar := range arList.Items {
		allRequests = append(allRequests, reconcile.Request{NamespacedName: utils.NamespacedName(&ar)})
	}

	// Second, find ApisixPluginConfigs that reference this secret
	if err := r.List(ctx, &pcList, client.MatchingFields{
		indexer.SecretIndexRef: indexer.GenIndexKey(secret.GetNamespace(), secret.GetName()),
	}); err != nil {
		r.Log.Error(err, "failed to list apisixpluginconfigs by secret", "secret", secret.Name)
		return nil
	}

	// Then find ApisixRoutes that reference these PluginConfigs
	for _, pc := range pcList.Items {
		var arListForPC apiv2.ApisixRouteList
		if err := r.List(ctx, &arListForPC, client.MatchingFields{
			indexer.PluginConfigIndexRef: indexer.GenIndexKey(pc.GetNamespace(), pc.GetName()),
		}); err != nil {
			r.Log.Error(err, "failed to list apisixroutes by plugin config", "pluginconfig", pc.Name)
			continue
		}
		for _, ar := range arListForPC.Items {
			allRequests = append(allRequests, reconcile.Request{NamespacedName: utils.NamespacedName(&ar)})
		}
	}

	return pkgutils.DedupComparable(allRequests)
}

func (r *ApisixRouteReconciler) listApisixRouteForIngressClass(ctx context.Context, object client.Object) (requests []reconcile.Request) {
	ingressClass, ok := object.(*networkingv1.IngressClass)
	if !ok {
		return nil
	}

	return ListMatchingRequests(
		ctx,
		r.Client,
		r.Log,
		&apiv2.ApisixRouteList{},
		func(obj client.Object) bool {
			ar, ok := obj.(*apiv2.ApisixRoute)
			if !ok {
				r.Log.Error(fmt.Errorf("expected ApisixRoute, got %T", obj), "failed to match object type")
				return false
			}
			return (IsDefaultIngressClass(ingressClass) && ar.Spec.IngressClassName == "") || ar.Spec.IngressClassName == ingressClass.Name
		},
	)
}

func (r *ApisixRouteReconciler) listApisixRouteForGatewayProxy(ctx context.Context, object client.Object) (requests []reconcile.Request) {
	return listIngressClassRequestsForGatewayProxy(ctx, r.Client, object, r.Log, r.listApisixRouteForIngressClass)
}

func (r *ApisixRouteReconciler) listApisixRouteForApisixUpstream(ctx context.Context, object client.Object) (requests []reconcile.Request) {
	au, ok := object.(*apiv2.ApisixUpstream)
	if !ok {
		return nil
	}

	var arList apiv2.ApisixRouteList
	if err := r.List(ctx, &arList, client.MatchingFields{indexer.ApisixUpstreamRef: indexer.GenIndexKey(au.GetNamespace(), au.GetName())}); err != nil {
		r.Log.Error(err, "failed to list ApisixRoutes")
		return nil
	}

	for _, ar := range arList.Items {
		requests = append(requests, reconcile.Request{NamespacedName: utils.NamespacedName(&ar)})
	}
	return pkgutils.DedupComparable(requests)
}

func (r *ApisixRouteReconciler) updateStatus(ar *apiv2.ApisixRoute, err error) {
	SetApisixCRDConditionAccepted(&ar.Status, ar.GetGeneration(), err)
	r.Updater.Update(status.Update{
		NamespacedName: utils.NamespacedName(ar),
		Resource:       &apiv2.ApisixRoute{},
		Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
			cp := obj.(*apiv2.ApisixRoute).DeepCopy()
			cp.Status = ar.Status
			return cp
		}),
	})
}

func (r *ApisixRouteReconciler) listApisixRoutesForPluginConfig(ctx context.Context, obj client.Object) []reconcile.Request {
	pc, ok := obj.(*apiv2.ApisixPluginConfig)
	if !ok {
		return nil
	}

	// First check if the ApisixPluginConfig has matching IngressClassName
	if pc.Spec.IngressClassName != "" {
		var ic networkingv1.IngressClass
		if err := r.Get(ctx, client.ObjectKey{Name: pc.Spec.IngressClassName}, &ic); err != nil {
			if client.IgnoreNotFound(err) != nil {
				r.Log.Error(err, "failed to get IngressClass for ApisixPluginConfig", "pluginconfig", pc.Name)
			}
			return nil
		}
		if !matchesController(ic.Spec.Controller) {
			return nil
		}
	}

	var arList apiv2.ApisixRouteList
	if err := r.List(ctx, &arList, client.MatchingFields{
		indexer.PluginConfigIndexRef: indexer.GenIndexKey(pc.GetNamespace(), pc.GetName()),
	}); err != nil {
		r.Log.Error(err, "failed to list apisixroutes by plugin config", "pluginconfig", pc.Name)
		return nil
	}
	requests := make([]reconcile.Request, 0, len(arList.Items))
	for _, ar := range arList.Items {
		requests = append(requests, reconcile.Request{NamespacedName: utils.NamespacedName(&ar)})
	}
	return pkgutils.DedupComparable(requests)
}

func (r *ApisixRouteReconciler) getSubsetLabels(tctx *provider.TranslateContext, auNN types.NamespacedName, backend apiv2.ApisixRouteHTTPBackend) map[string]string {
	if backend.Subset == "" {
		return nil
	}

	au, ok := tctx.Upstreams[auNN]
	if !ok {
		return nil
	}

	// try to get the subset labels from the ApisixUpstream subsets
	for _, subset := range au.Spec.Subsets {
		if backend.Subset == subset.Name {
			return subset.Labels
		}
	}

	return nil
}

func (r *ApisixRouteReconciler) filterEndpointSlicesBySubsetLabels(ctx context.Context, in []discoveryv1.EndpointSlice, labels map[string]string) []discoveryv1.EndpointSlice {
	if len(labels) == 0 {
		return in
	}

	for i := range in {
		in[i] = r.filterEndpointSliceByTargetPod(ctx, in[i], labels)
	}

	return utils.Filter(in, func(v discoveryv1.EndpointSlice) bool {
		return len(v.Endpoints) > 0
	})
}

// filterEndpointSliceByTargetPod filters item.Endpoints which is not a subset of labels
func (r *ApisixRouteReconciler) filterEndpointSliceByTargetPod(ctx context.Context, item discoveryv1.EndpointSlice, labels map[string]string) discoveryv1.EndpointSlice {
	item.Endpoints = utils.Filter(item.Endpoints, func(v discoveryv1.Endpoint) bool {
		if v.TargetRef == nil || v.TargetRef.Kind != KindPod {
			return true
		}

		var (
			pod   corev1.Pod
			podNN = types.NamespacedName{
				Namespace: v.TargetRef.Namespace,
				Name:      v.TargetRef.Name,
			}
		)
		if err := r.Get(ctx, podNN, &pod); err != nil {
			return false
		}

		return utils.IsSubsetOf(labels, pod.GetLabels())
	})

	return item
}
