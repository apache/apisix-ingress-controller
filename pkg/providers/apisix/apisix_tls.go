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
	"sync"
	"time"

	"go.uber.org/zap"
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
	configv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta3"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/providers/utils"
	"github.com/apache/apisix-ingress-controller/pkg/types"
	v1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

type apisixTlsController struct {
	*apisixCommon

	workqueue workqueue.RateLimitingInterface
	workers   int

	secretInformer    cache.SharedIndexInformer
	apisixTlsLister   kube.ApisixTlsLister
	apisixTlsInformer cache.SharedIndexInformer

	// this map enrolls which ApisixTls objects refer to a Kubernetes
	// Secret object.
	// type: Map<SecretKey, Map<ApisixTlsKey, ApisixTls>>
	// SecretKey is `namespace_name`, ApisixTlsKey is kube style meta key: `namespace/name`
	secretSSLMap *sync.Map
}

func newApisixTlsController(common *apisixCommon) *apisixTlsController {
	c := &apisixTlsController{
		apisixCommon: common,
		workqueue:    workqueue.NewNamedRateLimitingQueue(workqueue.NewItemFastSlowRateLimiter(1*time.Second, 60*time.Second, 5), "ApisixTls"),
		workers:      1,

		secretInformer:    common.SecretInformer,
		apisixTlsLister:   common.ApisixTlsLister,
		apisixTlsInformer: common.ApisixTlsInformer,

		secretSSLMap: new(sync.Map),
	}

	c.apisixTlsInformer.AddEventHandler(
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

	if ok := cache.WaitForCacheSync(ctx.Done(), c.apisixTlsInformer.HasSynced, c.secretInformer.HasSynced); !ok {
		log.Errorf("informers sync failed")
		return
	}
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
	key := event.Key
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		log.Errorf("found ApisixTls resource with invalid meta namespace key %s: %s", key, err)
		return err
	}

	var multiVersionedTls kube.ApisixTls
	switch event.GroupVersion {
	case config.ApisixV2beta3:
		multiVersionedTls, err = c.apisixTlsLister.V2beta3(namespace, name)
	case config.ApisixV2:
		multiVersionedTls, err = c.apisixTlsLister.V2(namespace, name)
	default:
		return fmt.Errorf("unsupported ApisixTls group version %s", event.GroupVersion)
	}

	if err != nil {
		if !k8serrors.IsNotFound(err) {
			log.Errorw("failed to get ApisixTls",
				zap.Error(err),
				zap.String("key", key),
				zap.String("version", event.GroupVersion),
			)
			return err
		}
		if ev.Type != types.EventDelete {
			log.Warnw("ApisixTls %s was deleted before it can be delivered",
				zap.String("key", key),
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
			log.Warnf("discard the stale ApisixTls delete event since the %s exists", key)
			return nil
		}
		multiVersionedTls = ev.Tombstone.(kube.ApisixTls)
	}

	switch event.GroupVersion {
	case config.ApisixV2beta3:
		tls := multiVersionedTls.V2beta3()
		ssl, err := c.translator.TranslateSSLV2Beta3(tls)
		if err != nil {
			log.Errorw("failed to translate ApisixTls",
				zap.Error(err),
				zap.Any("ApisixTls", tls),
			)
			c.RecordEvent(tls, corev1.EventTypeWarning, utils.ResourceSyncAborted, err)
			c.recordStatus(tls, utils.ResourceSyncAborted, err, metav1.ConditionFalse, tls.GetGeneration())
			return err
		}
		log.Debugw("got SSL object from ApisixTls",
			zap.Any("ssl", ssl),
			zap.Any("ApisixTls", tls),
		)

		secretKey := tls.Spec.Secret.Namespace + "_" + tls.Spec.Secret.Name
		c.syncSecretSSL(secretKey, key, ssl, ev.Type)
		if tls.Spec.Client != nil {
			caSecretKey := tls.Spec.Client.CASecret.Namespace + "_" + tls.Spec.Client.CASecret.Name
			if caSecretKey != secretKey {
				c.syncSecretSSL(caSecretKey, key, ssl, ev.Type)
			}
		}

		if err := c.SyncSSL(ctx, ssl, ev.Type); err != nil {
			log.Errorw("failed to sync SSL to APISIX",
				zap.Error(err),
				zap.Any("ssl", ssl),
			)
			c.RecordEvent(tls, corev1.EventTypeWarning, utils.ResourceSyncAborted, err)
			c.recordStatus(tls, utils.ResourceSyncAborted, err, metav1.ConditionFalse, tls.GetGeneration())
			return err
		}
		c.RecordEvent(tls, corev1.EventTypeNormal, utils.ResourceSynced, nil)
		c.recordStatus(tls, utils.ResourceSynced, nil, metav1.ConditionTrue, tls.GetGeneration())
		return err
	case config.ApisixV2:
		tls := multiVersionedTls.V2()
		ssl, err := c.translator.TranslateSSLV2(tls)
		if err != nil {
			log.Errorw("failed to translate ApisixTls",
				zap.Error(err),
				zap.Any("ApisixTls", tls),
			)
			c.RecordEvent(tls, corev1.EventTypeWarning, utils.ResourceSyncAborted, err)
			c.recordStatus(tls, utils.ResourceSyncAborted, err, metav1.ConditionFalse, tls.GetGeneration())
			return err
		}
		log.Debugw("got SSL object from ApisixTls",
			zap.Any("ssl", ssl),
			zap.Any("ApisixTls", tls),
		)

		secretKey := tls.Spec.Secret.Namespace + "_" + tls.Spec.Secret.Name
		c.syncSecretSSL(secretKey, key, ssl, ev.Type)
		if tls.Spec.Client != nil {
			caSecretKey := tls.Spec.Client.CASecret.Namespace + "_" + tls.Spec.Client.CASecret.Name
			if caSecretKey != secretKey {
				c.syncSecretSSL(caSecretKey, key, ssl, ev.Type)
			}
		}

		if err := c.SyncSSL(ctx, ssl, ev.Type); err != nil {
			log.Errorw("failed to sync SSL to APISIX",
				zap.Error(err),
				zap.Any("ssl", ssl),
			)
			c.RecordEvent(tls, corev1.EventTypeWarning, utils.ResourceSyncAborted, err)
			c.recordStatus(tls, utils.ResourceSyncAborted, err, metav1.ConditionFalse, tls.GetGeneration())
			return err
		}
		c.RecordEvent(tls, corev1.EventTypeNormal, utils.ResourceSynced, nil)
		c.recordStatus(tls, utils.ResourceSynced, nil, metav1.ConditionTrue, tls.GetGeneration())
		return err
	default:
		return fmt.Errorf("unsupported ApisixTls group version %s", event.GroupVersion)
	}
}

func (c *apisixTlsController) syncSecretSSL(secretKey string, apisixTlsKey string, ssl *v1.Ssl, event types.EventType) {
	if ssls, ok := c.secretSSLMap.Load(secretKey); ok {
		sslMap := ssls.(*sync.Map)
		switch event {
		case types.EventDelete:
			sslMap.Delete(apisixTlsKey)
			c.secretSSLMap.Store(secretKey, sslMap)
		default:
			sslMap.Store(apisixTlsKey, ssl)
			c.secretSSLMap.Store(secretKey, sslMap)
		}
	} else if event != types.EventDelete {
		sslMap := new(sync.Map)
		sslMap.Store(apisixTlsKey, ssl)
		c.secretSSLMap.Store(secretKey, sslMap)
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

func (c *apisixTlsController) onUpdate(prev, curr interface{}) {
	oldTls, err := kube.NewApisixTls(prev)
	if err != nil {
		log.Errorw("found ApisixTls resource with bad type", zap.Error(err))
		return
	}
	newTls, err := kube.NewApisixTls(curr)
	if err != nil {
		log.Errorw("found ApisixTls resource with bad type", zap.Error(err))
		return
	}
	if oldTls.ResourceVersion() >= newTls.ResourceVersion() {
		return
	}
	key, err := cache.MetaNamespaceKeyFunc(curr)
	if err != nil {
		log.Errorf("found ApisixTls object with bad namespace/name: %s, ignore it", err)
		return
	}
	if !c.namespaceProvider.IsWatchingNamespace(key) {
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
			OldObject:    oldTls,
			GroupVersion: newTls.GroupVersion(),
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

func (c *apisixTlsController) ResourceSync() {
	objs := c.apisixTlsInformer.GetIndexer().List()
	for _, obj := range objs {
		key, err := cache.MetaNamespaceKeyFunc(obj)
		if err != nil {
			log.Errorw("ApisixTls sync failed, found ApisixTls object with bad namespace/name ignore it", zap.String("error", err.Error()))
			continue
		}
		if !c.namespaceProvider.IsWatchingNamespace(key) {
			continue
		}
		tls, err := kube.NewApisixTls(obj)
		if err != nil {
			log.Errorw("ApisixTls sync failed, found ApisixTls resource with bad type", zap.Error(err))
			continue
		}
		c.workqueue.Add(&types.Event{
			Type: types.EventAdd,
			Object: kube.ApisixTlsEvent{
				Key:          key,
				GroupVersion: tls.GroupVersion(),
			},
		})
	}
}

// recordStatus record resources status
func (c *apisixTlsController) recordStatus(at interface{}, reason string, err error, status metav1.ConditionStatus, generation int64) {
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
	case *configv2beta3.ApisixTls:
		// set to status
		if v.Status.Conditions == nil {
			conditions := make([]metav1.Condition, 0)
			v.Status.Conditions = conditions
		}
		if utils.VerifyGeneration(&v.Status.Conditions, condition) {
			meta.SetStatusCondition(&v.Status.Conditions, condition)
			if _, errRecord := apisixClient.ApisixV2beta3().ApisixTlses(v.Namespace).
				UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
				log.Errorw("failed to record status change for ApisixTls",
					zap.Error(errRecord),
					zap.String("name", v.Name),
					zap.String("namespace", v.Namespace),
				)
			}
		}
	case *configv2.ApisixTls:
		// set to status
		if v.Status.Conditions == nil {
			conditions := make([]metav1.Condition, 0)
			v.Status.Conditions = conditions
		}
		if utils.VerifyGeneration(&v.Status.Conditions, condition) {
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
