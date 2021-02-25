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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	configv1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v1"
	configv2alpha1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2alpha1"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/seven/state"
	"github.com/apache/apisix-ingress-controller/pkg/types"
	"github.com/api7/ingress-controller/pkg/ingress/apisix"
)

type apisixRouteController struct {
	controller *Controller
	workqueue  workqueue.RateLimitingInterface
	workers    int
}

type routeObject struct {
	key     string
	version string
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
	key := ev.Object.(string)
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		log.Errorf("invalid resource key: %s", key)
		return fmt.Errorf("invalid resource key: %s", key)
	}
	switch {
	case rqo.Ope == UPDATE:
		apisixIngressRoute, err := c.apisixRouteList.ApisixRoutes(namespace).Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				log.Errorf("apisixRoute %s is removed", key)
				return nil
			}
			return err // if error occurred, return
		}
		oldRoutes, _, _ := c.controller.translator.TranslateRouteV1(rqo.OldObj)
		newRoutes, _, _ := c.controller.translator.TranslateRouteV1(apisixIngressRoute)

		rc := &state.RouteCompare{OldRoutes: oldRoutes, NewRoutes: newRoutes}
		return rc.Sync()
	case rqo.Ope == DELETE:
		apisixIngressRoute, _ := c.apisixRouteList.ApisixRoutes(namespace).Get(name)
		if apisixIngressRoute != nil && apisixIngressRoute.ResourceVersion > rqo.OldObj.ResourceVersion {
			log.Warnf("Route %s has been covered when retry", rqo.Key)
			return nil
		}
		routes, upstreams, _ := c.controller.translator.TranslateRouteV1(rqo.OldObj)
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
	default:
		return fmt.Errorf("not expected in (ApisixRouteController) sync")
	}
}

func (c *ApisixRouteController) add(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		log.Errorf("invalid resource key: %s", key)
		return fmt.Errorf("invalid resource key: %s", key)
	}

	apisixIngressRoute, err := c.apisixRouteList.ApisixRoutes(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Infof("apisixRoute %s is removed", key)
			return nil
		}
		log.Errorf("failed to list ApisixRoute %s: %s", key, err.Error())
		runtime.HandleError(fmt.Errorf("failed to list ApisixRoute %s: %s", key, err.Error()))
		return err
	}
	apisixRoute := apisix.ApisixRoute(*apisixIngressRoute)
	routes, services, upstreams, _ := apisixRoute.Convert(c.controller.translator)
	comb := state.ApisixCombination{Routes: routes, Services: services, Upstreams: upstreams}
	_, err = comb.Solver()
	return err

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
