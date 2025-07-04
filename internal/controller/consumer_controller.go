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
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/controller/indexer"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/internal/utils"
)

// ConsumerReconciler  reconciles a Gateway object.
type ConsumerReconciler struct { //nolint:revive
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger

	Provider provider.Provider

	Updater status.Updater
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConsumerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Consumer{},
			builder.WithPredicates(
				predicate.NewPredicateFuncs(r.checkGatewayRef),
			),
		).
		WithEventFilter(
			predicate.Or(
				predicate.GenerationChangedPredicate{},
				predicate.NewPredicateFuncs(TypePredicate[*corev1.Secret]()),
			),
		).
		Watches(&gatewayv1.Gateway{},
			handler.EnqueueRequestsFromMapFunc(r.listConsumersForGateway),
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
		Watches(&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.listConsumersForSecret),
		).
		Watches(&v1alpha1.GatewayProxy{},
			handler.EnqueueRequestsFromMapFunc(r.listConsumersForGatewayProxy),
		).
		Complete(r)
}

func (r *ConsumerReconciler) listConsumersForSecret(ctx context.Context, obj client.Object) []reconcile.Request {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		r.Log.Error(nil, "failed to convert to Secret", "object", obj)
		return nil
	}
	return ListRequests(
		ctx,
		r.Client,
		r.Log,
		&v1alpha1.ConsumerList{},
		client.MatchingFields{
			indexer.SecretIndexRef: indexer.GenIndexKey(secret.GetNamespace(), secret.GetName()),
		},
	)
}

func (r *ConsumerReconciler) listConsumersForGateway(ctx context.Context, obj client.Object) []reconcile.Request {
	gateway, ok := obj.(*gatewayv1.Gateway)
	if !ok {
		r.Log.Error(nil, "failed to convert to Gateway", "object", obj)
		return nil
	}
	consumerList := &v1alpha1.ConsumerList{}
	if err := r.List(ctx, consumerList, client.MatchingFields{
		indexer.ConsumerGatewayRef: indexer.GenIndexKey(gateway.GetNamespace(), gateway.GetName()),
	}); err != nil {
		r.Log.Error(err, "failed to list consumers")
		return nil
	}
	requests := make([]reconcile.Request, 0, len(consumerList.Items))
	for _, consumer := range consumerList.Items {
		requests = append(requests, reconcile.Request{
			NamespacedName: client.ObjectKey{
				Name:      consumer.Name,
				Namespace: consumer.Namespace,
			},
		})
	}
	return requests
}

func (r *ConsumerReconciler) listConsumersForGatewayProxy(ctx context.Context, obj client.Object) []reconcile.Request {
	gatewayProxy, ok := obj.(*v1alpha1.GatewayProxy)
	if !ok {
		r.Log.Error(nil, "failed to convert to GatewayProxy", "object", obj)
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

	for _, gateway := range gatewayList.Items {
		consumerList := &v1alpha1.ConsumerList{}
		if err := r.List(ctx, consumerList, client.MatchingFields{
			indexer.ConsumerGatewayRef: indexer.GenIndexKey(gateway.Namespace, gateway.Name),
		}); err != nil {
			r.Log.Error(err, "failed to list consumers for gateway", "gateway", gateway.Name)
			continue
		}

		for _, consumer := range consumerList.Items {
			requests = append(requests, reconcile.Request{
				NamespacedName: client.ObjectKey{
					Namespace: consumer.Namespace,
					Name:      consumer.Name,
				},
			})
		}
	}

	return requests
}

func (r *ConsumerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	consumer := new(v1alpha1.Consumer)
	if err := r.Get(ctx, req.NamespacedName, consumer); err != nil {
		if client.IgnoreNotFound(err) == nil {
			consumer.Namespace = req.Namespace
			consumer.Name = req.Name

			consumer.TypeMeta = metav1.TypeMeta{
				Kind:       "Consumer",
				APIVersion: v1alpha1.GroupVersion.String(),
			}

			if err := r.Provider.Delete(ctx, consumer); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	var statusErr error
	tctx := provider.NewDefaultTranslateContext(ctx)

	gateway, err := r.getGateway(ctx, consumer)
	if err != nil {
		r.Log.Error(err, "failed to get gateway", "consumer", consumer)
		statusErr = err
	}

	rk := utils.NamespacedNameKind(consumer)

	if err := ProcessGatewayProxy(r.Client, tctx, gateway, rk); err != nil {
		r.Log.Error(err, "failed to process gateway proxy", "gateway", gateway)
		statusErr = err
	}

	if err := r.processSpec(ctx, tctx, consumer); err != nil {
		r.Log.Error(err, "failed to process consumer spec", "consumer", consumer)
		statusErr = err
	}

	if err := r.Provider.Update(ctx, tctx, consumer); err != nil {
		r.Log.Error(err, "failed to update consumer", "consumer", consumer)
		statusErr = err
	}

	r.updateStatus(consumer, statusErr)

	return ctrl.Result{}, nil
}

func (r *ConsumerReconciler) processSpec(ctx context.Context, tctx *provider.TranslateContext, consumer *v1alpha1.Consumer) error {
	for _, credential := range consumer.Spec.Credentials {
		if credential.SecretRef == nil {
			continue
		}
		ns := consumer.GetNamespace()
		if credential.SecretRef.Namespace != nil {
			ns = *credential.SecretRef.Namespace
		}
		secret := corev1.Secret{}
		if err := r.Get(ctx, client.ObjectKey{
			Name:      credential.SecretRef.Name,
			Namespace: ns,
		}, &secret); err != nil {
			if client.IgnoreNotFound(err) == nil {
				continue
			}
			r.Log.Error(err, "failed to get secret", "secret", credential.SecretRef.Name)
			return err
		}

		tctx.Secrets[types.NamespacedName{
			Namespace: ns,
			Name:      credential.SecretRef.Name,
		}] = &secret

	}
	return nil
}

func (r *ConsumerReconciler) updateStatus(consumer *v1alpha1.Consumer, err error) {
	condition := NewCondition(consumer.Generation, true, "Successfully")
	if err != nil {
		condition = NewCondition(consumer.Generation, false, err.Error())
	}
	if !VerifyConditions(&consumer.Status.Conditions, condition) {
		return
	}
	meta.SetStatusCondition(&consumer.Status.Conditions, condition)

	r.Updater.Update(status.Update{
		NamespacedName: utils.NamespacedName(consumer),
		Resource:       consumer.DeepCopy(),
		Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
			t, ok := obj.(*v1alpha1.Consumer)
			if !ok {
				err := fmt.Errorf("unsupported object type %T", obj)
				panic(err)
			}
			t.Status = consumer.Status
			return t
		}),
	})
}

func (r *ConsumerReconciler) getGateway(ctx context.Context, consumer *v1alpha1.Consumer) (*gatewayv1.Gateway, error) {
	ns := consumer.GetNamespace()
	if consumer.Spec.GatewayRef.Namespace != nil {
		ns = *consumer.Spec.GatewayRef.Namespace
	}
	gateway := &gatewayv1.Gateway{}
	if err := r.Get(ctx, client.ObjectKey{
		Name:      consumer.Spec.GatewayRef.Name,
		Namespace: ns,
	}, gateway); err != nil {
		r.Log.Error(err, "failed to get gateway", "gateway", consumer.Spec.GatewayRef.Name)
		return nil, err
	}
	return gateway, nil
}

func (r *ConsumerReconciler) checkGatewayRef(object client.Object) bool {
	consumer, ok := object.(*v1alpha1.Consumer)
	if !ok {
		return false
	}
	if consumer.Spec.GatewayRef.Name == "" {
		return false
	}
	if consumer.Spec.GatewayRef.Kind != nil && *consumer.Spec.GatewayRef.Kind != KindGateway {
		return false
	}
	if consumer.Spec.GatewayRef.Group != nil && *consumer.Spec.GatewayRef.Group != gatewayv1.GroupName {
		return false
	}
	ns := consumer.GetNamespace()
	if consumer.Spec.GatewayRef.Namespace != nil {
		ns = *consumer.Spec.GatewayRef.Namespace
	}
	gateway := &gatewayv1.Gateway{}
	if err := r.Get(context.Background(), client.ObjectKey{
		Name:      consumer.Spec.GatewayRef.Name,
		Namespace: ns,
	}, gateway); err != nil {
		r.Log.Error(err, "failed to get gateway", "gateway", consumer.Spec.GatewayRef.Name)
		return false
	}
	gatewayClass := &gatewayv1.GatewayClass{}
	if err := r.Get(context.Background(), client.ObjectKey{Name: string(gateway.Spec.GatewayClassName)}, gatewayClass); err != nil {
		r.Log.Error(err, "failed to get gateway class", "gateway", gateway.GetName(), "gatewayclass", gateway.Spec.GatewayClassName)
		return false
	}
	return matchesController(string(gatewayClass.Spec.ControllerName))
}
