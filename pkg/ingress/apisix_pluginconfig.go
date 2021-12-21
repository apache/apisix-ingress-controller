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
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	apisixcache "github.com/apache/apisix-ingress-controller/pkg/apisix/cache"
	configv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta3"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/types"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

type apisixPluginConfigController struct {
	controller *Controller
	workqueue  workqueue.RateLimitingInterface
	workers    int
}

func (c *Controller) newapisixPluginConfigController() *apisixPluginConfigController {
	ctl := &apisixPluginConfigController{
		controller: c,
		workqueue:  workqueue.NewNamedRateLimitingQueue(workqueue.NewItemFastSlowRateLimiter(1*time.Second, 60*time.Second, 5), "ApisixPluginConfig"),
		workers:    1,
	}
	ctl.controller.apisixPluginConfigInformer.AddEventHandler(
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

	if ok := cache.WaitForCacheSync(ctx.Done(), c.controller.apisixPluginConfigInformer.HasSynced, c.controller.svcInformer.HasSynced); !ok {
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

// sync Used to synchronize ApisixPluginConfig resources, because pluginConfig alone exists in APISIX and will not be affected,
// the synchronization logic only includes pluginConfig's unique configuration management
// So when ApisixPluginConfig was deleted, only the scheme / load balancer / healthcheck / retry / timeout
// on ApisixPluginConfig was cleaned up
func (c *apisixPluginConfigController) sync(ctx context.Context, ev *types.Event) error {
	key := ev.Object.(string)
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		log.Errorf("found ApisixPluginConfig resource with invalid meta namespace key %s: %s", key, err)
		return err
	}

	apc, err := c.controller.apisixPluginConfigLister.ApisixPluginConfigs(namespace).Get(name)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			log.Errorf("failed to get ApisixPluginConfig %s: %s", key, err)
			return err
		}
		if ev.Type != types.EventDelete {
			log.Warnf("ApisixPluginConfig %s was deleted before it can be delivered", key)
			// Don't need to retry.
			return nil
		}
	}
	if ev.Type == types.EventDelete {
		if apc != nil {
			// We still find the resource while we are processing the DELETE event,
			// that means object with same namespace and name was created, discarding
			// this stale DELETE event.
			log.Warnf("discard the stale ApisixPluginConfig delete event since the %s exists", key)
			return nil
		}
		apc = ev.Tombstone.(*configv2beta3.ApisixPluginConfig)
	}

	svc, err := c.controller.svcLister.Services(namespace).Get(name)
	if err != nil {
		log.Errorf("failed to get service %s: %s", key, err)
		c.controller.recorderEvent(apc, corev1.EventTypeWarning, _resourceSyncAborted, err)
		c.controller.recordStatus(apc, _resourceSyncAborted, err, metav1.ConditionFalse, apc.GetGeneration())
		return err
	}

	clusterName := c.controller.cfg.APISIX.DefaultClusterName
	for _, port := range svc.Spec.Ports {
		_ = port
		upsName := apisixv1.ComposePluginConfigName(namespace, name)
		// TODO: multiple cluster
		ups, err := c.controller.apisix.Cluster(clusterName).PluginConfig().Get(ctx, upsName)
		if err != nil {
			if err == apisixcache.ErrNotFound {
				continue
			}
			log.Errorf("failed to get plugin_config %s: %s", upsName, err)
			c.controller.recorderEvent(apc, corev1.EventTypeWarning, _resourceSyncAborted, err)
			c.controller.recordStatus(apc, _resourceSyncAborted, err, metav1.ConditionFalse, apc.GetGeneration())
			return err
		}
		var newUps *apisixv1.PluginConfig
		if ev.Type != types.EventDelete {
			// FIXME Same ApisixPluginConfig might be translated multiple times.
			newUps, err = c.controller.translator.TranslateApisixPluginConfig(apc)
			if err != nil {
				log.Errorw("found malformed ApisixPluginConfig",
					zap.Any("object", apc),
					zap.Error(err),
				)
				c.controller.recorderEvent(apc, corev1.EventTypeWarning, _resourceSyncAborted, err)
				c.controller.recordStatus(apc, _resourceSyncAborted, err, metav1.ConditionFalse, apc.GetGeneration())
				return err
			}
		} else {
			newUps = apisixv1.NewDefaultPluginConfig()
		}

		newUps.Metadata = ups.Metadata
		newUps.Plugins = ups.Plugins
		log.Debugw("updating plugin_config since ApisixPluginConfig changed",
			zap.String("event", ev.Type.String()),
			zap.Any("plugin_config", newUps),
			zap.Any("ApisixPluginConfig", apc),
		)
		if _, err := c.controller.apisix.Cluster(clusterName).PluginConfig().Update(ctx, newUps); err != nil {
			log.Errorw("failed to update plugin_config",
				zap.Error(err),
				zap.Any("plugin_config", newUps),
				zap.Any("ApisixPluginConfig", apc),
				zap.String("cluster", clusterName),
			)
			c.controller.recorderEvent(apc, corev1.EventTypeWarning, _resourceSyncAborted, err)
			c.controller.recordStatus(apc, _resourceSyncAborted, err, metav1.ConditionFalse, apc.GetGeneration())
			return err
		}
	}
	if ev.Type != types.EventDelete {
		c.controller.recorderEvent(apc, corev1.EventTypeNormal, _resourceSynced, nil)
		c.controller.recordStatus(apc, _resourceSynced, nil, metav1.ConditionTrue, apc.GetGeneration())
	}
	return err
}

func (c *apisixPluginConfigController) handleSyncErr(obj interface{}, err error) {
	if err == nil {
		c.workqueue.Forget(obj)
		c.controller.MetricsCollector.IncrSyncOperation("plugin_config", "success")
		return
	}
	log.Warnw("sync ApisixPluginConfig failed, will retry",
		zap.Any("object", obj),
		zap.Error(err),
	)
	c.workqueue.AddRateLimited(obj)
	c.controller.MetricsCollector.IncrSyncOperation("plugin_config", "failure")
}

func (c *apisixPluginConfigController) onAdd(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found ApisixPluginConfig resource with bad meta namespace key: %s", err)
		return
	}
	if !c.controller.namespaceWatching(key) {
		return
	}
	log.Debugw("ApisixPluginConfig add event arrived",
		zap.Any("object", obj))

	c.workqueue.Add(&types.Event{
		Type:   types.EventAdd,
		Object: key,
	})

	c.controller.MetricsCollector.IncrEvents("plugin_config", "add")
}

func (c *apisixPluginConfigController) onUpdate(oldObj, newObj interface{}) {
	prev := oldObj.(*configv2beta3.ApisixClusterConfig)
	curr := newObj.(*configv2beta3.ApisixClusterConfig)
	if prev.ResourceVersion >= curr.ResourceVersion {
		return
	}
	key, err := cache.MetaNamespaceKeyFunc(newObj)
	if err != nil {
		log.Errorf("found ApisixPluginConfig resource with bad meta namespace key: %s", err)
		return
	}
	if !c.controller.namespaceWatching(key) {
		return
	}
	log.Debugw("ApisixPluginConfig update event arrived",
		zap.Any("new object", curr),
		zap.Any("old object", prev),
	)

	c.workqueue.Add(&types.Event{
		Type:   types.EventUpdate,
		Object: key,
	})

	c.controller.MetricsCollector.IncrEvents("plugin_config", "update")
}

func (c *apisixPluginConfigController) onDelete(obj interface{}) {
	au, ok := obj.(*configv2beta3.ApisixPluginConfig)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			return
		}
		au = tombstone.Obj.(*configv2beta3.ApisixPluginConfig)
	}

	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found ApisixPluginConfig resource with bad meta namespace key: %s", err)
		return
	}
	if !c.controller.namespaceWatching(key) {
		return
	}
	log.Debugw("ApisixPluginConfig delete event arrived",
		zap.Any("final state", au),
	)
	c.workqueue.Add(&types.Event{
		Type:      types.EventDelete,
		Object:    key,
		Tombstone: au,
	})

	c.controller.MetricsCollector.IncrEvents("plugin_config", "delete")
}
