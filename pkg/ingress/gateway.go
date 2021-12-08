// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package ingress

import (
	"context"
	"time"

	"go.uber.org/zap"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/types"
)

type gatewayController struct {
	controller *Controller
	workqueue  workqueue.RateLimitingInterface
	workers    int
}

func (c *Controller) newGatewayController() *gatewayController {
	ctl := &gatewayController{
		controller: c,
		workqueue:  workqueue.NewNamedRateLimitingQueue(workqueue.NewItemFastSlowRateLimiter(1*time.Second, 60*time.Second, 5), "Gateway"),
		workers:    1,
	}

	ctl.controller.gatewayInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    ctl.onAdd,
		UpdateFunc: ctl.onUpdate,
		DeleteFunc: ctl.OnDelete,
	})
	return ctl
}

func (c *gatewayController) run(ctx context.Context) {
	log.Info("gateway controller started")
	defer log.Infof("gateway controller exited")
	defer c.workqueue.ShutDown()

	log.Info("gateway controller started")
	if !cache.WaitForCacheSync(ctx.Done(), c.controller.gatewayInformer.HasSynced) {
		log.Error("cache sync failed")
		return
	}

	for i := 0; i < c.workers; i++ {
		go c.runWorker(ctx)
	}
	<-ctx.Done()
}

func (c *gatewayController) runWorker(ctx context.Context) {
	for {
		obj, quit := c.workqueue.Get()
		if quit {
			return
		}
		err := c.sync(ctx, obj.(*types.Event))
		c.workqueue.Done(obj)
		c.handleSyncErr(obj, err)
	}
}

func (c *gatewayController) sync(ctx context.Context, ev *types.Event) error {
	key := ev.Object.(string)
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		log.Errorf("found Gateway resource with invalid meta namespace key %s: %s", key, err)
		return err
	}

	gateway, err := c.controller.gatewayLister.Gateways(namespace).Get(name)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			log.Errorf("failed to get Gateway %s: %s", key, err)
			return err
		}
		if ev.Type != types.EventDelete {
			log.Warnf("Gateway %s was deleted before it can be delivered", key)
			// Don't need to retry.
			return nil
		}
	}

	if ev.Type == types.EventDelete {
		if gateway != nil {
			// We still find the resource while we are processing the DELETE event,
			// that means object with same namespace and name was created, discarding
			// this stale DELETE event.
			log.Warnf("discard the stale Gateway delete event since the %s exists", key)
			return nil
		}
		gateway = ev.Tombstone.(*gatewayv1alpha2.Gateway)
	}

	// TODO The current implementation does not fully support the definition of Gateway.
	// We can update `spec.addresses` with the current data plane information.
	// At present, we choose to directly update `GatewayStatus.Addresses`
	// to indicate that we have picked the Gateway resource.

	c.controller.recordStatus(gateway, string(gatewayv1alpha2.ListenerReasonReady), nil, metav1.ConditionTrue, gateway.Generation)
	return nil
}

func (c *gatewayController) handleSyncErr(obj interface{}, err error) {
	if err == nil {
		c.workqueue.Forget(obj)
		c.controller.MetricsCollector.IncrSyncOperation("gateway", "success")
		return
	}
	log.Warnw("sync gateway failed, will retry",
		zap.Any("object", obj),
		zap.Error(err),
	)
	c.workqueue.AddRateLimited(obj)
	c.controller.MetricsCollector.IncrSyncOperation("gateway", "failure")
}

func (c *gatewayController) onAdd(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found gateway resource with bad meta namespace key: %s", err)
		return
	}
	if !c.controller.namespaceWatching(key) {
		return
	}
	log.Debugw("gateway add event arrived",
		zap.Any("object", obj),
	)

	c.workqueue.AddRateLimited(&types.Event{
		Type:   types.EventAdd,
		Object: key,
	})
}
func (c *gatewayController) onUpdate(oldObj, newObj interface{}) {}
func (c *gatewayController) OnDelete(obj interface{})            {}
