package controller

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"github.com/gxthrj/apisix-ingress-types/pkg/client/listers/config/v1"
	clientSet "github.com/gxthrj/apisix-ingress-types/pkg/client/clientset/versioned"
	api6Informers "github.com/gxthrj/apisix-ingress-types/pkg/client/informers/externalversions/config/v1"
	api6Scheme "github.com/gxthrj/apisix-ingress-types/pkg/client/clientset/versioned/scheme"
	api6V1 "github.com/gxthrj/apisix-ingress-types/pkg/apis/config/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"fmt"
	"k8s.io/apimachinery/pkg/util/wait"
	"time"
	"k8s.io/apimachinery/pkg/api/errors"
	"github.com/iresty/ingress-controller/pkg/ingress/apisix"
	"github.com/gxthrj/seven/state"
)

type ApisixRouteController struct{
	kubeclientset kubernetes.Interface
	apisixRouteClientset clientSet.Interface
	apisixRouteList v1.ApisixRouteLister
	apisixRouteSynced cache.InformerSynced
	workqueue workqueue.RateLimitingInterface
}

func BuildApisixRouteController(
	kubeclientset kubernetes.Interface,
	api6RouteClientset clientSet.Interface,
	api6RouteInformer api6Informers.ApisixRouteInformer) *ApisixRouteController{

	runtime.Must(api6Scheme.AddToScheme(scheme.Scheme))
	controller := &ApisixRouteController{
		kubeclientset:    kubeclientset,
		apisixRouteClientset: api6RouteClientset,
		apisixRouteList:   api6RouteInformer.Lister(),
		apisixRouteSynced:   api6RouteInformer.Informer().HasSynced,
		workqueue:        workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ApisixRoutes"),
	}
	api6RouteInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: controller.addFunc,
			UpdateFunc: controller.updateFunc,
			DeleteFunc: controller.deleteFunc,
		})
	return controller
}

func (c *ApisixRouteController) addFunc(obj interface{}){
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}

func (c *ApisixRouteController) updateFunc(oldObj, newObj interface{}){
	oldRoute := oldObj.(*api6V1.ApisixRoute)
	newRoute := newObj.(*api6V1.ApisixRoute)
	if oldRoute.ResourceVersion == newRoute.ResourceVersion {
		return
	}
	c.addFunc(newObj)
}

func (c *ApisixRouteController) deleteFunc(obj interface{}){
	var key string
	var err error
	key, err = cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}

func (c *ApisixRouteController) Run(stop <-chan struct{}) error {
	//defer c.workqueue.ShutDown()
	// 同步缓存
	if ok := cache.WaitForCacheSync(stop); !ok {
		logger.Errorf("同步缓存失败")
		return fmt.Errorf("failed to wait for caches to sync")
	}
	go wait.Until(c.runWorker, time.Second, stop)
	return nil
}

func (c *ApisixRouteController) runWorker(){
	for c.processNextWorkItem() {}
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

func (c *ApisixRouteController) syncHandler(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		logger.Error("invalid resource key: %s", key)
		return fmt.Errorf("invalid resource key: %s", key)
	}

	apisixIngressRoute, err := c.apisixRouteList.ApisixRoutes(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err){
			logger.Info("apisixRoute %s is removed", key)
			return nil
		}
		runtime.HandleError(fmt.Errorf("failed to list apisixRoute %s/%s", key, err.Error()))
		return err
	}
	logger.Info(namespace)
	logger.Info(name)
	apisixRoute := apisix.ApisixRoute(*apisixIngressRoute)
	routes, services, upstreams, _ := apisixRoute.Convert()
	comb := state.ApisixCombination{Routes: routes, Services: services, Upstreams: upstreams}
	_, err = comb.Solver()
	return err
}