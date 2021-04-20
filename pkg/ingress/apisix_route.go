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
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"github.com/apache/apisix-ingress-controller/pkg/kube"
	"github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/clientset/versioned/typed/config/v2alpha1"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/types"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

const _routeController = "RouteController"

type apisixRouteController struct {
	controller *Controller
	workqueue  workqueue.RateLimitingInterface
	workers    int
	recorder   record.EventRecorder
}

func (c *Controller) newApisixRouteController() *apisixRouteController {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kube.GetKubeClient().CoreV1().Events("")})
	ctl := &apisixRouteController{
		controller: c,
		workqueue:  workqueue.NewNamedRateLimitingQueue(workqueue.NewItemFastSlowRateLimiter(1*time.Second, 60*time.Second, 5), "ApisixRoute"),
		workers:    1,
		recorder:   eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: _routeController}),
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
	ok := cache.WaitForCacheSync(ctx.Done(), c.controller.apisixRouteInformer.HasSynced)
	if !ok {
		log.Error("cache sync failed")
		return
	}

	for i := 0; i < c.workers; i++ {
		go c.runWorker(ctx)
	}
	<-ctx.Done()
	c.workqueue.ShutDown()
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
		ar        kube.ApisixRoute
		routes    []*apisixv1.Route
		upstreams []*apisixv1.Upstream
	)
	if obj.GroupVersion == kube.ApisixRouteV1 {
		ar, err = c.controller.apisixRouteLister.V1(namespace, name)
	} else {
		ar, err = c.controller.apisixRouteLister.V2alpha1(namespace, name)
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
	if obj.GroupVersion == kube.ApisixRouteV1 {
		routes, upstreams, err = c.controller.translator.TranslateRouteV1(ar.V1())
		if err != nil {
			log.Errorw("failed to translate ApisixRoute v1",
				zap.Error(err),
				zap.Any("object", ar),
			)
			return err
		}
	} else {
		routes, upstreams, err = c.controller.translator.TranslateRouteV2alpha1(ar.V2alpha1())
		if err != nil {
			log.Errorw("failed to translate ApisixRoute v2alpha1",
				zap.Error(err),
				zap.Any("object", ar),
			)
			return err
		}
	}

	log.Debugw("translated ApisixRoute",
		zap.Any("routes", routes),
		zap.Any("upstreams", upstreams),
		zap.Any("apisix_route", ar),
	)

	m := &manifest{
		routes:    routes,
		upstreams: upstreams,
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
		var (
			oldRoutes    []*apisixv1.Route
			oldUpstreams []*apisixv1.Upstream
		)
		if obj.GroupVersion == kube.ApisixRouteV1 {
			oldRoutes, oldUpstreams, err = c.controller.translator.TranslateRouteV1(obj.OldObject.V1())
		} else {
			oldRoutes, oldUpstreams, err = c.controller.translator.TranslateRouteV2alpha1(obj.OldObject.V2alpha1())
		}
		if err != nil {
			log.Errorw("failed to translate old ApisixRoute v2alpha1",
				zap.String("event", "update"),
				zap.Error(err),
				zap.Any("ApisixRoute", ar),
			)
			return err
		}

		om := &manifest{
			routes:    oldRoutes,
			upstreams: oldUpstreams,
		}
		added, updated, deleted = m.diff(om)
	}

	return c.controller.syncManifests(ctx, added, updated, deleted)
}

func (c *apisixRouteController) handleSyncErr(obj interface{}, err error) {
	event := obj.(*types.Event)
	route := event.Object.(kube.ApisixRouteEvent).OldObject
	// conditions
	condition := metav1.Condition{
		Type: "APISIXSynced",
	}
	if err == nil {
		message := fmt.Sprintf(_messageResourceSynced, _routeController)
		if route.GroupVersion() == kube.ApisixRouteV1 {
			c.recorder.Event(route.V1(), v1.EventTypeNormal, _successSynced, message)
		} else if route.GroupVersion() == kube.ApisixRouteV2alpha1 {
			c.recorder.Event(route.V2alpha1(), v1.EventTypeNormal, _successSynced, message)
			// build condition
			condition.Reason = _successSynced
			condition.Status = metav1.ConditionTrue
			condition.Message = "Sync Successfully"
			// set to status
			routev2 := route.V2alpha1()
			meta.SetStatusCondition(routev2.Status.Conditions, condition)
			v2alpha1.New(kube.GetApisixClient().ApisixV2alpha1().RESTClient()).ApisixRoutes(routev2.Namespace).
				UpdateStatus(context.TODO(), routev2, metav1.UpdateOptions{})
		}
		c.workqueue.Forget(obj)
		return
	}
	log.Warnw("sync ApisixRoute failed, will retry",
		zap.Any("object", obj),
		zap.Error(err),
	)
	message := fmt.Sprintf(_messageResourceFailed, _routeController, err.Error())
	if route.GroupVersion() == kube.ApisixRouteV1 {
		c.recorder.Event(route.V1(), v1.EventTypeWarning, _failedSynced, message)
	} else if route.GroupVersion() == kube.ApisixRouteV2alpha1 {
		c.recorder.Event(route.V2alpha1(), v1.EventTypeWarning, _failedSynced, message)
		// build condition
		condition.Reason = "_failedSynced"
		condition.Status = metav1.ConditionFalse
		condition.Message = err.Error()
		// set to status
		routev2 := route.V2alpha1()
		meta.SetStatusCondition(routev2.Status.Conditions, condition)
		v2alpha1.New(kube.GetApisixClient().ApisixV2alpha1().RESTClient()).ApisixRoutes(routev2.Namespace).
			UpdateStatus(context.TODO(), routev2, metav1.UpdateOptions{})
	}
	c.workqueue.AddRateLimited(obj)
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
	c.workqueue.AddRateLimited(&types.Event{
		Type: types.EventAdd,
		Object: kube.ApisixRouteEvent{
			Key:          key,
			GroupVersion: ar.GroupVersion(),
		},
	})
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
	c.workqueue.AddRateLimited(&types.Event{
		Type: types.EventUpdate,
		Object: kube.ApisixRouteEvent{
			Key:          key,
			GroupVersion: curr.GroupVersion(),
			OldObject:    prev,
		},
	})
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
	c.workqueue.AddRateLimited(&types.Event{
		Type: types.EventDelete,
		Object: kube.ApisixRouteEvent{
			Key:          key,
			GroupVersion: ar.GroupVersion(),
		},
		Tombstone: ar,
	})
}
