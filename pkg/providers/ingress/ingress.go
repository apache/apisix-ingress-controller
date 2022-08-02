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
	apiv1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/apache/apisix-ingress-controller/pkg/kube"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/providers/utils"
	"github.com/apache/apisix-ingress-controller/pkg/types"
)

const (
	_ingressKey = "kubernetes.io/ingress.class"
)

type ingressController struct {
	*ingressCommon

	workqueue workqueue.RateLimitingInterface
	workers   int

	ingressLister   kube.IngressLister
	ingressInformer cache.SharedIndexInformer
}

func newIngressController(common *ingressCommon, ingressLister kube.IngressLister, ingressInformer cache.SharedIndexInformer) *ingressController {
	c := &ingressController{
		ingressCommon: common,

		workqueue: workqueue.NewNamedRateLimitingQueue(workqueue.NewItemFastSlowRateLimiter(1*time.Second, 60*time.Second, 5), "ingress"),
		workers:   1,

		ingressLister:   ingressLister,
		ingressInformer: ingressInformer,
	}

	c.ingressInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.onAdd,
		UpdateFunc: c.onUpdate,
		DeleteFunc: c.OnDelete,
	})
	return c
}

func (c *ingressController) run(ctx context.Context) {
	log.Info("ingress controller started")
	defer log.Infof("ingress controller exited")
	defer c.workqueue.ShutDown()

	if !cache.WaitForCacheSync(ctx.Done(), c.ingressInformer.HasSynced) {
		log.Errorf("cache sync failed")
		return
	}
	for i := 0; i < c.workers; i++ {
		go c.runWorker(ctx)
	}
	<-ctx.Done()
}

func (c *ingressController) runWorker(ctx context.Context) {
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

func (c *ingressController) sync(ctx context.Context, ev *types.Event) error {
	ingEv := ev.Object.(kube.IngressEvent)
	namespace, name, err := cache.SplitMetaNamespaceKey(ingEv.Key)
	if err != nil {
		log.Errorf("found ingress resource with invalid meta namespace key %s: %s", ingEv.Key, err)
		return err
	}

	var ing kube.Ingress
	switch ingEv.GroupVersion {
	case kube.IngressV1:
		ing, err = c.ingressLister.V1(namespace, name)
	case kube.IngressV1beta1:
		ing, err = c.ingressLister.V1beta1(namespace, name)
	case kube.IngressExtensionsV1beta1:
		ing, err = c.ingressLister.ExtensionsV1beta1(namespace, name)
	default:
		err = fmt.Errorf("unsupported group version %s, one of (%s/%s/%s) is expected", ingEv.GroupVersion,
			kube.IngressV1, kube.IngressV1beta1, kube.IngressExtensionsV1beta1)
	}

	if err != nil {
		if !k8serrors.IsNotFound(err) {
			log.Errorf("failed to get ingress %s (group version: %s): %s", ingEv.Key, ingEv.GroupVersion, err)
			return err
		}

		if ev.Type != types.EventDelete {
			log.Warnf("ingress %s (group version: %s) was deleted before it can be delivered", ingEv.Key, ingEv.GroupVersion)
			// Don't need to retry.
			return nil
		}
	}
	if ev.Type == types.EventDelete {
		if ing != nil {
			// We still find the resource while we are processing the DELETE event,
			// that means object with same namespace and name was created, discarding
			// this stale DELETE event.
			log.Warnf("discard the stale ingress delete event since the %s exists", ingEv.Key)
			return nil
		}
		ing = ev.Tombstone.(kube.Ingress)
	}

	tctx, err := c.translator.TranslateIngress(ing)
	if err != nil {
		log.Errorw("failed to translate ingress",
			zap.Error(err),
			zap.Any("ingress", ing),
		)
		return err
	}

	log.Debugw("translated ingress resource to a couple of routes, upstreams and pluginConfigs",
		zap.Any("ingress", ing),
		zap.Any("routes", tctx.Routes),
		zap.Any("upstreams", tctx.Upstreams),
		zap.Any("ssl", tctx.SSL),
		zap.Any("pluginConfigs", tctx.PluginConfigs),
	)

	m := &utils.Manifest{
		SSLs:          tctx.SSL,
		Routes:        tctx.Routes,
		Upstreams:     tctx.Upstreams,
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
		oldCtx, err := c.translator.TranslateOldIngress(ingEv.OldObject)
		if err != nil {
			log.Errorw("failed to translate ingress",
				zap.String("event", "update"),
				zap.Error(err),
				zap.Any("ingress", ingEv.OldObject),
			)
			return err
		}
		om := &utils.Manifest{
			Routes:        oldCtx.Routes,
			Upstreams:     oldCtx.Upstreams,
			SSLs:          oldCtx.SSL,
			PluginConfigs: oldCtx.PluginConfigs,
		}
		added, updated, deleted = m.Diff(om)
	}
	if err := c.SyncManifests(ctx, added, updated, deleted); err != nil {
		log.Errorw("failed to sync ingress artifacts",
			zap.Error(err),
		)
		return err
	}
	return nil
}

func (c *ingressController) handleSyncErr(obj interface{}, err error) {
	ev := obj.(*types.Event)
	event := ev.Object.(kube.IngressEvent)
	if k8serrors.IsNotFound(err) && ev.Type != types.EventDelete {
		log.Infow("sync ingress but not found, ignore",
			zap.String("event_type", ev.Type.String()),
			zap.String("ingress", event.Key),
		)
		c.workqueue.Forget(event)
		return
	}
	namespace, name, errLocal := cache.SplitMetaNamespaceKey(event.Key)
	if errLocal != nil {
		log.Errorw("invalid resource key",
			zap.Error(errLocal),
		)
		return
	}

	var ing kube.Ingress
	switch event.GroupVersion {
	case kube.IngressV1:
		ing, errLocal = c.ingressLister.V1(namespace, name)
	case kube.IngressV1beta1:
		ing, errLocal = c.ingressLister.V1beta1(namespace, name)
	case kube.IngressExtensionsV1beta1:
		ing, errLocal = c.ingressLister.ExtensionsV1beta1(namespace, name)
	}

	if err == nil {
		// add status
		if ev.Type != types.EventDelete {
			if errLocal == nil {
				switch ing.GroupVersion() {
				case kube.IngressV1:
					c.recordStatus(ing.V1(), utils.ResourceSynced, nil, metav1.ConditionTrue, ing.V1().GetGeneration())
				case kube.IngressV1beta1:
					c.recordStatus(ing.V1beta1(), utils.ResourceSynced, nil, metav1.ConditionTrue, ing.V1beta1().GetGeneration())
				case kube.IngressExtensionsV1beta1:
					c.recordStatus(ing.ExtensionsV1beta1(), utils.ResourceSynced, nil, metav1.ConditionTrue, ing.ExtensionsV1beta1().GetGeneration())
				}
			} else {
				log.Errorw("failed to list ingress resource",
					zap.Error(errLocal),
				)
			}
		}
		c.workqueue.Forget(obj)
		c.MetricsCollector.IncrSyncOperation("ingress", "success")
		return
	}
	log.Warnw("sync ingress failed, will retry",
		zap.Any("object", obj),
		zap.Error(err),
	)

	if errLocal == nil {
		switch ing.GroupVersion() {
		case kube.IngressV1:
			c.recordStatus(ing.V1(), utils.ResourceSyncAborted, err, metav1.ConditionTrue, ing.V1().GetGeneration())
		case kube.IngressV1beta1:
			c.recordStatus(ing.V1beta1(), utils.ResourceSyncAborted, err, metav1.ConditionTrue, ing.V1beta1().GetGeneration())
		case kube.IngressExtensionsV1beta1:
			c.recordStatus(ing.ExtensionsV1beta1(), utils.ResourceSyncAborted, err, metav1.ConditionTrue, ing.ExtensionsV1beta1().GetGeneration())
		}
	} else {
		log.Errorw("failed to list ingress resource",
			zap.Error(errLocal),
		)
	}
	c.workqueue.AddRateLimited(obj)
	c.MetricsCollector.IncrSyncOperation("ingress", "failure")
}

func (c *ingressController) onAdd(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found ingress resource with bad meta namespace key: %s", err)
		return
	}
	if !c.namespaceProvider.IsWatchingNamespace(key) {
		return
	}

	ing := kube.MustNewIngress(obj)
	valid := c.isIngressEffective(ing)
	if valid {
		log.Debugw("ingress add event arrived",
			zap.Any("object", obj),
		)
	} else {
		log.Debugw("ignore noneffective ingress add event",
			zap.Any("object", obj),
		)
		return
	}

	c.workqueue.Add(&types.Event{
		Type: types.EventAdd,
		Object: kube.IngressEvent{
			Key:          key,
			GroupVersion: ing.GroupVersion(),
		},
	})

	c.MetricsCollector.IncrEvents("ingress", "add")
}

func (c *ingressController) onUpdate(oldObj, newObj interface{}) {
	prev := kube.MustNewIngress(oldObj)
	curr := kube.MustNewIngress(newObj)
	if prev.ResourceVersion() >= curr.ResourceVersion() {
		return
	}

	key, err := cache.MetaNamespaceKeyFunc(newObj)
	if err != nil {
		log.Errorf("found ingress resource with bad meta namespace key: %s", err)
		return
	}
	if !c.namespaceProvider.IsWatchingNamespace(key) {
		return
	}
	valid := c.isIngressEffective(curr)
	if valid {
		log.Debugw("ingress update event arrived",
			zap.Any("new object", newObj),
			zap.Any("old object", oldObj),
		)
	} else {
		log.Debugw("ignore noneffective ingress update event",
			zap.Any("new object", oldObj),
			zap.Any("old object", newObj),
		)
		return
	}

	c.workqueue.Add(&types.Event{
		Type: types.EventUpdate,
		Object: kube.IngressEvent{
			Key:          key,
			GroupVersion: curr.GroupVersion(),
			OldObject:    prev,
		},
	})

	c.MetricsCollector.IncrEvents("ingress", "update")
}

func (c *ingressController) OnDelete(obj interface{}) {
	ing, err := kube.NewIngress(obj)
	if err != nil {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			return
		}
		ing = kube.MustNewIngress(tombstone)
	}

	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found ingress resource with bad meta namespace key: %s", err)
		return
	}
	if !c.namespaceProvider.IsWatchingNamespace(key) {
		return
	}
	valid := c.isIngressEffective(ing)
	if valid {
		log.Debugw("ingress delete event arrived",
			zap.Any("final state", ing),
		)
	} else {
		log.Debugw("ignore noneffective ingress delete event",
			zap.Any("object", ing),
		)
		return
	}
	c.workqueue.Add(&types.Event{
		Type: types.EventDelete,
		Object: kube.IngressEvent{
			Key:          key,
			GroupVersion: ing.GroupVersion(),
		},
		Tombstone: ing,
	})

	c.MetricsCollector.IncrEvents("ingress", "delete")
}

func (c *ingressController) isIngressEffective(ing kube.Ingress) bool {
	var (
		ic  *string
		ica string
	)
	if ing.GroupVersion() == kube.IngressV1 {
		ic = ing.V1().Spec.IngressClassName
		ica = ing.V1().GetAnnotations()[_ingressKey]
	} else if ing.GroupVersion() == kube.IngressV1beta1 {
		ic = ing.V1beta1().Spec.IngressClassName
		ica = ing.V1beta1().GetAnnotations()[_ingressKey]
	} else {
		ic = ing.ExtensionsV1beta1().Spec.IngressClassName
		ica = ing.ExtensionsV1beta1().GetAnnotations()[_ingressKey]
	}

	// kubernetes.io/ingress.class takes the precedence.
	if ica != "" {
		return ica == c.Kubernetes.IngressClass
	}
	if ic != nil {
		return *ic == c.Kubernetes.IngressClass
	}
	return false
}

func (c *ingressController) ResourceSync() {
	objs := c.ingressInformer.GetIndexer().List()
	for _, obj := range objs {
		key, err := cache.MetaNamespaceKeyFunc(obj)
		if err != nil {
			log.Errorw("found Ingress resource with bad meta namespace key", zap.String("error", err.Error()))
			continue
		}
		if !c.namespaceProvider.IsWatchingNamespace(key) {
			continue
		}
		ing := kube.MustNewIngress(obj)
		c.workqueue.Add(&types.Event{
			Type: types.EventAdd,
			Object: kube.IngressEvent{
				Key:          key,
				GroupVersion: ing.GroupVersion(),
			},
		})
	}
}

// recordStatus record resources status
func (c *ingressController) recordStatus(at interface{}, reason string, err error, status metav1.ConditionStatus, generation int64) {
	client := c.KubeClient.Client

	if kubeObj, ok := at.(runtime.Object); ok {
		at = kubeObj.DeepCopyObject()
	}

	switch v := at.(type) {
	case *networkingv1.Ingress:
		// set to status
		lbips, err := c.ingressLBStatusIPs()
		if err != nil {
			log.Errorw("failed to get APISIX gateway external IPs",
				zap.Error(err),
			)

		}

		v.ObjectMeta.Generation = generation
		v.Status.LoadBalancer.Ingress = lbips
		if _, errRecord := client.NetworkingV1().Ingresses(v.Namespace).UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
			log.Errorw("failed to record status change for IngressV1",
				zap.Error(errRecord),
				zap.String("name", v.Name),
				zap.String("namespace", v.Namespace),
			)
		}

	case *networkingv1beta1.Ingress:
		// set to status
		lbips, err := c.ingressLBStatusIPs()
		if err != nil {
			log.Errorw("failed to get APISIX gateway external IPs",
				zap.Error(err),
			)

		}

		v.ObjectMeta.Generation = generation
		v.Status.LoadBalancer.Ingress = lbips
		if _, errRecord := client.NetworkingV1beta1().Ingresses(v.Namespace).UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
			log.Errorw("failed to record status change for IngressV1",
				zap.Error(errRecord),
				zap.String("name", v.Name),
				zap.String("namespace", v.Namespace),
			)
		}
	case *extensionsv1beta1.Ingress:
		// set to status
		lbips, err := c.ingressLBStatusIPs()
		if err != nil {
			log.Errorw("failed to get APISIX gateway external IPs",
				zap.Error(err),
			)

		}

		v.ObjectMeta.Generation = generation
		v.Status.LoadBalancer.Ingress = lbips
		if _, errRecord := client.ExtensionsV1beta1().Ingresses(v.Namespace).UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
			log.Errorw("failed to record status change for IngressV1",
				zap.Error(errRecord),
				zap.String("name", v.Name),
				zap.String("namespace", v.Namespace),
			)
		}
	default:
		// This should not be executed
		log.Errorf("unsupported resource record: %s", v)
	}
}

// ingressLBStatusIPs organizes the available addresses
func (c *ingressController) ingressLBStatusIPs() ([]apiv1.LoadBalancerIngress, error) {
	return utils.IngressLBStatusIPs(c.IngressPublishService, c.IngressStatusAddress, c.KubeClient.Client)
}
