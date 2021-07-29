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
	"sync"
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/apache/apisix-ingress-controller/pkg/kube/translation"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/types"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

type secretController struct {
	controller *Controller
	workqueue  workqueue.RateLimitingInterface
	workers    int
}

func (c *Controller) newSecretController() *secretController {
	ctl := &secretController{
		controller: c,
		workqueue:  workqueue.NewNamedRateLimitingQueue(workqueue.NewItemFastSlowRateLimiter(1*time.Second, 60*time.Second, 5), "Secrets"),
		workers:    1,
	}

	ctl.controller.secretInformer.AddEventHandler(
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

	if ok := cache.WaitForCacheSync(ctx.Done(), c.controller.secretInformer.HasSynced); !ok {
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
	sec, err := c.controller.secretLister.Secrets(namespace).Get(name)

	secretMapkey := namespace + "_" + name
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			log.Errorw("failed to get Secret",
				zap.String("key", secretMapkey),
				zap.Error(err),
			)
			return err
		}

		if ev.Type != types.EventDelete {
			log.Warnw("Secret was deleted before it can be delivered",
				zap.String("key", secretMapkey),
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
				zap.String("key", secretMapkey),
			)
			return nil
		}
		sec = ev.Tombstone.(*corev1.Secret)
	}
	// sync SSL in APISIX which is store in secretSSLMap
	// FixMe Need to update the status of CRD ApisixTls
	ssls, ok := c.controller.secretSSLMap.Load(secretMapkey)
	if !ok {
		// This secret is not concerned.
		return nil
	}
	sslMap := ssls.(*sync.Map)
	sslMap.Range(func(k, v interface{}) bool {
		ssl := v.(*apisixv1.Ssl)
		tlsMetaKey := k.(string)
		tlsNamespace, tlsName, err := cache.SplitMetaNamespaceKey(tlsMetaKey)
		if err != nil {
			log.Errorf("invalid cached ApisixTls key: %s", tlsMetaKey)
			return true
		}
		tls, err := c.controller.apisixTlsLister.ApisixTlses(tlsNamespace).Get(tlsName)
		if err != nil {
			log.Warnw("secret related ApisixTls resource not found, skip",
				zap.String("ApisixTls", tlsMetaKey),
			)
			return true
		}
		if tls.Spec.Secret.Namespace == sec.Namespace && tls.Spec.Secret.Name == sec.Name {
			cert, ok := sec.Data["cert"]
			if !ok {
				log.Warnw("secret required by ApisixTls invalid",
					zap.String("ApisixTls", tlsMetaKey),
					zap.Error(translation.ErrEmptyCert),
				)
				return true
			}
			pkey, ok := sec.Data["key"]
			if !ok {
				log.Warnw("secret required by ApisixTls invalid",
					zap.String("ApisixTls", tlsMetaKey),
					zap.Error(translation.ErrEmptyPrivKey),
				)
				return true
			}
			// sync ssl
			ssl.Cert = string(cert)
			ssl.Key = string(pkey)
		} else if tls.Spec.Client != nil &&
			tls.Spec.Client.CASecret.Namespace == sec.Namespace && tls.Spec.Client.CASecret.Name == sec.Name {
			ca, ok := sec.Data["cert"]
			if !ok {
				log.Warnw("secret required by ApisixTls invalid",
					zap.String("resource", tlsMetaKey),
					zap.Error(translation.ErrEmptyCert),
				)
				return true
			}
			ssl.Client = &apisixv1.MutualTLSClientConfig{
				CA: string(ca),
			}
		} else {
			log.Warnw("stale secret cache, ApisixTls doesn't requires target secret",
				zap.String("ApisixTls", tlsMetaKey),
				zap.String("secret", key),
			)
			return true
		}
		// Use another goroutine to send requests, to avoid
		// long time lock occupying.
		go func(ssl *apisixv1.Ssl) {
			err := c.controller.syncSSL(ctx, ssl, ev.Type)
			if err != nil {
				log.Errorw("failed to sync ssl to APISIX",
					zap.Error(err),
					zap.Any("ssl", ssl),
					zap.Any("secret", sec),
				)
				c.controller.recorderEventS(tls, corev1.EventTypeWarning, _resourceSyncAborted,
					fmt.Sprintf("sync from secret %s changes failed, error: %s", key, err.Error()))
				c.controller.recordStatus(tls, _resourceSyncAborted, err, metav1.ConditionFalse)
			} else {
				c.controller.recorderEventS(tls, corev1.EventTypeNormal, _resourceSynced,
					fmt.Sprintf("sync from secret %s changes", key))
				c.controller.recordStatus(tls, _resourceSynced, nil, metav1.ConditionTrue)
			}
		}(ssl)
		return true
	})
	return err
}

func (c *secretController) handleSyncErr(obj interface{}, err error) {
	if err == nil {
		c.workqueue.Forget(obj)
		return
	}
	log.Warnw("sync ApisixTls failed, will retry",
		zap.Any("object", obj),
		zap.Error(err),
	)
	c.workqueue.AddRateLimited(obj)
}

func (c *secretController) onAdd(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found secret object with bad namespace/name: %s, ignore it", err)
		return
	}
	if !c.controller.namespaceWatching(key) {
		return
	}

	log.Debugw("secret add event arrived",
		zap.String("object-key", key),
	)
	c.workqueue.AddRateLimited(&types.Event{
		Type:   types.EventAdd,
		Object: key,
	})
}

func (c *secretController) onUpdate(prev, curr interface{}) {
	prevSec := prev.(*corev1.Secret)
	currSec := curr.(*corev1.Secret)

	if prevSec.GetResourceVersion() == currSec.GetResourceVersion() {
		return
	}
	key, err := cache.MetaNamespaceKeyFunc(currSec)
	if err != nil {
		log.Errorf("found secrets object with bad namespace/name: %s, ignore it", err)
		return
	}
	if !c.controller.namespaceWatching(key) {
		return
	}
	log.Debugw("secret update event arrived",
		zap.Any("new object", curr),
		zap.Any("old object", prev),
	)
	c.workqueue.AddRateLimited(&types.Event{
		Type:   types.EventUpdate,
		Object: key,
	})
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
	// FIXME Refactor Controller.namespaceWatching to just use
	// namespace after all controllers use the same way to fetch
	// the object.
	if !c.controller.namespaceWatching(key) {
		return
	}
	log.Debugw("secret delete event arrived",
		zap.Any("final state", sec),
	)
	c.workqueue.AddRateLimited(&types.Event{
		Type:      types.EventDelete,
		Object:    key,
		Tombstone: sec,
	})
}
