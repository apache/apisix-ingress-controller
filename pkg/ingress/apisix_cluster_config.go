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

	"github.com/apache/apisix-ingress-controller/pkg/apisix"
	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/kube"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/types"
)

type apisixClusterConfigController struct {
	controller *Controller
	workqueue  workqueue.RateLimitingInterface
	workers    int
}

func (c *Controller) newApisixClusterConfigController() *apisixClusterConfigController {
	ctl := &apisixClusterConfigController{
		controller: c,
		workqueue:  workqueue.NewNamedRateLimitingQueue(workqueue.NewItemFastSlowRateLimiter(time.Second, 60*time.Second, 5), "ApisixClusterConfig"),
		workers:    1,
	}
	c.apisixClusterConfigInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctl.onAdd,
			UpdateFunc: ctl.onUpdate,
			DeleteFunc: ctl.onDelete,
		},
	)
	return ctl
}

func (c *apisixClusterConfigController) run(ctx context.Context) {
	log.Info("ApisixClusterConfig controller started")
	defer log.Info("ApisixClusterConfig controller exited")
	defer c.workqueue.ShutDown()

	if ok := cache.WaitForCacheSync(ctx.Done(), c.controller.apisixClusterConfigInformer.HasSynced); !ok {
		log.Error("cache sync failed")
		return
	}
	for i := 0; i < c.workers; i++ {
		go c.runWorker(ctx)
	}
	<-ctx.Done()
}

func (c *apisixClusterConfigController) runWorker(ctx context.Context) {
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

func (c *apisixClusterConfigController) sync(ctx context.Context, ev *types.Event) error {
	event := ev.Object.(kube.ApisixClusterConfigEvent)
	key := event.Key
	_, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		log.Errorf("found ApisixClusterConfig resource with invalid meta key %s: %s", key, err)
		return err
	}

	var multiVersioned kube.ApisixClusterConfig
	switch event.GroupVersion {
	case config.ApisixV2beta3:
		multiVersioned, err = c.controller.apisixClusterConfigLister.V2beta3(name)
	case config.ApisixV2:
		multiVersioned, err = c.controller.apisixClusterConfigLister.V2(name)
	default:
		return fmt.Errorf("unsupported ApisixClusterConfig group version %s", event.GroupVersion)
	}

	if err != nil {
		if !k8serrors.IsNotFound(err) {
			log.Errorw("failed to get ApisixClusterConfig",
				zap.Error(err),
				zap.String("key", key),
				zap.String("version", event.GroupVersion),
			)
			return err
		}
		if ev.Type != types.EventDelete {
			log.Warnw("ApisixClusterConfig was deleted before it can be delivered",
				zap.String("key", key),
				zap.String("version", event.GroupVersion),
			)
			return nil
		}
	}
	if ev.Type == types.EventDelete {
		if multiVersioned != nil {
			// We still find the resource while we are processing the DELETE event,
			// that means object with same namespace and name was created, discarding
			// this stale DELETE event.
			log.Warnf("discard the stale ApisixClusterConfig delete event since the %s exists", key)
			return nil
		}
		multiVersioned = ev.Tombstone.(kube.ApisixClusterConfig)
	}

	switch event.GroupVersion {
	case config.ApisixV2beta3:
		acc := multiVersioned.V2beta3()
		// Currently we don't handle multiple cluster, so only process
		// the default apisix cluster.
		if acc.Name != c.controller.cfg.APISIX.DefaultClusterName {
			log.Infow("ignore non-default apisix cluster config",
				zap.String("default_cluster_name", c.controller.cfg.APISIX.DefaultClusterName),
				zap.Any("ApisixClusterConfig", acc),
			)
			return nil
		}
		// Cluster delete is dangerous.
		// TODO handle delete?
		if ev.Type == types.EventDelete {
			log.Error("ApisixClusterConfig delete event for default apisix cluster will be ignored")
			return nil
		}

		if acc.Spec.Admin != nil {
			clusterOpts := &apisix.ClusterOptions{
				Name:     acc.Name,
				BaseURL:  acc.Spec.Admin.BaseURL,
				AdminKey: acc.Spec.Admin.AdminKey,
			}
			log.Infow("updating cluster",
				zap.Any("opts", clusterOpts),
			)
			// TODO we may first call AddCluster.
			// Since now we already have the default cluster, we just call UpdateCluster.
			if err := c.controller.apisix.UpdateCluster(ctx, clusterOpts); err != nil {
				log.Errorw("failed to update cluster",
					zap.String("cluster_name", acc.Name),
					zap.Error(err),
					zap.Any("opts", clusterOpts),
				)
				c.controller.recorderEvent(acc, corev1.EventTypeWarning, _resourceSyncAborted, err)
				c.controller.recordStatus(acc, _resourceSyncAborted, err, metav1.ConditionFalse, acc.GetGeneration())
				return err
			}
		}

		globalRule, err := c.controller.translator.TranslateClusterConfigV2beta3(acc)
		if err != nil {
			log.Errorw("failed to translate ApisixClusterConfig",
				zap.Error(err),
				zap.String("key", key),
				zap.Any("object", acc),
			)
			c.controller.recorderEvent(acc, corev1.EventTypeWarning, _resourceSyncAborted, err)
			c.controller.recordStatus(acc, _resourceSyncAborted, err, metav1.ConditionFalse, acc.GetGeneration())
			return err
		}
		log.Debugw("translated global_rule",
			zap.Any("object", globalRule),
		)

		// TODO multiple cluster support
		if ev.Type == types.EventAdd {
			_, err = c.controller.apisix.Cluster(acc.Name).GlobalRule().Create(ctx, globalRule)
		} else {
			_, err = c.controller.apisix.Cluster(acc.Name).GlobalRule().Update(ctx, globalRule)
		}
		if err != nil {
			log.Errorw("failed to reflect global_rule changes to apisix cluster",
				zap.Any("global_rule", globalRule),
				zap.Any("cluster", acc.Name),
			)
			c.controller.recorderEvent(acc, corev1.EventTypeWarning, _resourceSyncAborted, err)
			c.controller.recordStatus(acc, _resourceSyncAborted, err, metav1.ConditionFalse, acc.GetGeneration())
			return err
		}
		c.controller.recorderEvent(acc, corev1.EventTypeNormal, _resourceSynced, nil)
		c.controller.recordStatus(acc, _resourceSynced, nil, metav1.ConditionTrue, acc.GetGeneration())
		return nil
	case config.ApisixV2:
		acc := multiVersioned.V2()
		// Currently we don't handle multiple cluster, so only process
		// the default apisix cluster.
		if acc.Name != c.controller.cfg.APISIX.DefaultClusterName {
			log.Infow("ignore non-default apisix cluster config",
				zap.String("default_cluster_name", c.controller.cfg.APISIX.DefaultClusterName),
				zap.Any("ApisixClusterConfig", acc),
			)
			return nil
		}
		// Cluster delete is dangerous.
		// TODO handle delete?
		if ev.Type == types.EventDelete {
			log.Error("ApisixClusterConfig delete event for default apisix cluster will be ignored")
			return nil
		}

		if acc.Spec.Admin != nil {
			clusterOpts := &apisix.ClusterOptions{
				Name:     acc.Name,
				BaseURL:  acc.Spec.Admin.BaseURL,
				AdminKey: acc.Spec.Admin.AdminKey,
			}
			log.Infow("updating cluster",
				zap.Any("opts", clusterOpts),
			)
			// TODO we may first call AddCluster.
			// Since now we already have the default cluster, we just call UpdateCluster.
			if err := c.controller.apisix.UpdateCluster(ctx, clusterOpts); err != nil {
				log.Errorw("failed to update cluster",
					zap.String("cluster_name", acc.Name),
					zap.Error(err),
					zap.Any("opts", clusterOpts),
				)
				c.controller.recorderEvent(acc, corev1.EventTypeWarning, _resourceSyncAborted, err)
				c.controller.recordStatus(acc, _resourceSyncAborted, err, metav1.ConditionFalse, acc.GetGeneration())
				return err
			}
		}

		globalRule, err := c.controller.translator.TranslateClusterConfigV2(acc)
		if err != nil {
			log.Errorw("failed to translate ApisixClusterConfig",
				zap.Error(err),
				zap.String("key", key),
				zap.Any("object", acc),
			)
			c.controller.recorderEvent(acc, corev1.EventTypeWarning, _resourceSyncAborted, err)
			c.controller.recordStatus(acc, _resourceSyncAborted, err, metav1.ConditionFalse, acc.GetGeneration())
			return err
		}
		log.Debugw("translated global_rule",
			zap.Any("object", globalRule),
		)

		// TODO multiple cluster support
		if ev.Type == types.EventAdd {
			_, err = c.controller.apisix.Cluster(acc.Name).GlobalRule().Create(ctx, globalRule)
		} else {
			_, err = c.controller.apisix.Cluster(acc.Name).GlobalRule().Update(ctx, globalRule)
		}
		if err != nil {
			log.Errorw("failed to reflect global_rule changes to apisix cluster",
				zap.Any("global_rule", globalRule),
				zap.Any("cluster", acc.Name),
			)
			c.controller.recorderEvent(acc, corev1.EventTypeWarning, _resourceSyncAborted, err)
			c.controller.recordStatus(acc, _resourceSyncAborted, err, metav1.ConditionFalse, acc.GetGeneration())
			return err
		}
		c.controller.recorderEvent(acc, corev1.EventTypeNormal, _resourceSynced, nil)
		c.controller.recordStatus(acc, _resourceSynced, nil, metav1.ConditionTrue, acc.GetGeneration())
		return nil
	default:
		return fmt.Errorf("unsupported ApisixClusterConfig group version %s", event.GroupVersion)
	}
}

func (c *apisixClusterConfigController) handleSyncErr(obj interface{}, err error) {
	if err == nil {
		c.workqueue.Forget(obj)
		c.controller.MetricsCollector.IncrSyncOperation("clusterConfig", "success")
		return
	}
	event := obj.(*types.Event)
	if k8serrors.IsNotFound(err) && event.Type != types.EventDelete {
		log.Infow("sync ApisixClusterConfig but not found, ignore",
			zap.String("event_type", event.Type.String()),
			zap.String("ApisixClusterConfig", event.Object.(string)),
		)
		c.workqueue.Forget(event)
		return
	}
	log.Warnw("sync ApisixClusterConfig failed, will retry",
		zap.Any("object", obj),
		zap.Error(err),
	)

	c.workqueue.AddRateLimited(obj)
	c.controller.MetricsCollector.IncrSyncOperation("clusterConfig", "failure")
}

func (c *apisixClusterConfigController) onAdd(obj interface{}) {
	acc, err := kube.NewApisixClusterConfig(obj)
	if err != nil {
		log.Errorw("found ApisixClusterConfig resource with bad type", zap.Error(err))
		return
	}
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found ApisixClusterConfig resource with bad meta key: %s", err.Error())
		return
	}
	log.Debugw("ApisixClusterConfig add event arrived",
		zap.String("key", key),
		zap.Any("object", obj),
	)

	c.workqueue.Add(&types.Event{
		Type: types.EventAdd,
		Object: kube.ApisixClusterConfigEvent{
			Key:          key,
			GroupVersion: acc.GroupVersion(),
		},
	})

	c.controller.MetricsCollector.IncrEvents("clusterConfig", "add")
}

func (c *apisixClusterConfigController) onUpdate(oldObj, newObj interface{}) {
	prev, err := kube.NewApisixClusterConfig(oldObj)
	if err != nil {
		log.Errorw("found ApisixClusterConfig resource with bad type", zap.Error(err))
		return
	}
	curr, err := kube.NewApisixClusterConfig(newObj)
	if err != nil {
		log.Errorw("found ApisixClusterConfig resource with bad type", zap.Error(err))
		return
	}
	if prev.ResourceVersion() >= curr.ResourceVersion() {
		return
	}
	key, err := cache.MetaNamespaceKeyFunc(newObj)
	if err != nil {
		log.Errorf("found ApisixClusterConfig with bad meta key: %s", err)
		return
	}
	log.Debugw("ApisixClusterConfig update event arrived",
		zap.Any("new object", curr),
		zap.Any("old object", prev),
	)

	c.workqueue.Add(&types.Event{
		Type: types.EventUpdate,
		Object: kube.ApisixClusterConfigEvent{
			Key:          key,
			OldObject:    prev,
			GroupVersion: curr.GroupVersion(),
		},
	})

	c.controller.MetricsCollector.IncrEvents("clusterConfig", "update")
}

func (c *apisixClusterConfigController) onDelete(obj interface{}) {
	acc, err := kube.NewApisixClusterConfig(obj)
	if err != nil {
		tombstone, ok := obj.(*cache.DeletedFinalStateUnknown)
		if !ok {
			return
		}
		acc, err = kube.NewApisixClusterConfig(tombstone)
		if err != nil {
			log.Errorw("found ApisixClusterConfig resource with bad type", zap.Error(err))
			return
		}
	}

	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found ApisixClusterConfig resource with bad meta key: %s", err)
		return
	}
	log.Debugw("ApisixClusterConfig delete event arrived",
		zap.Any("final state", acc),
	)
	c.workqueue.Add(&types.Event{
		Type: types.EventDelete,
		Object: kube.ApisixClusterConfigEvent{
			Key:          key,
			GroupVersion: acc.GroupVersion(),
		},
		Tombstone: acc,
	})

	c.controller.MetricsCollector.IncrEvents("clusterConfig", "delete")
}

func (c *apisixClusterConfigController) ResourceSync() {
	objs := c.controller.apisixClusterConfigInformer.GetIndexer().List()
	for _, obj := range objs {
		key, err := cache.MetaNamespaceKeyFunc(obj)
		if err != nil {
			log.Errorw("ApisixClusterConfig sync failed, found ApisixClusterConfig resource with bad meta namespace key", zap.String("error", err.Error()))
			continue
		}
		if !c.controller.isWatchingNamespace(key) {
			continue
		}
		acc, err := kube.NewApisixClusterConfig(obj)
		if err != nil {
			log.Errorw("found ApisixClusterConfig resource with bad type", zap.String("error", err.Error()))
			return
		}
		c.workqueue.Add(&types.Event{
			Type: types.EventAdd,
			Object: kube.ApisixClusterConfigEvent{
				Key:          key,
				GroupVersion: acc.GroupVersion(),
			},
		})
	}
}
