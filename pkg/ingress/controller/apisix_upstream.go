package controller

import (
	"fmt"
	"github.com/golang/glog"
	apisixV1 "github.com/gxthrj/apisix-ingress-types/pkg/apis/config/v1"
	clientSet "github.com/gxthrj/apisix-ingress-types/pkg/client/clientset/versioned"
	apisixScheme "github.com/gxthrj/apisix-ingress-types/pkg/client/clientset/versioned/scheme"
	informers "github.com/gxthrj/apisix-ingress-types/pkg/client/informers/externalversions/config/v1"
	"github.com/gxthrj/apisix-ingress-types/pkg/client/listers/config/v1"
	"github.com/gxthrj/seven/state"
	"github.com/iresty/ingress-controller/pkg/ingress/apisix"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"time"
)

type ApisixUpstreamController struct {
	kubeclientset        kubernetes.Interface
	apisixRouteClientset clientSet.Interface
	apisixRouteList      v1.ApisixUpstreamLister
	apisixRouteSynced    cache.InformerSynced
	workqueue            workqueue.RateLimitingInterface
}

func BuildApisixUpstreamController(
	kubeclientset kubernetes.Interface,
	apisixUpstreamClientset clientSet.Interface,
	apisixUpstreamInformer informers.ApisixUpstreamInformer) *ApisixUpstreamController {

	runtime.Must(apisixScheme.AddToScheme(scheme.Scheme))
	controller := &ApisixUpstreamController{
		kubeclientset:        kubeclientset,
		apisixRouteClientset: apisixUpstreamClientset,
		apisixRouteList:      apisixUpstreamInformer.Lister(),
		apisixRouteSynced:    apisixUpstreamInformer.Informer().HasSynced,
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
		glog.Errorf("同步ApisixUpstream缓存失败")
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
		logger.Error("invalid resource key: %s", key)
		return fmt.Errorf("invalid resource key: %s", key)
	}

	apisixUpstreamYaml, err := c.apisixRouteList.ApisixUpstreams(namespace).Get(name)
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
	apisixUpstream := apisix.ApisixUpstreamCRD(*apisixUpstreamYaml)
	upstreams, _ := apisixUpstream.Convert()
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
