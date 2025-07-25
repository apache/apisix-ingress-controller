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

package status

import (
	"context"
	"sync"

	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	v2 "github.com/apache/apisix-ingress-controller/api/v2"
	types "github.com/apache/apisix-ingress-controller/internal/types"
	pkgmetrics "github.com/apache/apisix-ingress-controller/pkg/metrics"
)

const UpdateChannelBufferSize = 1000

type Update struct {
	NamespacedName k8stypes.NamespacedName
	Resource       client.Object
	Mutator        Mutator
}

type Mutator interface {
	Mutate(obj client.Object) client.Object
}

type MutatorFunc func(client.Object) client.Object

func (m MutatorFunc) Mutate(obj client.Object) client.Object {
	if m == nil {
		return nil
	}

	return m(obj)
}

var cmpIgnoreLastTT = cmp.Options{
	cmpopts.IgnoreFields(metav1.Condition{}, "LastTransitionTime"),
	cmpopts.IgnoreMapEntries(func(k string, _ any) bool {
		return k == "lastTransitionTime"
	}),
}

type UpdateHandler struct {
	log           logr.Logger
	client        client.Client
	updateChannel chan Update
	wg            *sync.WaitGroup
}

func NewStatusUpdateHandler(log logr.Logger, client client.Client) *UpdateHandler {
	u := &UpdateHandler{
		log:           log,
		client:        client,
		updateChannel: make(chan Update, UpdateChannelBufferSize),
		wg:            new(sync.WaitGroup),
	}

	u.wg.Add(1)
	return u
}

func (u *UpdateHandler) apply(ctx context.Context, update Update) {
	if err := retry.OnError(retry.DefaultBackoff, func(err error) bool {
		return k8serrors.IsConflict(err)
	}, func() error {
		return u.updateStatus(ctx, update)
	}); err != nil {
		u.log.Error(err, "unable to update status", "name", update.NamespacedName.Name,
			"namespace", update.NamespacedName.Namespace)
	}
}

func (u *UpdateHandler) updateStatus(ctx context.Context, update Update) error {
	var obj = update.Resource
	if err := u.client.Get(ctx, update.NamespacedName, obj); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	newObj := update.Mutator.Mutate(obj)
	if newObj == nil {
		return nil
	}

	if statusEqual(obj, newObj, cmpIgnoreLastTT) {
		u.log.V(1).Info("status is equal, skipping update", "name", update.NamespacedName.Name,
			"namespace", update.NamespacedName.Namespace,
			"kind", types.KindOf(obj))
		return nil
	}

	newObj.SetUID(obj.GetUID())

	u.log.Info("updating status", "name", update.NamespacedName.Name,
		"namespace", update.NamespacedName.Namespace,
		"kind", types.KindOf(newObj),
	)

	return u.client.Status().Update(ctx, newObj)
}

func (u *UpdateHandler) Start(ctx context.Context) error {
	u.log.Info("started status update handler")
	defer u.log.Info("stopped status update handler")

	u.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return nil
		case update := <-u.updateChannel:
			// Decrement queue length after removing item from queue
			pkgmetrics.DecStatusQueueLength()
			u.log.V(1).Info("received a status update", "namespace", update.NamespacedName.Namespace,
				"name", update.NamespacedName.Name,
				"kind", types.KindOf(update.Resource),
			)

			u.apply(ctx, update)
		}
	}
}

func (u *UpdateHandler) NeedsLeaderElection() bool {
	return true
}

func (u *UpdateHandler) Writer() Updater {
	return &UpdateWriter{
		updateChannel: u.updateChannel,
		wg:            u.wg,
	}
}

type Updater interface {
	Update(u Update)
}

type UpdateWriter struct {
	updateChannel chan<- Update
	wg            *sync.WaitGroup
}

func (u *UpdateWriter) Update(update Update) {
	u.wg.Wait()
	u.updateChannel <- update
	// Increment queue length after adding new item
	pkgmetrics.IncStatusQueueLength()
}

func statusEqual(a, b any, opts ...cmp.Option) bool {
	var statusA, statusB any

	switch a := a.(type) {
	case *gatewayv1.GatewayClass:
		b, ok := b.(*gatewayv1.GatewayClass)
		if !ok {
			return false
		}
		statusA, statusB = a.Status, b.Status

	case *gatewayv1.Gateway:
		b, ok := b.(*gatewayv1.Gateway)
		if !ok {
			return false
		}
		statusA, statusB = a.Status, b.Status

	case *gatewayv1.HTTPRoute:
		b, ok := b.(*gatewayv1.HTTPRoute)
		if !ok {
			return false
		}
		statusA, statusB = a.Status, b.Status
	case *v2.ApisixRoute:
		b, ok := b.(*v2.ApisixRoute)
		if !ok {
			return false
		}
		statusA, statusB = a.Status, b.Status
	case *v2.ApisixGlobalRule:
		b, ok := b.(*v2.ApisixGlobalRule)
		if !ok {
			return false
		}
		statusA, statusB = a.Status, b.Status
	case *v2.ApisixPluginConfig:
		b, ok := b.(*v2.ApisixPluginConfig)
		if !ok {
			return false
		}
		statusA, statusB = a.Status, b.Status
	case *v2.ApisixTls:
		b, ok := b.(*v2.ApisixTls)
		if !ok {
			return false
		}
		statusA, statusB = a.Status, b.Status
	case *v2.ApisixConsumer:
		b, ok := b.(*v2.ApisixConsumer)
		if !ok {
			return false
		}
		statusA, statusB = a.Status, b.Status
	case *v1alpha1.HTTPRoutePolicy:
		b, ok := b.(*v1alpha1.HTTPRoutePolicy)
		if !ok {
			return false
		}
		statusA, statusB = a.Status, b.Status
	case *v1alpha1.BackendTrafficPolicy:
		b, ok := b.(*v1alpha1.BackendTrafficPolicy)
		if !ok {
			return false
		}
		statusA, statusB = a.Status, b.Status
	case *v1alpha1.Consumer:
		b, ok := b.(*v1alpha1.Consumer)
		if !ok {
			return false
		}
		statusA, statusB = a.Status, b.Status
	default:
		return false
	}

	return cmp.Equal(statusA, statusB, opts...)
}
