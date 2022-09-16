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
	"fmt"
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	apisixcache "github.com/apache/apisix-ingress-controller/pkg/apisix/cache"
	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/kube"
	configv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	configv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta3"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/providers/utils"
	"github.com/apache/apisix-ingress-controller/pkg/types"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

type apisixUpstreamController struct {
	*apisixCommon

	workqueue workqueue.RateLimitingInterface
	workers   int

	svcInformer            cache.SharedIndexInformer
	svcLister              listerscorev1.ServiceLister
	apisixUpstreamInformer cache.SharedIndexInformer
	apisixUpstreamLister   kube.ApisixUpstreamLister
}

func newApisixUpstreamController(common *apisixCommon) *apisixUpstreamController {
	c := &apisixUpstreamController{
		apisixCommon: common,
		workqueue:    workqueue.NewNamedRateLimitingQueue(workqueue.NewItemFastSlowRateLimiter(1*time.Second, 60*time.Second, 5), "ApisixUpstream"),
		workers:      1,

		svcInformer:            common.SvcInformer,
		svcLister:              common.SvcLister,
		apisixUpstreamLister:   common.ApisixUpstreamLister,
		apisixUpstreamInformer: common.ApisixUpstreamInformer,
	}

	c.apisixUpstreamInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.onAdd,
			UpdateFunc: c.onUpdate,
			DeleteFunc: c.onDelete,
		},
	)
	return c
}

func (c *apisixUpstreamController) run(ctx context.Context) {
	log.Info("ApisixUpstream controller started")
	defer log.Info("ApisixUpstream controller exited")
	defer c.workqueue.ShutDown()

	if ok := cache.WaitForCacheSync(ctx.Done(), c.apisixUpstreamInformer.HasSynced, c.svcInformer.HasSynced); !ok {
		log.Error("cache sync failed")
		return
	}
	for i := 0; i < c.workers; i++ {
		go c.runWorker(ctx)
	}

	<-ctx.Done()
}

func (c *apisixUpstreamController) runWorker(ctx context.Context) {
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

// sync Used to synchronize ApisixUpstream resources, because upstream alone exists in APISIX and will not be affected,
// the synchronization logic only includes upstream's unique configuration management
// So when ApisixUpstream was deleted, only the scheme / load balancer / healthcheck / retry / timeout
// on ApisixUpstream was cleaned up
func (c *apisixUpstreamController) sync(ctx context.Context, ev *types.Event) error {
	event := ev.Object.(kube.ApisixUpstreamEvent)
	key := event.Key
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		log.Errorf("found ApisixUpstream resource with invalid meta namespace key %s: %s", key, err)
		return err
	}

	var multiVersioned kube.ApisixUpstream
	switch event.GroupVersion {
	case config.ApisixV2beta3:
		multiVersioned, err = c.apisixUpstreamLister.V2beta3(namespace, name)
	case config.ApisixV2:
		multiVersioned, err = c.apisixUpstreamLister.V2(namespace, name)
	default:
		return fmt.Errorf("unsupported ApisixUpstream group version %s", event.GroupVersion)
	}

	if err != nil {
		if !k8serrors.IsNotFound(err) {
			log.Errorw("failed to get ApisixUpstream",
				zap.Error(err),
				zap.String("key", key),
				zap.String("version", event.GroupVersion),
			)
			return err
		}
		if ev.Type != types.EventDelete {
			log.Warnw("ApisixUpstream was deleted before it can be delivered",
				zap.String("key", key),
				zap.String("version", event.GroupVersion),
			)
			// Don't need to retry.
			return nil
		}
	}
	if ev.Type == types.EventDelete {
		if multiVersioned != nil {
			// We still find the resource while we are processing the DELETE event,
			// that means object with same namespace and name was created, discarding
			// this stale DELETE event.
			log.Warnf("discard the stale ApisixUpstream delete event since the %s exists", key)
			return nil
		}
		multiVersioned = ev.Tombstone.(kube.ApisixUpstream)
	}

	switch event.GroupVersion {
	case config.ApisixV2beta3:
		au := multiVersioned.V2beta3()

		var portLevelSettings map[int32]configv2beta3.ApisixUpstreamConfig
		if au.Spec != nil && len(au.Spec.PortLevelSettings) > 0 {
			portLevelSettings = make(map[int32]configv2beta3.ApisixUpstreamConfig, len(au.Spec.PortLevelSettings))
			for _, port := range au.Spec.PortLevelSettings {
				portLevelSettings[port.Port] = port.ApisixUpstreamConfig
			}
		}

		svc, err := c.svcLister.Services(namespace).Get(name)
		if err != nil {
			log.Errorf("failed to get service %s: %s", key, err)
			c.RecordEvent(au, corev1.EventTypeWarning, utils.ResourceSyncAborted, err)
			c.recordStatus(au, utils.ResourceSyncAborted, err, metav1.ConditionFalse, au.GetGeneration())
			return err
		}

		var subsets []configv2beta3.ApisixUpstreamSubset
		subsets = append(subsets, configv2beta3.ApisixUpstreamSubset{})
		if au.Spec != nil && len(au.Spec.Subsets) > 0 {
			subsets = append(subsets, au.Spec.Subsets...)
		}
		clusterName := c.Config.APISIX.DefaultClusterName
		for _, port := range svc.Spec.Ports {
			for _, subset := range subsets {
				upsName := apisixv1.ComposeUpstreamName(namespace, name, subset.Name, port.Port, "")
				// TODO: multiple cluster
				ups, err := c.APISIX.Cluster(clusterName).Upstream().Get(ctx, upsName)
				if err != nil {
					if err == apisixcache.ErrNotFound {
						continue
					}
					log.Errorf("failed to get upstream %s: %s", upsName, err)
					c.RecordEvent(au, corev1.EventTypeWarning, utils.ResourceSyncAborted, err)
					c.recordStatus(au, utils.ResourceSyncAborted, err, metav1.ConditionFalse, au.GetGeneration())
					return err
				}
				var newUps *apisixv1.Upstream
				if au.Spec != nil && ev.Type != types.EventDelete {
					cfg, ok := portLevelSettings[port.Port]
					if !ok {
						cfg = au.Spec.ApisixUpstreamConfig
					}
					// FIXME Same ApisixUpstreamConfig might be translated multiple times.
					newUps, err = c.translator.TranslateUpstreamConfigV2beta3(&cfg)
					if err != nil {
						log.Errorw("found malformed ApisixUpstream",
							zap.Any("object", au),
							zap.Error(err),
						)
						c.RecordEvent(au, corev1.EventTypeWarning, utils.ResourceSyncAborted, err)
						c.recordStatus(au, utils.ResourceSyncAborted, err, metav1.ConditionFalse, au.GetGeneration())
						return err
					}
				} else {
					newUps = apisixv1.NewDefaultUpstream()
				}

				newUps.Metadata = ups.Metadata
				newUps.Nodes = ups.Nodes
				log.Debugw("updating upstream since ApisixUpstream changed",
					zap.String("event", ev.Type.String()),
					zap.Any("upstream", newUps),
					zap.Any("ApisixUpstream", au),
				)
				if _, err := c.APISIX.Cluster(clusterName).Upstream().Update(ctx, newUps); err != nil {
					log.Errorw("failed to update upstream",
						zap.Error(err),
						zap.Any("upstream", newUps),
						zap.Any("ApisixUpstream", au),
						zap.String("cluster", clusterName),
					)
					c.RecordEvent(au, corev1.EventTypeWarning, utils.ResourceSyncAborted, err)
					c.recordStatus(au, utils.ResourceSyncAborted, err, metav1.ConditionFalse, au.GetGeneration())
					return err
				}
			}
		}
		if ev.Type != types.EventDelete {
			c.RecordEvent(au, corev1.EventTypeNormal, utils.ResourceSynced, nil)
			c.recordStatus(au, utils.ResourceSynced, nil, metav1.ConditionTrue, au.GetGeneration())
		}
	case config.ApisixV2:
		au := multiVersioned.V2()

		var portLevelSettings map[int32]configv2.ApisixUpstreamConfig
		if au.Spec != nil && len(au.Spec.PortLevelSettings) > 0 {
			portLevelSettings = make(map[int32]configv2.ApisixUpstreamConfig, len(au.Spec.PortLevelSettings))
			for _, port := range au.Spec.PortLevelSettings {
				portLevelSettings[port.Port] = port.ApisixUpstreamConfig
			}
		}

		svc, err := c.svcLister.Services(namespace).Get(name)
		if err != nil {
			log.Errorf("failed to get service %s: %s", key, err)
			c.RecordEvent(au, corev1.EventTypeWarning, utils.ResourceSyncAborted, err)
			c.recordStatus(au, utils.ResourceSyncAborted, err, metav1.ConditionFalse, au.GetGeneration())
			return err
		}

		var subsets []configv2.ApisixUpstreamSubset
		subsets = append(subsets, configv2.ApisixUpstreamSubset{})
		if au.Spec != nil && len(au.Spec.Subsets) > 0 {
			subsets = append(subsets, au.Spec.Subsets...)
		}
		clusterName := c.Config.APISIX.DefaultClusterName
		for _, port := range svc.Spec.Ports {
			for _, subset := range subsets {
				// TODO: multiple cluster
				update := func(upsName string) error {
					ups, err := c.APISIX.Cluster(clusterName).Upstream().Get(ctx, upsName)
					if err != nil {
						if err == apisixcache.ErrNotFound {
							return nil
						}
						log.Errorf("failed to get upstream %s: %s", upsName, err)
						c.RecordEvent(au, corev1.EventTypeWarning, utils.ResourceSyncAborted, err)
						c.recordStatus(au, utils.ResourceSyncAborted, err, metav1.ConditionFalse, au.GetGeneration())
						return err
					}
					var newUps *apisixv1.Upstream
					if au.Spec != nil && ev.Type != types.EventDelete {
						cfg, ok := portLevelSettings[port.Port]
						if !ok {
							cfg = au.Spec.ApisixUpstreamConfig
						}
						// FIXME Same ApisixUpstreamConfig might be translated multiple times.
						newUps, err = c.translator.TranslateUpstreamConfigV2(&cfg)
						if err != nil {
							log.Errorw("ApisixUpstream conversion cannot be completed, or the format is incorrect",
								zap.Any("object", au),
								zap.Error(err),
							)
							c.RecordEvent(au, corev1.EventTypeWarning, utils.ResourceSyncAborted, err)
							c.recordStatus(au, utils.ResourceSyncAborted, err, metav1.ConditionFalse, au.GetGeneration())
							return err
						}
					} else {
						newUps = apisixv1.NewDefaultUpstream()
					}

					newUps.Metadata = ups.Metadata
					newUps.Nodes = ups.Nodes
					log.Debugw("updating upstream since ApisixUpstream changed",
						zap.String("event", ev.Type.String()),
						zap.Any("upstream", newUps),
						zap.Any("ApisixUpstream", au),
					)
					if _, err := c.APISIX.Cluster(clusterName).Upstream().Update(ctx, newUps); err != nil {
						log.Errorw("failed to update upstream",
							zap.Error(err),
							zap.Any("upstream", newUps),
							zap.Any("ApisixUpstream", au),
							zap.String("cluster", clusterName),
						)
						c.RecordEvent(au, corev1.EventTypeWarning, utils.ResourceSyncAborted, err)
						c.recordStatus(au, utils.ResourceSyncAborted, err, metav1.ConditionFalse, au.GetGeneration())
						return err
					}
					return nil
				}

				err := update(apisixv1.ComposeUpstreamName(namespace, name, subset.Name, port.Port, types.ResolveGranularity.Endpoint))
				if err != nil {
					return err
				}
				err = update(apisixv1.ComposeUpstreamName(namespace, name, subset.Name, port.Port, types.ResolveGranularity.Service))
				if err != nil {
					return err
				}
			}
		}
		if ev.Type != types.EventDelete {
			c.RecordEvent(au, corev1.EventTypeNormal, utils.ResourceSynced, nil)
			c.recordStatus(au, utils.ResourceSynced, nil, metav1.ConditionTrue, au.GetGeneration())
		}
	}

	return err
}

func (c *apisixUpstreamController) handleSyncErr(obj interface{}, err error) {
	if err == nil {
		c.workqueue.Forget(obj)
		c.MetricsCollector.IncrSyncOperation("upstream", "success")
		return
	}

	event := obj.(*types.Event)
	if k8serrors.IsNotFound(err) && event.Type != types.EventDelete {
		log.Infow("sync ApisixUpstream but not found, ignore",
			zap.String("event_type", event.Type.String()),
			zap.Any("ApisixUpstream", event.Object.(kube.ApisixUpstreamEvent)),
		)
		c.workqueue.Forget(event)
		return
	}
	log.Warnw("sync ApisixUpstream failed, will retry",
		zap.Any("object", obj),
		zap.Error(err),
	)
	c.workqueue.AddRateLimited(obj)
	c.MetricsCollector.IncrSyncOperation("upstream", "failure")
}

func (c *apisixUpstreamController) onAdd(obj interface{}) {
	au, err := kube.NewApisixUpstream(obj)
	if err != nil {
		log.Errorw("found ApisixUpstream resource with bad type", zap.Error(err))
		return
	}

	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found ApisixUpstream resource with bad meta namespace key: %s", err)
		return
	}
	if !c.namespaceProvider.IsWatchingNamespace(key) {
		return
	}
	log.Debugw("ApisixUpstream add event arrived",
		zap.Any("object", obj))

	c.workqueue.Add(&types.Event{
		Type: types.EventAdd,
		Object: kube.ApisixUpstreamEvent{
			Key:          key,
			GroupVersion: au.GroupVersion(),
		},
	})

	c.MetricsCollector.IncrEvents("upstream", "add")
}

func (c *apisixUpstreamController) onUpdate(oldObj, newObj interface{}) {
	prev, err := kube.NewApisixUpstream(oldObj)
	if err != nil {
		log.Errorw("found ApisixUpstream resource with bad type", zap.Error(err))
		return
	}
	curr, err := kube.NewApisixUpstream(newObj)
	if err != nil {
		log.Errorw("found ApisixUpstream resource with bad type", zap.Error(err))
		return
	}
	if prev.ResourceVersion() >= curr.ResourceVersion() {
		return
	}
	key, err := cache.MetaNamespaceKeyFunc(newObj)
	if err != nil {
		log.Errorf("found ApisixUpstream resource with bad meta namespace key: %s", err)
		return
	}
	if !c.namespaceProvider.IsWatchingNamespace(key) {
		return
	}
	log.Debugw("ApisixUpstream update event arrived",
		zap.Any("new object", curr),
		zap.Any("old object", prev),
	)

	c.workqueue.Add(&types.Event{
		Type: types.EventUpdate,
		Object: kube.ApisixUpstreamEvent{
			Key:          key,
			OldObject:    prev,
			GroupVersion: curr.GroupVersion(),
		},
	})

	c.MetricsCollector.IncrEvents("upstream", "update")
}

func (c *apisixUpstreamController) onDelete(obj interface{}) {
	au, err := kube.NewApisixUpstream(obj)
	if err != nil {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			return
		}
		au, err = kube.NewApisixUpstream(tombstone.Obj)
		if err != nil {
			log.Errorw("found ApisixUpstream resource with bad type", zap.Error(err))
			return
		}
	}

	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found ApisixUpstream resource with bad meta namespace key: %s", err)
		return
	}
	if !c.namespaceProvider.IsWatchingNamespace(key) {
		return
	}
	log.Debugw("ApisixUpstream delete event arrived",
		zap.Any("final state", au),
	)
	c.workqueue.Add(&types.Event{
		Type: types.EventDelete,
		Object: kube.ApisixUpstreamEvent{
			Key:          key,
			GroupVersion: au.GroupVersion(),
		},
		Tombstone: au,
	})

	c.MetricsCollector.IncrEvents("upstream", "delete")
}

func (c *apisixUpstreamController) ResourceSync() {
	objs := c.apisixUpstreamInformer.GetIndexer().List()
	for _, obj := range objs {
		key, err := cache.MetaNamespaceKeyFunc(obj)
		if err != nil {
			log.Errorw("ApisixUpstream sync failed, found ApisixUpstream resource with bad meta namespace key", zap.String("error", err.Error()))
			continue
		}
		if !c.namespaceProvider.IsWatchingNamespace(key) {
			continue
		}
		au, err := kube.NewApisixUpstream(obj)
		if err != nil {
			log.Errorw("ApisixUpstream sync failed, found ApisixUpstream resource with bad type", zap.Error(err))
			return
		}
		c.workqueue.Add(&types.Event{
			Type: types.EventAdd,
			Object: kube.ApisixUpstreamEvent{
				Key:          key,
				GroupVersion: au.GroupVersion(),
			},
		})
	}
}

// recordStatus record resources status
func (c *apisixUpstreamController) recordStatus(at interface{}, reason string, err error, status metav1.ConditionStatus, generation int64) {
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
	case *configv2beta3.ApisixUpstream:
		// set to status
		if v.Status.Conditions == nil {
			conditions := make([]metav1.Condition, 0)
			v.Status.Conditions = conditions
		}
		if utils.VerifyGeneration(&v.Status.Conditions, condition) {
			meta.SetStatusCondition(&v.Status.Conditions, condition)
			if _, errRecord := apisixClient.ApisixV2beta3().ApisixUpstreams(v.Namespace).
				UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
				log.Errorw("failed to record status change for ApisixUpstream",
					zap.Error(errRecord),
					zap.String("name", v.Name),
					zap.String("namespace", v.Namespace),
				)
			}
		}

	case *configv2.ApisixUpstream:
		// set to status
		if v.Status.Conditions == nil {
			conditions := make([]metav1.Condition, 0)
			v.Status.Conditions = conditions
		}
		if utils.VerifyGeneration(&v.Status.Conditions, condition) {
			meta.SetStatusCondition(&v.Status.Conditions, condition)
			if _, errRecord := apisixClient.ApisixV2().ApisixUpstreams(v.Namespace).
				UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
				log.Errorw("failed to record status change for ApisixUpstream",
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
