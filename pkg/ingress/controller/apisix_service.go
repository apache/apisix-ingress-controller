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
	"github.com/api7/ingress-controller/pkg/ingress/endpoint"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/api7/ingress-controller/pkg/ingress/apisix"
	configv1 "github.com/api7/ingress-controller/pkg/kube/apisix/apis/config/v1"
	clientset "github.com/api7/ingress-controller/pkg/kube/apisix/client/clientset/versioned"
	apisixscheme "github.com/api7/ingress-controller/pkg/kube/apisix/client/clientset/versioned/scheme"
	informersv1 "github.com/api7/ingress-controller/pkg/kube/apisix/client/informers/externalversions/config/v1"
	listersv1 "github.com/api7/ingress-controller/pkg/kube/apisix/client/listers/config/v1"
	"github.com/api7/ingress-controller/pkg/log"
	"github.com/api7/ingress-controller/pkg/seven/state"
)

type ApisixServiceController struct {
	controller          *Controller
	kubeclientset       kubernetes.Interface
	apisixClientset     clientset.Interface
	apisixServiceList   listersv1.ApisixServiceLister
	apisixServiceSynced cache.InformerSynced
	workqueue           workqueue.RateLimitingInterface
}

func BuildApisixServiceController(
	kubeclientset kubernetes.Interface,
	apisixServiceClientset clientset.Interface,
	apisixServiceInformer informersv1.ApisixServiceInformer,
	root *Controller) *ApisixServiceController {

	runtime.Must(apisixscheme.AddToScheme(scheme.Scheme))
	controller := &ApisixServiceController{
		controller:          root,
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
	OldObj *configv1.ApisixService `json:"old_obj"`
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
			log.Warnf("Service %s has been covered when retry", sqo.Key)
			return nil
		}
	} else {
		apisixServiceYaml, err = c.apisixServiceList.ApisixServices(namespace).Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				log.Infof("apisixService %s is removed", sqo.Key)
				return nil
			}
			runtime.HandleError(fmt.Errorf("failed to list apisixService %s/%s", sqo.Key, err.Error()))
			return err
		}
	}
	asb := apisix.ApisixServiceBuilder{CRD: apisixServiceYaml, Ep: &endpoint.EndpointRequest{}, EnableEndpointSlice: c.controller.cfg.EnableEndpointSlice}
	services, upstreams, _ := asb.Convert()
	comb := state.ApisixCombination{Routes: nil, Services: services, Upstreams: upstreams}
	if sqo.Ope == DELETE {
		return comb.Remove()
	} else {
		_, err = comb.Solver()
		return err
	}

}

func (c *ApisixServiceController) addFunc(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	if !c.controller.namespaceWatching(key) {
		return
	}
	sqo := &ServiceQueueObj{Key: key, OldObj: nil, Ope: ADD}
	c.workqueue.AddRateLimited(sqo)
}

func (c *ApisixServiceController) updateFunc(oldObj, newObj interface{}) {
	oldService := oldObj.(*configv1.ApisixService)
	newService := newObj.(*configv1.ApisixService)
	if oldService.ResourceVersion >= newService.ResourceVersion {
		return
	}
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(newObj); err != nil {
		runtime.HandleError(err)
		return
	}
	if !c.controller.namespaceWatching(key) {
		return
	}
	sqo := &ServiceQueueObj{Key: key, OldObj: oldService, Ope: UPDATE}
	c.workqueue.AddRateLimited(sqo)
}

func (c *ApisixServiceController) deleteFunc(obj interface{}) {
	oldService, ok := obj.(*configv1.ApisixService)
	if !ok {
		oldState, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			return
		}
		oldService, ok = oldState.Obj.(*configv1.ApisixService)
		if !ok {
			return
		}
	}

	var key string
	var err error
	key, err = cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(err)
		return
	}
	if !c.controller.namespaceWatching(key) {
		return
	}
	sqo := &ServiceQueueObj{Key: key, OldObj: oldService, Ope: DELETE}
	c.workqueue.AddRateLimited(sqo)
}
