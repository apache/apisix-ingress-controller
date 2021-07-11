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
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/types"
)

const (
	_endpointSlicesManagedBy = "endpointslice-controller.k8s.io"
)

type endpointSliceEvent struct {
	Key         string
	ServiceName string
}

type endpointSliceController struct {
	controller *Controller
	workqueue  workqueue.RateLimitingInterface
	workers    int
}

func (c *Controller) newEndpointSliceController() *endpointSliceController {
	ctl := &endpointSliceController{
		controller: c,
		workqueue:  workqueue.NewNamedRateLimitingQueue(workqueue.NewItemFastSlowRateLimiter(time.Second, 60*time.Second, 5), "endpointSlice"),
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

func (c *endpointSliceController) run(ctx context.Context) {
	log.Info("endpointSlice controller started")
	defer log.Info("endpointSlice controller exited")
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

func (c *endpointSliceController) sync(ctx context.Context, ev *types.Event) error {
	epEvent := ev.Object.(endpointSliceEvent)
	namespace, _, err := cache.SplitMetaNamespaceKey(epEvent.Key)
	if err != nil {
		log.Errorf("found endpointSlice object with bad namespace/name: %s, ignore it", epEvent.Key)
		return nil
	}
	ep, err := c.controller.epLister.GetEndpointSlices(namespace, epEvent.ServiceName)
	if err != nil {
		log.Errorf("failed to get all endpointSlices for service %s: %s",
			epEvent.ServiceName, err)
		return err
	}
	return c.controller.syncEndpoint(ctx, ep)
}

func (c *endpointSliceController) handleSyncErr(obj interface{}, err error) {
	if err == nil {
		c.workqueue.Forget(obj)
		return
	}
	log.Warnw("sync endpointSlice failed, will retry",
		zap.Any("object", obj),
	)
	c.workqueue.AddRateLimited(obj)
}

func (c *endpointSliceController) onAdd(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found endpointSlice object with bad namespace")
	}
	if !c.controller.namespaceWatching(key) {
		return
	}
	ep := obj.(*discoveryv1.EndpointSlice)
	svcName := ep.Labels[discoveryv1.LabelServiceName]
	if svcName == "" {
		return
	}
	if ep.Labels[discoveryv1.LabelManagedBy] != _endpointSlicesManagedBy {
		// We only care about endpointSlice objects managed by the EndpointSlices
		// controller.
		return
	}

	log.Debugw("endpointSlice add event arrived",
		zap.String("object-key", key),
	)

	c.workqueue.AddRateLimited(&types.Event{
		Type: types.EventAdd,
		Object: endpointSliceEvent{
			Key:         key,
			ServiceName: svcName,
		},
	})
}

func (c *endpointSliceController) onUpdate(prev, curr interface{}) {
	prevEp := prev.(*discoveryv1.EndpointSlice)
	currEp := curr.(*discoveryv1.EndpointSlice)

	if prevEp.GetResourceVersion() == currEp.GetResourceVersion() {
		return
	}
	key, err := cache.MetaNamespaceKeyFunc(currEp)
	if err != nil {
		log.Errorf("found endpointSlice object with bad namespace/name: %s, ignore it", err)
		return
	}
	if !c.controller.namespaceWatching(key) {
		return
	}
	if currEp.Labels[discoveryv1.LabelManagedBy] != _endpointSlicesManagedBy {
		// We only care about endpointSlice objects managed by the EndpointSlices
		// controller.
		return
	}
	svcName := currEp.Labels[discoveryv1.LabelServiceName]
	if svcName == "" {
		return
	}

	log.Debugw("endpointSlice update event arrived",
		zap.Any("new object", currEp),
		zap.Any("old object", prevEp),
	)
	c.workqueue.AddRateLimited(&types.Event{
		Type: types.EventUpdate,
		// TODO pass key.
		Object: endpointSliceEvent{
			Key:         key,
			ServiceName: svcName,
		},
	})
}

func (c *endpointSliceController) onDelete(obj interface{}) {
	ep, ok := obj.(*discoveryv1.EndpointSlice)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			log.Errorf("found endpoints: %+v in bad tombstone state", obj)
			return
		}
		ep = tombstone.Obj.(*discoveryv1.EndpointSlice)
	}
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found endpointSlice object with bad namespace/name: %s, ignore it", err)
		return
	}
	if !c.controller.namespaceWatching(key) {
		return
	}
	if ep.Labels[discoveryv1.LabelManagedBy] != _endpointSlicesManagedBy {
		// We only care about endpointSlice objects managed by the EndpointSlices
		// controller.
		return
	}
	svcName := ep.Labels[discoveryv1.LabelServiceName]
	log.Debugw("endpoints delete event arrived",
		zap.Any("object-key", key),
	)
	c.workqueue.AddRateLimited(&types.Event{
		Type: types.EventDelete,
		Object: endpointSliceEvent{
			Key:         key,
			ServiceName: svcName,
		},
	})
}
