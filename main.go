package main

import (
	"github.com/iresty/ingress-controller/pkg/ingress/controller"
	"github.com/iresty/ingress-controller/conf"
	informers "github.com/gxthrj/apisix-ingress-types/pkg/client/informers/externalversions"
	coreInformers "k8s.io/client-go/informers"
	"fmt"
	"net/http"
	"github.com/iresty/ingress-controller/pkg"
	"github.com/iresty/ingress-controller/log"
)

func main(){
	var logger = log.GetLogger()
	fmt.Println("123")
	//election.Elect()
	kubeClient := conf.InitKubeClient()
	apisixRouteClientset := conf.InitApisixRoute()
	sharedInformerFactory := informers.NewSharedInformerFactory(apisixRouteClientset, 0)
	coreSharedInformerFactory := coreInformers.NewSharedInformerFactory(kubeClient, 0)
	controller := controller.NewApisixRouteController(
		kubeClient,
		apisixRouteClientset,
		sharedInformerFactory.Apisix().V1().ApisixRoutes())
	stop := make(chan struct{})
	err := controller.Run(stop)
	go sharedInformerFactory.Start(stop)
	// endpoint informer
	conf.EndpointsInformer = coreSharedInformerFactory.Core().V1().Endpoints()
	go conf.EndpointsInformer.Informer().Run(stop)

	if err != nil {
		fmt.Println(err.Error())
	}
	// web
	router := pkg.Route()
	err = http.ListenAndServe(":8080", router)
	if err != nil {
		logger.Fatal("ListenAndServe: ", err)
	}
}

