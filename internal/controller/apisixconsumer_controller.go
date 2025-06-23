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
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
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
)

// ApisixConsumerReconciler reconciles a ApisixConsumer object
type ApisixConsumerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger

	Provider provider.Provider
	Updater  status.Updater
}

// Reconcile FIXME: implement the reconcile logic (For now, it dose nothing other than directly accepting)
func (r *ApisixConsumerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Log.Info("reconcile", "request", req.NamespacedName)

	ac := &apiv2.ApisixConsumer{}
	if err := r.Get(ctx, req.NamespacedName, ac); err != nil {
		if k8serrors.IsNotFound(err) {
			ac.Namespace = req.Namespace
			ac.Name = req.Name
			ac.TypeMeta = metav1.TypeMeta{
				Kind:       KindApisixConsumer,
				APIVersion: apiv2.GroupVersion.String(),
			}
			if err := r.Provider.Delete(ctx, ac); err != nil {
				r.Log.Error(err, "failed to delete provider", "ApisixConsumer", ac)
				return ctrl.Result{}, err
			}
		}
		r.Log.Error(err, "failed to get ApisixConsumer", "request", req.NamespacedName)
		return ctrl.Result{}, err
	}

	var (
		tctx         = provider.NewDefaultTranslateContext(ctx)
		ingressClass *networkingv1.IngressClass
		err          error
	)
	defer func() {
		r.updateStatus(ac, err)
	}()

	ingressClass, err = GetIngressClass(tctx, r.Client, r.Log, ac.Spec.IngressClassName)
	if err != nil {
		r.Log.Error(err, "failed to get IngressClass")
		return ctrl.Result{}, err
	}

	if err = ProcessIngressClassParameters(tctx, r.Client, r.Log, ac, ingressClass); err != nil {
		r.Log.Error(err, "failed to process IngressClass parameters", "ingressClass", ingressClass.Name)
		return ctrl.Result{}, err
	}

	if err = r.processSpec(ctx, tctx, ac); err != nil {
		return ctrl.Result{}, err
	}

	if err = r.Provider.Update(ctx, tctx, ac); err != nil {
		r.Log.Error(err, "failed to update provider", "ApisixConsumer", ac)
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApisixConsumerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv2.ApisixConsumer{},
			builder.WithPredicates(
				predicate.NewPredicateFuncs(r.checkIngressClass),
			)).
		WithEventFilter(
			predicate.Or(
				predicate.GenerationChangedPredicate{},
				predicate.AnnotationChangedPredicate{},
			),
		).
		Watches(
			&networkingv1.IngressClass{},
			handler.EnqueueRequestsFromMapFunc(r.listApisixConsumerForIngressClass),
			builder.WithPredicates(
				predicate.NewPredicateFuncs(matchesIngressController),
			),
		).
		Watches(&v1alpha1.GatewayProxy{},
			handler.EnqueueRequestsFromMapFunc(r.listApisixConsumerForGatewayProxy),
		).
		Watches(&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.listApisixConsumerForSecret),
		).
		Named("apisixconsumer").
		Complete(r)
}

func (r *ApisixConsumerReconciler) checkIngressClass(obj client.Object) bool {
	ac, ok := obj.(*apiv2.ApisixConsumer)
	if !ok {
		return false
	}

	return matchesIngressClass(r.Client, r.Log, ac.Spec.IngressClassName)
}

func (r *ApisixConsumerReconciler) listApisixConsumerForGatewayProxy(ctx context.Context, obj client.Object) []reconcile.Request {
	return listIngressClassRequestsForGatewayProxy(ctx, r.Client, obj, r.Log, r.listApisixConsumerForIngressClass)
}

func (r *ApisixConsumerReconciler) listApisixConsumerForIngressClass(ctx context.Context, obj client.Object) []reconcile.Request {
	ingressClass, ok := obj.(*networkingv1.IngressClass)
	if !ok {
		return nil
	}

	return ListMatchingRequests(
		ctx,
		r.Client,
		r.Log,
		&apiv2.ApisixConsumerList{},
		func(obj client.Object) bool {
			ac, ok := obj.(*apiv2.ApisixConsumer)
			if !ok {
				r.Log.Error(fmt.Errorf("expected ApisixConsumer, got %T", obj), "failed to match object type")
				return false
			}
			return (IsDefaultIngressClass(ingressClass) && ac.Spec.IngressClassName == "") || ac.Spec.IngressClassName == ingressClass.Name
		},
	)
}

func (r *ApisixConsumerReconciler) listApisixConsumerForSecret(ctx context.Context, obj client.Object) []reconcile.Request {
	secret, ok := obj.(*corev1.Secret)
	if !ok {
		r.Log.Error(nil, "failed to convert to Secret", "object", obj)
		return nil
	}
	return ListRequests(
		ctx,
		r.Client,
		r.Log,
		&apiv2.ApisixConsumerList{},
		client.MatchingFields{
			indexer.SecretIndexRef: indexer.GenIndexKey(secret.GetNamespace(), secret.GetName()),
		},
	)
}

func (r *ApisixConsumerReconciler) processSpec(ctx context.Context, tctx *provider.TranslateContext, ac *apiv2.ApisixConsumer) error {
	var secretRef *corev1.LocalObjectReference
	if ac.Spec.AuthParameter.KeyAuth != nil {
		secretRef = ac.Spec.AuthParameter.KeyAuth.SecretRef
	} else if ac.Spec.AuthParameter.BasicAuth != nil {
		secretRef = ac.Spec.AuthParameter.BasicAuth.SecretRef
	} else if ac.Spec.AuthParameter.JwtAuth != nil {
		secretRef = ac.Spec.AuthParameter.JwtAuth.SecretRef
	} else if ac.Spec.AuthParameter.WolfRBAC != nil {
		secretRef = ac.Spec.AuthParameter.WolfRBAC.SecretRef
	} else if ac.Spec.AuthParameter.HMACAuth != nil {
		secretRef = ac.Spec.AuthParameter.HMACAuth.SecretRef
	} else if ac.Spec.AuthParameter.LDAPAuth != nil {
		secretRef = ac.Spec.AuthParameter.LDAPAuth.SecretRef
	}
	if secretRef == nil {
		return nil
	}

	namespacedName := types.NamespacedName{
		Name:      secretRef.Name,
		Namespace: ac.Namespace,
	}

	secret := &corev1.Secret{}
	if err := r.Get(ctx, namespacedName, secret); err != nil {
		if k8serrors.IsNotFound(err) {
			r.Log.Info("secret not found", "secret", namespacedName.String())
			return nil
		} else {
			r.Log.Error(err, "failed to get secret", "secret", namespacedName.String())
			return err
		}
	}
	tctx.Secrets[namespacedName] = secret
	return nil
}

func (r *ApisixConsumerReconciler) updateStatus(consumer *apiv2.ApisixConsumer, err error) {
	SetApisixCRDConditionAccepted(&consumer.Status, consumer.GetGeneration(), err)
	r.Updater.Update(status.Update{
		NamespacedName: utils.NamespacedName(consumer),
		Resource:       &apiv2.ApisixConsumer{},
		Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
			ac := obj.(*apiv2.ApisixConsumer)
			acCopy := ac.DeepCopy()
			acCopy.Status = consumer.Status
			return acCopy
		}),
	})
}
