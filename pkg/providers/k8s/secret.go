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

package k8s

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/kube"
	configv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	configv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta3"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/providers/k8s/namespace"
	"github.com/apache/apisix-ingress-controller/pkg/providers/translation"
	providertypes "github.com/apache/apisix-ingress-controller/pkg/providers/types"
	"github.com/apache/apisix-ingress-controller/pkg/providers/utils"
	"github.com/apache/apisix-ingress-controller/pkg/types"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

type secretController struct {
	*providertypes.Common

	workqueue workqueue.RateLimitingInterface
	workers   int

	secretInformer  cache.SharedIndexInformer
	secretLister    listerscorev1.SecretLister
	apisixTlsLister kube.ApisixTlsLister

	NamespaceProvider namespace.WatchingNamespaceProvider

	translator translation.Translator
}

func newSecretController() *secretController {
	ctl := &secretController{
		workqueue: workqueue.NewNamedRateLimitingQueue(workqueue.NewItemFastSlowRateLimiter(1*time.Second, 60*time.Second, 5), "Secrets"),
		workers:   1,
	}

	ctl.secretInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctl.onAdd,
			UpdateFunc: ctl.onUpdate,
			DeleteFunc: ctl.onDelete,
		},
	)

	return ctl
}

func (c *secretController) run(ctx context.Context) {
	log.Info("secret controller started")
	defer log.Info("secret controller exited")
	defer c.workqueue.ShutDown()

	if ok := cache.WaitForCacheSync(ctx.Done(), c.secretInformer.HasSynced); !ok {
		log.Error("informers sync failed")
		return
	}

	for i := 0; i < c.workers; i++ {
		go c.runWorker(ctx)
	}

	<-ctx.Done()
}

func (c *secretController) runWorker(ctx context.Context) {
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

func (c *secretController) sync(ctx context.Context, ev *types.Event) error {
	key := ev.Object.(string)
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		log.Errorf("invalid resource key: %s", key)
		return err
	}
	sec, err := c.secretLister.Secrets(namespace).Get(name)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			log.Errorw("failed to get Secret",
				zap.String("key", key),
				zap.Error(err),
			)
			return err
		}

		if ev.Type != types.EventDelete {
			log.Warnw("Secret was deleted before it can be delivered",
				zap.String("key", key),
			)
			return nil
		}
	}
	if ev.Type == types.EventDelete {
		if sec != nil {
			// We still find the resource while we are processing the DELETE event,
			// that means object with same namespace and name was created, discarding
			// this stale DELETE event.
			log.Warnw("discard the stale secret delete event since the resource still exists",
				zap.String("key", key),
			)
			return nil
		}
		sec = ev.Tombstone.(*corev1.Secret)
	}

	secretMapKey := namespace + "_" + name
	// sync SSL in APISIX which is store in secretSSLMap
	ssls, ok := c.secretSSLMap.Load(secretMapKey)
	if !ok {
		// This secret is not concerned.
		return nil
	}
	sslMap := ssls.(*sync.Map)

	switch c.Kubernetes.APIVersion {
	case config.ApisixV2beta3:
		sslMap.Range(c.syncV2Beta3Handler(ctx, ev, sec, key))
	case config.ApisixV2:
		sslMap.Range(c.syncV2Handler(ctx, ev, sec, key))
	}
	return err
}

func (c *secretController) syncV2Beta3Handler(ctx context.Context, ev *types.Event, secret *corev1.Secret, secretKey string) func(k, v interface{}) bool {
	return func(k, v interface{}) bool {
		ssl := v.(*apisixv1.Ssl)
		tlsMetaKey := k.(string)
		tlsNamespace, tlsName, err := cache.SplitMetaNamespaceKey(tlsMetaKey)
		if err != nil {
			log.Errorf("invalid cached ApisixTls key: %s", tlsMetaKey)
			return true
		}

		multiVersioned, err := c.apisixTlsLister.V2beta3(tlsNamespace, tlsName)
		if err != nil {
			log.Warnw("secret related ApisixTls resource not found, skip",
				zap.String("ApisixTls", tlsMetaKey),
			)
			return true
		}
		tls := multiVersioned.V2beta3()

		// We don't expect a secret to be used as both SSL and mTLS in ApisixTls
		if tls.Spec.Secret.Namespace == secret.Namespace && tls.Spec.Secret.Name == secret.Name {
			cert, pkey, err := translation.ExtractKeyPair(secret, true)
			if err != nil {
				log.Errorw("secret required by ApisixTls invalid",
					zap.String("ApisixTls", tlsMetaKey),
					zap.Error(err),
				)
				go func(tls *configv2beta3.ApisixTls) {
					c.RecordEventS(tls, corev1.EventTypeWarning, utils.ResourceSyncAborted,
						fmt.Sprintf("sync from secret %s changes failed, error: %s", secretKey, err.Error()))
					c.controller.recordStatus(tls, utils.ResourceSyncAborted, err, metav1.ConditionFalse, tls.GetGeneration())
				}(tls)
				return true
			}
			// sync ssl
			ssl.Cert = string(cert)
			ssl.Key = string(pkey)
		} else if tls.Spec.Client != nil &&
			tls.Spec.Client.CASecret.Namespace == secret.Namespace && tls.Spec.Client.CASecret.Name == secret.Name {
			ca, _, err := translation.ExtractKeyPair(secret, false)
			if err != nil {
				log.Errorw("ca secret required by ApisixTls invalid",
					zap.String("ApisixTls", tlsMetaKey),
					zap.Error(err),
				)
				go func(tls *configv2beta3.ApisixTls) {
					c.RecordEventS(tls, corev1.EventTypeWarning, utils.ResourceSyncAborted,
						fmt.Sprintf("sync from ca secret %s changes failed, error: %s", secretKey, err.Error()))
					c.controller.recordStatus(tls, utils.ResourceSyncAborted, err, metav1.ConditionFalse, tls.GetGeneration())
				}(tls)
				return true
			}
			ssl.Client = &apisixv1.MutualTLSClientConfig{
				CA: string(ca),
			}
		} else {
			log.Warnw("stale secret cache, ApisixTls doesn't requires target secret",
				zap.String("ApisixTls", tlsMetaKey),
				zap.String("secret", secretKey),
			)
			return true
		}
		// Use another goroutine to send requests, to avoid
		// long time lock occupying.
		go func(ssl *apisixv1.Ssl, tls *configv2beta3.ApisixTls) {
			err := c.SyncSSL(ctx, ssl, ev.Type)
			if err != nil {
				log.Errorw("failed to sync ssl to APISIX",
					zap.Error(err),
					zap.Any("ssl", ssl),
					zap.Any("secret", secret),
				)
				c.RecordEventS(tls, corev1.EventTypeWarning, utils.ResourceSyncAborted,
					fmt.Sprintf("sync from secret %s changes failed, error: %s", secretKey, err.Error()))
				c.controller.recordStatus(tls, utils.ResourceSyncAborted, err, metav1.ConditionFalse, tls.GetGeneration())
			} else {
				c.RecordEventS(tls, corev1.EventTypeNormal, utils.ResourceSynced,
					fmt.Sprintf("sync from secret %s changes", secretKey))
				c.controller.recordStatus(tls, utils.ResourceSynced, nil, metav1.ConditionTrue, tls.GetGeneration())
			}
		}(ssl, tls)
		return true
	}
}

func (c *secretController) syncV2Handler(ctx context.Context, ev *types.Event, secret *corev1.Secret, secretKey string) func(k, v interface{}) bool {
	return func(k, v interface{}) bool {
		ssl := v.(*apisixv1.Ssl)
		tlsMetaKey := k.(string)
		tlsNamespace, tlsName, err := cache.SplitMetaNamespaceKey(tlsMetaKey)
		if err != nil {
			log.Errorf("invalid cached ApisixTls key: %s", tlsMetaKey)
			return true
		}

		multiVersioned, err := c.apisixTlsLister.V2(tlsNamespace, tlsName)
		if err != nil {
			log.Warnw("secret related ApisixTls resource not found, skip",
				zap.String("ApisixTls", tlsMetaKey),
			)
			return true
		}
		tls := multiVersioned.V2()

		// We don't expect a secret to be used as both SSL and mTLS in ApisixTls
		if tls.Spec.Secret.Namespace == secret.Namespace && tls.Spec.Secret.Name == secret.Name {
			cert, pkey, err := translation.ExtractKeyPair(secret, true)
			if err != nil {
				log.Errorw("secret required by ApisixTls invalid",
					zap.String("ApisixTls", tlsMetaKey),
					zap.Error(err),
				)
				go func(tls *configv2.ApisixTls) {
					c.RecordEventS(tls, corev1.EventTypeWarning, utils.ResourceSyncAborted,
						fmt.Sprintf("sync from secret %s changes failed, error: %s", secretKey, err.Error()))
					c.controller.recordStatus(tls, utils.ResourceSyncAborted, err, metav1.ConditionFalse, tls.GetGeneration())
				}(tls)
				return true
			}
			// sync ssl
			ssl.Cert = string(cert)
			ssl.Key = string(pkey)
		} else if tls.Spec.Client != nil &&
			tls.Spec.Client.CASecret.Namespace == secret.Namespace && tls.Spec.Client.CASecret.Name == secret.Name {
			ca, _, err := translation.ExtractKeyPair(secret, false)
			if err != nil {
				log.Errorw("ca secret required by ApisixTls invalid",
					zap.String("ApisixTls", tlsMetaKey),
					zap.Error(err),
				)
				go func(tls *configv2.ApisixTls) {
					c.RecordEventS(tls, corev1.EventTypeWarning, utils.ResourceSyncAborted,
						fmt.Sprintf("sync from ca secret %s changes failed, error: %s", secretKey, err.Error()))
					c.controller.recordStatus(tls, utils.ResourceSyncAborted, err, metav1.ConditionFalse, tls.GetGeneration())
				}(tls)
				return true
			}
			ssl.Client = &apisixv1.MutualTLSClientConfig{
				CA: string(ca),
			}
		} else {
			log.Warnw("stale secret cache, ApisixTls doesn't requires target secret",
				zap.String("ApisixTls", tlsMetaKey),
				zap.String("secret", secretKey),
			)
			return true
		}
		// Use another goroutine to send requests, to avoid
		// long time lock occupying.
		go func(ssl *apisixv1.Ssl, tls *configv2.ApisixTls) {
			err := c.SyncSSL(ctx, ssl, ev.Type)
			if err != nil {
				log.Errorw("failed to sync ssl to APISIX",
					zap.Error(err),
					zap.Any("ssl", ssl),
					zap.Any("secret", secret),
				)
				c.RecordEventS(tls, corev1.EventTypeWarning, utils.ResourceSyncAborted,
					fmt.Sprintf("sync from secret %s changes failed, error: %s", secretKey, err.Error()))
				c.controller.recordStatus(tls, utils.ResourceSyncAborted, err, metav1.ConditionFalse, tls.GetGeneration())
			} else {
				c.RecordEventS(tls, corev1.EventTypeNormal, utils.ResourceSynced,
					fmt.Sprintf("sync from secret %s changes", secretKey))
				c.controller.recordStatus(tls, utils.ResourceSynced, nil, metav1.ConditionTrue, tls.GetGeneration())
			}
		}(ssl, tls)
		return true
	}
}

func (c *secretController) handleSyncErr(obj interface{}, err error) {
	if err == nil {
		c.workqueue.Forget(obj)
		c.MetricsCollector.IncrSyncOperation("secret", "success")
		return
	}
	event := obj.(*types.Event)
	if k8serrors.IsNotFound(err) && event.Type != types.EventDelete {
		log.Infow("sync secret but not found, ignore",
			zap.String("event_type", event.Type.String()),
			zap.String("secret", event.Object.(string)),
		)
		c.workqueue.Forget(event)
		return
	}
	log.Warnw("sync secret failed, will retry",
		zap.Any("object", obj),
		zap.Error(err),
	)
	c.workqueue.AddRateLimited(obj)
	c.MetricsCollector.IncrSyncOperation("secret", "failure")
}

func (c *secretController) onAdd(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found secret object with bad namespace/name: %s, ignore it", err)
		return
	}
	if !c.NamespaceProvider.IsWatchingNamespace(key) {
		return
	}

	log.Debugw("secret add event arrived",
		zap.String("object-key", key),
	)
	c.workqueue.Add(&types.Event{
		Type:   types.EventAdd,
		Object: key,
	})

	c.MetricsCollector.IncrEvents("secret", "add")
}

func (c *secretController) onUpdate(prev, curr interface{}) {
	prevSec := prev.(*corev1.Secret)
	currSec := curr.(*corev1.Secret)

	if prevSec.GetResourceVersion() >= currSec.GetResourceVersion() {
		return
	}
	key, err := cache.MetaNamespaceKeyFunc(currSec)
	if err != nil {
		log.Errorf("found secrets object with bad namespace/name: %s, ignore it", err)
		return
	}
	if !c.NamespaceProvider.IsWatchingNamespace(key) {
		return
	}
	log.Debugw("secret update event arrived",
		zap.Any("new object", curr),
		zap.Any("old object", prev),
	)
	c.workqueue.Add(&types.Event{
		Type:   types.EventUpdate,
		Object: key,
	})

	c.MetricsCollector.IncrEvents("secret", "update")
}

func (c *secretController) onDelete(obj interface{}) {
	sec, ok := obj.(*corev1.Secret)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			log.Errorf("found secrets: %+v in bad tombstone state", obj)
			return
		}
		sec = tombstone.Obj.(*corev1.Secret)
	}

	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found secret resource with bad meta namespace key: %s", err)
		return
	}
	// FIXME Refactor Controller.isWatchingNamespace to just use
	// namespace after all controllers use the same way to fetch
	// the object.
	if !c.NamespaceProvider.IsWatchingNamespace(key) {
		return
	}
	log.Debugw("secret delete event arrived",
		zap.Any("final state", sec),
	)
	c.workqueue.Add(&types.Event{
		Type:      types.EventDelete,
		Object:    key,
		Tombstone: sec,
	})

	c.MetricsCollector.IncrEvents("secret", "delete")
}
