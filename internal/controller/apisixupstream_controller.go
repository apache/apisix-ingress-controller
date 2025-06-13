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

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
)

// ApisixUpstreamReconciler reconciles a ApisixUpstream object
type ApisixUpstreamReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

// Reconcile FIXME: implement the reconcile logic (For now, it dose nothing other than directly accepting)
func (r *ApisixUpstreamReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Log.Info("reconcile", "request", req.NamespacedName)

	var obj apiv2.ApisixUpstream
	if err := r.Get(ctx, req.NamespacedName, &obj); err != nil {
		r.Log.Error(err, "failed to get ApisixConsumer", "request", req.NamespacedName)
		return ctrl.Result{}, err
	}

	obj.Status.Conditions = []metav1.Condition{
		{
			Type:               string(gatewayv1.RouteConditionAccepted),
			Status:             metav1.ConditionTrue,
			ObservedGeneration: obj.GetGeneration(),
			LastTransitionTime: metav1.Now(),
			Reason:             string(gatewayv1.RouteReasonAccepted),
		},
	}

	if err := r.Status().Update(ctx, &obj); err != nil {
		r.Log.Error(err, "failed to update status", "request", req.NamespacedName)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApisixUpstreamReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv2.ApisixUpstream{}).
		Named("apisixupstream").
		Complete(r)
}
