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
	"sync"
	"time"

	"go.uber.org/zap"
	"gopkg.in/go-playground/pool.v3"
	corev1 "k8s.io/api/core/v1"
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

type apisixTlsController struct {
	*apisixCommon

	workqueue workqueue.RateLimitingInterface
	workers   int
	pool      pool.Pool

	// secretRefMap stores reference from K8s secret to ApisixTls
	// type: Map<SecretKey, Map<ApisixTlsKey, empty struct>>
	// SecretKey -> ApisixTlsKey -> empty object in APISIX
	// SecretKey and ApisixTlsKey are kube-style meta key: `namespace/name`
	secretRefMap *sync.Map
}

func newApisixTlsController(common *apisixCommon) *apisixTlsController {
	c := &apisixTlsController{
		apisixCommon: common,
		workqueue:    workqueue.NewNamedRateLimitingQueue(workqueue.NewItemFastSlowRateLimiter(1*time.Second, 60*time.Second, 5), "ApisixTls"),
		workers:      1,
		pool:         pool.NewLimited(2),

		secretRefMap: new(sync.Map),
	}

	c.ApisixTlsInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.onAdd,
			UpdateFunc: c.onUpdate,
			DeleteFunc: c.onDelete,
		},
	)
	return c
}

func (c *apisixTlsController) run(ctx context.Context) {
	log.Info("ApisixTls controller started")
	defer log.Info("ApisixTls controller exited")
	defer c.workqueue.ShutDown()

	for i := 0; i < c.workers; i++ {
		go c.runWorker(ctx)
	}

	<-ctx.Done()
}

func (c *apisixTlsController) runWorker(ctx context.Context) {
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

func (c *apisixTlsController) sync(ctx context.Context, ev *types.Event) error {
	event := ev.Object.(kube.ApisixTlsEvent)
	apisixTlsKey := event.Key
	namespace, name, err := cache.SplitMetaNamespaceKey(apisixTlsKey)
	if err != nil {
		log.Errorf("found ApisixTls resource with invalid meta namespace key %s: %s", apisixTlsKey, err)
		return err
	}

	var multiVersionedTls kube.ApisixTls
	switch event.GroupVersion {
	case config.ApisixV2:
		multiVersionedTls, err = c.ApisixTlsLister.V2(namespace, name)
	default:
		return fmt.Errorf("unsupported ApisixTls group version %s", event.GroupVersion)
	}

	if err != nil {
		if !k8serrors.IsNotFound(err) {
			log.Errorw("failed to get ApisixTls",
				zap.Error(err),
				zap.String("key", apisixTlsKey),
				zap.String("version", event.GroupVersion),
			)
			return err
		}
		if ev.Type == types.EventSync {
			// ignore not found error in delay sync
			return nil
		}
		if ev.Type != types.EventDelete {
			log.Warnw("ApisixTls %s was deleted before it can be delivered",
				zap.String("key", apisixTlsKey),
				zap.String("version", event.GroupVersion),
			)
			// Don't need to retry.
			return nil
		}
	}
	if ev.Type == types.EventDelete {
		if multiVersionedTls != nil {
			// We still find the resource while we are processing the DELETE event,
			// that means object with same namespace and name was created, discarding
			// this stale DELETE event.
			log.Warnf("discard the stale ApisixTls delete event since the %s exists", apisixTlsKey)
			return nil
		}
		multiVersionedTls = ev.Tombstone.(kube.ApisixTls)
	}

	var errRecord error
	switch event.GroupVersion {
	case config.ApisixV2:
		tls := multiVersionedTls.V2()
		secretKey := tls.Spec.Secret.Namespace + "/" + tls.Spec.Secret.Name
		c.storeSecretCache(secretKey, apisixTlsKey, ev.Type)
		if tls.Spec.Client != nil {
			caSecretKey := tls.Spec.Client.CASecret.Namespace + "/" + tls.Spec.Client.CASecret.Name
			if caSecretKey != secretKey {
				c.storeSecretCache(caSecretKey, apisixTlsKey, ev.Type)
			}
		}

		ssl, err := c.translator.TranslateSSLV2(tls)
		if err != nil {
			log.Errorw("failed to translate ApisixTls",
				zap.Error(err),
				zap.Any("ApisixTls", tls),
			)
			errRecord = err
			goto updateStatus
		}
		log.Debugw("got SSL object from ApisixTls",
			zap.Any("ssl", ssl),
			zap.Any("ApisixTls", tls),
		)

		if err := c.SyncSSL(ctx, ssl, ev.Type); err != nil {
			log.Errorw("failed to sync SSL to APISIX",
				zap.Error(err),
				zap.Any("ssl", ssl),
			)
			errRecord = err
			goto updateStatus
		}
	}
updateStatus:
	c.pool.Queue(func(wu pool.WorkUnit) (interface{}, error) {
		if wu.IsCancelled() {
			return nil, nil
		}
		c.updateStatus(multiVersionedTls, errRecord)
		return true, nil
	})
	return errRecord
}

func (c *apisixTlsController) updateStatus(obj kube.ApisixTls, statusErr error) {
	if obj == nil || c.Kubernetes.DisableStatusUpdates || !c.Elector.IsLeader() {
		return
	}
	var (
		at        kube.ApisixTls
		err       error
		namespace = obj.GetNamespace()
		name      = obj.GetName()
	)

	switch obj.GroupVersion() {
	case config.ApisixV2:
		at, err = c.ApisixTlsLister.V2(namespace, name)
	}
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			log.Warnw("Failed to update status, unable to get ApisixTls",
				zap.Error(err),
				zap.String("name", name),
				zap.String("namespace", namespace),
			)
		}
		return
	}
	if at.ResourceVersion() != obj.ResourceVersion() {
		return
	}
	var (
		reason    = utils.ResourceSynced
		condition = metav1.ConditionTrue
		eventType = corev1.EventTypeNormal
	)
	if statusErr != nil {
		reason = utils.ResourceSyncAborted
		condition = metav1.ConditionFalse
		eventType = corev1.EventTypeWarning
	}
	switch obj.GroupVersion() {
	case config.ApisixV2:
		c.RecordEvent(obj.V2(), eventType, reason, statusErr)
		c.recordStatus(obj.V2(), reason, statusErr, condition, at.GetGeneration())
	}
}

func (c *apisixTlsController) storeSecretCache(secretKey string, apisixTlsKey string, evType types.EventType) {
	if refs, ok := c.secretRefMap.Load(secretKey); ok {
		refMap := refs.(*sync.Map)
		switch evType {
		case types.EventDelete:
			refMap.Delete(apisixTlsKey)
			c.secretRefMap.Store(secretKey, refMap)
		default:
			refMap.Store(apisixTlsKey, struct{}{})
			c.secretRefMap.Store(secretKey, refMap)
		}
	} else if evType != types.EventDelete {
		refMap := new(sync.Map)
		refMap.Store(apisixTlsKey, struct{}{})
		c.secretRefMap.Store(secretKey, refMap)
	}
}

func (c *apisixTlsController) handleSyncErr(obj interface{}, err error) {
	if err == nil {
		c.workqueue.Forget(obj)
		c.MetricsCollector.IncrSyncOperation("TLS", "success")
		return
	}

	event := obj.(*types.Event)
	ev := event.Object.(kube.ApisixTlsEvent)
	if k8serrors.IsNotFound(err) && event.Type != types.EventDelete {
		log.Infow("sync ApisixTls but not found, ignore",
			zap.String("event_type", event.Type.String()),
			zap.String("ApisixTls", ev.Key),
			zap.String("version", ev.GroupVersion),
		)
		c.workqueue.Forget(event)
		return
	}
	log.Warnw("sync ApisixTls failed, will retry",
		zap.Any("object", obj),
		zap.Error(err),
	)
	c.workqueue.AddRateLimited(obj)
	c.MetricsCollector.IncrSyncOperation("TLS", "failure")
}

func (c *apisixTlsController) onAdd(obj interface{}) {
	tls, err := kube.NewApisixTls(obj)
	if err != nil {
		log.Errorw("found ApisixTls resource with bad type", zap.Error(err))
		return
	}
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found ApisixTls object with bad namespace/name: %s, ignore it", err)
		return
	}
	if !c.namespaceProvider.IsWatchingNamespace(key) {
		return
	}
	if !c.isEffective(tls) {
		return
	}
	log.Debugw("ApisixTls add event arrived",
		zap.Any("object", obj),
	)

	c.workqueue.Add(&types.Event{
		Type: types.EventAdd,
		Object: kube.ApisixTlsEvent{
			Key:          key,
			GroupVersion: tls.GroupVersion(),
		},
	})

	c.MetricsCollector.IncrEvents("TLS", "add")
}

func (c *apisixTlsController) onUpdate(oldObj, newObj interface{}) {
	prev, err := kube.NewApisixTls(oldObj)
	if err != nil {
		log.Errorw("found ApisixTls resource with bad type", zap.Error(err))
		return
	}
	curr, err := kube.NewApisixTls(newObj)
	if err != nil {
		log.Errorw("found ApisixTls resource with bad type", zap.Error(err))
		return
	}
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
	key, err := cache.MetaNamespaceKeyFunc(curr)
	if err != nil {
		log.Errorf("found ApisixTls object with bad namespace/name: %s, ignore it", err)
		return
	}
	if !c.namespaceProvider.IsWatchingNamespace(key) {
		return
	}
	if !c.isEffective(curr) {
		return
	}
	log.Debugw("ApisixTls update event arrived",
		zap.Any("new object", curr),
		zap.Any("old object", prev),
	)

	c.workqueue.Add(&types.Event{
		Type: types.EventUpdate,
		Object: kube.ApisixTlsEvent{
			Key:          key,
			OldObject:    prev,
			GroupVersion: curr.GroupVersion(),
		},
	})

	c.MetricsCollector.IncrEvents("TLS", "update")
}

func (c *apisixTlsController) onDelete(obj interface{}) {
	tls, err := kube.NewApisixTls(obj)
	if err != nil {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			return
		}
		tls, err = kube.NewApisixTls(tombstone)
		if err != nil {
			log.Errorw("found ApisixTls resource with bad type", zap.Error(err))
			return
		}
	}
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found ApisixTls resource with bad meta namespace key: %s", err)
		return
	}
	if !c.namespaceProvider.IsWatchingNamespace(key) {
		return
	}
	if !c.isEffective(tls) {
		return
	}
	log.Debugw("ApisixTls delete event arrived",
		zap.Any("final state", obj),
	)
	c.workqueue.Add(&types.Event{
		Type: types.EventDelete,
		Object: kube.ApisixTlsEvent{
			Key:          key,
			GroupVersion: tls.GroupVersion(),
		},
		Tombstone: tls,
	})

	c.MetricsCollector.IncrEvents("TLS", "delete")
}

// ResourceSync syncs ApisixTls resources within namespace to workqueue.
// If namespace is "", it syncs all namespaces ApisixTls resources.
func (c *apisixTlsController) ResourceSync(interval time.Duration, namespace string) {
	objs := c.ApisixTlsInformer.GetIndexer().List()
	delay := GetSyncDelay(interval, len(objs))

	for i, obj := range objs {
		key, err := cache.MetaNamespaceKeyFunc(obj)
		if err != nil {
			log.Errorw("ApisixTls sync failed, found ApisixTls object with bad namespace/name ignore it", zap.String("error", err.Error()))
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
		tls, err := kube.NewApisixTls(obj)
		if err != nil {
			log.Errorw("ApisixTls sync failed, found ApisixTls resource with bad type", zap.Error(err))
			continue
		}
		log.Debugw("ResourceSync",
			zap.String("resource", "ApisixTls"),
			zap.String("key", key),
			zap.Duration("calc_delay", delay),
			zap.Int("i", i),
			zap.Duration("delay", delay*time.Duration(i)),
		)
		c.workqueue.AddAfter(&types.Event{
			Type: types.EventSync,
			Object: kube.ApisixTlsEvent{
				Key:          key,
				GroupVersion: tls.GroupVersion(),
			},
		}, delay*time.Duration(i))
	}
}

// recordStatus record resources status
func (c *apisixTlsController) recordStatus(at interface{}, reason string, err error, status metav1.ConditionStatus, generation int64) {
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
	case *configv2.ApisixTls:
		// set to status
		if v.Status.Conditions == nil {
			conditions := make([]metav1.Condition, 0)
			v.Status.Conditions = conditions
		}
		if utils.VerifyConditions(&v.Status.Conditions, condition) {
			meta.SetStatusCondition(&v.Status.Conditions, condition)
			if _, errRecord := apisixClient.ApisixV2().ApisixTlses(v.Namespace).
				UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
				log.Errorw("failed to record status change for ApisixTls",
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

func (c *apisixTlsController) SyncSecretChange(ctx context.Context, ev *types.Event, secret *corev1.Secret, secretKey string) {
	refs, ok := c.secretRefMap.Load(secretKey)
	if !ok {
		log.Debugw("ApisixTls: sync secret change, not concerned", zap.String("key", secretKey))
		// This secret is not concerned.
		return
	}

	refMap, ok := refs.(*sync.Map) // apisix tls key -> empty struct
	if !ok {
		log.Debugw("ApisixTls: sync secret change, not such SSls map", zap.String("key", secretKey))
		return
	}

	log.Debugw("ApisixTls: sync secret change", zap.String("key", secretKey))
	switch c.Config.Kubernetes.APIVersion {
	case config.ApisixV2:
		refMap.Range(func(k, v interface{}) bool {
			tlsMetaKey := k.(string)
			c.workqueue.Add(&types.Event{
				Type: types.EventSync,
				Object: kube.ApisixTlsEvent{
					Key:          tlsMetaKey,
					GroupVersion: config.ApisixV2,
				},
			})
			return true
		})
	}
}

func (c *apisixTlsController) isEffective(atls kube.ApisixTls) bool {
	if atls.GroupVersion() == config.ApisixV2 {
		var ingClassName string
		if atls.V2().Spec != nil {
			ingClassName = atls.V2().Spec.IngressClassName
		}
		return utils.MatchCRDsIngressClass(ingClassName, c.Kubernetes.IngressClass)
	}
	// Compatible with legacy versions
	return true
}
