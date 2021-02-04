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

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/apache/apisix-ingress-controller/pkg/ingress/apisix"
	configv1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v1"
	clientset "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/clientset/versioned"
	apisixscheme "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/clientset/versioned/scheme"
	informersv1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/informers/externalversions/config/v1"
	listersv1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/listers/config/v1"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/seven/state"
)

type ApisixTLSController struct {
	controller      *Controller
	kubeclientset   kubernetes.Interface
	apisixClientset clientset.Interface
	apisixTLSList   listersv1.ApisixTlsLister
	apisixTLSSynced cache.InformerSynced
	workqueue       workqueue.RateLimitingInterface
}

type TlsQueueObj struct {
	Key    string              `json:"key"`
	OldObj *configv1.ApisixTls `json:"old_obj"`
	Ope    string              `json:"ope"` // add / update / delete
}

func BuildApisixTlsController(
	kubeclientset kubernetes.Interface,
	apisixTLSClientset clientset.Interface,
	apisixTLSInformer informersv1.ApisixTlsInformer,
	root *Controller) *ApisixTLSController {

	runtime.Must(apisixscheme.AddToScheme(scheme.Scheme))
	controller := &ApisixTLSController{
		controller:      root,
		kubeclientset:   kubeclientset,
		apisixClientset: apisixTLSClientset,
		apisixTLSList:   apisixTLSInformer.Lister(),
		apisixTLSSynced: apisixTLSInformer.Informer().HasSynced,
		workqueue:       workqueue.NewNamedRateLimitingQueue(workqueue.NewItemFastSlowRateLimiter(1*time.Second, 60*time.Second, 5), "ApisixTlses"),
	}
	apisixTLSInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    controller.addFunc,
			UpdateFunc: controller.updateFunc,
			DeleteFunc: controller.deleteFunc,
		})
	return controller
}

func (c *ApisixTLSController) Run(stop <-chan struct{}) error {
	if ok := cache.WaitForCacheSync(stop); !ok {
		log.Errorf("sync ApisixService cache failed")
		return fmt.Errorf("failed to wait for caches to sync")
	}
	go wait.Until(c.runWorker, time.Second, stop)
	return nil
}

func (c *ApisixTLSController) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *ApisixTLSController) processNextWorkItem() bool {
	defer recoverException()
	obj, shutdown := c.workqueue.Get()
	if shutdown {
		return false
	}
	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var key string
		var ok bool

		var tqo *TlsQueueObj
		if tqo, ok = obj.(*TlsQueueObj); !ok {
			c.workqueue.Forget(obj)
			return fmt.Errorf("expected TlsQueueObj in workqueue but got %#v", obj)
		}
		if err := c.syncHandler(tqo); err != nil {
			c.workqueue.AddRateLimited(tqo)
			log.Errorf("sync tls %s failed", tqo.Key)
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

func (c *ApisixTLSController) syncHandler(tqo *TlsQueueObj) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(tqo.Key)
	if err != nil {
		log.Errorf("invalid resource key: %s", tqo.Key)
		return fmt.Errorf("invalid resource key: %s", tqo.Key)
	}
	apisixTlsYaml := tqo.OldObj
	if tqo.Ope == state.Delete {
		apisixIngressTls, _ := c.apisixTLSList.ApisixTlses(namespace).Get(name)
		if apisixIngressTls != nil && apisixIngressTls.ResourceVersion > tqo.OldObj.ResourceVersion {
			log.Warnf("TLS %s has been covered when retry", tqo.Key)
			return nil
		}
	} else {
		apisixTlsYaml, err = c.apisixTLSList.ApisixTlses(namespace).Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				log.Infof("apisixTls %s is removed", tqo.Key)
				return nil
			}
			runtime.HandleError(fmt.Errorf("failed to list apisixTls %s/%s", tqo.Key, err.Error()))
			return err
		}
	}

	apisixTls := apisix.ApisixTLSCRD(*apisixTlsYaml)
	sc := &apisix.SecretClient{}
	if tls, err := apisixTls.Convert(sc); err != nil {
		return err
	} else {
		// sync to apisix
		log.Debug(tls)
		log.Debug(tqo)
		return state.SyncSsl(tls, tqo.Ope)
	}
}

func (c *ApisixTLSController) addFunc(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	if !c.controller.namespaceWatching(key) {
		return
	}
	rqo := &TlsQueueObj{Key: key, OldObj: nil, Ope: state.Create}
	c.workqueue.AddRateLimited(rqo)
}

func (c *ApisixTLSController) updateFunc(oldObj, newObj interface{}) {
	oldTls := oldObj.(*configv1.ApisixTls)
	newTls := newObj.(*configv1.ApisixTls)
	if oldTls.ResourceVersion == newTls.ResourceVersion {
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
	rqo := &TlsQueueObj{Key: key, OldObj: oldTls, Ope: state.Update}
	c.workqueue.AddRateLimited(rqo)
}

func (c *ApisixTLSController) deleteFunc(obj interface{}) {
	oldTls, ok := obj.(*configv1.ApisixTls)
	if !ok {
		oldState, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			return
		}
		oldTls, ok = oldState.Obj.(*configv1.ApisixTls)
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
	rqo := &TlsQueueObj{Key: key, OldObj: oldTls, Ope: state.Delete}
	c.workqueue.AddRateLimited(rqo)
}
