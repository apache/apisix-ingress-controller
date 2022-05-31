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
package gateway

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/types"
)

const (
	GatewayClassName = "apisix-ingress-controller"
)

type gatewayClassController struct {
	controller *GatewayProvider
	workqueue  workqueue.RateLimitingInterface
	workers    int
}

func newGatewayClassController(c *GatewayProvider) *gatewayClassController {
	ctrl := &gatewayClassController{
		controller: c,
		workqueue:  workqueue.NewNamedRateLimitingQueue(workqueue.NewItemFastSlowRateLimiter(1*time.Second, 60*time.Second, 5), "GatewayClass"),
		workers:    1,
	}

	ctrl.init()

	ctrl.controller.gatewayClassInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    ctrl.onAdd,
		UpdateFunc: ctrl.onUpdate,
		DeleteFunc: ctrl.OnDelete,
	})
	return ctrl
}

func (c *gatewayClassController) init() error {
	classes, err := c.controller.gatewayClassLister.List(labels.Everything())
	if err != nil {
		return err
	}

	for _, gatewayClass := range classes {
		if gatewayClass.Spec.ControllerName == GatewayClassName {

			err := c.recordStatus(gatewayClass, metav1.Condition{
				Type:               string(v1alpha2.GatewayClassConditionStatusAccepted),
				Status:             metav1.ConditionTrue,
				Reason:             "Updated",
				Message:            fmt.Sprintf("Updated by apisix-ingress-controller, sync at %v", time.Now()),
				LastTransitionTime: metav1.Now(),
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *gatewayClassController) recordStatus(gatewayClass *v1alpha2.GatewayClass, condition metav1.Condition) error {
	gc := gatewayClass.DeepCopy()

	var newConditions []metav1.Condition
	for _, cond := range gc.Status.Conditions {
		if cond.Type == condition.Type {
			if cond.Status == condition.Status {
				// Update message to record last sync time, don't change LastTransitionTime
				cond.Message = condition.Message
			} else {
				newConditions = append(newConditions, condition)
			}
		}

		if cond.Type != condition.Type {
			newConditions = append(newConditions, cond)
		}
	}

	gc.Status.Conditions = newConditions

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.controller.gatewayClient.GatewayV1alpha2().GatewayClasses().UpdateStatus(ctx, gc, metav1.UpdateOptions{})
	if err != nil {
		log.Errorw("failed to update GatewayClass status",
			zap.Error(err),
			zap.String("name", gatewayClass.Name),
		)
		return err
	}

	return nil
}

func (c *gatewayClassController) run(ctx context.Context) {
	log.Info("gateway HTTPRoute controller started")
	defer log.Info("gateway HTTPRoute controller exited")
	defer c.workqueue.ShutDown()

	if !cache.WaitForCacheSync(ctx.Done(), c.controller.gatewayClassInformer.HasSynced) {
		log.Error("sync Gateway HTTPRoute cache failed")
		return
	}

	for i := 0; i < c.workers; i++ {
		go c.runWorker(ctx)
	}
	<-ctx.Done()
}

func (c *gatewayClassController) runWorker(ctx context.Context) {
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

func (c *gatewayClassController) sync(ctx context.Context, ev *types.Event) error {
	key := ev.Object.(string)
	_ = key
	return nil
}

func (c *gatewayClassController) handleSyncErr(obj interface{}, err error) {
	if err == nil {
		c.workqueue.Forget(obj)
		c.controller.MetricsCollector.IncrSyncOperation("gateway_class", "success")
		return
	}
	event := obj.(*types.Event)
	if k8serrors.IsNotFound(err) && event.Type != types.EventDelete {
		log.Infow("sync gateway HTTPRoute but not found, ignore",
			zap.String("event_type", event.Type.String()),
			zap.String("HTTPRoute ", event.Object.(string)),
		)
		c.workqueue.Forget(event)
		return
	}
	log.Warnw("sync gateway HTTPRoute failed, will retry",
		zap.Any("object", obj),
		zap.Error(err),
	)
	c.workqueue.AddRateLimited(obj)
	c.controller.MetricsCollector.IncrSyncOperation("gateway_class", "failure")
}

func (c *gatewayClassController) onAdd(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found gateway HTTPRoute resource with bad meta namespace key: %s", err)
		return
	}
	if !c.controller.NamespaceProvider.IsWatchingNamespace(key) {
		return
	}
	log.Debugw("gateway HTTPRoute add event arrived",
		zap.Any("object", obj),
	)

	c.workqueue.Add(&types.Event{
		Type:   types.EventAdd,
		Object: key,
	})
}
func (c *gatewayClassController) onUpdate(oldObj, newObj interface{}) {}
func (c *gatewayClassController) OnDelete(obj interface{})            {}
