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

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/types"
)

type podController struct {
	controller *Controller
}

func (c *Controller) newPodController() *podController {
	ctl := &podController{
		controller: c,
	}
	ctl.controller.podInformer.AddEventHandler(
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

	if ok := cache.WaitForCacheSync(ctx.Done(), c.controller.podInformer.HasSynced); !ok {
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
	if !c.controller.namespaceWatching(key) {
		return
	}
	log.Debugw("pod add event arrived",
		zap.String("obj.key", key),
	)
	pod := obj.(*corev1.Pod)
	if err := c.controller.podCache.Add(pod); err != nil {
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
}

func (c *podController) onUpdate(_, cur interface{}) {
	pod := cur.(*corev1.Pod)

	if !c.controller.namespaceWatching(pod.Namespace + "/" + pod.Name) {
		return
	}
	log.Debugw("pod update event arrived",
		zap.Any("final state", pod),
	)
	if pod.DeletionTimestamp != nil {
		if err := c.controller.podCache.Delete(pod); err != nil {
			log.Errorw("failed to delete pod from cache",
				zap.Error(err),
				zap.Any("pod", pod),
			)
		}
	}
	if pod.Status.PodIP != "" {
		if err := c.controller.podCache.Add(pod); err != nil {
			log.Errorw("failed to add pod to cache",
				zap.Error(err),
				zap.Any("pod", pod),
			)
		}
	}
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

	if !c.controller.namespaceWatching(pod.Namespace + "/" + pod.Name) {
		return
	}
	log.Debugw("pod delete event arrived",
		zap.Any("final state", pod),
	)
	if err := c.controller.podCache.Delete(pod); err != nil {
		log.Errorw("failed to delete pod from cache",
			zap.Error(err),
			zap.Any("pod", pod),
		)
	}
}
