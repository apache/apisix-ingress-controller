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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	"github.com/apache/apisix-ingress-controller/internal/utils"
)

// ApisixUpstreamReconciler reconciles a ApisixUpstream object
type ApisixUpstreamReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	Log     logr.Logger
	Updater status.Updater
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApisixUpstreamReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv2.ApisixUpstream{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Named("apisixupstream").
		Complete(r)
}

func (r *ApisixUpstreamReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var au apiv2.ApisixUpstream
	if err := r.Get(ctx, req.NamespacedName, &au); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Only update status
	r.updateStatus(&au, nil)
	return ctrl.Result{}, nil
}

func (r *ApisixUpstreamReconciler) updateStatus(au *apiv2.ApisixUpstream, err error) {
	SetApisixCRDConditionAccepted(&au.Status, au.GetGeneration(), err)
	r.Updater.Update(status.Update{
		NamespacedName: utils.NamespacedName(au),
		Resource:       &apiv2.ApisixUpstream{},
		Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
			cp := obj.(*apiv2.ApisixUpstream).DeepCopy()
			cp.Status = au.Status
			return cp
		}),
	})
}
