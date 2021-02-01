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
package controller

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/apache/apisix-ingress-controller/pkg/apisix"
	apisixcache "github.com/apache/apisix-ingress-controller/pkg/apisix/cache"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/seven/state"
	"github.com/apache/apisix-ingress-controller/pkg/types"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

const (
	_defaultNodeWeight = 100
	// maxRetries is the number of times an object will be retried before it is dropped out of the queue.
	_maxRetries = 10
)

type endpointsController struct {
	controller *Controller
	informer   cache.SharedIndexInformer
	lister     listerscorev1.EndpointsLister
	workqueue  workqueue.RateLimitingInterface
	workers    int
}

func (c *Controller) newEndpointsController(informer cache.SharedIndexInformer, lister listerscorev1.EndpointsLister) *endpointsController {
	ctl := &endpointsController{
		controller: c,
		informer:   informer,
		lister:     lister,
		workqueue:  workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "endpoints"),
		workers:    1,
	}

	ctl.informer.AddEventHandler(
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

	if ok := cache.WaitForCacheSync(ctx.Done(), c.informer.HasSynced); !ok {
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
	c.workqueue.ShutDown()
}

func (c *endpointsController) sync(ctx context.Context, ev *types.Event) error {
	ep := ev.Object.(*corev1.Endpoints)
	clusters := c.controller.apisix.ListClusters()
	for _, s := range ep.Subsets {
		for _, port := range s.Ports {
			// FIXME this is wrong, we should use the port name as the key.
			upstream := fmt.Sprintf("%s_%s_%d", ep.Namespace, ep.Name, port.Port)
			for _, cluster := range clusters {
				var addresses []corev1.EndpointAddress
				if ev.Type != types.EventDelete {
					addresses = s.Addresses
				}
				if err := c.syncToCluster(ctx, upstream, cluster, addresses, int(port.Port)); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (c *endpointsController) syncToCluster(ctx context.Context, upstreamName string,
	cluster apisix.Cluster, addresses []corev1.EndpointAddress, port int) error {
	upstream, err := cluster.Upstream().Get(ctx, upstreamName)
	if err != nil {
		if err == apisixcache.ErrNotFound {
			log.Warnw("upstream is not referenced",
				zap.String("cluster", cluster.String()),
				zap.String("upstream", upstreamName),
			)
			return nil
		} else {
			log.Errorw("failed to get upstream",
				zap.String("upstream", upstreamName),
				zap.String("cluster", cluster.String()),
				zap.Error(err),
			)
			return err
		}
	}

	nodes := make([]apisixv1.Node, 0, len(addresses))
	for _, address := range addresses {
		nodes = append(nodes, apisixv1.Node{
			IP:     address.IP,
			Port:   port,
			Weight: _defaultNodeWeight,
		})
	}
	log.Debugw("upstream binds new nodes",
		zap.String("upstream", upstreamName),
		zap.Any("nodes", nodes),
	)

	upstream.Nodes = nodes
	upstream.FromKind = WatchFromKind
	upstreams := []*apisixv1.Upstream{upstream}
	comb := state.ApisixCombination{Routes: nil, Services: nil, Upstreams: upstreams}

	if _, err = comb.Solver(); err != nil {
		log.Errorw("failed to sync upstream",
			zap.String("upstream", upstreamName),
			zap.String("cluster", cluster.String()),
			zap.Error(err),
		)
		return err
	}
	return nil
}

func (c *endpointsController) handleSyncErr(obj interface{}, err error) {
	if err == nil {
		c.workqueue.Forget(obj)
		return
	}
	if c.workqueue.NumRequeues(obj) < _maxRetries {
		log.Infow("sync endpoints failed, will retry",
			zap.Any("object", obj),
		)
		c.workqueue.AddRateLimited(obj)
	} else {
		c.workqueue.Forget(obj)
		log.Warnf("drop endpoints %+v out of the queue", obj)
	}
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

	c.workqueue.AddRateLimited(&types.Event{
		Type:   types.EventAdd,
		Object: obj,
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
	c.workqueue.AddRateLimited(&types.Event{
		Type:   types.EventUpdate,
		Object: curr,
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
	c.workqueue.AddRateLimited(&types.Event{
		Type:   types.EventDelete,
		Object: ep,
	})
}
