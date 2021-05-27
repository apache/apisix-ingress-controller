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
	configv1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v1"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/types"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

type apisixUpstreamController struct {
	controller *Controller
	workqueue  workqueue.RateLimitingInterface
	workers    int
}

func (c *Controller) newApisixUpstreamController() *apisixUpstreamController {
	ctl := &apisixUpstreamController{
		controller: c,
		workqueue:  workqueue.NewNamedRateLimitingQueue(workqueue.NewItemFastSlowRateLimiter(1*time.Second, 60*time.Second, 5), "ApisixUpstream"),
		workers:    1,
	}
	ctl.controller.apisixUpstreamInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctl.onAdd,
			UpdateFunc: ctl.onUpdate,
			DeleteFunc: ctl.onDelete,
		},
	)
	return ctl
}

func (c *apisixUpstreamController) run(ctx context.Context) {
	log.Info("ApisixUpstream controller started")
	defer log.Info("ApisixUpstream controller exited")
	defer c.workqueue.ShutDown()

	if ok := cache.WaitForCacheSync(ctx.Done(), c.controller.apisixUpstreamInformer.HasSynced, c.controller.svcInformer.HasSynced); !ok {
		log.Error("cache sync failed")
		return
	}
	for i := 0; i < c.workers; i++ {
		go c.runWorker(ctx)
	}

	<-ctx.Done()
}

func (c *apisixUpstreamController) runWorker(ctx context.Context) {
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

func (c *apisixUpstreamController) sync(ctx context.Context, ev *types.Event) error {
	key := ev.Object.(string)
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		log.Errorf("found ApisixUpstream resource with invalid meta namespace key %s: %s", key, err)
		return err
	}

	au, err := c.controller.apisixUpstreamLister.ApisixUpstreams(namespace).Get(name)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			log.Errorf("failed to get ApisixUpstream %s: %s", key, err)
			return err
		}
		if ev.Type != types.EventDelete {
			log.Warnf("ApisixUpstream %s was deleted before it can be delivered", key)
			// Don't need to retry.
			return nil
		}
	}
	if ev.Type == types.EventDelete {
		if au != nil {
			// We still find the resource while we are processing the DELETE event,
			// that means object with same namespace and name was created, discarding
			// this stale DELETE event.
			log.Warnf("discard the stale ApisixUpstream delete event since the %s exists", key)
			return nil
		}
		au = ev.Tombstone.(*configv1.ApisixUpstream)
	}

	var portLevelSettings map[int32]*configv1.ApisixUpstreamConfig
	if len(au.Spec.PortLevelSettings) > 0 {
		portLevelSettings = make(map[int32]*configv1.ApisixUpstreamConfig, len(au.Spec.PortLevelSettings))
		for _, port := range au.Spec.PortLevelSettings {
			portLevelSettings[port.Port] = &port.ApisixUpstreamConfig
		}
	}

	svc, err := c.controller.svcLister.Services(namespace).Get(name)
	if err != nil {
		log.Errorf("failed to get service %s: %s", key, err)
		c.controller.recorderEvent(au, corev1.EventTypeWarning, _resourceSyncAborted, err)
		c.controller.recordStatus(au, _resourceSyncAborted, err, metav1.ConditionFalse)
		return err
	}

	var subsets []configv1.ApisixUpstreamSubset
	subsets = append(subsets, configv1.ApisixUpstreamSubset{})
	if len(au.Spec.Subsets) > 0 {
		subsets = append(subsets, au.Spec.Subsets...)
	}
	clusterName := c.controller.cfg.APISIX.DefaultClusterName
	for _, port := range svc.Spec.Ports {
		for _, subset := range subsets {
			upsName := apisixv1.ComposeUpstreamName(namespace, name, subset.Name, port.Port)
			// TODO: multiple cluster
			ups, err := c.controller.apisix.Cluster(clusterName).Upstream().Get(ctx, upsName)
			if err != nil {
				if err == apisixcache.ErrNotFound {
					continue
				}
				log.Errorf("failed to get upstream %s: %s", upsName, err)
				c.controller.recorderEvent(au, corev1.EventTypeWarning, _resourceSyncAborted, err)
				c.controller.recordStatus(au, _resourceSyncAborted, err, metav1.ConditionFalse)
				return err
			}
			var newUps *apisixv1.Upstream
			if ev.Type != types.EventDelete {
				cfg, ok := portLevelSettings[port.Port]
				if !ok {
					cfg = &au.Spec.ApisixUpstreamConfig
				}
				// FIXME Same ApisixUpstreamConfig might be translated multiple times.
				newUps, err = c.controller.translator.TranslateUpstreamConfig(cfg)
				if err != nil {
					log.Errorw("found malformed ApisixUpstream",
						zap.Any("object", au),
						zap.Error(err),
					)
					c.controller.recorderEvent(au, corev1.EventTypeWarning, _resourceSyncAborted, err)
					c.controller.recordStatus(au, _resourceSyncAborted, err, metav1.ConditionFalse)
					return err
				}
			} else {
				newUps = apisixv1.NewDefaultUpstream()
			}

			newUps.Metadata = ups.Metadata
			newUps.Nodes = ups.Nodes
			log.Debugw("updating upstream since ApisixUpstream changed",
				zap.String("event", ev.Type.String()),
				zap.Any("upstream", newUps),
				zap.Any("ApisixUpstream", au),
			)
			if _, err := c.controller.apisix.Cluster(clusterName).Upstream().Update(ctx, newUps); err != nil {
				log.Errorw("failed to update upstream",
					zap.Error(err),
					zap.Any("upstream", newUps),
					zap.Any("ApisixUpstream", au),
					zap.String("cluster", clusterName),
				)
				c.controller.recorderEvent(au, corev1.EventTypeWarning, _resourceSyncAborted, err)
				c.controller.recordStatus(au, _resourceSyncAborted, err, metav1.ConditionFalse)
				return err
			}
		}
	}
	c.controller.recorderEvent(au, corev1.EventTypeNormal, _resourceSynced, nil)
	c.controller.recordStatus(au, _resourceSynced, nil, metav1.ConditionTrue)
	return err
}

func (c *apisixUpstreamController) handleSyncErr(obj interface{}, err error) {
	if err == nil {
		c.workqueue.Forget(obj)
		return
	}
	log.Warnw("sync ApisixUpstream failed, will retry",
		zap.Any("object", obj),
		zap.Error(err),
	)
	c.workqueue.AddRateLimited(obj)
}

func (c *apisixUpstreamController) onAdd(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found ApisixUpstream resource with bad meta namespace key: %s", err)
		return
	}
	if !c.controller.namespaceWatching(key) {
		return
	}
	log.Debugw("ApisixUpstream add event arrived",
		zap.Any("object", obj))

	c.workqueue.AddRateLimited(&types.Event{
		Type:   types.EventAdd,
		Object: key,
	})
}

func (c *apisixUpstreamController) onUpdate(oldObj, newObj interface{}) {
	prev := oldObj.(*configv1.ApisixUpstream)
	curr := newObj.(*configv1.ApisixUpstream)
	if prev.ResourceVersion >= curr.ResourceVersion {
		return
	}
	key, err := cache.MetaNamespaceKeyFunc(newObj)
	if err != nil {
		log.Errorf("found ApisixUpstream resource with bad meta namespace key: %s", err)
		return
	}
	if !c.controller.namespaceWatching(key) {
		return
	}
	log.Debugw("ApisixUpstream update event arrived",
		zap.Any("new object", curr),
		zap.Any("old object", prev),
	)

	c.workqueue.AddRateLimited(&types.Event{
		Type:   types.EventUpdate,
		Object: key,
	})
}

func (c *apisixUpstreamController) onDelete(obj interface{}) {
	au, ok := obj.(*configv1.ApisixUpstream)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			return
		}
		au = tombstone.Obj.(*configv1.ApisixUpstream)
	}

	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found ApisixUpstream resource with bad meta namespace key: %s", err)
		return
	}
	if !c.controller.namespaceWatching(key) {
		return
	}
	log.Debugw("ApisixUpstream delete event arrived",
		zap.Any("final state", au),
	)
	c.workqueue.AddRateLimited(&types.Event{
		Type:      types.EventDelete,
		Object:    key,
		Tombstone: au,
	})
}
