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
	"time"

	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/apache/apisix-ingress-controller/pkg/kube"
	v2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	"github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta3"
	"github.com/apache/apisix-ingress-controller/pkg/kube/translation"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/types"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

type apisixRouteController struct {
	controller *Controller
	workqueue  workqueue.RateLimitingInterface
	workers    int
}

func (c *Controller) newApisixRouteController() *apisixRouteController {
	ctl := &apisixRouteController{
		controller: c,
		workqueue:  workqueue.NewNamedRateLimitingQueue(workqueue.NewItemFastSlowRateLimiter(1*time.Second, 60*time.Second, 5), "ApisixRoute"),
		workers:    1,
	}
	c.apisixRouteInformer.AddEventHandler(
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
	defer c.workqueue.ShutDown()

	ok := cache.WaitForCacheSync(ctx.Done(), c.controller.apisixRouteInformer.HasSynced)
	if !ok {
		log.Error("cache sync failed")
		return
	}

	for i := 0; i < c.workers; i++ {
		go c.runWorker(ctx)
	}
	<-ctx.Done()
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
	obj := ev.Object.(kube.ApisixRouteEvent)
	namespace, name, err := cache.SplitMetaNamespaceKey(obj.Key)
	if err != nil {
		log.Errorf("invalid resource key: %s", obj.Key)
		return err
	}
	var (
		ar         kube.ApisixRoute
		replaced   *v2beta3.ApisixRoute
		replacedV2 *v2.ApisixRoute
		tctx       *translation.TranslateContext
	)
	switch obj.GroupVersion {
	case kube.ApisixRouteV2beta1:
		ar, err = c.controller.apisixRouteLister.V2beta1(namespace, name)
	case kube.ApisixRouteV2beta2:
		ar, err = c.controller.apisixRouteLister.V2beta2(namespace, name)
	case kube.ApisixRouteV2beta3:
		ar, err = c.controller.apisixRouteLister.V2beta3(namespace, name)
	case kube.ApisixRouteV2:
		ar, err = c.controller.apisixRouteLister.V2(namespace, name)
	}
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			log.Errorw("failed to get ApisixRoute",
				zap.String("version", obj.GroupVersion),
				zap.String("key", obj.Key),
				zap.Error(err),
			)
			return err
		}

		if ev.Type != types.EventDelete {
			log.Warnw("ApisixRoute was deleted before it can be delivered",
				zap.String("key", obj.Key),
				zap.String("version", obj.GroupVersion),
			)
			return nil
		}
	}
	if ev.Type == types.EventDelete {
		if ar != nil {
			// We still find the resource while we are processing the DELETE event,
			// that means object with same namespace and name was created, discarding
			// this stale DELETE event.
			log.Warnw("discard the stale ApisixRoute delete event since the resource still exists",
				zap.String("key", obj.Key),
			)
			return nil
		}
		ar = ev.Tombstone.(kube.ApisixRoute)
	}

	switch obj.GroupVersion {
	case kube.ApisixRouteV2beta1:
		if ev.Type != types.EventDelete {
			tctx, err = c.controller.translator.TranslateRouteV2beta1(ar.V2beta1())
		} else {
			tctx, err = c.controller.translator.TranslateRouteV2beta1NotStrictly(ar.V2beta1())
		}
		if err != nil {
			log.Errorw("failed to translate ApisixRoute v2beta1",
				zap.Error(err),
				zap.Any("object", ar),
			)
			return err
		}
	case kube.ApisixRouteV2beta2:
		if ev.Type != types.EventDelete {
			tctx, err = c.controller.translator.TranslateRouteV2beta2(ar.V2beta2())
		} else {
			tctx, err = c.controller.translator.TranslateRouteV2beta2NotStrictly(ar.V2beta2())
		}
		if err != nil {
			log.Errorw("failed to translate ApisixRoute v2beta2",
				zap.Error(err),
				zap.Any("object", ar),
			)
			return err
		}
	case kube.ApisixRouteV2beta3:
		if ev.Type != types.EventDelete {
			if replaced, err = c.replacePluginNameWithIdIfNotEmptyV2beta3(ctx, ar.V2beta3()); err == nil {
				tctx, err = c.controller.translator.TranslateRouteV2beta3(replaced)
			}
		} else {
			tctx, err = c.controller.translator.TranslateRouteV2beta3NotStrictly(ar.V2beta3())
		}
		if err != nil {
			log.Errorw("failed to translate ApisixRoute v2beta3",
				zap.Error(err),
				zap.Any("object", ar),
			)
			return err
		}
	case kube.ApisixRouteV2:
		if ev.Type != types.EventDelete {
			if replacedV2, err = c.replacePluginNameWithIdIfNotEmptyV2(ctx, ar.V2()); err == nil {
				tctx, err = c.controller.translator.TranslateRouteV2(replacedV2)
			}
		} else {
			tctx, err = c.controller.translator.TranslateRouteV2NotStrictly(ar.V2())
		}
		if err != nil {
			log.Errorw("failed to translate ApisixRoute v2",
				zap.Error(err),
				zap.Any("object", ar),
			)
			return err
		}
	}

	log.Debugw("translated ApisixRoute",
		zap.Any("routes", tctx.Routes),
		zap.Any("upstreams", tctx.Upstreams),
		zap.Any("apisix_route", ar),
		zap.Any("pluginConfigs", tctx.PluginConfigs),
	)

	m := &manifest{
		routes:        tctx.Routes,
		upstreams:     tctx.Upstreams,
		streamRoutes:  tctx.StreamRoutes,
		pluginConfigs: tctx.PluginConfigs,
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
		var oldCtx *translation.TranslateContext
		switch obj.GroupVersion {
		case kube.ApisixRouteV2beta1:
			oldCtx, err = c.controller.translator.TranslateRouteV2beta1(obj.OldObject.V2beta1())
		case kube.ApisixRouteV2beta2:
			oldCtx, err = c.controller.translator.TranslateRouteV2beta2(obj.OldObject.V2beta2())
		case kube.ApisixRouteV2beta3:
			if replaced, err = c.replacePluginNameWithIdIfNotEmptyV2beta3(ctx, obj.OldObject.V2beta3()); err == nil {
				oldCtx, err = c.controller.translator.TranslateRouteV2beta3(replaced)
			}
		case kube.ApisixRouteV2:
			if replacedV2, err = c.replacePluginNameWithIdIfNotEmptyV2(ctx, obj.OldObject.V2()); err == nil {
				oldCtx, err = c.controller.translator.TranslateRouteV2(replacedV2)
			}
		}
		if err != nil {
			log.Errorw("failed to translate old ApisixRoute",
				zap.String("version", obj.GroupVersion),
				zap.String("event", "update"),
				zap.Error(err),
				zap.Any("ApisixRoute", ar),
			)
			return err
		}

		om := &manifest{
			routes:        oldCtx.Routes,
			upstreams:     oldCtx.Upstreams,
			streamRoutes:  oldCtx.StreamRoutes,
			pluginConfigs: oldCtx.PluginConfigs,
		}
		added, updated, deleted = m.diff(om)
	}

	return c.controller.syncManifests(ctx, added, updated, deleted)
}

func (c *apisixRouteController) replacePluginNameWithIdIfNotEmptyV2beta3(ctx context.Context, in *v2beta3.ApisixRoute) (*v2beta3.ApisixRoute, error) {
	clusterName := c.controller.cfg.APISIX.DefaultClusterName
	news := make([]v2beta3.ApisixRouteHTTP, 0)
	for _, v := range in.Spec.HTTP {
		pluginConfigId := ""
		if v.PluginConfigName != "" {
			pc, err := c.controller.apisix.Cluster(clusterName).PluginConfig().Get(ctx, apisixv1.ComposePluginConfigName(in.Namespace, v.PluginConfigName))
			if err != nil {
				log.Errorw("replacePluginNameWithIdIfNotEmptyV2beta3 error:  plugin_config not found",
					zap.String("name", apisixv1.ComposePluginConfigName(in.Namespace, v.PluginConfigName)),
					zap.Any("obj", in),
					zap.Error(err))
			} else {
				pluginConfigId = pc.ID
			}
		}
		v.PluginConfigName = pluginConfigId
		news = append(news, v)
	}
	in.Spec.HTTP = news
	return in, nil
}

func (c *apisixRouteController) replacePluginNameWithIdIfNotEmptyV2(ctx context.Context, in *v2.ApisixRoute) (*v2.ApisixRoute, error) {
	clusterName := c.controller.cfg.APISIX.DefaultClusterName
	news := make([]v2.ApisixRouteHTTP, 0)
	for _, v := range in.Spec.HTTP {
		pluginConfigId := ""
		if v.PluginConfigName != "" {
			pc, err := c.controller.apisix.Cluster(clusterName).PluginConfig().Get(ctx, apisixv1.ComposePluginConfigName(in.Namespace, v.PluginConfigName))
			if err != nil {
				log.Errorw("replacePluginNameWithIdIfNotEmptyV2 error:  plugin_config not found",
					zap.String("name", apisixv1.ComposePluginConfigName(in.Namespace, v.PluginConfigName)),
					zap.Any("obj", in),
					zap.Error(err))
			} else {
				pluginConfigId = pc.ID
			}
		}
		v.PluginConfigName = pluginConfigId
		news = append(news, v)
	}
	in.Spec.HTTP = news
	return in, nil
}

func (c *apisixRouteController) handleSyncErr(obj interface{}, errOrigin error) {
	ev := obj.(*types.Event)
	event := ev.Object.(kube.ApisixRouteEvent)
	namespace, name, errLocal := cache.SplitMetaNamespaceKey(event.Key)
	if errLocal != nil {
		log.Errorf("invalid resource key: %s", event.Key)
		c.controller.MetricsCollector.IncrSyncOperation("route", "failure")
		return
	}
	var ar kube.ApisixRoute
	switch event.GroupVersion {
	case kube.ApisixRouteV2beta3:
		ar, errLocal = c.controller.apisixRouteLister.V2beta3(namespace, name)
	case kube.ApisixRouteV2beta1:
		ar, errLocal = c.controller.apisixRouteLister.V2beta1(namespace, name)
	case kube.ApisixRouteV2beta2:
		ar, errLocal = c.controller.apisixRouteLister.V2beta2(namespace, name)
	case kube.ApisixRouteV2:
		ar, errLocal = c.controller.apisixRouteLister.V2(namespace, name)
	}
	if errOrigin == nil {
		if ev.Type != types.EventDelete {
			if errLocal == nil {
				switch ar.GroupVersion() {
				case kube.ApisixRouteV2beta1:
					c.controller.recorderEvent(ar.V2beta1(), v1.EventTypeNormal, _resourceSynced, nil)
					c.controller.recordStatus(ar.V2beta1(), _resourceSynced, nil, metav1.ConditionTrue, ar.V2beta1().GetGeneration())
				case kube.ApisixRouteV2beta2:
					c.controller.recorderEvent(ar.V2beta2(), v1.EventTypeNormal, _resourceSynced, nil)
					c.controller.recordStatus(ar.V2beta2(), _resourceSynced, nil, metav1.ConditionTrue, ar.V2beta2().GetGeneration())
				case kube.ApisixRouteV2beta3:
					c.controller.recorderEvent(ar.V2beta3(), v1.EventTypeNormal, _resourceSynced, nil)
					c.controller.recordStatus(ar.V2beta3(), _resourceSynced, nil, metav1.ConditionTrue, ar.V2beta3().GetGeneration())
				case kube.ApisixRouteV2:
					c.controller.recorderEvent(ar.V2(), v1.EventTypeNormal, _resourceSynced, nil)
					c.controller.recordStatus(ar.V2(), _resourceSynced, nil, metav1.ConditionTrue, ar.V2().GetGeneration())
				}
			} else {
				log.Errorw("failed list ApisixRoute",
					zap.Error(errLocal),
					zap.String("name", name),
					zap.String("namespace", namespace),
				)
			}
		}
		c.workqueue.Forget(obj)
		c.controller.MetricsCollector.IncrSyncOperation("route", "success")
		return
	}
	log.Warnw("sync ApisixRoute failed, will retry",
		zap.Any("object", obj),
		zap.Error(errOrigin),
	)
	if errLocal == nil {
		switch ar.GroupVersion() {
		case kube.ApisixRouteV2beta1:
			c.controller.recorderEvent(ar.V2beta1(), v1.EventTypeWarning, _resourceSyncAborted, errOrigin)
			c.controller.recordStatus(ar.V2beta1(), _resourceSyncAborted, errOrigin, metav1.ConditionFalse, ar.V2beta1().GetGeneration())
		case kube.ApisixRouteV2beta2:
			c.controller.recorderEvent(ar.V2beta2(), v1.EventTypeWarning, _resourceSyncAborted, errOrigin)
			c.controller.recordStatus(ar.V2beta2(), _resourceSyncAborted, errOrigin, metav1.ConditionFalse, ar.V2beta2().GetGeneration())
		case kube.ApisixRouteV2beta3:
			c.controller.recorderEvent(ar.V2beta3(), v1.EventTypeWarning, _resourceSyncAborted, errOrigin)
			c.controller.recordStatus(ar.V2beta3(), _resourceSyncAborted, errOrigin, metav1.ConditionFalse, ar.V2beta3().GetGeneration())
		case kube.ApisixRouteV2:
			c.controller.recorderEvent(ar.V2(), v1.EventTypeWarning, _resourceSyncAborted, errOrigin)
			c.controller.recordStatus(ar.V2(), _resourceSyncAborted, errOrigin, metav1.ConditionFalse, ar.V2().GetGeneration())
		}

	} else {
		log.Errorw("failed list ApisixRoute",
			zap.Error(errLocal),
			zap.String("name", name),
			zap.String("namespace", namespace),
		)
	}
	c.workqueue.AddRateLimited(obj)
	c.controller.MetricsCollector.IncrSyncOperation("route", "failure")
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

	ar := kube.MustNewApisixRoute(obj)
	c.workqueue.Add(&types.Event{
		Type: types.EventAdd,
		Object: kube.ApisixRouteEvent{
			Key:          key,
			GroupVersion: ar.GroupVersion(),
		},
	})

	c.controller.MetricsCollector.IncrEvents("route", "add")
}

func (c *apisixRouteController) onUpdate(oldObj, newObj interface{}) {
	prev := kube.MustNewApisixRoute(oldObj)
	curr := kube.MustNewApisixRoute(newObj)
	if prev.ResourceVersion() >= curr.ResourceVersion() {
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
	log.Debugw("ApisixRoute update event arrived",
		zap.Any("new object", curr),
		zap.Any("old object", prev),
	)
	c.workqueue.Add(&types.Event{
		Type: types.EventUpdate,
		Object: kube.ApisixRouteEvent{
			Key:          key,
			GroupVersion: curr.GroupVersion(),
			OldObject:    prev,
		},
	})

	c.controller.MetricsCollector.IncrEvents("route", "update")
}

func (c *apisixRouteController) onDelete(obj interface{}) {
	ar, err := kube.NewApisixRoute(obj)
	if err != nil {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			return
		}
		ar = kube.MustNewApisixRoute(tombstone)
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
		zap.Any("final state", ar),
	)
	c.workqueue.Add(&types.Event{
		Type: types.EventDelete,
		Object: kube.ApisixRouteEvent{
			Key:          key,
			GroupVersion: ar.GroupVersion(),
		},
		Tombstone: ar,
	})

	c.controller.MetricsCollector.IncrEvents("route", "delete")
}
