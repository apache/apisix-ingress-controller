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
	"github.com/api7/ingress-controller/pkg/ingress/endpoint"
	"github.com/api7/ingress-controller/pkg/log"
)

type ApisixUpstreamController struct {
	kubeclientset        kubernetes.Interface
	apisixClientset      clientSet.Interface
	apisixUpstreamList   v1.ApisixUpstreamLister
	apisixUpstreamSynced cache.InformerSynced
	workqueue            workqueue.RateLimitingInterface
}

func BuildApisixUpstreamController(
	kubeclientset kubernetes.Interface,
	apisixUpstreamClientset clientSet.Interface,
	apisixUpstreamInformer informers.ApisixUpstreamInformer) *ApisixUpstreamController {

	runtime.Must(apisixScheme.AddToScheme(scheme.Scheme))
	controller := &ApisixUpstreamController{
		kubeclientset:        kubeclientset,
		apisixClientset:      apisixUpstreamClientset,
		apisixUpstreamList:   apisixUpstreamInformer.Lister(),
		apisixUpstreamSynced: apisixUpstreamInformer.Informer().HasSynced,
		workqueue:            workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ApisixUpstreams"),
	}
	apisixUpstreamInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    controller.addFunc,
			UpdateFunc: controller.updateFunc,
			DeleteFunc: controller.deleteFunc,
		})
	return controller
}

func (c *ApisixUpstreamController) Run(stop <-chan struct{}) error {
	// 同步缓存
	if ok := cache.WaitForCacheSync(stop); !ok {
		log.Error("同步ApisixUpstream缓存失败")
		return fmt.Errorf("failed to wait for caches to sync")
	}
	go wait.Until(c.runWorker, time.Second, stop)
	return nil
}

func (c *ApisixUpstreamController) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *ApisixUpstreamController) processNextWorkItem() bool {
	defer recoverException()
	obj, shutdown := c.workqueue.Get()
	if shutdown {
		return false
	}
	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var key string
		var ok bool

		if key, ok = obj.(string); !ok {
			c.workqueue.Forget(obj)
			return fmt.Errorf("expected string in workqueue but got %#v", obj)
		}
		// 在syncHandler中处理业务
		if err := c.syncHandler(key); err != nil {
			return fmt.Errorf("error syncing '%s': %s", key, err.Error())
		}

		c.workqueue.Forget(obj)
		return nil
	}(obj)
	if err != nil {
		runtime.HandleError(err)
	}
	return true
}

func (c *ApisixUpstreamController) syncHandler(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		log.Errorf("invalid resource key: %s", key)
		return fmt.Errorf("invalid resource key: %s", key)
	}

	apisixUpstreamYaml, err := c.apisixUpstreamList.ApisixUpstreams(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Infof("apisixUpstream %s is removed", key)
			return nil
		}
		runtime.HandleError(fmt.Errorf("failed to list apisixUpstream %s/%s", key, err.Error()))
		return err
	}
	log.Info(namespace)
	log.Info(name)
	//apisixUpstream := apisix.ApisixUpstreamCRD(*apisixUpstreamYaml)
	aub := apisix.ApisixUpstreamBuilder{CRD: apisixUpstreamYaml, Ep: &endpoint.EndpointRequest{}}
	upstreams, _ := aub.Convert()
	comb := state.ApisixCombination{Routes: nil, Services: nil, Upstreams: upstreams}
	_, err = comb.Solver()
	return err
}

func (c *ApisixUpstreamController) addFunc(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}

func (c *ApisixUpstreamController) updateFunc(oldObj, newObj interface{}) {
	oldRoute := oldObj.(*apisixV1.ApisixUpstream)
	newRoute := newObj.(*apisixV1.ApisixUpstream)
	if oldRoute.ResourceVersion == newRoute.ResourceVersion {
		return
	}
	c.addFunc(newObj)
}

func (c *ApisixUpstreamController) deleteFunc(obj interface{}) {
	var key string
	var err error
	key, err = cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}
