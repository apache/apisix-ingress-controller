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
	"fmt"
	"time"

	"go.uber.org/zap"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/apache/apisix-ingress-controller/pkg/kube"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/types"
)

const (
	_knativeIngressClassKey = "networking.knative.dev/ingress.class"
)

type knativeIngressController struct {
	controller *Controller
	workqueue  workqueue.RateLimitingInterface
	workers    int
}

func (c *Controller) newKnativeIngressController() *knativeIngressController {
	ctl := &knativeIngressController{
		controller: c,
		workqueue:  workqueue.NewNamedRateLimitingQueue(workqueue.NewItemFastSlowRateLimiter(1*time.Second, 60*time.Second, 5), "KnativeIngress"),
		workers:    1,
	}

	c.knativeIngressInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    ctl.onAdd,
		UpdateFunc: ctl.onUpdate,
		DeleteFunc: ctl.OnDelete,
	})
	return ctl
}

func (c *knativeIngressController) run(ctx context.Context) {
	log.Info("knative ingress controller started")
	defer log.Infof("knative ingress controller exited")
	defer c.workqueue.ShutDown()

	if !cache.WaitForCacheSync(ctx.Done(), c.controller.knativeIngressInformer.HasSynced) {
		log.Errorf("cache sync failed")
		return
	}
	for i := 0; i < c.workers; i++ {
		go c.runWorker(ctx)
	}
	<-ctx.Done()
}

func (c *knativeIngressController) runWorker(ctx context.Context) {
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

func (c *knativeIngressController) sync(ctx context.Context, ev *types.Event) error {
	ingEv := ev.Object.(kube.KnativeIngressEvent)
	namespace, name, err := cache.SplitMetaNamespaceKey(ingEv.Key)
	if err != nil {
		log.Errorf("found knative ingress resource with invalid meta namespace key %s: %s", ingEv.Key, err)
		return err
	}

	var ing kube.KnativeIngress
	switch ingEv.GroupVersion {
	case kube.KnativeIngressV1alpha1:
		ing, err = c.controller.knativeIngressLister.V1alpha1(namespace, name)
	default:
		err = fmt.Errorf("unsupported group version %s, one of (%s) is expected", ingEv.GroupVersion,
			kube.KnativeIngressV1alpha1)
	}

	if err != nil {
		if !k8serrors.IsNotFound(err) {
			log.Errorf("failed to get knative ingress %s (group version: %s): %s", ingEv.Key, ingEv.GroupVersion, err)
			return err
		}

		if ev.Type != types.EventDelete {
			log.Warnf("knative ingress %s (group version: %s) was deleted before it can be delivered", ingEv.Key, ingEv.GroupVersion)
			// Don't need to retry.
			return nil
		}
	}
	if ev.Type == types.EventDelete {
		if ing != nil {
			// We still find the resource while we are processing the DELETE event,
			// that means object with same namespace and name was created, discarding
			// this stale DELETE event.
			log.Warnf("discard the stale knative ingress delete event since the %s exists", ingEv.Key)
			return nil
		}
		ing = ev.Tombstone.(kube.KnativeIngress)
	}

	tctx, err := c.controller.translator.TranslateKnativeIngress(ing)
	if err != nil {
		log.Errorw("failed to translate knative ingress",
			zap.Error(err),
			zap.Any("knative ingress", ing),
		)
		return err
	}

	log.Debugw("translated knative ingress resource to a couple of routes and upstreams",
		zap.Any("knative ingress", ing),
		zap.Any("routes", tctx.Routes),
		zap.Any("upstreams", tctx.Upstreams),
	)

	m := &manifest{
		routes:    tctx.Routes,
		upstreams: tctx.Upstreams,
	}

	var (
		added   *manifest
		updated *manifest
		deleted *manifest
	)

	if ev.Type == types.EventDelete {
		deleted = m
	} else if ev.Type == types.EventAdd {
		added = m
	} else {
		oldCtx, err := c.controller.translator.TranslateKnativeIngress(ingEv.OldObject)
		if err != nil {
			log.Errorw("failed to translate knative ingress",
				zap.String("event", "update"),
				zap.Error(err),
				zap.Any("knative ingress", ingEv.OldObject),
			)
			return err
		}
		om := &manifest{
			routes:    oldCtx.Routes,
			upstreams: oldCtx.Upstreams,
		}
		added, updated, deleted = m.diff(om)
	}
	if err := c.controller.syncManifests(ctx, added, updated, deleted); err != nil {
		log.Errorw("failed to sync knative ingress artifacts",
			zap.Error(err),
		)
		return err
	}
	return nil
}

func (c *knativeIngressController) handleSyncErr(obj interface{}, err error) {
	if err == nil {
		c.workqueue.Forget(obj)
		return
	}
	log.Warnw("sync knative ingress failed, will retry",
		zap.Any("object", obj),
		zap.Error(err),
	)
	c.workqueue.AddRateLimited(obj)
}

func (c *knativeIngressController) onAdd(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found knative ingress resource with bad meta namespace key: %s", err)
		return
	}
	if !c.controller.namespaceWatching(key) {
		return
	}

	ing := kube.MustNewKnativeIngress(obj)
	valid := c.isKnativeIngressEffective(ing)
	if valid {
		log.Debugw("knative ingress add event arrived",
			zap.Any("object", ing),
		)
	} else {
		log.Debugw("ignore noneffective knative ingress add event",
			zap.Any("object", ing),
		)
		return
	}

	c.workqueue.AddRateLimited(&types.Event{
		Type: types.EventAdd,
		Object: kube.KnativeIngressEvent{
			Key:          key,
			GroupVersion: ing.GroupVersion(),
		},
	})
}

func (c *knativeIngressController) onUpdate(oldObj, newObj interface{}) {
	prev := kube.MustNewKnativeIngress(oldObj)
	curr := kube.MustNewKnativeIngress(newObj)
	if prev.ResourceVersion() >= curr.ResourceVersion() {
		return
	}

	key, err := cache.MetaNamespaceKeyFunc(newObj)
	if err != nil {
		log.Errorf("found knative ingress resource with bad meta namespace key: %s", err)
		return
	}
	valid := c.isKnativeIngressEffective(curr)
	if valid {
		log.Debugw("knative ingress update event arrived",
			zap.Any("new object", oldObj),
			zap.Any("old object", newObj),
		)
	} else {
		log.Debugw("ignore noneffective knative  ingress update event",
			zap.Any("new object", oldObj),
			zap.Any("old object", newObj),
		)
		return
	}

	c.workqueue.AddRateLimited(&types.Event{
		Type: types.EventUpdate,
		Object: kube.KnativeIngressEvent{
			Key:          key,
			GroupVersion: curr.GroupVersion(),
			OldObject:    prev,
		},
	})
}

func (c *knativeIngressController) OnDelete(obj interface{}) {
	ing, err := kube.NewKnativeIngress(obj)
	if err != nil {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			return
		}
		ing = kube.MustNewKnativeIngress(tombstone)
	}

	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found knative ingress resource with bad meta namespace key: %s", err)
		return
	}
	if !c.controller.namespaceWatching(key) {
		return
	}
	valid := c.isKnativeIngressEffective(ing)
	if valid {
		log.Debugw("knative ingress delete event arrived",
			zap.Any("final state", ing),
		)
	} else {
		log.Debugw("ignore noneffective knative ingress delete event",
			zap.Any("object", ing),
		)
		return
	}
	c.workqueue.AddRateLimited(&types.Event{
		Type: types.EventDelete,
		Object: kube.KnativeIngressEvent{
			Key:          key,
			GroupVersion: ing.GroupVersion(),
		},
		Tombstone: ing,
	})
}

func (c *knativeIngressController) isKnativeIngressEffective(ing kube.KnativeIngress) bool {
	var ica string
	if ing.GroupVersion() == kube.KnativeIngressV1alpha1 {
		ica = ing.V1alpha1().GetAnnotations()[_knativeIngressClassKey]
	}
	if ica != "" {
		return ica == c.controller.cfg.Kubernetes.IngressClass
	}
	return false
}
