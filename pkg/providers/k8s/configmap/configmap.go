// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package configmap

import (
	"context"
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/providers/k8s/configmap/translation"
	providertypes "github.com/apache/apisix-ingress-controller/pkg/providers/types"
	"github.com/apache/apisix-ingress-controller/pkg/providers/utils"
	"github.com/apache/apisix-ingress-controller/pkg/types"
)

type subscripKey struct {
	namespace string
	name      string
}

type configmapController struct {
	*providertypes.Common
	workqueue workqueue.RateLimitingInterface
	workers   int

	subscriptionList map[subscripKey]struct{}
}

func newConfigMapController(common *providertypes.Common) *configmapController {
	ctl := &configmapController{

		workqueue: workqueue.NewNamedRateLimitingQueue(workqueue.NewItemFastSlowRateLimiter(1*time.Second, 60*time.Second, 5), "ConfigMap"),
		workers:   1,

		subscriptionList: map[subscripKey]struct{}{},

		Common: common,
	}
	ctl.ConfigMapInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctl.onAdd,
			UpdateFunc: ctl.onUpdate,
			DeleteFunc: ctl.onDelete,
		},
	)
	return ctl
}

func (c *configmapController) Subscribe(namespace, configName string) {
	c.subscriptionList[subscripKey{
		namespace: namespace,
		name:      configName,
	}] = struct{}{}
}

func (c *configmapController) IsSubscribing(key string) bool {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return false
	}
	_, ok := c.subscriptionList[subscripKey{
		namespace: namespace,
		name:      name,
	}]
	return ok
}

func (c *configmapController) run(ctx context.Context) {
	log.Info("configmap controller started")
	defer log.Info("configmap controller exited")
	for i := 0; i < c.workers; i++ {
		go c.runWorker(ctx)
	}
	<-ctx.Done()
}

func (c *configmapController) runWorker(ctx context.Context) {
	for {
		obj, quit := c.workqueue.Get()
		if quit {
			return
		}
		err := c.sync(ctx, obj.(*types.Event))
		c.workqueue.Done(obj)
		c.handleSyncErr(obj.(*types.Event), err)
	}
}

func (c *configmapController) sync(ctx context.Context, ev *types.Event) error {
	key := ev.Object.(string)
	log.Debugw("configmap sync event arrived",
		zap.String("event_type", ev.Type.String()),
		zap.Any("key", ev.Object),
	)

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		log.Errorf("invalid resource key: %s", key)
		return err
	}
	cm, err := c.ConfigMapLister.ConfigMaps(namespace).Get(name)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			log.Errorw("sync failed, unable to get ConfigMap",
				zap.String("key", key),
				zap.Error(err),
			)
			return err
		}
		if ev.Type != types.EventDelete {
			log.Warnw("configmap was deleted before it can be delivered",
				zap.String("key", key),
			)
			return nil
		}
		cm = ev.Tombstone.(*corev1.ConfigMap)
	}

	var (
		configmap    *translation.ConfigMap
		oldConfigmap *translation.ConfigMap
	)

	configmap, err = translation.TranslateConfigMap(cm)
	if err != nil {
		return err
	}
	if ev.Type == types.EventUpdate {
		oldConfigmap, _ = translation.TranslateConfigMap(ev.OldObject.(*corev1.ConfigMap))
	}

	for clusterName, pluginMetadatas := range configmap.ConfigYaml.Data {
		m := &utils.Manifest{
			PluginMetadatas: pluginMetadatas,
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
			if oldConfigmap != nil {
				oldPluginMetadatas := oldConfigmap.ConfigYaml.Data[clusterName]
				om := &utils.Manifest{
					PluginMetadatas: oldPluginMetadatas,
				}
				added, updated, deleted = m.Diff(om)
			}
		}
		log.Debugw("sync ApisixGlobalRule to cluster",
			zap.String("event_type", ev.Type.String()),
			zap.Any("add", added),
			zap.Any("update", updated),
			zap.Any("delete", deleted),
		)
		if err := c.SyncClusterManifests(ctx, clusterName, added, updated, deleted, false); err != nil {
			log.Errorw("sync cluster failed", zap.Error(err))
			return err
		}
	}
	if ev.Type == types.EventUpdate && oldConfigmap != nil {
		if oldConfigmap == nil {
			return nil
		}
		for clusterName, pluginMetadatas := range oldConfigmap.ConfigYaml.Data {
			if _, ok := configmap.ConfigYaml.Data[clusterName]; !ok {

				deleted := &utils.Manifest{
					PluginMetadatas: pluginMetadatas,
				}
				log.Debugw("sync configmap to cluster",
					zap.String("event_type", ev.Type.String()),
					zap.Any("delete", deleted),
				)
				if err := c.SyncClusterManifests(ctx, clusterName, nil, nil, deleted, false); err != nil {
					log.Errorw("sync cluster failed", zap.Error(err))
				}
			}
		}
	}
	return nil
}

func (c *configmapController) handleSyncErr(event *types.Event, err error) {
	key := event.Object.(string)
	if err != nil {
		if k8serrors.IsNotFound(err) && event.Type != types.EventDelete {
			log.Infow("sync configmap but not found, ignore",
				zap.String("event_type", event.Type.String()),
				zap.Any("key", key),
			)
			c.workqueue.Forget(event)
			return
		}
		log.Warnw("sync configmap info failed, will retry",
			zap.String("key", key),
			zap.Error(err),
		)
		c.workqueue.AddRateLimited(event)
	} else {
		c.workqueue.Forget(event)
	}
}

func (c *configmapController) onAdd(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found ConfigMap resource with error: %v", err)
		return
	}
	if !c.IsSubscribing(key) {
		return
	}
	log.Debugw("configmap add event arrived",
		zap.String("key", key),
		zap.Any("object", obj),
	)
	c.workqueue.Add(&types.Event{
		Type:   types.EventAdd,
		Object: key,
	})
}

func (c *configmapController) onUpdate(pre, cur interface{}) {
	old := pre.(*corev1.ConfigMap)
	new := cur.(*corev1.ConfigMap)
	if old.ResourceVersion >= new.ResourceVersion {
		return
	}
	key, err := cache.MetaNamespaceKeyFunc(cur)
	if err != nil {
		log.Errorf("found ConfigMap resource with error: %v", err)
		return
	}
	if !c.IsSubscribing(key) {
		return
	}
	log.Debugw("configmap update event arrived",
		zap.String("key", key),
		zap.Any("object", new),
	)
	c.workqueue.Add(&types.Event{
		Type:      types.EventUpdate,
		Object:    key,
		OldObject: old,
	})
}

func (c *configmapController) onDelete(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found ConfigMap resource with error: %v", err)
		return
	}
	if !c.IsSubscribing(key) {
		return
	}
	log.Debugw("configmap delete event arrived",
		zap.String("key", key),
		zap.Any("object", obj),
	)
	c.workqueue.Add(&types.Event{
		Type:      types.EventDelete,
		Object:    key,
		Tombstone: obj,
	})
}
