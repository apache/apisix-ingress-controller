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

	"go.uber.org/zap"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	apisixcache "github.com/apache/apisix-ingress-controller/pkg/apisix/cache"
	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/kube"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/types"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

const (
	_ingressKey = "kubernetes.io/ingress.class"
)

type ingressController struct {
	controller *Controller
	workqueue  workqueue.RateLimitingInterface
	workers    int
}

func (c *Controller) newIngressController() *ingressController {
	ctl := &ingressController{
		controller: c,
		workqueue:  workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ingress"),
		workers:    1,
	}

	c.ingressInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    ctl.onAdd,
		UpdateFunc: ctl.onUpdate,
		DeleteFunc: ctl.OnDelete,
	})
	return ctl
}

func (c *ingressController) run(ctx context.Context) {
	log.Info("ingress controller started")
	defer log.Infof("ingress controller exited")

	if !cache.WaitForCacheSync(ctx.Done(), c.controller.ingressInformer.HasSynced) {
		log.Errorf("cache sync failed")
		return
	}
	for i := 0; i < c.workers; i++ {
		go c.runWorker(ctx)
	}
	<-ctx.Done()
	c.workqueue.ShutDown()
}

func (c *ingressController) runWorker(ctx context.Context) {
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

func (c *ingressController) sync(ctx context.Context, ev *types.Event) error {
	ingEv := ev.Object.(kube.IngressEvent)
	namespace, name, err := cache.SplitMetaNamespaceKey(ingEv.Key)
	if err != nil {
		log.Errorf("found ingress resource with invalid meta namespace key %s: %s", ingEv.Key, err)
	}

	var ing kube.Ingress
	if ingEv.GroupVersion == kube.IngressV1 {
		ing, err = c.controller.ingressLister.V1(namespace, name)
	} else {
		ing, err = c.controller.ingressLister.V1beta1(namespace, name)
	}

	if err != nil {
		if !k8serrors.IsNotFound(err) {
			log.Errorf("failed to get ingress %s (group version: %s): %s", ingEv.Key, ingEv.GroupVersion, err)
			return err
		}

		if ev.Type != types.EventDelete {
			log.Warnf("ingress %s (group version: %s) was deleted before it can be delivered", ingEv.Key, ingEv.GroupVersion)
			// Don't need to retry.
			return nil
		}
	}
	if ev.Type == types.EventDelete {
		if ing != nil {
			// We still find the resource while we are processing the DELETE event,
			// that means object with same namespace and name was created, discarding
			// this stale DELETE event.
			log.Warnf("discard the stale ingress delete event since the %s exists", ingEv.Key)
			return nil
		}
		ing = ev.Tombstone.(kube.Ingress)
	}

	// Translator should generate ID, fullname and name.
	routes, upstreams, err := c.controller.translator.TranslateIngress(ing)
	if err != nil {
		log.Errorw("failed to translate ingress",
			zap.Error(err),
			zap.Any("ingress", ing),
		)
		return err
	}

	log.Debugw("translated ingress resource to a couple of routes and upstreams",
		zap.Any("ingress", ing),
		zap.Any("routes", routes),
		zap.Any("upstreams", upstreams),
	)

	if err := c.syncToCluster(ctx, "", routes, upstreams, ev.Type); err != nil {
		log.Errorw("failed to sync ingress artifacts (routes, upstreams)",
			zap.Error(err),
			zap.Any("routes", routes),
			zap.Any("upstreams", upstreams),
			zap.Any("ingress", ing),
		)
		return err
	}
	return nil
}

func (c *ingressController) syncToCluster(ctx context.Context, clusterName string, routes []*apisixv1.Route, upstreams []*apisixv1.Upstream, ev types.EventType) error {
	// TODO throttle if the number of routes and upstreams are huge.
	if ev == types.EventDelete {
		for _, r := range routes {
			if err := c.controller.apisix.Cluster(clusterName).Route().Delete(ctx, r); err != nil {
				return err
			}
		}
		for _, u := range upstreams {
			if err := c.controller.apisix.Cluster(clusterName).Upstream().Delete(ctx, u); err != nil {
				return err
			}
		}
		return nil
	}
	for _, u := range upstreams {
		old, err := c.controller.apisix.Cluster(clusterName).Upstream().Get(ctx, u.FullName)
		if err == nil && err != apisixcache.ErrNotFound {
			return err
		}
		if old == nil {
			_, err = c.controller.apisix.Cluster(clusterName).Upstream().Create(ctx, u)
		} else {
			// TODO diff route.
			_, err = c.controller.apisix.Cluster(clusterName).Upstream().Update(ctx, u)
		}
		if err != nil {
			return err
		}
	}
	for _, r := range routes {
		old, err := c.controller.apisix.Cluster(clusterName).Route().Get(ctx, r.FullName)
		if err == nil && err != apisixcache.ErrNotFound {
			return err
		}
		if old == nil {
			_, err = c.controller.apisix.Cluster(clusterName).Route().Create(ctx, r)
		} else {
			// TODO diff route.
			_, err = c.controller.apisix.Cluster(clusterName).Route().Update(ctx, r)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *ingressController) handleSyncErr(obj interface{}, err error) {
	if err == nil {
		c.workqueue.Forget(obj)
		return
	}
	if c.workqueue.NumRequeues(obj) < _maxRetries {
		log.Infow("sync ingress failed, will retry",
			zap.Any("object", obj),
		)
		c.workqueue.AddRateLimited(obj)
	} else {
		c.workqueue.Forget(obj)
		log.Warnf("drop ingress %+v out of the queue", obj)
	}
}

func (c *ingressController) onAdd(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found ingress resource with bad meta namesapce key: %s", err)
		return
	}
	if !c.controller.namespaceWatching(key) {
		return
	}

	ing := kube.MustNewIngress(obj)
	valid := c.isIngressEffective(ing)
	if valid {
		log.Debugw("ingress add event arrived",
			zap.Any("object", obj),
		)
	} else {
		log.Debugw("ignore noneffective ingress add event",
			zap.Any("object", obj),
		)
		return
	}

	c.workqueue.AddRateLimited(&types.Event{
		Type: types.EventAdd,
		Object: kube.IngressEvent{
			Key:          key,
			GroupVersion: ing.GroupVersion(),
		},
	})
}

func (c *ingressController) onUpdate(oldObj, newObj interface{}) {
	prev := kube.MustNewIngress(oldObj)
	curr := kube.MustNewIngress(newObj)
	if prev.ResourceVersion() >= curr.ResourceVersion() {
		return
	}

	key, err := cache.MetaNamespaceKeyFunc(newObj)
	if err != nil {
		log.Errorf("found ingress resource with bad meta namespace key: %s", err)
		return
	}
	valid := c.isIngressEffective(curr)
	if valid {
		log.Debugw("ingress update event arrived",
			zap.Any("new object", oldObj),
			zap.Any("old object", newObj),
		)
	} else {
		log.Debugw("ignore noneffective ingress update event",
			zap.Any("new object", oldObj),
			zap.Any("old object", newObj),
		)
		return
	}

	c.workqueue.AddRateLimited(&types.Event{
		Type: types.EventUpdate,
		Object: kube.IngressEvent{
			Key:          key,
			GroupVersion: curr.GroupVersion(),
		},
	})
}

func (c *ingressController) OnDelete(obj interface{}) {
	ing, err := kube.NewIngress(obj)
	if err != nil {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			return
		}
		ing = kube.MustNewIngress(tombstone)
	}

	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found ingress resource with bad meta namespace key: %s", err)
		return
	}
	if !c.controller.namespaceWatching(key) {
		return
	}
	valid := c.isIngressEffective(ing)
	if valid {
		log.Debugw("ingress delete event arrived",
			zap.Any("final state", ing),
		)
	} else {
		log.Debugw("ignore noneffective ingress delete event",
			zap.Any("object", ing),
		)
		return
	}
	c.workqueue.AddRateLimited(&types.Event{
		Type: types.EventDelete,
		Object: kube.IngressEvent{
			Key:          key,
			GroupVersion: ing.GroupVersion(),
		},
		Tombstone: ing,
	})
}

func (c *ingressController) isIngressEffective(ing kube.Ingress) bool {
	var ingressClass string
	if ing.GroupVersion() == kube.IngressV1 {
		ic, ok := ing.V1().GetAnnotations()[_ingressKey]
		if !ok && ing.V1().Spec.IngressClassName != nil {
			ic = *ing.V1().Spec.IngressClassName
		}
		ingressClass = ic
	} else {
		ingressClass = ing.V1beta1().GetAnnotations()[_ingressKey]
	}

	return ingressClass == config.IngressClass
}
