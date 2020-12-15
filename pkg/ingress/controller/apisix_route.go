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
	api6V1 "github.com/gxthrj/apisix-ingress-types/pkg/apis/config/v1"
	clientSet "github.com/gxthrj/apisix-ingress-types/pkg/client/clientset/versioned"
	api6Scheme "github.com/gxthrj/apisix-ingress-types/pkg/client/clientset/versioned/scheme"
	api6Informers "github.com/gxthrj/apisix-ingress-types/pkg/client/informers/externalversions/config/v1"
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

	"github.com/api7/ingress-controller/pkg/log"
)

type ApisixRouteController struct {
	kubeclientset        kubernetes.Interface
	apisixRouteClientset clientSet.Interface
	apisixRouteList      v1.ApisixRouteLister
	apisixRouteSynced    cache.InformerSynced
	workqueue            workqueue.RateLimitingInterface
}

type RouteQueueObj struct {
	Key    string              `json:"key"`
	OldObj *api6V1.ApisixRoute `json:"old_obj"`
	Ope    string              `json:"ope"` // add / update / delete
}

func BuildApisixRouteController(
	kubeclientset kubernetes.Interface,
	api6RouteClientset clientSet.Interface,
	api6RouteInformer api6Informers.ApisixRouteInformer) *ApisixRouteController {

	runtime.Must(api6Scheme.AddToScheme(scheme.Scheme))
	controller := &ApisixRouteController{
		kubeclientset:        kubeclientset,
		apisixRouteClientset: api6RouteClientset,
		apisixRouteList:      api6RouteInformer.Lister(),
		apisixRouteSynced:    api6RouteInformer.Informer().HasSynced,
		workqueue:            workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ApisixRoutes"),
	}
	api6RouteInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    controller.addFunc,
			UpdateFunc: controller.updateFunc,
			DeleteFunc: controller.deleteFunc,
		})
	return controller
}

func (c *ApisixRouteController) addFunc(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	rqo := &RouteQueueObj{Key: key, OldObj: nil, Ope: ADD}
	c.workqueue.AddRateLimited(rqo)
}

func (c *ApisixRouteController) updateFunc(oldObj, newObj interface{}) {
	oldRoute := oldObj.(*api6V1.ApisixRoute)
	newRoute := newObj.(*api6V1.ApisixRoute)
	if oldRoute.ResourceVersion == newRoute.ResourceVersion {
		return
	}
	//c.addFunc(newObj)
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(newObj); err != nil {
		runtime.HandleError(err)
		return
	}
	rqo := &RouteQueueObj{Key: key, OldObj: oldRoute, Ope: UPDATE}
	c.workqueue.AddRateLimited(rqo)
}

func (c *ApisixRouteController) deleteFunc(obj interface{}) {
	var key string
	var err error
	key, err = cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(err)
		return
	}
	rqo := &RouteQueueObj{Key: key, OldObj: nil, Ope: DELETE}
	c.workqueue.AddRateLimited(rqo)
}

func (c *ApisixRouteController) Run(stop <-chan struct{}) error {
	//defer c.workqueue.ShutDown()
	// 同步缓存
	if ok := cache.WaitForCacheSync(stop); !ok {
		log.Errorf("同步缓存失败")
		return fmt.Errorf("failed to wait for caches to sync")
	}
	go wait.Until(c.runWorker, time.Second, stop)
	return nil
}

func (c *ApisixRouteController) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *ApisixRouteController) processNextWorkItem() bool {
	defer recoverException()
	obj, shutdown := c.workqueue.Get()
	if shutdown {
		return false
	}
	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		var rqo *RouteQueueObj
		if rqo, ok = obj.(*RouteQueueObj); !ok {
			c.workqueue.Forget(obj)
			return fmt.Errorf("expected RouteQueueObj in workqueue but got %#v", obj)
		}
		// 在syncHandler中处理业务
		if err := c.syncHandler(rqo); err != nil {
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

func (c *ApisixRouteController) syncHandler(rqo *RouteQueueObj) error {
	key := rqo.Key
	switch {
	case rqo.Ope == ADD:
		return c.add(key)
	case rqo.Ope == UPDATE:
		// 1.first add new route config
		if err := c.add(key); err != nil {
			// log error
			return err
		} else {
			// 2.then delete routes not exist
			return c.sync(rqo)
		}
	case rqo.Ope == DELETE:
		return c.sync(rqo)
	default:
		// log error
		return fmt.Errorf("RouteQueueObj is not expected")
	}
}

func (c *ApisixRouteController) add(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		log.Errorf("invalid resource key: %s", key)
		return fmt.Errorf("invalid resource key: %s", key)
	}

	apisixIngressRoute, err := c.apisixRouteList.ApisixRoutes(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Infof("apisixRoute %s is removed", key)
			return nil
		}
		runtime.HandleError(fmt.Errorf("failed to list apisixRoute %s/%s", key, err.Error()))
		return err
	}
	apisixRoute := apisix.ApisixRoute(*apisixIngressRoute)
	routes, services, upstreams, _ := apisixRoute.Convert()
	comb := state.ApisixCombination{Routes: routes, Services: services, Upstreams: upstreams}
	_, err = comb.Solver()
	return err
}

// sync
// 1.diff routes between old and new objects
// 2.delete routes not exist
func (c *ApisixRouteController) sync(rqo *RouteQueueObj) error {
	key := rqo.Key
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		log.Errorf("invalid resource key: %s", key)
		return fmt.Errorf("invalid resource key: %s", key)
	}

	apisixIngressRoute, err := c.apisixRouteList.ApisixRoutes(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Infof("apisixRoute %s is removed", key)
			return nil
		}
		runtime.HandleError(fmt.Errorf("failed to list apisixRoute %s/%s", key, err.Error()))
		return err
	}
	switch {
	case rqo.Ope == UPDATE:
		oldApisixRoute := apisix.ApisixRoute(*rqo.OldObj)
		oldRoutes, _, _, _ := oldApisixRoute.Convert()

		newApisixRoute := apisix.ApisixRoute(*apisixIngressRoute)
		newRoutes, _, _, _ := newApisixRoute.Convert()

		rc := &state.RouteCompare{OldRoutes: oldRoutes, NewRoutes: newRoutes}
		return rc.Sync()
	case rqo.Ope == DELETE:
		apisixRoute := apisix.ApisixRoute(*apisixIngressRoute)
		routes, _, _, _ := apisixRoute.Convert()
		rc := &state.RouteCompare{OldRoutes: routes, NewRoutes: nil}
		return rc.Sync()
	default:
		return fmt.Errorf("not expected in (ApisixRouteController) sync")
	}
}
