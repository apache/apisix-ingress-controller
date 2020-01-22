package controller

import (
	"k8s.io/client-go/kubernetes"
	clientset "github.com/gxthrj/apisix-ingress-types/pkg/client/clientset/versioned"
	"github.com/gxthrj/apisix-ingress-types/pkg/client/listers/config/v1"
	apisixV1 "github.com/gxthrj/apisix-ingress-types/pkg/apis/config/v1"
	informers "github.com/gxthrj/apisix-ingress-types/pkg/client/informers/externalversions/config/v1"
	apisixScheme "github.com/gxthrj/apisix-ingress-types/pkg/client/clientset/versioned/scheme"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"github.com/iresty/ingress-controller/log"
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"time"
	"github.com/iresty/ingress-controller/pkg/ingress/apisix"
	"github.com/gxthrj/seven/state"
)

var logger = log.GetLogger()

type Controller struct{
	kubeclientset kubernetes.Interface
	apisixRouteClientset clientset.Interface
	apisixRouteList v1.ApisixRouteLister
	apisixRouteSynced cache.InformerSynced
	workqueue workqueue.RateLimitingInterface
}

func NewApisixRouteController(
	kubeclientset kubernetes.Interface,
	apisixRouteClientset clientset.Interface,
	apisixRouteInformer informers.ApisixRouteInformer) *Controller{

	runtime.Must(apisixScheme.AddToScheme(scheme.Scheme))
	controller := &Controller{
		kubeclientset:    kubeclientset,
		apisixRouteClientset: apisixRouteClientset,
		apisixRouteList:   apisixRouteInformer.Lister(),
		apisixRouteSynced:   apisixRouteInformer.Informer().HasSynced,
		workqueue:        workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Students"),
	}
	apisixRouteInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: controller.addFunc,
			UpdateFunc: controller.updateFunc,
			DeleteFunc: controller.deleteFunc,
		})
	return controller
}

func (c *Controller) Run(stop <-chan struct{}) error {
	//defer c.workqueue.ShutDown()
	// 同步缓存
	if ok := cache.WaitForCacheSync(stop); !ok {
		logger.Errorf("同步缓存失败")
		return fmt.Errorf("failed to wait for caches to sync")
	}
	go wait.Until(c.runWorker, time.Second, stop)
	return nil
}

func (c *Controller) runWorker(){
	for c.processNextWorkItem() {}
}


func (c *Controller) processNextWorkItem() bool {
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

func (c *Controller) syncHandler(key string) error {
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
	// 命名规则 host + path +
	//for _, rule := range apisixRoute.Spec.Rules {
	//	logger.Info(rule.Http.Paths)
	//	for _, path := range rule.Http.Paths {
	//		logger.Info(rule.Host + path.Path)
	//		logger.Info(path.Backend.ServiceName)
	//		logger.Info(path.Backend.ServicePort)
	//	}
	//}
	return err
}

func (c *Controller) addFunc(obj interface{}){
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}

func (c *Controller) updateFunc(oldObj, newObj interface{}){
	oldRoute := oldObj.(*apisixV1.ApisixRoute)
	newRoute := newObj.(*apisixV1.ApisixRoute)
	if oldRoute.ResourceVersion == newRoute.ResourceVersion {
		return
	}
	c.addFunc(newObj)
}

func (c *Controller) deleteFunc(obj interface{}){
	var key string
	var err error
	key, err = cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}