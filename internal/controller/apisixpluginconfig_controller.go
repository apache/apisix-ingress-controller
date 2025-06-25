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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	"github.com/apache/apisix-ingress-controller/internal/utils"
)

// ApisixPluginConfigReconciler reconciles a ApisixPluginConfig object
type ApisixPluginConfigReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	Log     logr.Logger
	Updater status.Updater
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApisixPluginConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv2.ApisixPluginConfig{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Named("apisixpluginconfig").
		Complete(r)
}

func (r *ApisixPluginConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var pc apiv2.ApisixPluginConfig
	if err := r.Get(ctx, req.NamespacedName, &pc); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Only update status
	r.updateStatus(&pc, nil)
	return ctrl.Result{}, nil
}

func (r *ApisixPluginConfigReconciler) updateStatus(pc *apiv2.ApisixPluginConfig, err error) {
	SetApisixCRDConditionAccepted(&pc.Status, pc.GetGeneration(), err)
	r.Updater.Update(status.Update{
		NamespacedName: utils.NamespacedName(pc),
		Resource:       &apiv2.ApisixPluginConfig{},
		Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
			cp, ok := obj.(*apiv2.ApisixPluginConfig)
			if !ok {
				err := fmt.Errorf("unsupported object type %T", obj)
				panic(err)
			}
			cpCopy := cp.DeepCopy()
			cpCopy.Status = pc.Status
			return cpCopy
		}),
	})
}
