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
package gateway

import (
	"context"
	"time"

	"go.uber.org/zap"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/apache/apisix-ingress-controller/pkg/ingress/utils"
	"github.com/apache/apisix-ingress-controller/pkg/kube/translation"
	"github.com/apache/apisix-ingress-controller/pkg/log"
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
			log.Warnf("discard the stale Gateway delete event since the %s exists", key)
			return nil
		}
		httpRoute = ev.Tombstone.(*gatewayv1alpha2.HTTPRoute)
	}

	tctx, err := c.controller.translator.TranslateGatewayHTTPRouteV1Alpha2(httpRoute)

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
		oldObj := ev.OldObject.(*gatewayv1alpha2.HTTPRoute)
		oldCtx, err = c.controller.translator.TranslateGatewayHTTPRouteV1Alpha2(oldObj)
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

	return utils.SyncManifests(ctx, c.controller.APISIX, c.controller.APISIXClusterName, added, updated, deleted)
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
		log.Errorf("found gateway HTTPRoute resource with bad meta namespace key: %s", err)
		return
	}
	if !c.controller.NamespaceProvider.IsWatchingNamespace(key) {
		return
	}
	log.Debugw("gateway HTTPRoute add event arrived",
		zap.Any("object", obj),
	)

	log.Debugw("add HTTPRoute", zap.String("key", key))
	c.workqueue.Add(&types.Event{
		Type:   types.EventAdd,
		Object: key,
	})
}
func (c *gatewayHTTPRouteController) onUpdate(oldObj, newObj interface{}) {}
func (c *gatewayHTTPRouteController) OnDelete(obj interface{})            {}
