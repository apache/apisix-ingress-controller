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
package apisix

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"go.uber.org/zap"
	"gopkg.in/go-playground/pool.v3"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/kube"
	configv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/providers/translation"
	"github.com/apache/apisix-ingress-controller/pkg/providers/utils"
	"github.com/apache/apisix-ingress-controller/pkg/types"
)

type apisixPluginConfigController struct {
	*apisixCommon

	workqueue workqueue.RateLimitingInterface
	workers   int
	pool      pool.Pool
}

func newApisixPluginConfigController(common *apisixCommon) *apisixPluginConfigController {
	c := &apisixPluginConfigController{
		apisixCommon: common,
		workqueue:    workqueue.NewNamedRateLimitingQueue(workqueue.NewItemFastSlowRateLimiter(1*time.Second, 60*time.Second, 5), "ApisixPluginConfig"),
		workers:      1,
		pool:         pool.NewLimited(1),
	}

	c.ApisixPluginConfigInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.onAdd,
			UpdateFunc: c.onUpdate,
			DeleteFunc: c.onDelete,
		},
	)
	return c
}

func (c *apisixPluginConfigController) run(ctx context.Context) {
	log.Info("ApisixPluginConfig controller started")
	defer log.Info("ApisixPluginConfig controller exited")
	defer c.workqueue.ShutDown()

	for i := 0; i < c.workers; i++ {
		go c.runWorker(ctx)
	}
	<-ctx.Done()
}

func (c *apisixPluginConfigController) runWorker(ctx context.Context) {
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

func (c *apisixPluginConfigController) sync(ctx context.Context, ev *types.Event) error {
	obj := ev.Object.(kube.ApisixPluginConfigEvent)
	namespace, name, err := cache.SplitMetaNamespaceKey(obj.Key)
	if err != nil {
		log.Errorf("invalid resource key: %s", obj.Key)
		return err
	}
	var (
		apc  kube.ApisixPluginConfig
		tctx *translation.TranslateContext
	)
	switch obj.GroupVersion {
	case config.ApisixV2:
		apc, err = c.ApisixPluginConfigLister.V2(namespace, name)
	default:
		return fmt.Errorf("unsupported ApisixPluginConfig group version %s", obj.GroupVersion)
	}
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			log.Errorw("failed to get ApisixPluginConfig",
				zap.String("version", obj.GroupVersion),
				zap.String("key", obj.Key),
				zap.Error(err),
			)
			return err
		}

		if ev.Type == types.EventSync {
			// ignore not found error in delay sync
			return nil
		}
		if ev.Type != types.EventDelete {
			log.Warnw("ApisixPluginConfig was deleted before it can be delivered",
				zap.String("key", obj.Key),
				zap.String("version", obj.GroupVersion),
			)
			return nil
		}
	}
	if ev.Type == types.EventDelete {
		if apc != nil {
			// We still find the resource while we are processing the DELETE event,
			// that means object with same namespace and name was created, discarding
			// this stale DELETE event.
			log.Warnw("discard the stale ApisixPluginConfig delete event since the resource still exists",
				zap.String("key", obj.Key),
			)
			return nil
		}
		apc = ev.Tombstone.(kube.ApisixPluginConfig)
	}
	// translator phase: translate resource, construction data plance context
	var errRecord error
	{
		switch obj.GroupVersion {
		case config.ApisixV2:
			if ev.Type != types.EventDelete {
				tctx, err = c.translator.TranslatePluginConfigV2(apc.V2())
			} else {
				tctx, err = c.translator.GeneratePluginConfigV2DeleteMark(apc.V2())
			}
			if err != nil {
				log.Errorw("failed to translate ApisixPluginConfig v2",
					zap.Error(err),
					zap.Any("object", apc),
				)
				errRecord = err
				goto updatestatus
			}
		}

	}
	// sync phase: Use context update data palne
	{
		log.Debugw("translated ApisixPluginConfig",
			zap.Any("pluginConfigs", tctx.PluginConfigs),
		)
		m := &utils.Manifest{
			PluginConfigs: tctx.PluginConfigs,
		}

		var (
			added   *utils.Manifest
			updated *utils.Manifest
			deleted *utils.Manifest
		)

		if ev.Type == types.EventDelete {
			deleted = m
		} else if ev.Type.IsAddEvent() {
			added = m
		} else {
			var oldCtx *translation.TranslateContext
			switch obj.GroupVersion {
			case config.ApisixV2:
				oldCtx, err = c.translator.TranslatePluginConfigV2(obj.OldObject.V2())
			}
			if err != nil {
				log.Errorw("failed to translate old ApisixPluginConfig",
					zap.String("version", obj.GroupVersion),
					zap.String("event", "update"),
					zap.Error(err),
					zap.Any("ApisixPluginConfig", apc),
				)
				errRecord = err
				goto updatestatus
			}

			om := &utils.Manifest{
				PluginConfigs: oldCtx.PluginConfigs,
			}
			added, updated, deleted = m.Diff(om)
		}

		if err := c.SyncManifests(ctx, added, updated, deleted, ev.Type.IsSyncEvent()); err != nil {
			log.Errorw("failed to sync ApisixPluginConfig to apisix",
				zap.Error(err),
			)
			errRecord = err
			goto updatestatus
		}
	}
updatestatus:
	c.pool.Queue(func(wu pool.WorkUnit) (interface{}, error) {
		if wu.IsCancelled() {
			return nil, nil
		}
		c.updateStatus(apc, errRecord)
		return true, nil
	})
	return errRecord
}

func (c *apisixPluginConfigController) updateStatus(obj kube.ApisixPluginConfig, statusErr error) {
	if obj == nil || c.Kubernetes.DisableStatusUpdates || !c.Elector.IsLeader() {
		return
	}
	var (
		apc       kube.ApisixPluginConfig
		err       error
		namespace = obj.GetNamespace()
		name      = obj.GetName()
	)

	switch obj.GroupVersion() {
	case config.ApisixV2:
		apc, err = c.ApisixPluginConfigLister.V2(namespace, name)
	}
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			log.Warnw("failed to update status, unable to get ApisixPluginConfig",
				zap.Error(err),
				zap.String("name", name),
				zap.String("namespace", namespace),
			)
		}
		return
	}
	if apc.ResourceVersion() != obj.ResourceVersion() {
		return
	}
	var (
		reason    = utils.ResourceSynced
		condition = metav1.ConditionTrue
	)
	if statusErr != nil {
		reason = utils.ResourceSyncAborted
		condition = metav1.ConditionFalse
	}
	switch obj.GroupVersion() {
	case config.ApisixV2:
		c.RecordEvent(apc.V2(), v1.EventTypeNormal, reason, statusErr)
		c.recordStatus(apc.V2(), reason, statusErr, condition, apc.GetGeneration())
	}
}

func (c *apisixPluginConfigController) handleSyncErr(obj interface{}, errOrigin error) {
	if errOrigin == nil {
		c.workqueue.Forget(obj)
		c.MetricsCollector.IncrSyncOperation("PluginConfig", "success")
		return
	}
	ev := obj.(*types.Event)
	if k8serrors.IsNotFound(errOrigin) && ev.Type != types.EventDelete {
		log.Infow("sync ApisixPluginConfig but not found, ignore",
			zap.String("event_type", ev.Type.String()),
			zap.String("ApisixPluginConfig", ev.Object.(kube.ApisixPluginConfigEvent).Key),
		)
		c.workqueue.Forget(obj)
		return
	}
	log.Warnw("sync ApisixPluginConfig failed, will retry",
		zap.Any("object", obj),
		zap.Error(errOrigin),
	)
	c.workqueue.Forget(obj)
	c.MetricsCollector.IncrSyncOperation("PluginConfig", "failure")
}

func (c *apisixPluginConfigController) onAdd(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found ApisixPluginConfig resource with bad meta namespace key: %s", err)
		return
	}
	apc := kube.MustNewApisixPluginConfig(obj)
	if !c.isEffective(apc) {
		return
	}
	if !c.namespaceProvider.IsWatchingNamespace(key) {
		return
	}
	log.Debugw("ApisixPluginConfig add event arrived",
		zap.Any("object", obj))

	c.workqueue.Add(&types.Event{
		Type: types.EventAdd,
		Object: kube.ApisixPluginConfigEvent{
			Key:          key,
			GroupVersion: apc.GroupVersion(),
		},
	})

	c.MetricsCollector.IncrEvents("PluginConfig", "add")
}

func (c *apisixPluginConfigController) onUpdate(oldObj, newObj interface{}) {
	prev := kube.MustNewApisixPluginConfig(oldObj)
	curr := kube.MustNewApisixPluginConfig(newObj)
	oldRV, _ := strconv.ParseInt(prev.ResourceVersion(), 0, 64)
	newRV, _ := strconv.ParseInt(curr.ResourceVersion(), 0, 64)
	if oldRV >= newRV {
		return
	}
	// Updates triggered by status are ignored.
	if prev.GetGeneration() == curr.GetGeneration() && prev.GetUID() == curr.GetUID() {
		switch curr.GroupVersion() {
		case config.ApisixV2:
			if reflect.DeepEqual(prev.V2().Spec, curr.V2().Spec) && !reflect.DeepEqual(prev.V2().Status, curr.V2().Status) {
				return
			}
		}
	}
	key, err := cache.MetaNamespaceKeyFunc(newObj)
	if err != nil {
		log.Errorf("found ApisixPluginConfig resource with bad meta namespace key: %s", err)
		return
	}
	if !c.isEffective(curr) {
		return
	}
	if !c.namespaceProvider.IsWatchingNamespace(key) {
		return
	}
	log.Debugw("ApisixPluginConfig update event arrived",
		zap.Any("new object", curr),
		zap.Any("old object", prev),
	)
	c.workqueue.Add(&types.Event{
		Type: types.EventUpdate,
		Object: kube.ApisixPluginConfigEvent{
			Key:          key,
			GroupVersion: curr.GroupVersion(),
			OldObject:    prev,
		},
	})

	c.MetricsCollector.IncrEvents("PluginConfig", "update")
}

func (c *apisixPluginConfigController) onDelete(obj interface{}) {
	apc, err := kube.NewApisixPluginConfig(obj)
	if err != nil {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			return
		}
		apc = kube.MustNewApisixPluginConfig(tombstone)
	}
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found ApisixPluginConfig resource with bad meta namesapce key: %s", err)
		return
	}
	if !c.isEffective(apc) {
		return
	}
	if !c.namespaceProvider.IsWatchingNamespace(key) {
		return
	}
	log.Debugw("ApisixPluginConfig delete event arrived",
		zap.Any("final state", apc),
	)

	c.workqueue.Add(&types.Event{
		Type: types.EventDelete,
		Object: kube.ApisixPluginConfigEvent{
			Key:          key,
			GroupVersion: apc.GroupVersion(),
		},
		Tombstone: apc,
	})

	c.MetricsCollector.IncrEvents("PluginConfig", "delete")
}

// ResourceSync syncs ApisixPluginConfig resources within namespace to workqueue.
// If namespace is "", it syncs all namespaces ApisixPluginConfig resources.
func (c *apisixPluginConfigController) ResourceSync(interval time.Duration, namespace string) {
	objs := c.ApisixPluginConfigInformer.GetIndexer().List()
	delay := GetSyncDelay(interval, len(objs))

	for i, obj := range objs {
		key, err := cache.MetaNamespaceKeyFunc(obj)
		if err != nil {
			log.Errorw("ApisixPluginConfig sync failed, found ApisixPluginConfig resource with bad meta namespace key", zap.String("error", err.Error()))
			continue
		}
		apc := kube.MustNewApisixPluginConfig(obj)
		if !c.isEffective(apc) {
			continue
		}
		if !c.namespaceProvider.IsWatchingNamespace(key) {
			continue
		}
		ns, _, err := cache.SplitMetaNamespaceKey(key)
		if err != nil {
			log.Errorw("split ApisixRoute meta key failed",
				zap.Error(err),
				zap.String("key", key),
			)
			continue
		}
		if namespace != "" && ns != namespace {
			continue
		}
		log.Debugw("ResourceSync",
			zap.String("resource", "ApisixPluginConfig"),
			zap.String("key", key),
			zap.Duration("calc_delay", delay),
			zap.Int("i", i),
			zap.Duration("delay", delay*time.Duration(i)),
		)
		c.workqueue.AddAfter(&types.Event{
			Type: types.EventSync,
			Object: kube.ApisixPluginConfigEvent{
				Key:          key,
				GroupVersion: apc.GroupVersion(),
			},
		}, delay*time.Duration(i))
	}
}

// recordStatus record resources status
func (c *apisixPluginConfigController) recordStatus(at interface{}, reason string, err error, status metav1.ConditionStatus, generation int64) {
	if c.Kubernetes.DisableStatusUpdates {
		return
	}
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
	case *configv2.ApisixPluginConfig:
		// set to status
		if v.Status.Conditions == nil {
			conditions := make([]metav1.Condition, 0)
			v.Status.Conditions = conditions
		}
		if utils.VerifyConditions(&v.Status.Conditions, condition) {
			meta.SetStatusCondition(&v.Status.Conditions, condition)
			if _, errRecord := apisixClient.ApisixV2().ApisixPluginConfigs(v.Namespace).
				UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
				log.Errorw("failed to record status change for ApisixPluginConfig",
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

func (c *apisixPluginConfigController) isEffective(apc kube.ApisixPluginConfig) bool {
	if apc.GroupVersion() == config.ApisixV2 {
		ingClassName := apc.V2().Spec.IngressClassName
		ok := utils.MatchCRDsIngressClass(ingClassName, c.Kubernetes.IngressClass)
		if !ok {
			log.Debugw("IngressClass: ApisixPluginConfig ignored",
				zap.String("key", apc.V2().Namespace+"/"+apc.V2().Name),
				zap.String("ingressClass", apc.V2().Spec.IngressClassName),
			)
		}

		return ok
	}
	// Compatible with legacy versions
	return true
}
