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
	"fmt"
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/kube"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/types"
)

type apisixConsumerController struct {
	controller *Controller
	workqueue  workqueue.RateLimitingInterface
	workers    int
}

func (c *Controller) newApisixConsumerController() *apisixConsumerController {
	ctl := &apisixConsumerController{
		controller: c,
		workqueue:  workqueue.NewNamedRateLimitingQueue(workqueue.NewItemFastSlowRateLimiter(1*time.Second, 60*time.Second, 5), "ApisixConsumer"),
		workers:    1,
	}
	ctl.controller.apisixConsumerInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctl.onAdd,
			UpdateFunc: ctl.onUpdate,
			DeleteFunc: ctl.onDelete,
		},
	)
	return ctl
}

func (c *apisixConsumerController) run(ctx context.Context) {
	log.Info("ApisixConsumer controller started")
	defer log.Info("ApisixConsumer controller exited")
	if ok := cache.WaitForCacheSync(ctx.Done(), c.controller.apisixConsumerInformer.HasSynced); !ok {
		log.Error("cache sync failed")
		return
	}
	for i := 0; i < c.workers; i++ {
		go c.runWorker(ctx)
	}
	<-ctx.Done()
	c.workqueue.ShutDown()
}

func (c *apisixConsumerController) runWorker(ctx context.Context) {
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

func (c *apisixConsumerController) sync(ctx context.Context, ev *types.Event) error {
	event := ev.Object.(kube.ApisixConsumerEvent)
	key := event.Key
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		log.Errorf("found ApisixConsumer resource with invalid meta namespace key %s: %s", key, err)
		return err
	}

	var multiVersioned kube.ApisixConsumer
	switch event.GroupVersion {
	case config.ApisixV2beta3:
		multiVersioned, err = c.controller.apisixConsumerLister.V2beta3(namespace, name)
	case config.ApisixV2:
		multiVersioned, err = c.controller.apisixConsumerLister.V2(namespace, name)
	default:
		return fmt.Errorf("unsupported ApisixConsumer group version %s", event.GroupVersion)
	}

	if err != nil {
		if !k8serrors.IsNotFound(err) {
			log.Errorw("failed to get ApisixConsumer",
				zap.Error(err),
				zap.String("key", key),
				zap.String("version", event.GroupVersion),
			)
			return err
		}
		if ev.Type != types.EventDelete {
			log.Warnw("ApisixConsumer was deleted before it can be delivered",
				zap.String("key", key),
				zap.String("version", event.GroupVersion),
			)
			// Don't need to retry.
			return nil
		}
	}
	if ev.Type == types.EventDelete {
		if multiVersioned != nil {
			// We still find the resource while we are processing the DELETE event,
			// that means object with same namespace and name was created, discarding
			// this stale DELETE event.
			log.Warnf("discard the stale ApisixConsumer delete event since the %s exists", key)
			return nil
		}
		multiVersioned = ev.Tombstone.(kube.ApisixConsumer)
	}

	switch event.GroupVersion {
	case config.ApisixV2beta3:
		ac := multiVersioned.V2beta3()

		consumer, err := c.controller.translator.TranslateApisixConsumerV2beta3(ac)
		if err != nil {
			log.Errorw("failed to translate ApisixConsumer",
				zap.Error(err),
				zap.Any("ApisixConsumer", ac),
			)
			c.controller.recorderEvent(ac, corev1.EventTypeWarning, _resourceSyncAborted, err)
			c.controller.recordStatus(ac, _resourceSyncAborted, err, metav1.ConditionFalse, ac.GetGeneration())
			return err
		}
		log.Debugw("got consumer object from ApisixConsumer",
			zap.Any("consumer", consumer),
			zap.Any("ApisixConsumer", ac),
		)

		if err := c.controller.syncConsumer(ctx, consumer, ev.Type); err != nil {
			log.Errorw("failed to sync Consumer to APISIX",
				zap.Error(err),
				zap.Any("consumer", consumer),
			)
			c.controller.recorderEvent(ac, corev1.EventTypeWarning, _resourceSyncAborted, err)
			c.controller.recordStatus(ac, _resourceSyncAborted, err, metav1.ConditionFalse, ac.GetGeneration())
			return err
		}

		c.controller.recorderEvent(ac, corev1.EventTypeNormal, _resourceSynced, nil)
	case config.ApisixV2:
		ac := multiVersioned.V2()

		consumer, err := c.controller.translator.TranslateApisixConsumerV2(ac)
		if err != nil {
			log.Errorw("failed to translate ApisixConsumer",
				zap.Error(err),
				zap.Any("ApisixConsumer", ac),
			)
			c.controller.recorderEvent(ac, corev1.EventTypeWarning, _resourceSyncAborted, err)
			c.controller.recordStatus(ac, _resourceSyncAborted, err, metav1.ConditionFalse, ac.GetGeneration())
			return err
		}
		log.Debugw("got consumer object from ApisixConsumer",
			zap.Any("consumer", consumer),
			zap.Any("ApisixConsumer", ac),
		)

		if err := c.controller.syncConsumer(ctx, consumer, ev.Type); err != nil {
			log.Errorw("failed to sync Consumer to APISIX",
				zap.Error(err),
				zap.Any("consumer", consumer),
			)
			c.controller.recorderEvent(ac, corev1.EventTypeWarning, _resourceSyncAborted, err)
			c.controller.recordStatus(ac, _resourceSyncAborted, err, metav1.ConditionFalse, ac.GetGeneration())
			return err
		}

		c.controller.recorderEvent(ac, corev1.EventTypeNormal, _resourceSynced, nil)
	}
	return nil
}

func (c *apisixConsumerController) handleSyncErr(obj interface{}, err error) {
	if err == nil {
		c.workqueue.Forget(obj)
		c.controller.MetricsCollector.IncrSyncOperation("consumer", "success")
		return
	}
	event := obj.(*types.Event)
	if k8serrors.IsNotFound(err) && event.Type != types.EventDelete {
		log.Infow("sync ApisixConsumer but not found, ignore",
			zap.String("event_type", event.Type.String()),
			zap.String("ApisixConsumer", event.Object.(string)),
		)
		c.workqueue.Forget(event)
		return
	}
	log.Warnw("sync ApisixConsumer failed, will retry",
		zap.Any("object", obj),
		zap.Error(err),
	)
	c.workqueue.AddRateLimited(obj)
	c.controller.MetricsCollector.IncrSyncOperation("consumer", "failure")
}

func (c *apisixConsumerController) onAdd(obj interface{}) {
	ac, err := kube.NewApisixConsumer(obj)
	if err != nil {
		log.Errorw("found ApisixConsumer resource with bad type", zap.Error(err))
		return
	}
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found ApisixConsumer resource with bad meta namespace key: %s", err)
		return
	}
	if !c.controller.isWatchingNamespace(key) {
		return
	}
	log.Debugw("ApisixConsumer add event arrived",
		zap.Any("object", obj),
	)

	c.workqueue.Add(&types.Event{
		Type: types.EventAdd,
		Object: kube.ApisixConsumerEvent{
			Key:          key,
			GroupVersion: ac.GroupVersion(),
		},
	})

	c.controller.MetricsCollector.IncrEvents("consumer", "add")
}

func (c *apisixConsumerController) onUpdate(oldObj, newObj interface{}) {
	prev, err := kube.NewApisixConsumer(oldObj)
	if err != nil {
		log.Errorw("found ApisixConsumer resource with bad type", zap.Error(err))
		return
	}
	curr, err := kube.NewApisixConsumer(newObj)
	if err != nil {
		log.Errorw("found ApisixConsumer resource with bad type", zap.Error(err))
		return
	}
	if prev.ResourceVersion() >= curr.ResourceVersion() {
		return
	}
	key, err := cache.MetaNamespaceKeyFunc(newObj)
	if err != nil {
		log.Errorf("found ApisixConsumer resource with bad meta namespace key: %s", err)
		return
	}
	if !c.controller.isWatchingNamespace(key) {
		return
	}
	log.Debugw("ApisixConsumer update event arrived",
		zap.Any("new object", curr),
		zap.Any("old object", prev),
	)

	c.workqueue.Add(&types.Event{
		Type: types.EventUpdate,
		Object: kube.ApisixConsumerEvent{
			Key:          key,
			OldObject:    prev,
			GroupVersion: curr.GroupVersion(),
		},
	})

	c.controller.MetricsCollector.IncrEvents("consumer", "update")
}

func (c *apisixConsumerController) onDelete(obj interface{}) {
	ac, err := kube.NewApisixConsumer(obj)
	if err != nil {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			return
		}
		ac, err = kube.NewApisixConsumer(tombstone.Obj)
		if err != nil {
			log.Errorw("found ApisixConsumer resource with bad type", zap.Error(err))
			return
		}
	}

	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found ApisixConsumer resource with bad meta namespace key: %s", err)
		return
	}
	if !c.controller.isWatchingNamespace(key) {
		return
	}
	log.Debugw("ApisixConsumer delete event arrived",
		zap.Any("final state", ac),
	)
	c.workqueue.Add(&types.Event{
		Type: types.EventDelete,
		Object: kube.ApisixConsumerEvent{
			Key:          key,
			GroupVersion: ac.GroupVersion(),
		},
		Tombstone: ac,
	})

	c.controller.MetricsCollector.IncrEvents("consumer", "delete")
}

func (c *apisixConsumerController) ResourceSync() {
	objs := c.controller.apisixConsumerInformer.GetIndexer().List()
	for _, obj := range objs {
		key, err := cache.MetaNamespaceKeyFunc(obj)
		if err != nil {
			log.Errorf("ApisixConsumer sync failed, found ApisixConsumer resource with bad meta namespace key: %s", err)
			continue
		}
		if !c.controller.isWatchingNamespace(key) {
			continue
		}
		c.workqueue.Add(&types.Event{
			Type:   types.EventAdd,
			Object: key,
		})
	}
}
