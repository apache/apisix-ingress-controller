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
	"errors"
	"fmt"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/informers"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/api7/ingress-controller/pkg/apisix"
	apisixcache "github.com/api7/ingress-controller/pkg/apisix/cache"
	"github.com/api7/ingress-controller/pkg/log"
	"github.com/api7/ingress-controller/pkg/seven/state"
	apisixv1 "github.com/api7/ingress-controller/pkg/types/apisix/v1"
)

const (
	_defaultNodeWeight = 100
)

type endpointsController struct {
	controller *Controller
	informer   cache.SharedIndexInformer
	lister     listerscorev1.EndpointsLister
	workqueue  workqueue.RateLimitingInterface
}

func (c *Controller) newEndpointsController(factory informers.SharedInformerFactory) *endpointsController {
	ctl := &endpointsController{
		controller: c,
		informer:   factory.Core().V1().Endpoints().Informer(),
		lister:     factory.Core().V1().Endpoints().Lister(),
		workqueue:  workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "endpoints"),
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

func (c *endpointsController) run(ctx context.Context) error {
	log.Info("endpoints controller started")
	defer log.Info("endpoints controller exited")

	go func() {
		c.informer.Run(ctx.Done())
	}()

	if ok := cache.WaitForCacheSync(ctx.Done(), c.informer.HasSynced); !ok {
		return errors.New("endpoints informers cache sync failed")
	}

	for {
		obj, shutdown := c.workqueue.Get()
		if shutdown {
			return nil
		}

		var (
			err error
		)

		key, ok := obj.(string)
		if !ok {
			log.Errorf("found endpoints object with unexpected type %T, ignore it", obj)
			c.workqueue.Forget(obj)
		} else {
			err = c.process(ctx, key)
		}

		c.workqueue.Done(obj)

		if err != nil {
			log.Warnf("endpoints %s retried since %s", key, err)
			c.retry(obj)
		}
	}
}

func (c *endpointsController) process(ctx context.Context, key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		log.Errorf("found endpoints objects with malformed namespace/name: %s, ignore it", err)
		return nil
	}

	ep, err := c.lister.Endpoints(namespace).Get(name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Warnf("endpoints %s was removed before it can be processed", key)
			return nil
		}
		log.Errorf("failed to get endpoints %s: %s", key, err)
		return err
	}
	return c.sync(ctx, ep)
}

func (c *endpointsController) sync(ctx context.Context, ep *corev1.Endpoints) error {
	clusters := c.controller.apisix.ListClusters()
	for _, s := range ep.Subsets {
		for _, port := range s.Ports {
			upstream := fmt.Sprintf("%s_%s_%d", ep.Namespace, ep.Name, port.Port)
			for _, cluster := range clusters {
				if err := c.syncToCluster(ctx, upstream, cluster, s.Addresses, int(port.Port)); err != nil {
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

func (c *endpointsController) retry(obj interface{}) {
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
	c.workqueue.AddRateLimited(key)
}

func (c *endpointsController) onUpdate(prev, curr interface{}) {
	prevEp := prev.(*corev1.Endpoints)
	currEp := curr.(*corev1.Endpoints)

	if prevEp.GetResourceVersion() == currEp.GetResourceVersion() {
		return
	}
	c.onAdd(currEp)
}

func (c *endpointsController) onDelete(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("failed to find the final state before deletion: %s", err)
		return
	}
	if !c.controller.namespaceWatching(key) {
		return
	}
	c.workqueue.AddRateLimited(key)
}
