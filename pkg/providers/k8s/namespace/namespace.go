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
package namespace

import (
	"context"
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/types"
)

// FIXME: Controller should be the Core Part,
// Provider should act as "EventHandler", register there functions to Controller
type EventHandler interface {
	OnAdd()
	OnUpdate()
	OnDelete()
}

type namespaceController struct {
	syncCh     chan string
	controller *watchingProvider
	workqueue  workqueue.RateLimitingInterface
	workers    int
}

func newNamespaceController(c *watchingProvider, syncCh chan string) *namespaceController {
	ctl := &namespaceController{
		syncCh:     syncCh,
		controller: c,
		workqueue:  workqueue.NewNamedRateLimitingQueue(workqueue.NewItemFastSlowRateLimiter(1*time.Second, 60*time.Second, 5), "Namespace"),
		workers:    1,
	}
	ctl.controller.namespaceInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctl.onAdd,
			UpdateFunc: ctl.onUpdate,
			DeleteFunc: ctl.onDelete,
		},
	)
	return ctl
}

func (c *namespaceController) run(ctx context.Context) {
	if ok := cache.WaitForCacheSync(ctx.Done(), c.controller.namespaceInformer.HasSynced); !ok {
		log.Error("namespace informers sync failed")
		return
	}
	log.Info("namespace controller started")
	defer log.Info("namespace controller exited")
	for i := 0; i < c.workers; i++ {
		go c.runWorker(ctx)
	}
	<-ctx.Done()
}

func (c *namespaceController) runWorker(ctx context.Context) {
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

func (c *namespaceController) sync(ctx context.Context, ev *types.Event) error {
	if ev.Type != types.EventDelete {
		// check the labels of specify namespace
		namespace, err := c.controller.kube.Client.CoreV1().Namespaces().Get(ctx, ev.Object.(string), metav1.GetOptions{})
		if err != nil {
			return err
		}

		// if labels of namespace contains the watchingLabels, the namespace should be set to controller.watchingNamespaces
		if c.controller.watchingLabels.IsSubsetOf(namespace.Labels) {
			log.Infow("watching namespace", zap.String("name", namespace.Name))
			if _, ok := c.controller.watchingNamespaces.Load(namespace.Name); !ok {
				c.controller.watchingNamespaces.Store(namespace.Name, struct{}{})
				if c.syncCh != nil {
					log.Infof("resync resource in namespace %s", namespace.Name)
					c.syncCh <- namespace.Name
				}
			}
		} else {
			log.Infow("un-watching namespace", zap.String("name", namespace.Name))
			c.controller.watchingNamespaces.Delete(namespace.Name)
		}

	} else { // type == types.EventDelete
		namespace := ev.Tombstone.(*corev1.Namespace)
		if _, ok := c.controller.watchingNamespaces.Load(namespace.Name); ok {
			log.Infow("un-watching namespace", zap.String("name", namespace.Name))
			c.controller.watchingNamespaces.Delete(namespace.Name)
		}
		// do nothing, if the namespace did not in controller.watchingNamespaces
	}
	return nil
}

func (c *namespaceController) handleSyncErr(event *types.Event, err error) {
	name := event.Object.(string)
	if err != nil {
		if k8serrors.IsNotFound(err) && event.Type != types.EventDelete {
			log.Infow("sync namespace but not found, ignore",
				zap.String("event_type", event.Type.String()),
				zap.String("namespace", event.Object.(string)),
			)
			c.workqueue.Forget(event)
			return
		}
		log.Warnw("sync namespace info failed, will retry",
			zap.String("namespace", name),
			zap.Error(err),
		)
		c.workqueue.AddRateLimited(event)
	} else {
		c.workqueue.Forget(event)
	}
}

func (c *namespaceController) onAdd(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found Namespace resource with error: %v", err)
		return
	}
	log.Debugw("namespace add event arrived",
		zap.Any("namespace", obj),
	)
	c.workqueue.Add(&types.Event{
		Type:   types.EventAdd,
		Object: key,
	})
}

func (c *namespaceController) onUpdate(pre, cur interface{}) {
	oldNamespace := pre.(*corev1.Namespace)
	newNamespace := cur.(*corev1.Namespace)
	if oldNamespace.ResourceVersion >= newNamespace.ResourceVersion {
		return
	}
	key, err := cache.MetaNamespaceKeyFunc(cur)
	if err != nil {
		log.Errorf("found Namespace resource with error: %v", err)
		return
	}
	c.workqueue.Add(&types.Event{
		Type:   types.EventUpdate,
		Object: key,
	})
}

func (c *namespaceController) onDelete(obj interface{}) {
	namespace := obj.(*corev1.Namespace)
	c.workqueue.Add(&types.Event{
		Type:      types.EventDelete,
		Object:    namespace.Name,
		Tombstone: namespace,
	})
}
