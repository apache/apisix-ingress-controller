// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
//
// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	tls://www.apache.org/licenses/LICENSE-2.0
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

type gatewayTLSRouteController struct {
	controller *Provider
	workqueue  workqueue.RateLimitingInterface
	workers    int
}

func newGatewayTLSRouteController(c *Provider) *gatewayTLSRouteController {
	ctrl := &gatewayTLSRouteController{
		controller: c,
		workqueue:  workqueue.NewNamedRateLimitingQueue(workqueue.NewItemFastSlowRateLimiter(1*time.Second, 60*time.Second, 5), "GatewayTLSRoute"),
		workers:    1,
	}

	ctrl.controller.gatewayTLSRouteInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    ctrl.onAdd,
		UpdateFunc: ctrl.onUpdate,
		DeleteFunc: ctrl.OnDelete,
	})
	return ctrl
}

func (c *gatewayTLSRouteController) run(ctx context.Context) {
	log.Info("gateway TLSRoute controller started")
	defer log.Info("gateway TLSRoute controller exited")
	defer c.workqueue.ShutDown()

	if !cache.WaitForCacheSync(ctx.Done(), c.controller.gatewayTLSRouteInformer.HasSynced) {
		log.Error("sync Gateway TLSRoute cache failed")
		return
	}

	for i := 0; i < c.workers; i++ {
		go c.runWorker(ctx)
	}
	<-ctx.Done()
}

func (c *gatewayTLSRouteController) runWorker(ctx context.Context) {
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

func (c *gatewayTLSRouteController) sync(ctx context.Context, ev *types.Event) error {
	key := ev.Object.(string)
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		log.Errorw("found Gateway TLSRoute resource with invalid key",
			zap.Error(err),
			zap.String("key", key),
		)
		return err
	}

	log.Debugw("sync TLSRoute", zap.String("key", key))

	tlsRoute, err := c.controller.gatewayTLSRouteLister.TLSRoutes(namespace).Get(name)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			log.Errorw("failed to get Gateway TLSRoute",
				zap.Error(err),
				zap.String("key", key),
			)
			return err
		}
		if ev.Type != types.EventDelete {
			log.Warnw("Gateway TLSRoute was deleted before process",
				zap.String("key", key),
			)
			// Don't need to retry.
			return nil
		}
	}

	if ev.Type == types.EventDelete {
		if tlsRoute != nil {
			// We still find the resource while we are processing the DELETE event,
			// that means object with same namespace and name was created, discarding
			// this stale DELETE event.
			log.Warnw("discard the stale Gateway delete event since it exists",
				zap.String("key", key),
			)
			return nil
		}
		tlsRoute = ev.Tombstone.(*gatewayv1alpha2.TLSRoute)
	}
	err = c.controller.validator.ValidateCommonRoute(tlsRoute)
	if err != nil {
		log.Errorw("failed to validate gateway HTTPRoute",
			zap.Error(err),
			zap.Any("object", tlsRoute),
		)
		return err
	}

	tctx, err := c.controller.translator.TranslateGatewayTLSRouteV1Alpha2(tlsRoute)
	if err != nil {
		log.Warnw("failed to translate gateway TLSRoute",
			zap.Error(err),
			zap.Any("object", tlsRoute),
		)
		return err
	}

	log.Debugw("translated TLSRoute",
		zap.Any("stream_routes", tctx.StreamRoutes),
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
		oldObj := ev.OldObject.(*gatewayv1alpha2.TLSRoute)
		oldCtx, err = c.controller.translator.TranslateGatewayTLSRouteV1Alpha2(oldObj)
		if err != nil {
			log.Errorw("failed to translate old TLSRoute",
				zap.String("version", oldObj.APIVersion),
				zap.String("event_type", "update"),
				zap.Any("TLSRoute", oldObj),
				zap.Error(err),
			)
			return err
		}

		om := &utils.Manifest{
			StreamRoutes: oldCtx.StreamRoutes,
			Upstreams:    oldCtx.Upstreams,
		}
		added, updated, deleted = m.Diff(om)
	}

	return utils.SyncManifests(ctx, c.controller.APISIX, c.controller.APISIXClusterName, added, updated, deleted, false)
}

func (c *gatewayTLSRouteController) handleSyncErr(obj interface{}, err error) {
	if err == nil {
		c.workqueue.Forget(obj)
		c.controller.MetricsCollector.IncrSyncOperation("gateway_tlsroute", "success")
		return
	}
	event := obj.(*types.Event)
	if k8serrors.IsNotFound(err) && event.Type != types.EventDelete {
		log.Infow("sync gateway TLSRoute but not found, ignore",
			zap.String("event_type", event.Type.String()),
			zap.String("TLSRoute ", event.Object.(string)),
		)
		c.workqueue.Forget(event)
		return
	}
	log.Warnw("sync gateway TLSRoute failed, will retry",
		zap.Any("object", obj),
		zap.Error(err),
	)
	c.workqueue.AddRateLimited(obj)
	c.controller.MetricsCollector.IncrSyncOperation("gateway_tlsroute", "failure")
}

func (c *gatewayTLSRouteController) onAdd(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorw("found gateway TLSRoute resource with bad meta namespace key",
			zap.Error(err),
		)
		return
	}
	if !c.controller.NamespaceProvider.IsWatchingNamespace(key) {
		return
	}
	log.Debugw("gateway TLSRoute add event arrived",
		zap.Any("object", obj),
	)

	log.Debugw("add TLSRoute", zap.String("key", key))
	c.workqueue.Add(&types.Event{
		Type:   types.EventAdd,
		Object: key,
	})
}
func (c *gatewayTLSRouteController) onUpdate(oldObj, newObj interface{}) {}
func (c *gatewayTLSRouteController) OnDelete(obj interface{})            {}
