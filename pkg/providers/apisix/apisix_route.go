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
package apisix

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	apisixcache "github.com/apache/apisix-ingress-controller/pkg/apisix/cache"
	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/kube"
	v2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	configv2beta2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta2"
	"github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta3"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/providers/translation"
	"github.com/apache/apisix-ingress-controller/pkg/providers/utils"
	"github.com/apache/apisix-ingress-controller/pkg/types"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

type apisixRouteController struct {
	*apisixCommon
	workqueue workqueue.RateLimitingInterface
	workers   int

	svcInformer         cache.SharedIndexInformer
	apisixRouteLister   kube.ApisixRouteLister
	apisixRouteInformer cache.SharedIndexInformer

	svcLock sync.RWMutex
	svcMap  map[string]map[string]struct{}
}

func newApisixRouteController(common *apisixCommon, svcInformer cache.SharedIndexInformer, apisixRouteInformer cache.SharedIndexInformer) *apisixRouteController {
	c := &apisixRouteController{
		apisixCommon: common,
		workqueue:    workqueue.NewNamedRateLimitingQueue(workqueue.NewItemFastSlowRateLimiter(1*time.Second, 60*time.Second, 5), "ApisixRoute"),
		workers:      1,

		svcInformer:         svcInformer,
		apisixRouteInformer: apisixRouteInformer,

		svcMap: make(map[string]map[string]struct{}),
	}

	apisixFactory := common.KubeClient.NewAPISIXSharedIndexInformerFactory()
	c.apisixRouteLister = kube.NewApisixRouteLister(
		apisixFactory.Apisix().V2beta2().ApisixRoutes().Lister(),
		apisixFactory.Apisix().V2beta3().ApisixRoutes().Lister(),
		apisixFactory.Apisix().V2().ApisixRoutes().Lister(),
	)

	c.apisixRouteInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.onAdd,
			UpdateFunc: c.onUpdate,
			DeleteFunc: c.onDelete,
		},
	)
	c.svcInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: c.onSvcAdd,
		},
	)

	return c
}

func (c *apisixRouteController) run(ctx context.Context) {
	log.Info("ApisixRoute controller started")
	defer log.Info("ApisixRoute controller exited")
	defer c.workqueue.ShutDown()

	ok := cache.WaitForCacheSync(ctx.Done(), c.apisixRouteInformer.HasSynced, c.svcInformer.HasSynced)
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

		switch val := obj.(type) {
		case *types.Event:
			err := c.sync(ctx, val)
			c.workqueue.Done(obj)
			c.handleSyncErr(obj, err)
		case string:
			err := c.handleSvcAdd(val)
			c.workqueue.Done(obj)
			c.handleSvcErr(val, err)
		}
	}
}

func (c *apisixRouteController) syncServiceRelationship(ev *types.Event, name string, ar kube.ApisixRoute) {
	obj := ev.Object.(kube.ApisixRouteEvent)

	var (
		oldBackends []string
		newBackends []string
	)
	switch obj.GroupVersion {
	case config.ApisixV2beta3:
		var (
			old    *v2beta3.ApisixRoute
			newObj *v2beta3.ApisixRoute
		)

		if ev.Type == types.EventUpdate {
			old = obj.OldObject.V2beta3()
		} else if ev.Type == types.EventDelete {
			old = ev.Tombstone.(kube.ApisixRoute).V2beta3()
		}

		if ev.Type != types.EventDelete {
			newObj = ar.V2beta3()
		}

		// calculate diff, so we don't need to care about the event order
		if old != nil {
			for _, rule := range old.Spec.HTTP {
				for _, backend := range rule.Backends {
					oldBackends = append(oldBackends, old.Namespace+"/"+backend.ServiceName)
				}
			}
		}
		if newObj != nil {
			for _, rule := range newObj.Spec.HTTP {
				for _, backend := range rule.Backends {
					newBackends = append(newBackends, newObj.Namespace+"/"+backend.ServiceName)
				}
			}
		}
	case config.ApisixV2:
		var (
			old    *v2.ApisixRoute
			newObj *v2.ApisixRoute
		)

		if ev.Type == types.EventUpdate {
			old = obj.OldObject.V2()
		} else if ev.Type == types.EventDelete {
			old = ev.Tombstone.(kube.ApisixRoute).V2()
		}

		if ev.Type != types.EventDelete {
			newObj = ar.V2()
		}

		// calculate diff, so we don't need to care about the event order
		if old != nil {
			for _, rule := range old.Spec.HTTP {
				for _, backend := range rule.Backends {
					oldBackends = append(oldBackends, old.Namespace+"/"+backend.ServiceName)
				}
			}
		}
		if newObj != nil {
			for _, rule := range newObj.Spec.HTTP {
				for _, backend := range rule.Backends {
					newBackends = append(newBackends, newObj.Namespace+"/"+backend.ServiceName)
				}
			}
		}
	}

	// NOTE:
	// This implementation MAY cause potential problem due to unstable event order
	// The last event processed MAY not be the logical last event, so it may override the logical previous event
	// We have a periodic full-sync, which reduce this problem, but it doesn't solve it completely.

	c.svcLock.Lock()
	defer c.svcLock.Unlock()
	toDelete := utils.Difference(oldBackends, newBackends)
	toAdd := utils.Difference(newBackends, oldBackends)
	for _, svc := range toDelete {
		delete(c.svcMap[svc], name)
	}

	for _, svc := range toAdd {
		if _, ok := c.svcMap[svc]; !ok {
			c.svcMap[svc] = make(map[string]struct{})
		}
		c.svcMap[svc][name] = struct{}{}
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
		ar   kube.ApisixRoute
		tctx *translation.TranslateContext
	)
	switch obj.GroupVersion {
	case config.ApisixV2beta2:
		ar, err = c.apisixRouteLister.V2beta2(namespace, name)
	case config.ApisixV2beta3:
		ar, err = c.apisixRouteLister.V2beta3(namespace, name)
	case config.ApisixV2:
		ar, err = c.apisixRouteLister.V2(namespace, name)
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

	c.syncServiceRelationship(ev, name, ar)

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
	case config.ApisixV2beta2:
		if ev.Type != types.EventDelete {
			tctx, err = c.translator.TranslateRouteV2beta2(ar.V2beta2())
		} else {
			tctx, err = c.translator.TranslateRouteV2beta2NotStrictly(ar.V2beta2())
		}
		if err != nil {
			log.Errorw("failed to translate ApisixRoute v2beta2",
				zap.Error(err),
				zap.Any("object", ar),
			)
			return err
		}
	case config.ApisixV2beta3:
		if ev.Type != types.EventDelete {
			if err = c.checkPluginNameIfNotEmptyV2beta3(ctx, ar.V2beta3()); err == nil {
				tctx, err = c.translator.TranslateRouteV2beta3(ar.V2beta3())
			}
		} else {
			tctx, err = c.translator.TranslateRouteV2beta3NotStrictly(ar.V2beta3())
		}
		if err != nil {
			log.Errorw("failed to translate ApisixRoute v2beta3",
				zap.Error(err),
				zap.Any("object", ar),
			)
			return err
		}
	case config.ApisixV2:
		if ev.Type != types.EventDelete {
			if err = c.checkPluginNameIfNotEmptyV2(ctx, ar.V2()); err == nil {
				tctx, err = c.translator.TranslateRouteV2(ar.V2())
			}
		} else {
			tctx, err = c.translator.TranslateRouteV2NotStrictly(ar.V2())
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

	m := &utils.Manifest{
		Routes:        tctx.Routes,
		Upstreams:     tctx.Upstreams,
		StreamRoutes:  tctx.StreamRoutes,
		PluginConfigs: tctx.PluginConfigs,
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
		oldCtx, _ := c.getOldTranslateContext(ctx, obj.OldObject)
		om := &utils.Manifest{
			Routes:        oldCtx.Routes,
			Upstreams:     oldCtx.Upstreams,
			StreamRoutes:  oldCtx.StreamRoutes,
			PluginConfigs: oldCtx.PluginConfigs,
		}
		added, updated, deleted = m.Diff(om)
	}

	return c.SyncManifests(ctx, added, updated, deleted)
}

func (c *apisixRouteController) checkPluginNameIfNotEmptyV2beta3(ctx context.Context, in *v2beta3.ApisixRoute) error {
	for _, v := range in.Spec.HTTP {
		if v.PluginConfigName != "" {
			_, err := c.APISIX.Cluster(c.Config.APISIX.DefaultClusterName).PluginConfig().Get(ctx, apisixv1.ComposePluginConfigName(in.Namespace, v.PluginConfigName))
			if err != nil {
				if err == apisixcache.ErrNotFound {
					log.Errorw("checkPluginNameIfNotEmptyV2beta3 error: plugin_config not found",
						zap.String("name", apisixv1.ComposePluginConfigName(in.Namespace, v.PluginConfigName)),
						zap.Any("obj", in),
						zap.Error(err))
				} else {
					log.Errorw("checkPluginNameIfNotEmptyV2beta3 PluginConfig get failed",
						zap.String("name", apisixv1.ComposePluginConfigName(in.Namespace, v.PluginConfigName)),
						zap.Any("obj", in),
						zap.Error(err))
				}
				return err
			}
		}
	}
	return nil
}

func (c *apisixRouteController) checkPluginNameIfNotEmptyV2(ctx context.Context, in *v2.ApisixRoute) error {
	for _, v := range in.Spec.HTTP {
		if v.PluginConfigName != "" {
			_, err := c.APISIX.Cluster(c.Config.APISIX.DefaultClusterName).PluginConfig().Get(ctx, apisixv1.ComposePluginConfigName(in.Namespace, v.PluginConfigName))
			if err != nil {
				if err == apisixcache.ErrNotFound {
					log.Errorw("checkPluginNameIfNotEmptyV2 error: plugin_config not found",
						zap.String("name", apisixv1.ComposePluginConfigName(in.Namespace, v.PluginConfigName)),
						zap.Any("obj", in),
						zap.Error(err))
				} else {
					log.Errorw("checkPluginNameIfNotEmptyV2 PluginConfig get failed",
						zap.String("name", apisixv1.ComposePluginConfigName(in.Namespace, v.PluginConfigName)),
						zap.Any("obj", in),
						zap.Error(err))
				}
				return err
			}
		}
	}
	return nil
}

func (c *apisixRouteController) handleSyncErr(obj interface{}, errOrigin error) {
	ev := obj.(*types.Event)
	event := ev.Object.(kube.ApisixRouteEvent)
	if k8serrors.IsNotFound(errOrigin) && ev.Type != types.EventDelete {
		log.Infow("sync ApisixRoute but not found, ignore",
			zap.String("event_type", ev.Type.String()),
			zap.String("ApisixRoute", event.Key),
		)
		c.workqueue.Forget(event)
		return
	}
	namespace, name, errLocal := cache.SplitMetaNamespaceKey(event.Key)
	if errLocal != nil {
		log.Errorf("invalid resource key: %s", event.Key)
		c.MetricsCollector.IncrSyncOperation("route", "failure")
		return
	}
	var ar kube.ApisixRoute
	switch event.GroupVersion {
	case config.ApisixV2beta2:
		ar, errLocal = c.apisixRouteLister.V2beta2(namespace, name)
	case config.ApisixV2beta3:
		ar, errLocal = c.apisixRouteLister.V2beta3(namespace, name)
	case config.ApisixV2:
		ar, errLocal = c.apisixRouteLister.V2(namespace, name)
	}
	if errOrigin == nil {
		if ev.Type != types.EventDelete {
			if errLocal == nil {
				switch ar.GroupVersion() {
				case config.ApisixV2beta2:
					c.RecordEvent(ar.V2beta2(), v1.EventTypeNormal, utils.ResourceSynced, nil)
					c.recordStatus(ar.V2beta2(), utils.ResourceSynced, nil, metav1.ConditionTrue, ar.V2beta2().GetGeneration())
				case config.ApisixV2beta3:
					c.RecordEvent(ar.V2beta3(), v1.EventTypeNormal, utils.ResourceSynced, nil)
					c.recordStatus(ar.V2beta3(), utils.ResourceSynced, nil, metav1.ConditionTrue, ar.V2beta3().GetGeneration())
				case config.ApisixV2:
					c.RecordEvent(ar.V2(), v1.EventTypeNormal, utils.ResourceSynced, nil)
					c.recordStatus(ar.V2(), utils.ResourceSynced, nil, metav1.ConditionTrue, ar.V2().GetGeneration())
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
		c.MetricsCollector.IncrSyncOperation("route", "success")
		return
	}
	log.Warnw("sync ApisixRoute failed, will retry",
		zap.Any("object", obj),
		zap.Error(errOrigin),
	)
	if errLocal == nil {
		switch ar.GroupVersion() {
		case config.ApisixV2beta2:
			c.RecordEvent(ar.V2beta2(), v1.EventTypeWarning, utils.ResourceSyncAborted, errOrigin)
			c.recordStatus(ar.V2beta2(), utils.ResourceSyncAborted, errOrigin, metav1.ConditionFalse, ar.V2beta2().GetGeneration())
		case config.ApisixV2beta3:
			c.RecordEvent(ar.V2beta3(), v1.EventTypeWarning, utils.ResourceSyncAborted, errOrigin)
			c.recordStatus(ar.V2beta3(), utils.ResourceSyncAborted, errOrigin, metav1.ConditionFalse, ar.V2beta3().GetGeneration())
		case config.ApisixV2:
			c.RecordEvent(ar.V2(), v1.EventTypeWarning, utils.ResourceSyncAborted, errOrigin)
			c.recordStatus(ar.V2(), utils.ResourceSyncAborted, errOrigin, metav1.ConditionFalse, ar.V2().GetGeneration())
		}
	} else {
		log.Errorw("failed list ApisixRoute",
			zap.Error(errLocal),
			zap.String("name", name),
			zap.String("namespace", namespace),
		)
	}
	c.workqueue.AddRateLimited(obj)
	c.MetricsCollector.IncrSyncOperation("route", "failure")
}

func (c *apisixRouteController) onAdd(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found ApisixRoute resource with bad meta namespace key: %s", err)
		return
	}
	if !c.namespaceProvider.IsWatchingNamespace(key) {
		return
	}
	log.Debugw("ApisixRoute add event arrived",
		zap.String("key", key),
		zap.Any("object", obj),
	)

	ar := kube.MustNewApisixRoute(obj)
	c.workqueue.Add(&types.Event{
		Type: types.EventAdd,
		Object: kube.ApisixRouteEvent{
			Key:          key,
			GroupVersion: ar.GroupVersion(),
		},
	})

	c.MetricsCollector.IncrEvents("route", "add")
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
	if !c.namespaceProvider.IsWatchingNamespace(key) {
		return
	}
	log.Debugw("ApisixRoute update event arrived",
		zap.String("key", key),
		zap.Any("new object", oldObj),
		zap.Any("old object", newObj),
	)
	c.workqueue.Add(&types.Event{
		Type: types.EventUpdate,
		Object: kube.ApisixRouteEvent{
			Key:          key,
			GroupVersion: curr.GroupVersion(),
			OldObject:    prev,
		},
	})

	c.MetricsCollector.IncrEvents("route", "update")
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
	if !c.namespaceProvider.IsWatchingNamespace(key) {
		return
	}
	log.Debugw("ApisixRoute delete event arrived",
		zap.String("key", key),
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

	c.MetricsCollector.IncrEvents("route", "delete")
}

func (c *apisixRouteController) ResourceSync() {
	objs := c.apisixRouteInformer.GetIndexer().List()

	c.svcLock.Lock()
	defer c.svcLock.Unlock()

	c.svcMap = make(map[string]map[string]struct{})

	for _, obj := range objs {
		key, err := cache.MetaNamespaceKeyFunc(obj)
		if err != nil {
			log.Errorw("ApisixRoute sync failed, found ApisixRoute resource with bad meta namespace key",
				zap.Error(err),
			)
			continue
		}
		if !c.namespaceProvider.IsWatchingNamespace(key) {
			continue
		}
		ar := kube.MustNewApisixRoute(obj)
		c.workqueue.Add(&types.Event{
			Type: types.EventAdd,
			Object: kube.ApisixRouteEvent{
				Key:          key,
				GroupVersion: ar.GroupVersion(),
			},
		})

		ns, name, err := cache.SplitMetaNamespaceKey(key)
		if err != nil {
			log.Errorw("split ApisixRoute meta key failed",
				zap.Error(err),
				zap.String("key", key),
			)
			continue
		}

		var backends []string
		switch ar.GroupVersion() {
		case config.ApisixV2beta3:
			for _, rule := range ar.V2beta3().Spec.HTTP {
				for _, backend := range rule.Backends {
					backends = append(backends, ns+"/"+backend.ServiceName)
				}
			}
		case config.ApisixV2:
			for _, rule := range ar.V2().Spec.HTTP {
				for _, backend := range rule.Backends {
					backends = append(backends, ns+"/"+backend.ServiceName)
				}
			}
		}
		for _, svcKey := range backends {
			if _, ok := c.svcMap[svcKey]; !ok {
				c.svcMap[svcKey] = make(map[string]struct{})
			}
			c.svcMap[svcKey][name] = struct{}{}
		}
	}
}

func (c *apisixRouteController) onSvcAdd(obj interface{}) {
	log.Debugw("Service add event arrived",
		zap.Any("object", obj),
	)
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorw("found Service with bad meta key",
			zap.Error(err),
			zap.String("key", key),
		)
		return
	}
	if !c.namespaceProvider.IsWatchingNamespace(key) {
		return
	}

	c.workqueue.Add(key)
}

func (c *apisixRouteController) handleSvcAdd(key string) error {
	ns, _, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		log.Errorw("failed to split Service meta key",
			zap.Error(err),
			zap.String("key", key),
		)
		return nil
	}

	c.svcLock.RLock()
	routes, ok := c.svcMap[key]
	c.svcLock.RUnlock()

	if ok {
		for route := range routes {
			c.workqueue.Add(&types.Event{
				Type: types.EventAdd,
				Object: kube.ApisixRouteEvent{
					Key:          ns + "/" + route,
					GroupVersion: c.Kubernetes.ApisixRouteVersion,
				},
			})
		}
	}
	return nil
}

func (c *apisixRouteController) handleSvcErr(key string, errOrigin error) {
	if errOrigin == nil {
		c.workqueue.Forget(key)

		return
	}

	log.Warnw("sync Service failed, will retry",
		zap.Any("key", key),
		zap.Error(errOrigin),
	)
	c.workqueue.AddRateLimited(key)
}

// Building objects from cache
// For old objects, you cannot use TranslateRoute to build. Because it needs to parse the latest service, which will cause data inconsistency
func (c *apisixRouteController) getOldTranslateContext(ctx context.Context, kar kube.ApisixRoute) (*translation.TranslateContext, error) {
	clusterName := c.Config.APISIX.DefaultClusterName
	oldCtx := translation.DefaultEmptyTranslateContext()

	switch c.Kubernetes.ApisixRouteVersion {
	case config.ApisixV2beta3:
		ar := kar.V2beta3()
		for _, part := range ar.Spec.Stream {
			name := apisixv1.ComposeStreamRouteName(ar.Namespace, ar.Name, part.Name)
			sr, err := c.APISIX.Cluster(clusterName).StreamRoute().Get(ctx, name)
			if err != nil {
				continue
			}
			if sr.UpstreamId != "" {
				ups := apisixv1.NewDefaultUpstream()
				ups.ID = sr.UpstreamId
				oldCtx.AddUpstream(ups)
			}
			oldCtx.AddStreamRoute(sr)
		}
		for _, part := range ar.Spec.HTTP {
			name := apisixv1.ComposeRouteName(ar.Namespace, ar.Name, part.Name)
			r, err := c.APISIX.Cluster(clusterName).Route().Get(ctx, name)
			if err != nil {
				continue
			}
			if r.UpstreamId != "" {
				ups := apisixv1.NewDefaultUpstream()
				ups.ID = r.UpstreamId
				oldCtx.AddUpstream(ups)
			}
			if r.PluginConfigId != "" {
				pc := apisixv1.NewDefaultPluginConfig()
				pc.ID = r.PluginConfigId
				oldCtx.AddPluginConfig(pc)
			}
			oldCtx.AddRoute(r)
		}
	case config.ApisixV2:
		ar := kar.V2()
		for _, part := range ar.Spec.Stream {
			name := apisixv1.ComposeStreamRouteName(ar.Namespace, ar.Name, part.Name)
			sr, err := c.APISIX.Cluster(clusterName).StreamRoute().Get(ctx, name)
			if err != nil {
				continue
			}
			if sr.UpstreamId != "" {
				ups := apisixv1.NewDefaultUpstream()
				ups.ID = sr.UpstreamId
				oldCtx.AddUpstream(ups)
			}
			oldCtx.AddStreamRoute(sr)
		}
		for _, part := range ar.Spec.HTTP {
			name := apisixv1.ComposeRouteName(ar.Namespace, ar.Name, part.Name)
			r, err := c.APISIX.Cluster(clusterName).Route().Get(ctx, name)
			if err != nil {
				continue
			}
			if r.UpstreamId != "" {
				ups := apisixv1.NewDefaultUpstream()
				ups.ID = r.UpstreamId
				oldCtx.AddUpstream(ups)
			}
			if r.PluginConfigId != "" {
				pc := apisixv1.NewDefaultPluginConfig()
				pc.ID = r.PluginConfigId
				oldCtx.AddPluginConfig(pc)
			}
			oldCtx.AddRoute(r)

		}
	}
	return oldCtx, nil
}

// recordStatus record resources status
func (c *apisixRouteController) recordStatus(at interface{}, reason string, err error, status metav1.ConditionStatus, generation int64) {
	// build condition
	message := utils.CommonSuccessMessage
	if err != nil {
		message = err.Error()
	}
	condition := metav1.Condition{
		Type:               utils.ConditionType,
		Reason:             reason,
		Status:             status,
		Message:            message,
		ObservedGeneration: generation,
	}
	apisixClient := c.KubeClient.APISIXClient

	if kubeObj, ok := at.(runtime.Object); ok {
		at = kubeObj.DeepCopyObject()
	}

	switch v := at.(type) {
	case *configv2beta2.ApisixRoute:
		// set to status
		if v.Status.Conditions == nil {
			conditions := make([]metav1.Condition, 0)
			v.Status.Conditions = conditions
		}
		if utils.VerifyGeneration(&v.Status.Conditions, condition) {
			meta.SetStatusCondition(&v.Status.Conditions, condition)
			if _, errRecord := apisixClient.ApisixV2beta2().ApisixRoutes(v.Namespace).
				UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
				log.Errorw("failed to record status change for ApisixRoute",
					zap.Error(errRecord),
					zap.String("name", v.Name),
					zap.String("namespace", v.Namespace),
				)
			}
		}
	case *v2beta3.ApisixRoute:
		// set to status
		if v.Status.Conditions == nil {
			conditions := make([]metav1.Condition, 0)
			v.Status.Conditions = conditions
		}
		if utils.VerifyGeneration(&v.Status.Conditions, condition) {
			meta.SetStatusCondition(&v.Status.Conditions, condition)
			if _, errRecord := apisixClient.ApisixV2beta3().ApisixRoutes(v.Namespace).
				UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
				log.Errorw("failed to record status change for ApisixRoute",
					zap.Error(errRecord),
					zap.String("name", v.Name),
					zap.String("namespace", v.Namespace),
				)
			}
		}
	case *v2.ApisixRoute:
		// set to status
		if v.Status.Conditions == nil {
			conditions := make([]metav1.Condition, 0)
			v.Status.Conditions = conditions
		}
		if utils.VerifyGeneration(&v.Status.Conditions, condition) {
			meta.SetStatusCondition(&v.Status.Conditions, condition)
			if _, errRecord := apisixClient.ApisixV2().ApisixRoutes(v.Namespace).
				UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
				log.Errorw("failed to record status change for ApisixRoute",
					zap.Error(errRecord),
					zap.String("name", v.Name),
					zap.String("namespace", v.Namespace),
				)
			}
		}
	default:
		// This should not be executed
		log.Errorf("unsupported resource record: %s", v)
	}
}
