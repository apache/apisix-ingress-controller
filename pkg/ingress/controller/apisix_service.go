package controller

import (
	clientSet "github.com/gxthrj/apisix-ingress-types/pkg/client/clientset/versioned"
	informers "github.com/gxthrj/apisix-ingress-types/pkg/client/informers/externalversions/config/v1"
	apisixScheme "github.com/gxthrj/apisix-ingress-types/pkg/client/clientset/versioned/scheme"
	apisixV1 "github.com/gxthrj/apisix-ingress-types/pkg/apis/config/v1"
	"github.com/gxthrj/apisix-ingress-types/pkg/client/listers/config/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"fmt"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/util/wait"
	"time"
	"k8s.io/apimachinery/pkg/api/errors"
	"github.com/api7/ingress-controller/pkg/ingress/apisix"
	"github.com/gxthrj/seven/state"
)

type ApisixServiceController struct {
	kubeclientset          kubernetes.Interface
	apisixClientset clientSet.Interface
	apisixServiceList      v1.ApisixServiceLister
	apisixServiceSynced    cache.InformerSynced
	workqueue              workqueue.RateLimitingInterface
}

func BuildApisixServiceController(
	kubeclientset kubernetes.Interface,
	apisixServiceClientset clientSet.Interface,
	apisixServiceInformer informers.ApisixServiceInformer) *ApisixServiceController {

	runtime.Must(apisixScheme.AddToScheme(scheme.Scheme))
	controller := &ApisixServiceController{
		kubeclientset:        kubeclientset,
		apisixClientset:      apisixServiceClientset,
		apisixServiceList:   apisixServiceInformer.Lister(),
		apisixServiceSynced: apisixServiceInformer.Informer().HasSynced,
		workqueue:            workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ApisixServices"),
	}
	apisixServiceInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    controller.addFunc,
			UpdateFunc: controller.updateFunc,
			DeleteFunc: controller.deleteFunc,
		})
	return controller
}

func (c *ApisixServiceController) Run(stop <-chan struct{}) error {
	// 同步缓存
	if ok := cache.WaitForCacheSync(stop); !ok {
		glog.Errorf("同步ApisixService缓存失败")
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

func (c *ApisixServiceController) syncHandler(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		logger.Error("invalid resource key: %s", key)
		return fmt.Errorf("invalid resource key: %s", key)
	}

	apisixServiceYaml, err := c.apisixServiceList.ApisixServices(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("apisixUpstream %s is removed", key)
			return nil
		}
		runtime.HandleError(fmt.Errorf("failed to list apisixUpstream %s/%s", key, err.Error()))
		return err
	}
	logger.Info(namespace)
	logger.Info(name)
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
	c.workqueue.AddRateLimited(key)
}

func (c *ApisixServiceController) updateFunc(oldObj, newObj interface{}) {
	oldRoute := oldObj.(*apisixV1.ApisixService)
	newRoute := newObj.(*apisixV1.ApisixService)
	if oldRoute.ResourceVersion == newRoute.ResourceVersion {
		return
	}
	c.addFunc(newObj)
}

func (c *ApisixServiceController) deleteFunc(obj interface{}) {
	var key string
	var err error
	key, err = cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}
