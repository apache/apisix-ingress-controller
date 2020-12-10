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
	"github.com/golang/glog"
	"github.com/gxthrj/seven/state"
	CoreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	CoreListerV1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"time"
	"strconv"
	"github.com/gxthrj/seven/apisix"
	sevenConf "github.com/gxthrj/seven/conf"
	apisixType "github.com/gxthrj/apisix-types/pkg/apis/apisix/v1"
	"github.com/api7/ingress-controller/conf"
)

type EndpointController struct {
	kubeclientset kubernetes.Interface
	endpointList  CoreListerV1.EndpointsLister
	endpointSynced cache.InformerSynced
	workqueue     workqueue.RateLimitingInterface
}

func BuildEndpointController(kubeclientset kubernetes.Interface) *EndpointController {
	controller := &EndpointController{
		kubeclientset: kubeclientset,
		endpointList: conf.EndpointsInformer.Lister(),
		endpointSynced: conf.EndpointsInformer.Informer().HasSynced,
		workqueue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "endpoints"),
	}
	conf.EndpointsInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    controller.addFunc,
			UpdateFunc: controller.updateFunc,
			DeleteFunc: controller.deleteFunc,
		})
	return controller
}

func (c *EndpointController) Run(stop <-chan struct{}) error {
	// 同步缓存
	if ok := cache.WaitForCacheSync(stop); !ok {
		glog.Errorf("同步Endpoint缓存失败")
		return fmt.Errorf("failed to wait for caches to sync")
	}
	go wait.Until(c.runWorker, time.Second, stop)
	return nil
}

func (c *EndpointController) runWorker() {
	for c.processNextWorkItem() {}
}

func (c *EndpointController) processNextWorkItem() bool {
	defer recoverException()
	obj, shutdown := c.workqueue.Get()
	if shutdown {
		glog.V(2).Info("shutdown")
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

func (c *EndpointController) syncHandler(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if name == "cinfoserver" || name == "file-resync2-server" {
		glog.V(2).Infof("find endpoint %s/%s", namespace, name)
	}
	if err != nil {
		logger.Error("invalid resource key: %s", key)
		return fmt.Errorf("invalid resource key: %s", key)
	}

	endpointYaml, err := c.endpointList.Endpoints(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("endpoint %s is removed", key)
			return nil
		}
		runtime.HandleError(fmt.Errorf("failed to list endpoint %s/%s", key, err.Error()))
		return err
	}
	// endpoint sync
	c.process(endpointYaml)
	return err
}

func (c *EndpointController) process(ep *CoreV1.Endpoints) {
	if ep.Namespace != "kube-system"{ // todo here is some ignore namespaces
		for _, s := range ep.Subsets{
			// if upstream need to watch
			// ips
			ips := make([]string, 0)
			for _, address := range s.Addresses{
				ips = append(ips, address.IP)
			}
			// ports
			for _, port := range s.Ports{
				upstreamName := ep.Namespace + "_" + ep.Name + "_" + strconv.Itoa(int(port.Port))
				// find upstreamName is in apisix
				// default
				syncWithGroup("", upstreamName, ips, port)
				// sync with all apisix group
				for g, _ := range sevenConf.UrlGroup {
					syncWithGroup(g, upstreamName, ips, port)
					//upstreams, err :=  apisix.ListUpstream(k)
					//if err == nil {
					//	for _, upstream := range upstreams {
					//		if *(upstream.Name) == upstreamName {
					//			nodes := make([]*apisixType.Node, 0)
					//			for _, ip := range ips {
					//				ipAddress := ip
					//				p := int(port.Port)
					//				weight := 100
					//				node := &apisixType.Node{IP: &ipAddress, Port: &p, Weight: &weight}
					//				nodes = append(nodes, node)
					//			}
					//			upstream.Nodes = nodes
					//			// update upstream nodes
					//			// add to seven solver queue
					//			//apisix.UpdateUpstream(upstream)
					//			fromKind := WatchFromKind
					//			upstream.FromKind = &fromKind
					//			upstreams := []*apisixType.Upstream{upstream}
					//			comb := state.ApisixCombination{Routes: nil, Services: nil, Upstreams: upstreams}
					//			if _, err = comb.Solver(); err != nil {
					//				glog.Errorf(err.Error())
					//			}
					//		}
					//	}
					//}
				}
			}
		}
	}
}

func syncWithGroup(group, upstreamName string, ips []string, port CoreV1.EndpointPort) {
	upstreams, err := apisix.ListUpstream(group)
	if err == nil {
		for _, upstream := range upstreams {
			if *(upstream.Name) == upstreamName {
				nodes := make([]*apisixType.Node, 0)
				for _, ip := range ips {
					ipAddress := ip
					p := int(port.Port)
					weight := 100
					node := &apisixType.Node{IP: &ipAddress, Port: &p, Weight: &weight}
					nodes = append(nodes, node)
				}
				upstream.Nodes = nodes
				// update upstream nodes
				// add to seven solver queue
				//apisix.UpdateUpstream(upstream)
				fromKind := WatchFromKind
				upstream.FromKind = &fromKind
				upstreams := []*apisixType.Upstream{upstream}
				comb := state.ApisixCombination{Routes: nil, Services: nil, Upstreams: upstreams}
				if _, err = comb.Solver(); err != nil {
					glog.Errorf(err.Error())
				}
			}
		}
	}
}

func (c *EndpointController) addFunc(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}

func (c *EndpointController) updateFunc(oldObj, newObj interface{}) {
	oldRoute := oldObj.(*CoreV1.Endpoints)
	newRoute := newObj.(*CoreV1.Endpoints)
	if oldRoute.ResourceVersion == newRoute.ResourceVersion {
		return
	}
	c.addFunc(newObj)
}

func (c *EndpointController) deleteFunc(obj interface{}) {
	var key string
	var err error
	key, err = cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}
