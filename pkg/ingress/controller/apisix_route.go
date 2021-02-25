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

	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"

	"go.uber.org/zap"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	configv1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v1"
	configv2alpha1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2alpha1"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/seven/state"
	"github.com/apache/apisix-ingress-controller/pkg/types"
)

type apisixRouteController struct {
	controller *Controller
	workqueue  workqueue.RateLimitingInterface
	workers    int
}

type routeObject struct {
	key     string
	version string
	old     interface{}
}

func (c *Controller) newApisixRouteController() *apisixRouteController {
	ctl := &apisixRouteController{
		controller: c,
		workqueue:  workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ApisixRoute"),
		workers:    1,
	}

	c.apisixRouteV1Informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctl.onAdd,
			UpdateFunc: ctl.onUpdate,
			DeleteFunc: ctl.onDelete,
		},
	)
	c.apisixRouteV2alpha1Informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctl.onAdd,
			UpdateFunc: ctl.onUpdate,
			DeleteFunc: ctl.onDelete,
		},
	)
	return ctl
}

func (c *apisixRouteController) run(ctx context.Context) {
	log.Info("ApisixRoute controller started")
	defer log.Info("ApisixRoute controller exited")
	ok := cache.WaitForCacheSync(ctx.Done(), c.controller.apisixRouteV2alpha1Informer.HasSynced,
		c.controller.apisixRouteV1Informer.HasSynced)
	if !ok {
		log.Error("cache sync failed")
		return
	}

	for i := 0; i < c.workers; i++ {
		go c.runWorker(ctx)
	}
}

func (c *apisixRouteController) runWorker(ctx context.Context) {
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

func (c *apisixRouteController) sync(ctx context.Context, ev *types.Event) error {
	obj := ev.Object.(routeObject)
	namespace, name, err := cache.SplitMetaNamespaceKey(obj.key)
	if err != nil {
		log.Errorf("invalid resource key: %s", obj.key)
		return fmt.Errorf("invalid resource key: %s", obj.key)
	}
	var (
		routeV1       *configv1.ApisixRoute
		routeV2alpha1 *configv2alpha1.ApisixRoute
		routes        []*apisixv1.Route
		upstreams     []*apisixv1.Upstream
	)
	if ev.Type != types.EventDelete {
		if obj.version == "v1" {
			routeV1, err = c.controller.apisixRouteV1Lister.ApisixRoutes(namespace).Get(name)
		} else {
			routeV2alpha1, err = c.controller.apisixRouteV2alpha1Lister.ApisixRoutes(namespace).Get(name)
		}
	}
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			log.Errorw("failed to get ApisixRoute",
				zap.String("version", obj.version),
				zap.String("key", obj.key),
				zap.Error(err),
			)
			return err
		}

		if ev.Type != types.EventDelete {
			log.Warnw("ApisixRoute was deleted before it can be delivered",
				zap.String("key", obj.key),
				zap.String("version", obj.version),
			)
			return nil
		}
	}
	if ev.Type == types.EventDelete {
		if (obj.version == "v1" && routeV1 != nil) || (obj.version == "v2alpha1" && routeV2alpha1 != nil) {
			// We still find the resource while we are processing the DELETE event,
			// that means object with same namespace and name was created, discarding
			// this stale DELETE event.
			log.Warnw("discard the stale ApisixRoute delete event since the resource still exists",
				zap.String("key", obj.key),
			)
			return nil
		}
		if obj.version == "v1" {
			routeV1 = ev.Tombstone.(*configv1.ApisixRoute)
		} else {
			routeV2alpha1 = ev.Tombstone.(*configv2alpha1.ApisixRoute)
		}
	}
	if obj.version == "v1" {
		routes, upstreams, err = c.controller.translator.TranslateRouteV1(routeV1)
		if err != nil {
			log.Errorw("failed to translate ApisixRoute v1",
				zap.Error(err),
				zap.Any("object", routeV1),
			)
			return err
		}
	} else {
		routes, upstreams, err = c.controller.translator.TranslateRouteV2alpha1(routeV2alpha1)
		if err != nil {
			log.Errorw("failed to translate ApisixRoute v2alpha1",
				zap.Error(err),
				zap.Any("object", routeV2alpha1),
			)
			return err
		}
	}

	if ev.Type == types.EventDelete {
		rc := &state.RouteCompare{OldRoutes: routes, NewRoutes: nil}
		if err := rc.Sync(); err != nil {
			return err
		} else {
			comb := state.ApisixCombination{Routes: nil, Upstreams: upstreams}
			if err := comb.Remove(); err != nil {
				return err
			}
		}
		return nil
	} else if ev.Type == types.EventAdd {
		comb := state.ApisixCombination{Routes: routes, Upstreams: upstreams}
		if _, err := comb.Solver(); err != nil {
			return err
		}
	} else {
		var oldRoutes []*apisixv1.Route
		if obj.version == "v1" {
			oldRoutes, _, _ = c.controller.translator.TranslateRouteV1(routeV1)
		} else {
			oldRoutes, _, _ = c.controller.translator.TranslateRouteV2alpha1(routeV2alpha1)
		}
		rc := &state.RouteCompare{OldRoutes: oldRoutes, NewRoutes: routes}
		return rc.Sync()
	}
	return nil
}

func (c *apisixRouteController) handleSyncErr(obj interface{}, err error) {
	if err == nil {
		c.workqueue.Forget(obj)
		return
	}
	if c.workqueue.NumRequeues(obj) < _maxRetries {
		log.Infow("sync ApisixRoute failed, will retry",
			zap.Any("object", obj),
		)
		c.workqueue.AddRateLimited(obj)
	} else {
		c.workqueue.Forget(obj)
		log.Warnf("drop ApisixRoute %+v out of the queue", obj)
	}
}

func (c *apisixRouteController) onAdd(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found ApisixRoute resource with bad meta namespace key: %s", err)
		return
	}
	if !c.controller.namespaceWatching(key) {
		return
	}
	log.Debugw("ApisixRoute add event arrived",
		zap.Any("object", obj))

	var version string
	switch obj.(type) {
	case *configv2alpha1.ApisixRoute:
		version = "v2alpha1"
	case *configv1.ApisixRoute:
		version = "v1"
	}

	c.workqueue.AddRateLimited(&types.Event{
		Type: types.EventAdd,
		Object: routeObject{
			key:     key,
			version: version,
		},
	})
}

func (c *apisixRouteController) onUpdate(oldObj, newObj interface{}) {
	var (
		rv1     string
		rv2     string
		version string
	)
	switch obj := oldObj.(type) {
	case *configv1.ApisixRoute:
		rv1 = obj.ResourceVersion
		rv2 = newObj.(*configv1.ApisixRoute).ResourceVersion
		version = "v1"
	case *configv2alpha1.ApisixRoute:
		rv1 = obj.ResourceVersion
		rv2 = newObj.(*configv2alpha1.ApisixRoute).ResourceVersion
		version = "v2"
	default:
		return
	}
	if rv1 >= rv2 {
		return
	}
	key, err := cache.MetaNamespaceKeyFunc(newObj)
	if err != nil {
		log.Errorf("found ApisixRoute resource with bad meta namespace key: %s", err)
		return
	}
	if !c.controller.namespaceWatching(key) {
		return
	}
	c.workqueue.AddRateLimited(&types.Event{
		Type: types.EventUpdate,
		Object: routeObject{
			key:     key,
			version: version,
			old:     oldObj,
		},
	})
}

func (c *apisixRouteController) onDelete(obj interface{}) {
	var (
		version string
	)
	ev := &types.Event{
		Type: types.EventDelete,
	}
	switch ar := obj.(type) {
	case *configv1.ApisixRoute:
		ev.Tombstone = ar
		version = "v1"
	case *configv2alpha1.ApisixRoute:
		ev.Tombstone = ar
		version = "v2alpha1"
	case cache.DeletedFinalStateUnknown:
		// *configv1.ApisixRoute
		// *configv2alpha1.ApisixRoute
		ev.Tombstone = ar.Obj
		switch ar.Obj.(type) {
		case *configv1.ApisixRoute:
			version = "v1"
		case *configv2alpha1.ApisixRoute:
			version = "v2alpha1"
		default:
			return
		}
	default:
		return
	}
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found ApisixRoute resource with bad meta namesapce key: %s", err)
		return
	}
	if !c.controller.namespaceWatching(key) {
		return
	}
	log.Debugw("ApisixRoute delete event arrived",
		zap.Any("final state", ev.Tombstone),
	)
	ev.Object = routeObject{
		key:     key,
		version: version,
	}
	c.workqueue.AddRateLimited(ev)
}
