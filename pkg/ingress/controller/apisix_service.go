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
package controller

import (
	"fmt"
	"time"

	apisixV1 "github.com/gxthrj/apisix-ingress-types/pkg/apis/config/v1"
	clientSet "github.com/gxthrj/apisix-ingress-types/pkg/client/clientset/versioned"
	apisixScheme "github.com/gxthrj/apisix-ingress-types/pkg/client/clientset/versioned/scheme"
	informers "github.com/gxthrj/apisix-ingress-types/pkg/client/informers/externalversions/config/v1"
	"github.com/gxthrj/apisix-ingress-types/pkg/client/listers/config/v1"
	"github.com/gxthrj/seven/state"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/api7/ingress-controller/pkg/ingress/apisix"
	"github.com/api7/ingress-controller/pkg/log"
)

type ApisixServiceController struct {
	kubeclientset       kubernetes.Interface
	apisixClientset     clientSet.Interface
	apisixServiceList   v1.ApisixServiceLister
	apisixServiceSynced cache.InformerSynced
	workqueue           workqueue.RateLimitingInterface
}

func BuildApisixServiceController(
	kubeclientset kubernetes.Interface,
	apisixServiceClientset clientSet.Interface,
	apisixServiceInformer informers.ApisixServiceInformer) *ApisixServiceController {

	runtime.Must(apisixScheme.AddToScheme(scheme.Scheme))
	controller := &ApisixServiceController{
		kubeclientset:       kubeclientset,
		apisixClientset:     apisixServiceClientset,
		apisixServiceList:   apisixServiceInformer.Lister(),
		apisixServiceSynced: apisixServiceInformer.Informer().HasSynced,
		workqueue:           workqueue.NewNamedRateLimitingQueue(workqueue.NewItemFastSlowRateLimiter(1*time.Second, 60*time.Second, 5), "ApisixServices"),
	}
	apisixServiceInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    controller.addFunc,
			UpdateFunc: controller.updateFunc,
			DeleteFunc: controller.deleteFunc,
		})
	return controller
}

type ServiceQueueObj struct {
	Key    string                  `json:"key"`
	OldObj *apisixV1.ApisixService `json:"old_obj"`
	Ope    string                  `json:"ope"` // add / update / delete
}

func (c *ApisixServiceController) Run(stop <-chan struct{}) error {
	// 同步缓存
	if ok := cache.WaitForCacheSync(stop); !ok {
		log.Error("同步ApisixService缓存失败")
		return fmt.Errorf("failed to wait for caches to sync")
	}
	go wait.Until(c.runWorker, time.Second, stop)
	return nil
}

func (c *ApisixServiceController) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *ApisixServiceController) processNextWorkItem() bool {
	defer recoverException()
	obj, shutdown := c.workqueue.Get()
	if shutdown {
		return false
	}
	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var sqo *ServiceQueueObj
		var ok bool
		if sqo, ok = obj.(*ServiceQueueObj); !ok {
			c.workqueue.Forget(obj)
			return fmt.Errorf("expected ServiceQueueObj in workqueue but got %#v", obj)
		}
		if err := c.syncHandler(sqo); err != nil {
			c.workqueue.AddRateLimited(obj)
			log.Errorf("sync service %s failed", sqo.Key)
			return fmt.Errorf("error syncing '%s': %s", sqo.Key, err.Error())
		}

		c.workqueue.Forget(obj)
		return nil
	}(obj)
	if err != nil {
		runtime.HandleError(err)
	}
	return true
}

func (c *ApisixServiceController) syncHandler(sqo *ServiceQueueObj) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(sqo.Key)
	if err != nil {
		log.Errorf("invalid resource key: %s", sqo.Key)
		return fmt.Errorf("invalid resource key: %s", sqo.Key)
	}
	apisixServiceYaml := sqo.OldObj
	if sqo.Ope == DELETE {
		apisixIngressService, _ := c.apisixServiceList.ApisixServices(namespace).Get(name)
		if apisixIngressService != nil && apisixIngressService.ResourceVersion > sqo.OldObj.ResourceVersion {
			return nil
		}
	} else {
		apisixServiceYaml, err = c.apisixServiceList.ApisixServices(namespace).Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				log.Infof("apisixUpstream %s is removed", sqo.Key)
			}
			runtime.HandleError(fmt.Errorf("failed to list apisixUpstream %s/%s", sqo.Key, err.Error()))
			return err
		}
	}
	apisixService := apisix.ApisixServiceCRD(*apisixServiceYaml)
	services, upstreams, _ := apisixService.Convert()
	comb := state.ApisixCombination{Routes: nil, Services: services, Upstreams: upstreams}
	_, err = comb.Solver()
	return err
}

func (c *ApisixServiceController) addFunc(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	sqo := &RouteQueueObj{Key: key, OldObj: nil, Ope: ADD}
	c.workqueue.AddRateLimited(sqo)
}

func (c *ApisixServiceController) updateFunc(oldObj, newObj interface{}) {
	oldService := oldObj.(*apisixV1.ApisixService)
	newService := newObj.(*apisixV1.ApisixService)
	if oldService.ResourceVersion >= newService.ResourceVersion {
		return
	}
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(newObj); err != nil {
		runtime.HandleError(err)
		return
	}
	sqo := &ServiceQueueObj{Key: key, OldObj: oldService, Ope: UPDATE}
	c.workqueue.AddRateLimited(sqo)
}

func (c *ApisixServiceController) deleteFunc(obj interface{}) {
	oldService := obj.(cache.DeletedFinalStateUnknown).Obj.(*apisixV1.ApisixService)
	var key string
	var err error
	key, err = cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(err)
		return
	}
	sqo := &ServiceQueueObj{Key: key, OldObj: oldService, Ope: DELETE}
	c.workqueue.AddRateLimited(sqo)
}
