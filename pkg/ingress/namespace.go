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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/types"
)

type namespaceController struct {
	controller *Controller
	workqueue  workqueue.RateLimitingInterface
	workers    int
}

func (c *Controller) newNamespaceController() *namespaceController {
	ctl := &namespaceController{
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

func (c *Controller) initWatchingNamespaceByLabels(ctx context.Context) error {
	labelSelector := metav1.LabelSelector{MatchLabels: c.watchingLabels}
	opts := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}
	namespaces, err := c.kubeClient.Client.CoreV1().Namespaces().List(ctx, opts)
	if err != nil {
		return err
	} else {
		for _, ns := range namespaces.Items {
			c.watchingNamespace.Store(ns.Name, struct{}{})
		}
	}
	return nil
}

func (c *namespaceController) run(ctx context.Context) {
	log.Info("namespace controller started")
	defer log.Info("namespace controller exited")

	if ok := cache.WaitForCacheSync(ctx.Done(), c.controller.namespaceInformer.HasSynced); !ok {
		log.Error("informers sync failed")
		return
	}
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
		namespace, err := c.controller.kubeClient.Client.CoreV1().Namespaces().Get(ctx, ev.Object.(string), metav1.GetOptions{})
		if err != nil {
			return err
		} else {
			// if labels of namespace contains the watchingLabels, the namespace should be set to controller.watchingNamespace
			if c.controller.watchingLabels.IsSubsetOf(namespace.Labels) {
				c.controller.watchingNamespace.Store(namespace.Name, struct{}{})
			}
		}
	} else { // type == types.EventDelete
		namespace := ev.Tombstone.(*corev1.Namespace)
		if _, ok := c.controller.watchingNamespace.Load(namespace.Name); ok {
			c.controller.watchingNamespace.Delete(namespace.Name)
			// need to compare
			err := c.controller.CompareResources(ctx)
			if err != nil {
				return err
			}
		}
		// do nothing, if the namespace did not in controller.watchingNamespace
	}
	return nil
}

func (c *namespaceController) handleSyncErr(event *types.Event, err error) {
	name := event.Object.(string)
	if err != nil {
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
	if err == nil {
		log.Debugw(key)
	}
	c.workqueue.AddRateLimited(&types.Event{
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
		log.Errorf("found Namespace resource with error: %s", err)
		return
	}
	c.workqueue.AddRateLimited(&types.Event{
		Type:   types.EventUpdate,
		Object: key,
	})
}

func (c *namespaceController) onDelete(obj interface{}) {
	namespace := obj.(*corev1.Namespace)
	c.workqueue.AddRateLimited(&types.Event{
		Type:      types.EventDelete,
		Object:    namespace.Name,
		Tombstone: namespace,
	})
}
