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
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/ingress/utils"
	"github.com/apache/apisix-ingress-controller/pkg/kube"
	"github.com/apache/apisix-ingress-controller/pkg/kube/translation"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/types"
)

type apisixPluginConfigController struct {
	controller *Controller
	workqueue  workqueue.RateLimitingInterface
	workers    int
}

func (c *Controller) newApisixPluginConfigController() *apisixPluginConfigController {
	ctl := &apisixPluginConfigController{
		controller: c,
		workqueue:  workqueue.NewNamedRateLimitingQueue(workqueue.NewItemFastSlowRateLimiter(1*time.Second, 60*time.Second, 5), "ApisixPluginConfig"),
		workers:    1,
	}
	c.apisixPluginConfigInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctl.onAdd,
			UpdateFunc: ctl.onUpdate,
			DeleteFunc: ctl.onDelete,
		},
	)
	return ctl
}

func (c *apisixPluginConfigController) run(ctx context.Context) {
	log.Info("ApisixPluginConfig controller started")
	defer log.Info("ApisixPluginConfig controller exited")
	defer c.workqueue.ShutDown()

	ok := cache.WaitForCacheSync(ctx.Done(), c.controller.apisixPluginConfigInformer.HasSynced)
	if !ok {
		log.Error("cache sync failed")
		return
	}

	for i := 0; i < c.workers; i++ {
		go c.runWorker(ctx)
	}
	<-ctx.Done()
}

func (c *apisixPluginConfigController) runWorker(ctx context.Context) {
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

func (c *apisixPluginConfigController) sync(ctx context.Context, ev *types.Event) error {
	obj := ev.Object.(kube.ApisixPluginConfigEvent)
	namespace, name, err := cache.SplitMetaNamespaceKey(obj.Key)
	if err != nil {
		log.Errorf("invalid resource key: %s", obj.Key)
		return err
	}
	var (
		apc  kube.ApisixPluginConfig
		tctx *translation.TranslateContext
	)
	switch obj.GroupVersion {
	case config.ApisixV2beta3:
		apc, err = c.controller.apisixPluginConfigLister.V2beta3(namespace, name)
	case config.ApisixV2:
		apc, err = c.controller.apisixPluginConfigLister.V2(namespace, name)
	default:
		return fmt.Errorf("unsupported ApisixPluginConfig group version %s", obj.GroupVersion)
	}
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			log.Errorw("failed to get ApisixPluginConfig",
				zap.String("version", obj.GroupVersion),
				zap.String("key", obj.Key),
				zap.Error(err),
			)
			return err
		}

		if ev.Type != types.EventDelete {
			log.Warnw("ApisixPluginConfig was deleted before it can be delivered",
				zap.String("key", obj.Key),
				zap.String("version", obj.GroupVersion),
			)
			return nil
		}
	}
	if ev.Type == types.EventDelete {
		if apc != nil {
			// We still find the resource while we are processing the DELETE event,
			// that means object with same namespace and name was created, discarding
			// this stale DELETE event.
			log.Warnw("discard the stale ApisixPluginConfig delete event since the resource still exists",
				zap.String("key", obj.Key),
			)
			return nil
		}
		apc = ev.Tombstone.(kube.ApisixPluginConfig)
	}

	switch obj.GroupVersion {
	case config.ApisixV2beta3:
		if ev.Type != types.EventDelete {
			tctx, err = c.controller.translator.TranslatePluginConfigV2beta3(apc.V2beta3())
		} else {
			tctx, err = c.controller.translator.TranslatePluginConfigV2beta3NotStrictly(apc.V2beta3())
		}
		if err != nil {
			log.Errorw("failed to translate ApisixPluginConfig v2beta3",
				zap.Error(err),
				zap.Any("object", apc),
			)
			return err
		}
	case config.ApisixV2:
		if ev.Type != types.EventDelete {
			tctx, err = c.controller.translator.TranslatePluginConfigV2(apc.V2())
		} else {
			tctx, err = c.controller.translator.TranslatePluginConfigV2NotStrictly(apc.V2())
		}
		if err != nil {
			log.Errorw("failed to translate ApisixPluginConfig v2",
				zap.Error(err),
				zap.Any("object", apc),
			)
			return err
		}
	}

	log.Debugw("translated ApisixPluginConfig",
		zap.Any("pluginConfigs", tctx.PluginConfigs),
	)

	m := &utils.Manifest{
		PluginConfigs: tctx.PluginConfigs,
	}

	var (
		added   *utils.Manifest
		updated *utils.Manifest
		deleted *utils.Manifest
	)

	if ev.Type == types.EventDelete {
		deleted = m
	} else if ev.Type == types.EventAdd {
		added = m
	} else {
		var oldCtx *translation.TranslateContext
		switch obj.GroupVersion {
		case config.ApisixV2beta3:
			oldCtx, err = c.controller.translator.TranslatePluginConfigV2beta3(obj.OldObject.V2beta3())
		case config.ApisixV2:
			oldCtx, err = c.controller.translator.TranslatePluginConfigV2(obj.OldObject.V2())
		}
		if err != nil {
			log.Errorw("failed to translate old ApisixPluginConfig",
				zap.String("version", obj.GroupVersion),
				zap.String("event", "update"),
				zap.Error(err),
				zap.Any("ApisixPluginConfig", apc),
			)
			return err
		}

		om := &utils.Manifest{
			PluginConfigs: oldCtx.PluginConfigs,
		}
		added, updated, deleted = m.Diff(om)
	}

	return c.controller.syncManifests(ctx, added, updated, deleted)
}

func (c *apisixPluginConfigController) handleSyncErr(obj interface{}, errOrigin error) {
	ev := obj.(*types.Event)
	event := ev.Object.(kube.ApisixPluginConfigEvent)
	if k8serrors.IsNotFound(errOrigin) && ev.Type != types.EventDelete {
		log.Infow("sync ApisixPluginConfig but not found, ignore",
			zap.String("event_type", ev.Type.String()),
			zap.String("ApisixPluginConfig", ev.Object.(kube.ApisixPluginConfigEvent).Key),
		)
		c.workqueue.Forget(event)
		return
	}
	namespace, name, errLocal := cache.SplitMetaNamespaceKey(event.Key)
	if errLocal != nil {
		log.Errorf("invalid resource key: %s", event.Key)
		c.controller.MetricsCollector.IncrSyncOperation("PluginConfig", "failure")
		return
	}
	var apc kube.ApisixPluginConfig
	switch event.GroupVersion {
	case config.ApisixV2beta3:
		apc, errLocal = c.controller.apisixPluginConfigLister.V2beta3(namespace, name)
	case config.ApisixV2:
		apc, errLocal = c.controller.apisixPluginConfigLister.V2(namespace, name)
	default:
		errLocal = fmt.Errorf("unsupported ApisixPluginConfig group version %s", event.GroupVersion)
	}
	if errOrigin == nil {
		if ev.Type != types.EventDelete {
			if errLocal == nil {
				switch apc.GroupVersion() {
				case config.ApisixV2beta3:
					c.controller.recorderEvent(apc.V2beta3(), v1.EventTypeNormal, _resourceSynced, nil)
					c.controller.recordStatus(apc.V2beta3(), _resourceSynced, nil, metav1.ConditionTrue, apc.V2beta3().GetGeneration())
				case config.ApisixV2:
					c.controller.recorderEvent(apc.V2(), v1.EventTypeNormal, _resourceSynced, nil)
					c.controller.recordStatus(apc.V2(), _resourceSynced, nil, metav1.ConditionTrue, apc.V2().GetGeneration())
				}
			} else {
				log.Errorw("failed list ApisixPluginConfig",
					zap.Error(errLocal),
					zap.String("name", name),
					zap.String("namespace", namespace),
				)
			}
		}
		c.workqueue.Forget(obj)
		c.controller.MetricsCollector.IncrSyncOperation("PluginConfig", "success")
		return
	}
	log.Warnw("sync ApisixPluginConfig failed, will retry",
		zap.Any("object", obj),
		zap.Error(errOrigin),
	)
	if errLocal == nil {
		switch apc.GroupVersion() {
		case config.ApisixV2beta3:
			c.controller.recorderEvent(apc.V2beta3(), v1.EventTypeWarning, _resourceSyncAborted, errOrigin)
			c.controller.recordStatus(apc.V2beta3(), _resourceSyncAborted, errOrigin, metav1.ConditionFalse, apc.V2beta3().GetGeneration())
		case config.ApisixV2:
			c.controller.recorderEvent(apc.V2(), v1.EventTypeWarning, _resourceSyncAborted, errOrigin)
			c.controller.recordStatus(apc.V2(), _resourceSyncAborted, errOrigin, metav1.ConditionFalse, apc.V2().GetGeneration())
		}
	} else {
		log.Errorw("failed list ApisixPluginConfig",
			zap.Error(errLocal),
			zap.String("name", name),
			zap.String("namespace", namespace),
		)
	}
	c.workqueue.AddRateLimited(obj)
	c.controller.MetricsCollector.IncrSyncOperation("PluginConfig", "failure")
}

func (c *apisixPluginConfigController) onAdd(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found ApisixPluginConfig resource with bad meta namespace key: %s", err)
		return
	}
	if !c.controller.isWatchingNamespace(key) {
		return
	}
	log.Debugw("ApisixPluginConfig add event arrived",
		zap.Any("object", obj))

	apc := kube.MustNewApisixPluginConfig(obj)
	c.workqueue.Add(&types.Event{
		Type: types.EventAdd,
		Object: kube.ApisixPluginConfigEvent{
			Key:          key,
			GroupVersion: apc.GroupVersion(),
		},
	})

	c.controller.MetricsCollector.IncrEvents("PluginConfig", "add")
}

func (c *apisixPluginConfigController) onUpdate(oldObj, newObj interface{}) {
	prev := kube.MustNewApisixPluginConfig(oldObj)
	curr := kube.MustNewApisixPluginConfig(newObj)
	if prev.ResourceVersion() >= curr.ResourceVersion() {
		return
	}
	key, err := cache.MetaNamespaceKeyFunc(newObj)
	if err != nil {
		log.Errorf("found ApisixPluginConfig resource with bad meta namespace key: %s", err)
		return
	}
	if !c.controller.isWatchingNamespace(key) {
		return
	}
	log.Debugw("ApisixPluginConfig update event arrived",
		zap.Any("new object", curr),
		zap.Any("old object", prev),
	)
	c.workqueue.Add(&types.Event{
		Type: types.EventUpdate,
		Object: kube.ApisixPluginConfigEvent{
			Key:          key,
			GroupVersion: curr.GroupVersion(),
			OldObject:    prev,
		},
	})

	c.controller.MetricsCollector.IncrEvents("PluginConfig", "update")
}

func (c *apisixPluginConfigController) onDelete(obj interface{}) {
	apc, err := kube.NewApisixPluginConfig(obj)
	if err != nil {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			return
		}
		apc = kube.MustNewApisixPluginConfig(tombstone)
	}
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found ApisixPluginConfig resource with bad meta namesapce key: %s", err)
		return
	}
	if !c.controller.isWatchingNamespace(key) {
		return
	}
	log.Debugw("ApisixPluginConfig delete event arrived",
		zap.Any("final state", apc),
	)
	c.workqueue.Add(&types.Event{
		Type: types.EventDelete,
		Object: kube.ApisixPluginConfigEvent{
			Key:          key,
			GroupVersion: apc.GroupVersion(),
		},
		Tombstone: apc,
	})

	c.controller.MetricsCollector.IncrEvents("PluginConfig", "delete")
}
