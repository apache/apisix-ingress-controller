package main

import (
	"github.com/api7/ingress-controller/pkg/ingress/controller"
	"github.com/api7/ingress-controller/conf"
	api6Informers "github.com/gxthrj/apisix-ingress-types/pkg/client/informers/externalversions"
	"net/http"
	"github.com/api7/ingress-controller/pkg"
	"github.com/api7/ingress-controller/log"
	"time"
	"flag"
	"github.com/golang/glog"
)

func main(){
	flag.Parse()
	defer glog.Flush()
	var logger = log.GetLogger()
	kubeClientSet := conf.GetKubeClient()
	apisixClientset := conf.InitApisixClient()
	sharedInformerFactory := api6Informers.NewSharedInformerFactory(apisixClientset, 0)
	stop := make(chan struct{})
	c := &controller.Api6Controller{
		KubeClientSet: kubeClientSet,
		Api6ClientSet: apisixClientset,
		SharedInformerFactory: sharedInformerFactory,
		CoreSharedInformerFactory: conf.CoreSharedInformerFactory,
		Stop: stop,
	}
	epInformer := c.CoreSharedInformerFactory.Core().V1().Endpoints()
	conf.EndpointsInformer = epInformer

	// endpoint
	c.Endpoint()
	go c.CoreSharedInformerFactory.Start(stop)

	// ApisixRoute
	c.ApisixRoute()
	// ApisixUpstream
	c.ApisixUpstream()
	// ApisixService
	c.ApisixService()

	go func(){
		time.Sleep(time.Duration(10)*time.Second)
		c.SharedInformerFactory.Start(stop)
	}()

	//if err != nil {
	//	fmt.Println(err.Error())
	//}
	// web
	router := pkg.Route()
	err := http.ListenAndServe(":8080", router)
	if err != nil {
		logger.Fatal("ListenAndServe: ", err)
	}
}

