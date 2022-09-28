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
	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/providers/k8s/configmap/translation"
	providertypes "github.com/apache/apisix-ingress-controller/pkg/providers/types"
	"github.com/apache/apisix-ingress-controller/pkg/providers/utils"
	"github.com/apache/apisix-ingress-controller/pkg/types"
)

// FIXME: Controller should be the Core Part,
// Provider should act as "EventHandler", register there functions to Controller
type EventHandler interface {
	OnAdd()
	OnUpdate()
	OnDelete()
}

type configmapController struct {
	*providertypes.Common
	workqueue workqueue.RateLimitingInterface
	workers   int

	configmapInformer cache.SharedIndexInformer
	configmapLister   v1.ConfigMapLister

	subscriptionList map[string]struct{}
}

func newConfigMapController(common *providertypes.Common) *configmapController {
	ctl := &configmapController{
		configmapInformer: common.ConfigMapInformer,
		configmapLister:   common.ConfigMapLister,

		workqueue: workqueue.NewNamedRateLimitingQueue(workqueue.NewItemFastSlowRateLimiter(1*time.Second, 60*time.Second, 5), "ConfigMap"),
		workers:   1,

		subscriptionList: map[string]struct{}{},

		Common: common,
	}
	ctl.configmapInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctl.onAdd,
			UpdateFunc: ctl.onUpdate,
			DeleteFunc: ctl.onDelete,
		},
	)
	return ctl
}

func (c *configmapController) Subscription(configName string) {
	c.subscriptionList[configName] = struct{}{}
}

func (c *configmapController) IsSubscribing(key string) bool {
	_, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return false
	}
	_, ok := c.subscriptionList[name]
	return ok
}

func (c *configmapController) run(ctx context.Context) {
	if ok := cache.WaitForCacheSync(ctx.Done(), c.configmapInformer.HasSynced); !ok {
		log.Error("namespace informers sync failed")
		return
	}
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
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		log.Errorf("invalid resource key: %s", key)
		return err
	}
	cm, err := c.configmapLister.ConfigMaps(namespace).Get(name)

	log.Debugw("configmap sync event arrived",
		zap.String("event_type", ev.Type.String()),
		zap.Any("configmap", ev.Object),
	)

	pluginMetadatas := translation.TranslateConfigMapToPluginMetadatas(cm)

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
		oldPluginMetadatas := translation.TranslateConfigMapToPluginMetadatas(ev.OldObject.(*corev1.ConfigMap))
		om := &utils.Manifest{
			PluginMetadatas: oldPluginMetadatas,
		}
		added, updated, deleted = m.Diff(om)
	}

	return c.SyncManifests(ctx, added, updated, deleted)
}

func (c *configmapController) handleSyncErr(event *types.Event, err error) {
	name := event.Object.(string)
	if err != nil {
		if k8serrors.IsNotFound(err) && event.Type != types.EventDelete {
			log.Infow("sync configmap but not found, ignore",
				zap.String("event_type", event.Type.String()),
				zap.Any("configmap", event.Object),
			)
			c.workqueue.Forget(event)
			return
		}
		log.Warnw("sync configmap info failed, will retry",
			zap.String("configmap", name),
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
	c.workqueue.Add(&types.Event{
		Type:      types.EventDelete,
		Object:    key,
		Tombstone: obj,
	})
}
