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
package endpoint

import (
	"context"
	"time"

	"go.uber.org/zap"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/apache/apisix-ingress-controller/pkg/kube"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/providers/k8s/namespace"
	"github.com/apache/apisix-ingress-controller/pkg/types"
)

const (
	_endpointSlicesManagedBy = "endpointslice-controller.k8s.io"
)

type endpointSliceController struct {
	*baseEndpointController

	workqueue workqueue.RateLimitingInterface
	workers   int

	namespaceProvider namespace.WatchingNamespaceProvider

	epInformer cache.SharedIndexInformer
	epLister   kube.EndpointLister
}

func newEndpointSliceController(base *baseEndpointController, namespaceProvider namespace.WatchingNamespaceProvider) *endpointSliceController {
	c := &endpointSliceController{
		baseEndpointController: base,

		workqueue: workqueue.NewNamedRateLimitingQueue(workqueue.NewItemFastSlowRateLimiter(time.Second, 60*time.Second, 5), "endpointSlice"),
		workers:   1,

		namespaceProvider: namespaceProvider,

		epLister:   base.EpLister,
		epInformer: base.EpInformer,
	}

	c.epInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.onAdd,
			UpdateFunc: c.onUpdate,
			DeleteFunc: c.onDelete,
		},
	)

	return c
}

func (c *endpointSliceController) run(ctx context.Context) {
	log.Info("endpointSlice controller started")
	defer log.Info("endpointSlice controller exited")
	defer c.workqueue.ShutDown()

	if ok := cache.WaitForCacheSync(ctx.Done(), c.epInformer.HasSynced); !ok {
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
	log.Debugw("process endpoint slice sync event",
		zap.Any("event", ev),
	)
	ep := ev.Object.(kube.Endpoint)
	ns, err := ep.Namespace()
	if err != nil {
		return err
	}

	newestEp, err := c.epLister.GetEndpoint(ns, ep.ServiceName())
	if err != nil {
		if errors.IsNotFound(err) {
			return c.syncEmptyEndpoint(ctx, ep)
		}
		return err
	}
	return c.syncEndpoint(ctx, newestEp)
}

func (c *endpointSliceController) handleSyncErr(obj interface{}, err error) {
	if err == nil {
		c.workqueue.Forget(obj)
		c.MetricsCollector.IncrSyncOperation("endpointSlice", "success")
		return
	}
	event := obj.(*types.Event)
	if k8serrors.IsNotFound(err) && event.Type != types.EventDelete {
		log.Infow("sync endpointSlice but not found, ignore",
			zap.String("event_type", event.Type.String()),
			zap.Any("endpointSlice", event.Object.(kube.Endpoint)),
		)
		c.workqueue.Forget(event)
		return
	}
	log.Warnw("sync endpointSlice failed, will retry",
		zap.Any("object", obj),
	)
	c.workqueue.AddRateLimited(obj)
	c.MetricsCollector.IncrSyncOperation("endpointSlice", "failure")
}

func (c *endpointSliceController) onAdd(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found endpointSlice object with bad namespace")
	}
	if !c.namespaceProvider.IsWatchingNamespace(key) {
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

	c.workqueue.Add(&types.Event{
		Type:   types.EventAdd,
		Object: kube.NewEndpointWithSlice(ep),
	})

	c.MetricsCollector.IncrEvents("endpointSlice", "add")
}

func (c *endpointSliceController) onUpdate(prev, curr interface{}) {
	prevEp := prev.(*discoveryv1.EndpointSlice)
	currEp := curr.(*discoveryv1.EndpointSlice)

	if prevEp.GetResourceVersion() >= currEp.GetResourceVersion() {
		return
	}
	key, err := cache.MetaNamespaceKeyFunc(currEp)
	if err != nil {
		log.Errorf("found endpointSlice object with bad namespace/name: %s, ignore it", err)
		return
	}
	if !c.namespaceProvider.IsWatchingNamespace(key) {
		return
	}
	svcName := currEp.Labels[discoveryv1.LabelServiceName]
	if svcName == "" {
		return
	}
	if currEp.Labels[discoveryv1.LabelManagedBy] != _endpointSlicesManagedBy {
		// We only care about endpointSlice objects managed by the EndpointSlices
		// controller.
		return
	}

	log.Debugw("endpointSlice update event arrived",
		zap.Any("new object", currEp),
		zap.Any("old object", prevEp),
	)
	c.workqueue.Add(&types.Event{
		Type: types.EventUpdate,
		// TODO pass key.
		Object: kube.NewEndpointWithSlice(currEp),
	})

	c.MetricsCollector.IncrEvents("endpointSlice", "update")
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
	if !c.namespaceProvider.IsWatchingNamespace(key) {
		return
	}
	svcName := ep.Labels[discoveryv1.LabelServiceName]
	if svcName == "" {
		return
	}
	if ep.Labels[discoveryv1.LabelManagedBy] != _endpointSlicesManagedBy {
		// We only care about endpointSlice objects managed by the EndpointSlices
		// controller.
		return
	}
	log.Debugw("endpoints delete event arrived",
		zap.Any("object-key", key),
	)
	c.workqueue.Add(&types.Event{
		Type:   types.EventDelete,
		Object: kube.NewEndpointWithSlice(ep),
	})

	c.MetricsCollector.IncrEvents("endpointSlice", "delete")
}
