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
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/providers/translation"
	"github.com/apache/apisix-ingress-controller/pkg/providers/utils"
	"github.com/apache/apisix-ingress-controller/pkg/types"
)

type gatewayUDPRouteController struct {
	controller *Provider
	workqueue  workqueue.RateLimitingInterface
	workers    int
}

func newGatewayUDPRouteController(c *Provider) *gatewayUDPRouteController {
	ctrl := &gatewayUDPRouteController{
		controller: c,
		workqueue:  workqueue.NewNamedRateLimitingQueue(workqueue.NewItemFastSlowRateLimiter(1*time.Second, 60*time.Second, 5), "GatewayUDPRoute"),
		workers:    1,
	}

	ctrl.controller.gatewayUDPRouteInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    ctrl.onAdd,
		UpdateFunc: ctrl.onUpdate,
		DeleteFunc: ctrl.OnDelete,
	})
	return ctrl
}

func (c *gatewayUDPRouteController) run(ctx context.Context) {
	log.Info("gateway UDPRoute controller started")
	defer log.Info("gateway UDPRoute controller exited")
	defer c.workqueue.ShutDown()

	if !cache.WaitForCacheSync(ctx.Done(), c.controller.gatewayUDPRouteInformer.HasSynced) {
		log.Error("sync Gateway UDPRoute cache failed")
		return
	}

	for i := 0; i < c.workers; i++ {
		go c.runWorker(ctx)
	}
	<-ctx.Done()
}

func (c *gatewayUDPRouteController) runWorker(ctx context.Context) {
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

func (c *gatewayUDPRouteController) sync(ctx context.Context, ev *types.Event) error {
	key := ev.Object.(string)
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		log.Errorw("found Gateway UDPRoute resource with invalid key",
			zap.Error(err),
			zap.String("key", key),
		)
		return err
	}

	log.Debugw("sync UDPRoute", zap.String("key", key))

	udpRoute, err := c.controller.gatewayUDPRouteLister.UDPRoutes(namespace).Get(name)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			log.Errorw("failed to get Gateway UDPRoute",
				zap.Error(err),
				zap.String("key", key),
			)
			return err
		}
		if ev.Type != types.EventDelete {
			log.Warnw("Gateway UDPRoute was deleted before process",
				zap.String("key", key),
			)
			// Don't need to retry.
			return nil
		}
	}

	if ev.Type == types.EventDelete {
		if udpRoute != nil {
			// We still find the resource while we are processing the DELETE event,
			// that means object with same namespace and name was created, discarding
			// this stale DELETE event.
			log.Warnw("discard the stale Gateway delete event since it exists",
				zap.String("key", key),
			)
			return nil
		}
		udpRoute = ev.Tombstone.(*gatewayv1alpha2.UDPRoute)
	}
	err = c.controller.validator.ValidateCommonRoute(udpRoute)
	if err != nil {
		log.Errorw("failed to validate gateway UDPRoute",
			zap.Error(err),
			zap.Any("object", udpRoute),
		)
		return err
	}
	tctx, err := c.controller.translator.TranslateGatewayUDPRouteV1Alpha2(udpRoute)
	if err != nil {
		log.Errorw("failed to translate gateway UDPRoute",
			zap.Error(err),
			zap.Any("object", udpRoute),
		)
		return err
	}

	log.Debugw("translated UDPRoute",
		zap.Any("streamroutes", tctx.StreamRoutes),
		zap.Any("upstreams", tctx.Upstreams),
	)
	m := &utils.Manifest{
		StreamRoutes: tctx.StreamRoutes,
		Upstreams:    tctx.Upstreams,
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
		oldObj := ev.OldObject.(*gatewayv1alpha2.UDPRoute)
		oldCtx, err = c.controller.translator.TranslateGatewayUDPRouteV1Alpha2(oldObj)
		if err != nil {
			log.Errorw("failed to translate old UDPRoute",
				zap.String("version", oldObj.APIVersion),
				zap.String("event_type", "update"),
				zap.Any("UDPRoute", oldObj),
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

func (c *gatewayUDPRouteController) handleSyncErr(obj interface{}, err error) {
	if err == nil {
		c.workqueue.Forget(obj)
		c.controller.MetricsCollector.IncrSyncOperation("gateway_udproute", "success")
		return
	}
	event := obj.(*types.Event)
	if k8serrors.IsNotFound(err) && event.Type != types.EventDelete {
		log.Infow("sync gateway UDPRoute but not found, ignore",
			zap.String("event_type", event.Type.String()),
			zap.String("UDPRoute ", event.Object.(string)),
		)
		c.workqueue.Forget(event)
		return
	}
	log.Warnw("sync gateway UDPRoute failed, will retry",
		zap.Any("object", obj),
		zap.Error(err),
	)
	c.workqueue.AddRateLimited(obj)
	c.controller.MetricsCollector.IncrSyncOperation("gateway_udproute", "failure")
}

func (c *gatewayUDPRouteController) onAdd(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorw("found gateway UDPRoute resource with bad meta namespace key",
			zap.Error(err),
		)
		return
	}
	if !c.controller.NamespaceProvider.IsWatchingNamespace(key) {
		return
	}
	log.Debugw("gateway UDPRoute add event arrived",
		zap.Any("object", obj),
	)

	log.Debugw("add UDPRoute", zap.String("key", key))
	c.workqueue.Add(&types.Event{
		Type:   types.EventAdd,
		Object: key,
	})
}
func (c *gatewayUDPRouteController) onUpdate(oldObj, newObj interface{}) {}
func (c *gatewayUDPRouteController) OnDelete(obj interface{})            {}
