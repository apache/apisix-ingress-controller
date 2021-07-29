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
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/apache/apisix-ingress-controller/pkg/kube"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/types"
)

type endpointsController struct {
	controller *Controller
	workqueue  workqueue.RateLimitingInterface
	workers    int
}

func (c *Controller) newEndpointsController() *endpointsController {
	ctl := &endpointsController{
		controller: c,
		workqueue:  workqueue.NewNamedRateLimitingQueue(workqueue.NewItemFastSlowRateLimiter(1*time.Second, 60*time.Second, 5), "endpoints"),
		workers:    1,
	}

	ctl.controller.epInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctl.onAdd,
			UpdateFunc: ctl.onUpdate,
			DeleteFunc: ctl.onDelete,
		},
	)

	return ctl
}

func (c *endpointsController) run(ctx context.Context) {
	log.Info("endpoints controller started")
	defer log.Info("endpoints controller exited")
	defer c.workqueue.ShutDown()

	if ok := cache.WaitForCacheSync(ctx.Done(), c.controller.epInformer.HasSynced); !ok {
		log.Error("informers sync failed")
		return
	}

	handler := func() {
		for {
			obj, shutdown := c.workqueue.Get()
			if shutdown {
				return
			}

			err := c.sync(ctx, obj.(*types.Event))
			c.workqueue.Done(obj)
			c.handleSyncErr(obj, err)
		}
	}

	for i := 0; i < c.workers; i++ {
		go handler()
	}

	<-ctx.Done()
}

func (c *endpointsController) sync(ctx context.Context, ev *types.Event) error {
	ep := ev.Object.(kube.Endpoint)
	return c.controller.syncEndpoint(ctx, ep)
}

func (c *endpointsController) handleSyncErr(obj interface{}, err error) {
	if err == nil {
		c.workqueue.Forget(obj)
		return
	}
	log.Warnw("sync endpoints failed, will retry",
		zap.Any("object", obj),
	)
	c.workqueue.AddRateLimited(obj)
}

func (c *endpointsController) onAdd(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found endpoints object with bad namespace/name: %s, ignore it", err)
		return
	}
	if !c.controller.namespaceWatching(key) {
		return
	}
	log.Debugw("endpoints add event arrived",
		zap.String("object-key", key))

	c.workqueue.AddRateLimited(&types.Event{
		Type: types.EventAdd,
		// TODO pass key.
		Object: kube.NewEndpoint(obj.(*corev1.Endpoints)),
	})
}

func (c *endpointsController) onUpdate(prev, curr interface{}) {
	prevEp := prev.(*corev1.Endpoints)
	currEp := curr.(*corev1.Endpoints)

	if prevEp.GetResourceVersion() == currEp.GetResourceVersion() {
		return
	}
	key, err := cache.MetaNamespaceKeyFunc(currEp)
	if err != nil {
		log.Errorf("found endpoints object with bad namespace/name: %s, ignore it", err)
		return
	}
	if !c.controller.namespaceWatching(key) {
		return
	}
	log.Debugw("endpoints update event arrived",
		zap.Any("new object", currEp),
		zap.Any("old object", prevEp),
	)
	c.workqueue.AddRateLimited(&types.Event{
		Type: types.EventUpdate,
		// TODO pass key.
		Object: kube.NewEndpoint(currEp),
	})
}

func (c *endpointsController) onDelete(obj interface{}) {
	ep, ok := obj.(*corev1.Endpoints)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			log.Errorf("found endpoints: %+v in bad tombstone state", obj)
			return
		}
		ep = tombstone.Obj.(*corev1.Endpoints)
	}

	// FIXME Refactor Controller.namespaceWatching to just use
	// namespace after all controllers use the same way to fetch
	// the object.
	if !c.controller.namespaceWatching(ep.Namespace + "/" + ep.Name) {
		return
	}
	log.Debugw("endpoints delete event arrived",
		zap.Any("final state", ep),
	)
	c.workqueue.AddRateLimited(&types.Event{
		Type:   types.EventDelete,
		Object: kube.NewEndpoint(ep),
	})
}
