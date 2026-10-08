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
	"slices"

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
	"github.com/apache/apisix-ingress-controller/internal/controller/config"
	"github.com/apache/apisix-ingress-controller/internal/controller/indexer"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	"github.com/apache/apisix-ingress-controller/internal/manager/readiness"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	internaltypes "github.com/apache/apisix-ingress-controller/internal/types"
	"github.com/apache/apisix-ingress-controller/internal/utils"
	pkgutils "github.com/apache/apisix-ingress-controller/pkg/utils"
)

const keyAuthType = "key-auth"

// ConsumerReconciler  reconciles a Gateway object.
type ConsumerReconciler struct { //nolint:revive
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger

	Provider provider.Provider

	Updater status.Updater
	Readier readiness.ReadinessManager
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConsumerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	hasGatewayAPI := false
	if !config.ControllerConfig.DisableGatewayAPI {
		var err error
		if hasGatewayAPI, err = pkgutils.HasAPIResource(mgr, &gatewayv1.Gateway{}); err != nil {
			return err
		}
	}
	if !hasGatewayAPI {
		r.Log.Info("skipping Consumer controller setup as Gateway API is not available")
		return nil
	}
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
		Watches(&v1alpha1.Consumer{},
			handler.EnqueueRequestsFromMapFunc(r.listConsumersSharingKeyAuthKeys),
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
	requests := ListRequests(
		ctx,
		r.Client,
		r.Log,
		&v1alpha1.ConsumerList{},
		client.MatchingFields{
			indexer.SecretIndexRef: indexer.GenIndexKey(secret.GetNamespace(), secret.GetName()),
		},
	)
	return append(requests, r.listKeyAuthKeyOwners(ctx, string(secret.Data["key"]))...)
}

// listConsumersSharingKeyAuthKeys requeues Consumers sharing a key-auth key with
// obj, so a credential skipped as a duplicate returns once the key is free.
func (r *ConsumerReconciler) listConsumersSharingKeyAuthKeys(ctx context.Context, obj client.Object) []reconcile.Request {
	consumer, ok := obj.(*v1alpha1.Consumer)
	if !ok {
		r.Log.Error(nil, "failed to convert to Consumer", "object", obj)
		return nil
	}
	keys := make([]string, 0, len(consumer.Spec.Credentials))
	for _, credential := range consumer.Spec.Credentials {
		if credential.Type != keyAuthType {
			continue
		}
		if credential.SecretRef == nil {
			keys = append(keys, indexer.InlineKeyAuthKey(credential))
			continue
		}
		var secret corev1.Secret
		if err := r.Get(ctx, credentialSecretNN(consumer, credential), &secret); err != nil {
			continue
		}
		keys = append(keys, string(secret.Data["key"]))
	}
	return r.listKeyAuthKeyOwners(ctx, keys...)
}

func (r *ConsumerReconciler) listKeyAuthKeyOwners(ctx context.Context, keys ...string) []reconcile.Request {
	var requests []reconcile.Request
	for _, key := range keys {
		if key == "" {
			continue
		}
		owners, err := r.keyAuthKeyOwners(ctx, key)
		if err != nil {
			r.Log.Error(err, "failed to list consumers sharing a key-auth key")
			continue
		}
		for i := range owners {
			requests = append(requests, reconcile.Request{NamespacedName: utils.NamespacedName(&owners[i])})
		}
	}
	return requests
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
	defer r.Readier.Done(&v1alpha1.Consumer{}, req.NamespacedName)
	consumer := new(v1alpha1.Consumer)
	if err := r.Get(ctx, req.NamespacedName, consumer); err != nil {
		if client.IgnoreNotFound(err) == nil {
			consumer.Namespace = req.Namespace
			consumer.Name = req.Name

			consumer.TypeMeta = metav1.TypeMeta{
				Kind:       internaltypes.KindConsumer,
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
		r.Log.V(1).Info("no matching Gateway available",
			"gatewayRef", consumer.Spec.GatewayRef,
			"error", err.Error())
		return ctrl.Result{}, nil
	}

	rk := utils.NamespacedNameKind(consumer)

	if err := ProcessGatewayProxy(r.Client, r.Log, tctx, gateway, rk); err != nil {
		r.Log.Error(err, "failed to process gateway proxy", "gateway", utils.NamespacedName(gateway))
		statusErr = err
	}

	if err := r.processSpec(ctx, tctx, consumer); err != nil {
		r.Log.Error(err, "failed to process consumer spec", "consumer", utils.NamespacedName(consumer))
		statusErr = err
	}

	published, skipped, err := r.skipDuplicateKeyAuthCredentials(ctx, tctx, consumer)
	if err != nil {
		return ctrl.Result{}, err
	}
	if skipped > 0 {
		statusErr = fmt.Errorf("%d key-auth credential(s) skipped: key already used by another Consumer", skipped)
	}

	if err := r.Provider.Update(ctx, tctx, published); err != nil {
		r.Log.Error(err, "failed to update consumer", "consumer", utils.NamespacedName(consumer))
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
		// A cross-namespace SecretRef needs a ReferenceGrant, same as routes.
		secretNN := types.NamespacedName{Namespace: ns, Name: credential.SecretRef.Name}
		permitted, err := CheckConsumerSecretRef(ctx, r.Client, consumer.GetNamespace(), secretNN)
		if err != nil {
			return err
		}
		if !permitted {
			r.Log.Error(nil, "cross-namespace secret reference not permitted by any ReferenceGrant",
				"consumer", utils.NamespacedName(consumer), "secret", client.ObjectKey{Namespace: ns, Name: credential.SecretRef.Name})
			return fmt.Errorf("cross-namespace secret reference from Consumer %s/%s to Secret %s/%s is not permitted by any ReferenceGrant",
				consumer.GetNamespace(), consumer.GetName(), ns, credential.SecretRef.Name)
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
	return loadPluginSecrets(ctx, r.Client, tctx, consumer.GetNamespace(), consumer.Spec.Plugins)
}

// skipDuplicateKeyAuthCredentials drops key-auth credentials whose key an older
// Consumer on the same Gateway already uses. Admission alone misses a duplicate
// that arrives later, e.g. via a Secret created after its Consumer.
func (r *ConsumerReconciler) skipDuplicateKeyAuthCredentials(ctx context.Context, tctx *provider.TranslateContext, consumer *v1alpha1.Consumer) (*v1alpha1.Consumer, int, error) {
	kept := make([]v1alpha1.Credential, 0, len(consumer.Spec.Credentials))
	for _, credential := range consumer.Spec.Credentials {
		key := publishedKeyAuthKey(tctx, consumer, credential)
		if key != "" {
			taken, err := r.keyAuthKeyTaken(ctx, consumer, key)
			if err != nil {
				return nil, 0, err
			}
			if taken {
				continue
			}
		}
		kept = append(kept, credential)
	}
	skipped := len(consumer.Spec.Credentials) - len(kept)
	if skipped == 0 {
		return consumer, 0, nil
	}
	published := consumer.DeepCopy()
	published.Spec.Credentials = kept
	return published, skipped, nil
}

// keyAuthKeyTaken reports whether an older Consumer on the same Gateway uses key.
func (r *ConsumerReconciler) keyAuthKeyTaken(ctx context.Context, consumer *v1alpha1.Consumer, key string) (bool, error) {
	owners, err := r.keyAuthKeyOwners(ctx, key)
	if err != nil {
		return false, err
	}
	gatewayRef := indexer.ConsumerGatewayRefIndexFunc(consumer)
	for i := range owners {
		owner := &owners[i]
		if owner.Namespace == consumer.Namespace && owner.Name == consumer.Name {
			continue
		}
		if !slices.Equal(indexer.ConsumerGatewayRefIndexFunc(owner), gatewayRef) {
			continue
		}
		if olderConsumer(owner, consumer) {
			return true, nil
		}
	}
	return false, nil
}

// keyAuthKeyOwners lists the Consumers whose key-auth credentials use key.
func (r *ConsumerReconciler) keyAuthKeyOwners(ctx context.Context, key string) ([]v1alpha1.Consumer, error) {
	index := client.MatchingFields{indexer.KeyAuthKey: indexer.GenKeyAuthKeyIndex(key)}

	var owners v1alpha1.ConsumerList
	if err := r.List(ctx, &owners, index); err != nil {
		return nil, err
	}
	var secrets corev1.SecretList
	if err := r.List(ctx, &secrets, index); err != nil {
		return nil, err
	}
	for _, secret := range secrets.Items {
		var refs v1alpha1.ConsumerList
		if err := r.List(ctx, &refs, client.MatchingFields{
			indexer.SecretIndexRef: indexer.GenIndexKey(secret.Namespace, secret.Name),
		}); err != nil {
			return nil, err
		}
		for i := range refs.Items {
			uses, err := usesKeyAuthSecret(ctx, r.Client, &refs.Items[i], utils.NamespacedName(&secret))
			if err != nil {
				return nil, err
			}
			if uses {
				owners.Items = append(owners.Items, refs.Items[i])
			}
		}
	}
	return owners.Items, nil
}

// usesKeyAuthSecret reports whether consumer loads secretNN as a key-auth credential.
func usesKeyAuthSecret(ctx context.Context, c client.Client, consumer *v1alpha1.Consumer, secretNN types.NamespacedName) (bool, error) {
	for _, credential := range consumer.Spec.Credentials {
		if credential.Type != keyAuthType || credential.SecretRef == nil || credentialSecretNN(consumer, credential) != secretNN {
			continue
		}
		if secretNN.Namespace == consumer.Namespace {
			return true, nil
		}
		return CheckConsumerSecretRef(ctx, c, consumer.Namespace, secretNN)
	}
	return false, nil
}

// publishedKeyAuthKey returns the key a key-auth credential publishes, or "".
func publishedKeyAuthKey(tctx *provider.TranslateContext, consumer *v1alpha1.Consumer, credential v1alpha1.Credential) string {
	if credential.Type != keyAuthType {
		return ""
	}
	if credential.SecretRef == nil {
		return indexer.InlineKeyAuthKey(credential)
	}
	secret := tctx.Secrets[credentialSecretNN(consumer, credential)]
	if secret == nil {
		return ""
	}
	return string(secret.Data["key"])
}

func credentialSecretNN(consumer *v1alpha1.Consumer, credential v1alpha1.Credential) types.NamespacedName {
	ns := consumer.Namespace
	if credential.SecretRef.Namespace != nil {
		ns = *credential.SecretRef.Namespace
	}
	return types.NamespacedName{Namespace: ns, Name: credential.SecretRef.Name}
}

// olderConsumer orders by creation time, then by namespace/name for a stable tie-break.
func olderConsumer(a, b *v1alpha1.Consumer) bool {
	if !a.CreationTimestamp.Equal(&b.CreationTimestamp) {
		return a.CreationTimestamp.Before(&b.CreationTimestamp)
	}
	return utils.NamespacedName(a).String() < utils.NamespacedName(b).String()
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
			cp := obj.(*v1alpha1.Consumer).DeepCopy()
			cp.Status = consumer.Status
			return cp
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
		return nil, fmt.Errorf("failed to get gateway %s/%s: %w", ns, consumer.Spec.GatewayRef.Name, err)
	}
	gatewayClass := gatewayv1.GatewayClass{}
	if err := r.Get(ctx, client.ObjectKey{
		Name: string(gateway.Spec.GatewayClassName),
	}, &gatewayClass); err != nil {
		return nil, fmt.Errorf("failed to retrieve gatewayclass for gateway: %w", err)
	}

	if string(gatewayClass.Spec.ControllerName) != config.ControllerConfig.ControllerName {
		return nil, fmt.Errorf("gateway %s/%s is not managed by this controller", gateway.Namespace, gateway.Name)
	}
	return gateway, nil
}

func (r *ConsumerReconciler) checkGatewayRef(object client.Object) bool {
	consumer, ok := object.(*v1alpha1.Consumer)
	if !ok {
		return false
	}
	return MatchConsumerGatewayRef(context.Background(), r.Client, r.Log, consumer)
}
