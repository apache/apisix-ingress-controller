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
package pod

import (
	"context"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/providers/k8s/namespace"
	providertypes "github.com/apache/apisix-ingress-controller/pkg/providers/types"
	"github.com/apache/apisix-ingress-controller/pkg/types"
)

type podController struct {
	*providertypes.Common

	namespaceProvider namespace.WatchingNamespaceProvider
	podInformer       cache.SharedIndexInformer

	podCache types.PodCache
}

func newPodController(common *providertypes.Common, nsProvider namespace.WatchingNamespaceProvider,
	podInformer cache.SharedIndexInformer) *podController {
	ctl := &podController{
		Common: common,

		namespaceProvider: nsProvider,
		podInformer:       podInformer,

		podCache: types.NewPodCache(),
	}
	ctl.podInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctl.onAdd,
			UpdateFunc: ctl.onUpdate,
			DeleteFunc: ctl.onDelete,
		},
	)
	return ctl
}

func (c *podController) run(ctx context.Context) {
	log.Info("pod controller started")
	defer log.Info("pod controller exited")

	if ok := cache.WaitForCacheSync(ctx.Done(), c.podInformer.HasSynced); !ok {
		log.Error("informers sync failed")
		return
	}

	<-ctx.Done()
}

func (c *podController) onAdd(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Errorf("found pod with bad namespace/name: %s, ignore it", err)
		return
	}
	if !c.namespaceProvider.IsWatchingNamespace(key) {
		return
	}
	log.Debugw("pod add event arrived",
		zap.String("obj.key", key),
	)
	pod := obj.(*corev1.Pod)
	if err := c.podCache.Add(pod); err != nil {
		if err == types.ErrPodNoAssignedIP {
			log.Debugw("pod no assigned ip, postpone the adding in subsequent update event",
				zap.Any("pod", pod),
			)
		} else {
			log.Errorw("failed to add pod to cache",
				zap.Error(err),
				zap.Any("pod", pod),
			)
		}
	}

	c.MetricsCollector.IncrEvents("pod", "add")
}

func (c *podController) onUpdate(oldObj, newObj interface{}) {
	prev := oldObj.(*corev1.Pod)
	curr := newObj.(*corev1.Pod)
	if prev.GetResourceVersion() >= curr.GetResourceVersion() {
		return
	}

	if !c.namespaceProvider.IsWatchingNamespace(curr.Namespace + "/" + curr.Name) {
		return
	}
	log.Debugw("pod update event arrived",
		zap.Any("pod namespace", curr.Namespace),
		zap.Any("pod name", curr.Name),
	)
	if curr.DeletionTimestamp != nil {
		if err := c.podCache.Delete(curr); err != nil {
			log.Errorw("failed to delete pod from cache",
				zap.Error(err),
				zap.Any("pod", curr),
			)
		}
	}
	if curr.Status.PodIP != "" {
		if err := c.podCache.Add(curr); err != nil {
			log.Errorw("failed to add pod to cache",
				zap.Error(err),
				zap.Any("pod", curr),
			)
		}
	}

	c.MetricsCollector.IncrEvents("pod", "update")
}

func (c *podController) onDelete(obj interface{}) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			log.Errorf("found pod: %+v in bad tombstone state", obj)
			return
		}
		pod = tombstone.Obj.(*corev1.Pod)
	}

	if !c.namespaceProvider.IsWatchingNamespace(pod.Namespace + "/" + pod.Name) {
		return
	}
	log.Debugw("pod delete event arrived",
		zap.Any("final state", pod),
	)
	if err := c.podCache.Delete(pod); err != nil {
		log.Errorw("failed to delete pod from cache",
			zap.Error(err),
			zap.Any("pod", pod),
		)
	}

	c.MetricsCollector.IncrEvents("pod", "delete")
}
