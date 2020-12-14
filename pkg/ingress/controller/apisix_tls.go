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
	"github.com/api7/ingress-controller/pkg/ingress/apisix"
	"github.com/golang/glog"
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
	"time"
)

type ApisixTlsController struct {
	kubeclientset   kubernetes.Interface
	apisixClientset clientSet.Interface
	apisixTlsList   v1.ApisixTlsLister
	apisixTlsSynced cache.InformerSynced
	workqueue       workqueue.RateLimitingInterface
}

func BuildApisixTlsController(
	kubeclientset kubernetes.Interface,
	apisixTlsClientset clientSet.Interface,
	apisixTlsInformer informers.ApisixTlsInformer) *ApisixTlsController {

	runtime.Must(apisixScheme.AddToScheme(scheme.Scheme))
	controller := &ApisixTlsController{
		kubeclientset:   kubeclientset,
		apisixClientset: apisixTlsClientset,
		apisixTlsList:   apisixTlsInformer.Lister(),
		apisixTlsSynced: apisixTlsInformer.Informer().HasSynced,
		workqueue:       workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ApisixTlses"),
	}
	apisixTlsInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    controller.addFunc,
			UpdateFunc: controller.updateFunc,
			DeleteFunc: controller.deleteFunc,
		})
	return controller
}

func (c *ApisixTlsController) Run(stop <-chan struct{}) error {
	if ok := cache.WaitForCacheSync(stop); !ok {
		glog.Errorf("sync ApisixService cache failed")
		return fmt.Errorf("failed to wait for caches to sync")
	}
	go wait.Until(c.runWorker, time.Second, stop)
	return nil
}

func (c *ApisixTlsController) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *ApisixTlsController) processNextWorkItem() bool {
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

func (c *ApisixTlsController) syncHandler(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		logger.Errorf("invalid resource key: %s", key)
		return fmt.Errorf("invalid resource key: %s", key)
	}

	apisixTlsYaml, err := c.apisixTlsList.ApisixTlses(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Infof("apisixTls %s is removed", key)
			return nil
		}
		runtime.HandleError(fmt.Errorf("failed to list apisixTls %s/%s", key, err.Error()))
		return err
	}
	logger.Info(namespace)
	logger.Info(name)
	apisixTls := apisix.ApisixTlsCRD(*apisixTlsYaml)
	tlses, _ := apisixTls.Convert()
	//comb := state.ApisixCombination{Routes: nil, Services: services, Upstreams: upstreams}
	//_, err = comb.Solver()
	return err
}

func (c *ApisixTlsController) addFunc(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}

func (c *ApisixTlsController) updateFunc(oldObj, newObj interface{}) {
	oldTls := oldObj.(*apisixV1.ApisixTls)
	newTls := newObj.(*apisixV1.ApisixTls)
	if oldTls.ResourceVersion == newTls.ResourceVersion {
		return
	}
	c.addFunc(newObj)
}

func (c *ApisixTlsController) deleteFunc(obj interface{}) {
	var key string
	var err error
	key, err = cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}
