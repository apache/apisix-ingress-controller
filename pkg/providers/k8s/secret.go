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
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/apache/apisix-ingress-controller/pkg/log"
	apisixprovider "github.com/apache/apisix-ingress-controller/pkg/providers/apisix"
	ingressprovider "github.com/apache/apisix-ingress-controller/pkg/providers/ingress"
	"github.com/apache/apisix-ingress-controller/pkg/providers/k8s/namespace"
	providertypes "github.com/apache/apisix-ingress-controller/pkg/providers/types"
	"github.com/apache/apisix-ingress-controller/pkg/types"
)

type secretController struct {
	*providertypes.Common

	workqueue workqueue.RateLimitingInterface
	workers   int

	secretLister   listerscorev1.SecretLister
	secretInformer cache.SharedIndexInformer

	namespaceProvider namespace.WatchingNamespaceProvider
	apisixProvider    apisixprovider.Provider
	ingressProvider   ingressprovider.Provider
}

func newSecretController(common *providertypes.Common, namespaceProvider namespace.WatchingNamespaceProvider,
	apisixProvider apisixprovider.Provider, ingressProvider ingressprovider.Provider) *secretController {
	c := &secretController{
		Common: common,

		workqueue: workqueue.NewNamedRateLimitingQueue(workqueue.NewItemFastSlowRateLimiter(1*time.Second, 60*time.Second, 5), "Secrets"),
		workers:   1,

		secretLister:   common.SecretLister,
		secretInformer: common.SecretInformer,

		namespaceProvider: namespaceProvider,
		apisixProvider:    apisixProvider,
		ingressProvider:   ingressProvider,
	}

	c.secretInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.onAdd,
			UpdateFunc: c.onUpdate,
			DeleteFunc: c.onDelete,
		},
	)

	return c
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

	log.Debugw("sync secret change",
		zap.String("key", key),
	)
	secretKey := namespace + "/" + name
	c.apisixProvider.SyncSecretChange(ctx, ev, sec, secretKey)
	c.ingressProvider.SyncSecretChange(ctx, ev, sec, secretKey)
	return nil
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
	if !c.namespaceProvider.IsWatchingNamespace(key) {
		return
	}

	log.Debugw("secret add event arrived",
		zap.String("key", key),
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
	if !c.namespaceProvider.IsWatchingNamespace(key) {
		return
	}
	log.Debugw("secret update event arrived",
		zap.Any("secret name", curr.(*corev1.Secret).ObjectMeta.Name),
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
	if !c.namespaceProvider.IsWatchingNamespace(key) {
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
