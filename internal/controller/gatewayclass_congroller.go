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
	"time"

	"github.com/go-logr/logr"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/internal/controller/config"
	"github.com/apache/apisix-ingress-controller/internal/controller/indexer"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
)

const (
	FinalizerGatewayClassProtection = "apisix.apache.org/gc-protection"
)

// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gatewayclasses,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gatewayclasses/status,verbs=get;update

// GatewayClassReconciler reconciles a GatewayClass object.
type GatewayClassReconciler struct { //nolint:revive
	client.Client
	Scheme *runtime.Scheme

	record.EventRecorder
	Log logr.Logger

	Updater status.Updater
}

// SetupWithManager sets up the controller with the Manager.
func (r *GatewayClassReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.EventRecorder = mgr.GetEventRecorderFor("gatewayclass-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&gatewayv1.GatewayClass{}).
		WithEventFilter(predicate.NewPredicateFuncs(r.GatewayClassFilter)).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}

func (r *GatewayClassReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	gc := new(gatewayv1.GatewayClass)
	if err := r.Get(ctx, req.NamespacedName, gc); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if gc.GetDeletionTimestamp().IsZero() {
		if !controllerutil.ContainsFinalizer(gc, FinalizerGatewayClassProtection) {
			controllerutil.AddFinalizer(gc, FinalizerGatewayClassProtection)
			if err := r.Update(ctx, gc); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(gc, FinalizerGatewayClassProtection) {
			var gatewayList gatewayv1.GatewayList
			if err := r.List(ctx, &gatewayList, client.MatchingFields{indexer.GatewayClassIndexRef: gc.Name}); err != nil {
				r.Log.Error(err, "failed to list gateways")
				return ctrl.Result{}, err
			}
			if len(gatewayList.Items) > 0 {
				var gateways []types.NamespacedName
				for _, item := range gatewayList.Items {
					gateways = append(gateways, types.NamespacedName{
						Namespace: item.GetNamespace(),
						Name:      item.GetName(),
					})
				}
				r.Eventf(gc, "Warning", "DeletionBlocked", "the GatewayClass is still used by Gateways: %v", gateways)
				return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
			} else {
				controllerutil.RemoveFinalizer(gc, FinalizerGatewayClassProtection)
				if err := r.Update(ctx, gc); err != nil {
					return ctrl.Result{}, err
				}
			}
		}

		return ctrl.Result{}, nil
	}

	condition := meta.Condition{
		Type:               string(gatewayv1.GatewayClassConditionStatusAccepted),
		Status:             meta.ConditionTrue,
		Reason:             string(gatewayv1.GatewayClassReasonAccepted),
		ObservedGeneration: gc.Generation,
		Message:            "the gatewayclass has been accepted by the apisix-ingress-controller",
		LastTransitionTime: meta.Now(),
	}

	if !IsConditionPresentAndEqual(gc.Status.Conditions, condition) {
		r.Log.Info("gatewayclass has been accepted", "gatewayclass", gc.Name)
		setGatewayClassCondition(gc, condition)
		r.Updater.Update(status.Update{
			NamespacedName: NamespacedName(gc),
			Resource:       gc.DeepCopy(),
			Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
				t, ok := obj.(*gatewayv1.GatewayClass)
				if !ok {
					err := fmt.Errorf("unsupported object type %T", obj)
					panic(err)
				}
				t.Status = gc.Status
				return t
			}),
		})
	}
	return ctrl.Result{}, nil
}

func (r *GatewayClassReconciler) GatewayClassFilter(obj client.Object) bool {
	gatewayClass, ok := obj.(*gatewayv1.GatewayClass)
	if !ok {
		r.Log.Error(fmt.Errorf("unexpected object type"), "failed to convert object to GatewayClass")
		return false
	}

	return matchesController(string(gatewayClass.Spec.ControllerName))
}

func matchesController(controllerName string) bool {
	return controllerName == config.ControllerConfig.ControllerName
}

func setGatewayClassCondition(gwc *gatewayv1.GatewayClass, newCondition meta.Condition) {
	newConditions := []meta.Condition{}
	for _, condition := range gwc.Status.Conditions {
		if condition.Type != newCondition.Type {
			newConditions = append(newConditions, condition)
		}
	}
	newConditions = append(newConditions, newCondition)
	gwc.Status.Conditions = newConditions
}
