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
package gateway

import (
	"context"
	"time"

	"go.uber.org/zap"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/providers/translation"
	"github.com/apache/apisix-ingress-controller/pkg/providers/utils"
	"github.com/apache/apisix-ingress-controller/pkg/types"
)

type gatewayHTTPRouteController struct {
	controller *Provider
	workqueue  workqueue.RateLimitingInterface
	workers    int
}

func newGatewayHTTPRouteController(c *Provider) *gatewayHTTPRouteController {
	ctrl := &gatewayHTTPRouteController{
		controller: c,
		workqueue:  workqueue.NewNamedRateLimitingQueue(workqueue.NewItemFastSlowRateLimiter(1*time.Second, 60*time.Second, 5), "GatewayHTTPRoute"),
		workers:    1,
	}

	ctrl.controller.gatewayHTTPRouteInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    ctrl.onAdd,
		UpdateFunc: ctrl.onUpdate,
		DeleteFunc: ctrl.OnDelete,
	})
	return ctrl
}

func (c *gatewayHTTPRouteController) run(ctx context.Context) {
	log.Info("gateway HTTPRoute controller started")
	defer log.Info("gateway HTTPRoute controller exited")
	defer c.workqueue.ShutDown()

	if !cache.WaitForCacheSync(ctx.Done(), c.controller.gatewayHTTPRouteInformer.HasSynced) {
		log.Error("sync Gateway HTTPRoute cache failed")
		return
	}

	for i := 0; i < c.workers; i++ {
		go c.runWorker(ctx)
	}
	<-ctx.Done()
}

func (c *gatewayHTTPRouteController) runWorker(ctx context.Context) {
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

func (c *gatewayHTTPRouteController) sync(ctx context.Context, ev *types.Event) error {
	key := ev.Object.(string)
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		log.Errorw("found Gateway HTTPRoute resource with invalid key",
			zap.Error(err),
			zap.String("key", key),
		)
		return err
	}

	log.Debugw("sync HTTPRoute", zap.String("key", key))

	httpRoute, err := c.controller.gatewayHTTPRouteLister.HTTPRoutes(namespace).Get(name)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			log.Errorw("failed to get Gateway HTTPRoute",
				zap.Error(err),
				zap.String("key", key),
			)
			return err
		}
		if ev.Type != types.EventDelete {
			log.Warnw("Gateway HTTPRoute was deleted before process",
				zap.String("key", key),
			)
			// Don't need to retry.
			return nil
		}
	}

	if ev.Type == types.EventDelete {
		if httpRoute != nil {
			// We still find the resource while we are processing the DELETE event,
			// that means object with same namespace and name was created, discarding
			// this stale DELETE event.
			log.Warnw("discard the stale Gateway delete event since it exists",
				zap.String("key", key),
			)
			return nil
		}
		httpRoute = ev.Tombstone.(*gatewayv1beta1.HTTPRoute)
	}

	tctx, err := c.controller.translator.TranslateGatewayHTTPRouteV1beta1(httpRoute)

	if err != nil {
		log.Errorw("failed to translate gateway HTTPRoute",
			zap.Error(err),
			zap.Any("object", httpRoute),
		)
		return err
	}

	log.Debugw("translated HTTPRoute",
		zap.Any("routes", tctx.Routes),
		zap.Any("upstreams", tctx.Upstreams),
	)
	m := &utils.Manifest{
		Routes:    tctx.Routes,
		Upstreams: tctx.Upstreams,
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
		var oldCtx *translation.TranslateContext
		oldObj := ev.OldObject.(*gatewayv1beta1.HTTPRoute)
		oldCtx, err = c.controller.translator.TranslateGatewayHTTPRouteV1beta1(oldObj)
		if err != nil {
			log.Errorw("failed to translate old HTTPRoute",
				zap.String("version", oldObj.APIVersion),
				zap.String("event_type", "update"),
				zap.Any("HTTPRoute", oldObj),
				zap.Error(err),
			)
			return err
		}

		om := &utils.Manifest{
			Routes:    oldCtx.Routes,
			Upstreams: oldCtx.Upstreams,
		}
		added, updated, deleted = m.Diff(om)
	}

	return utils.SyncManifests(ctx, c.controller.APISIX, c.controller.APISIXClusterName, added, updated, deleted, false)
}

func (c *gatewayHTTPRouteController) handleSyncErr(obj interface{}, err error) {
	if err == nil {
		c.workqueue.Forget(obj)
		c.controller.MetricsCollector.IncrSyncOperation("gateway_httproute", "success")
		return
	}
	event := obj.(*types.Event)
	if k8serrors.IsNotFound(err) && event.Type != types.EventDelete {
		log.Infow("sync gateway HTTPRoute but not found, ignore",
			zap.String("event_type", event.Type.String()),
			zap.String("HTTPRoute ", event.Object.(string)),
		)
		c.workqueue.Forget(event)
		return
	}
	log.Warnw("sync gateway HTTPRoute failed, will retry",
		zap.Any("object", obj),
		zap.Error(err),
	)
	c.workqueue.AddRateLimited(obj)
	c.controller.MetricsCollector.IncrSyncOperation("gateway_httproute", "failure")
}

func (c *gatewayHTTPRouteController) onAdd(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorw("found gateway HTTPRoute resource with bad meta namespace key",
			zap.Error(err),
		)
		return
	}
	if !c.controller.NamespaceProvider.IsWatchingNamespace(key) {
		return
	}
	log.Debugw("gateway HTTPRoute add event arrived",
		zap.String("key", key),
		zap.Any("object", obj),
	)

	c.workqueue.Add(&types.Event{
		Type:   types.EventAdd,
		Object: key,
	})
}

func (c *gatewayHTTPRouteController) onUpdate(oldObj, newObj interface{}) {
	oldHTTPRoute := oldObj.(*gatewayv1beta1.HTTPRoute)
	newHTTPRoute := newObj.(*gatewayv1beta1.HTTPRoute)
	if oldHTTPRoute.ResourceVersion >= newHTTPRoute.ResourceVersion {
		return
	}
	key, err := cache.MetaNamespaceKeyFunc(oldObj)
	if err != nil {
		log.Errorw("found gateway HTTPRoute resource with bad meta namespace key",
			zap.Error(err),
		)
		return
	}
	if !c.controller.NamespaceProvider.IsWatchingNamespace(key) {
		return
	}
	log.Debugw("Gateway HTTPRoute update event arrived",
		zap.String("key", key),
		zap.Any("old object", oldObj),
		zap.Any("new object", newObj),
	)

	c.workqueue.Add(&types.Event{
		Type:   types.EventUpdate,
		Object: key,
	})
}

func (c *gatewayHTTPRouteController) OnDelete(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorw("found Gateway HTTPRoute resource with bad meta namespace key",
			zap.Error(err),
		)
		return
	}
	if !c.controller.NamespaceProvider.IsWatchingNamespace(key) {
		return
	}
	log.Debugw("Gateway HTTPRoute delete event arrived",
		zap.String("key", key),
		zap.Any("object", obj),
	)

	c.workqueue.Add(&types.Event{
		Type:      types.EventDelete,
		Object:    key,
		Tombstone: obj,
	})
}
