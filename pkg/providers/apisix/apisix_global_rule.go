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
	"strconv"
	"time"

	"go.uber.org/zap"
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
	"github.com/apache/apisix-ingress-controller/pkg/providers/utils"
	"github.com/apache/apisix-ingress-controller/pkg/types"
)

type apisixGlobalRuleController struct {
	*apisixCommon

	workqueue workqueue.RateLimitingInterface
	workers   int
}

func newApisixGlobalRuleController(common *apisixCommon) *apisixGlobalRuleController {
	c := &apisixGlobalRuleController{
		apisixCommon: common,
		workqueue:    workqueue.NewNamedRateLimitingQueue(workqueue.NewItemFastSlowRateLimiter(1*time.Second, 60*time.Second, 5), "ApisixGlobalRule"),
		workers:      1,
	}

	c.ApisixGlobalRuleInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.onAdd,
			UpdateFunc: c.onUpdate,
			DeleteFunc: c.onDelete,
		},
	)
	return c
}

func (c *apisixGlobalRuleController) run(ctx context.Context) {
	log.Info("ApisixGlobalRule controller started")
	defer log.Info("ApisixGlobalRule controller exited")
	defer c.workqueue.ShutDown()

	for i := 0; i < c.workers; i++ {
		go c.runWorker(ctx)
	}
	<-ctx.Done()
}

func (c *apisixGlobalRuleController) runWorker(ctx context.Context) {
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

func (c *apisixGlobalRuleController) sync(ctx context.Context, ev *types.Event) error {
	obj := ev.Object.(kube.ApisixGlobalRuleEvent)
	namespace, name, err := cache.SplitMetaNamespaceKey(obj.Key)
	if err != nil {
		log.Errorf("invalid resource key: %s", obj.Key)
		return err
	}
	var (
		agr kube.ApisixGlobalRule
	)
	agr, err = c.ApisixGlobalRuleLister.ApisixGlobalRule(namespace, name)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			log.Errorw("failed to get ApisixGlobalRule",
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
			log.Warnw("ApisixGlobalRule was deleted before it can be delivered",
				zap.String("key", obj.Key),
				zap.String("version", obj.GroupVersion),
			)
			return nil
		}
	}
	if ev.Type == types.EventDelete {
		if agr != nil {
			// We still find the resource while we are processing the DELETE event,
			// that means object with same namespace and name was created, discarding
			// this stale DELETE event.
			log.Warnw("discard the stale ApisixGlobalRule delete event since the resource still exists",
				zap.String("key", obj.Key),
			)
			return nil
		}
		agr = ev.Tombstone.(kube.ApisixGlobalRule)
	}

	tctx, err := c.translator.TranslateGlobalRule(agr)
	if err != nil {
		log.Errorw("failed to translate ApisixGlobalRule v2",
			zap.Error(err),
			zap.Any("object", agr),
		)
		return err
	}

	m := &utils.Manifest{
		GlobalRules: tctx.GlobalRules,
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
		oldCtx, err := c.translator.TranslateGlobalRule(obj.OldObject)
		if err != nil {
			log.Errorw("failed to translate old ApisixGlobalRule",
				zap.String("version", obj.GroupVersion),
				zap.String("event", "update"),
				zap.Error(err),
				zap.Any("ApisixGlobalRule", agr),
			)
		} else {
			om := &utils.Manifest{
				GlobalRules: oldCtx.GlobalRules,
			}
			added, updated, deleted = m.Diff(om)
		}
	}
	log.Debugw("sync ApisixGlobalRule to cluster",
		zap.String("event_type", ev.Type.String()),
		zap.Any("add", added),
		zap.Any("update", updated),
		zap.Any("delete", deleted),
	)
	return c.SyncManifests(ctx, added, updated, deleted, ev.Type.IsSyncEvent())
}

func (c *apisixGlobalRuleController) handleSyncErr(obj interface{}, errOrigin error) {
	ev := obj.(*types.Event)
	event := ev.Object.(kube.ApisixGlobalRuleEvent)
	if k8serrors.IsNotFound(errOrigin) && ev.Type != types.EventDelete {
		log.Infow("sync ApisixGlobalRule but not found, ignore",
			zap.String("event_type", ev.Type.String()),
			zap.String("ApisixGlobalRule", ev.Object.(kube.ApisixGlobalRuleEvent).Key),
		)
		c.workqueue.Forget(event)
		return
	}
	if !c.Kubernetes.DisableStatusUpdates && c.Elector.IsLeader() {
		namespace, name, errLocal := cache.SplitMetaNamespaceKey(event.Key)
		if errLocal != nil {
			log.Errorf("invalid resource key: %s", event.Key)
			c.MetricsCollector.IncrSyncOperation("GlobalRule", "failure")
			return
		}
		var agr kube.ApisixGlobalRule
		switch event.GroupVersion {
		case config.ApisixV2:
			agr, errLocal = c.ApisixGlobalRuleLister.V2(namespace, name)
		default:
			errLocal = fmt.Errorf("unsupported ApisixGlobalRule group version %s", event.GroupVersion)
		}
		if errOrigin == nil {
			if ev.Type != types.EventDelete {
				if errLocal == nil {
					switch agr.GroupVersion() {
					case config.ApisixV2:
						c.RecordEvent(agr.V2(), v1.EventTypeNormal, utils.ResourceSynced, nil)
						c.recordStatus(agr.V2(), utils.ResourceSynced, nil, metav1.ConditionTrue, agr.GetGeneration())
					}
				} else {
					log.Errorw("failed list ApisixGlobalRule",
						zap.Error(errLocal),
						zap.String("name", name),
						zap.String("namespace", namespace),
					)
				}
			}
			c.workqueue.Forget(obj)
			c.MetricsCollector.IncrSyncOperation("GlobalRule", "success")
			return
		}
		log.Warnw("sync ApisixGlobalRule failed, will retry",
			zap.Any("object", obj),
			zap.Error(errOrigin),
		)
		if errLocal == nil {
			switch agr.GroupVersion() {
			case config.ApisixV2:
				c.RecordEvent(agr.V2(), v1.EventTypeWarning, utils.ResourceSyncAborted, errOrigin)
				c.recordStatus(agr.V2(), utils.ResourceSyncAborted, errOrigin, metav1.ConditionFalse, agr.GetGeneration())
			}
		} else {
			log.Errorw("failed list ApisixGlobalRule",
				zap.Error(errLocal),
				zap.String("name", name),
				zap.String("namespace", namespace),
			)
		}
	}
	c.workqueue.AddRateLimited(obj)
	c.MetricsCollector.IncrSyncOperation("GlobalRule", "failure")
}

func (c *apisixGlobalRuleController) onAdd(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found ApisixGlobalRule resource with bad meta namespace key: %s", err)
		return
	}
	agr := kube.MustNewApisixGlobalRule(obj)
	if !c.isEffective(agr) {
		return
	}
	if !c.namespaceProvider.IsWatchingNamespace(key) {
		return
	}
	log.Debugw("ApisixGlobalRule add event arrived",
		zap.Any("object", obj))

	c.workqueue.Add(&types.Event{
		Type: types.EventAdd,
		Object: kube.ApisixGlobalRuleEvent{
			Key:          key,
			GroupVersion: agr.GroupVersion(),
		},
	})

	c.MetricsCollector.IncrEvents("GlobalRule", "add")
}

func (c *apisixGlobalRuleController) onUpdate(oldObj, newObj interface{}) {
	prev := kube.MustNewApisixGlobalRule(oldObj)
	curr := kube.MustNewApisixGlobalRule(newObj)
	oldRV, _ := strconv.ParseInt(prev.ResourceVersion(), 0, 64)
	newRV, _ := strconv.ParseInt(curr.ResourceVersion(), 0, 64)
	if oldRV >= newRV {
		return
	}
	key, err := cache.MetaNamespaceKeyFunc(newObj)
	if err != nil {
		log.Errorf("found ApisixGlobalRule resource with bad meta namespace key: %s", err)
		return
	}
	if !c.isEffective(curr) {
		return
	}
	if !c.namespaceProvider.IsWatchingNamespace(key) {
		return
	}
	log.Debugw("ApisixGlobalRule update event arrived",
		zap.Any("new object", curr),
		zap.Any("old object", prev),
	)
	c.workqueue.Add(&types.Event{
		Type: types.EventUpdate,
		Object: kube.ApisixGlobalRuleEvent{
			Key:          key,
			GroupVersion: curr.GroupVersion(),
			OldObject:    prev,
		},
	})

	c.MetricsCollector.IncrEvents("GlobalRule", "update")
}

func (c *apisixGlobalRuleController) onDelete(obj interface{}) {
	agr, err := kube.NewApisixGlobalRule(obj)
	if err != nil {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			return
		}
		agr = kube.MustNewApisixGlobalRule(tombstone)
	}
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found ApisixGlobalRule resource with bad meta namesagre key: %s", err)
		return
	}
	if !c.isEffective(agr) {
		return
	}
	if !c.namespaceProvider.IsWatchingNamespace(key) {
		return
	}
	log.Debugw("ApisixGlobalRule delete event arrived",
		zap.Any("final state", agr),
	)

	c.workqueue.Add(&types.Event{
		Type: types.EventDelete,
		Object: kube.ApisixGlobalRuleEvent{
			Key:          key,
			GroupVersion: agr.GroupVersion(),
		},
		Tombstone: agr,
	})

	c.MetricsCollector.IncrEvents("GlobalRule", "delete")
}

// ResourceSync syncs ApisixGlobalRule resources within namespace to workqueue.
// If namespace is "", it syncs all namespaces ApisixGlobalRule resources.
func (c *apisixGlobalRuleController) ResourceSync(interval time.Duration, namespace string) {
	objs := c.ApisixGlobalRuleInformer.GetIndexer().List()
	delay := GetSyncDelay(interval, len(objs))

	for i, obj := range objs {
		key, err := cache.MetaNamespaceKeyFunc(obj)
		if err != nil {
			log.Errorw("ApisixGlobalRule sync failed, found ApisixGlobalRule resource with bad meta namespace key", zap.String("error", err.Error()))
			continue
		}
		agr := kube.MustNewApisixGlobalRule(obj)
		if !c.isEffective(agr) {
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
			zap.String("resource", "ApisixGlobalRule"),
			zap.String("key", key),
			zap.Duration("calc_delay", delay),
			zap.Int("i", i),
			zap.Duration("delay", delay*time.Duration(i)),
		)
		c.workqueue.AddAfter(&types.Event{
			Type: types.EventSync,
			Object: kube.ApisixGlobalRuleEvent{
				Key:          key,
				GroupVersion: agr.GroupVersion(),
			},
		}, delay*time.Duration(i))
	}
}

// recordStatus record resources status
func (c *apisixGlobalRuleController) recordStatus(at interface{}, reason string, err error, status metav1.ConditionStatus, generation int64) {
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
	case *configv2.ApisixGlobalRule:
		// set to status
		if v.Status.Conditions == nil {
			conditions := make([]metav1.Condition, 0)
			v.Status.Conditions = conditions
		}
		if utils.VerifyGeneration(&v.Status.Conditions, condition) && !meta.IsStatusConditionPresentAndEqual(v.Status.Conditions, condition.Type, condition.Status) {
			meta.SetStatusCondition(&v.Status.Conditions, condition)
			if _, errRecord := apisixClient.ApisixV2().ApisixGlobalRules(v.Namespace).
				UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
				log.Errorw("failed to record status change for ApisixGlobalRule",
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

func (c *apisixGlobalRuleController) isEffective(agr kube.ApisixGlobalRule) bool {
	if agr.GroupVersion() == config.ApisixV2 {
		ingClassName := agr.V2().Spec.IngressClassName
		ok := utils.MatchCRDsIngressClass(ingClassName, c.Kubernetes.IngressClass)
		if !ok {
			log.Debugw("IngressClass: ApisixGlobalRule ignored",
				zap.String("key", agr.V2().Namespace+"/"+agr.V2().Name),
				zap.String("ingressClass", agr.V2().Spec.IngressClassName),
			)
		}

		return ok
	}
	// Compatible with legacy versions
	return true
}
