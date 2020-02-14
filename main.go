package main

import (
	"github.com/iresty/ingress-controller/pkg/ingress/controller"
	"github.com/iresty/ingress-controller/conf"
	api6Informers "github.com/gxthrj/apisix-ingress-types/pkg/client/informers/externalversions"
	coreInformers "k8s.io/client-go/informers"
	"net/http"
	"github.com/iresty/ingress-controller/pkg"
	"github.com/iresty/ingress-controller/log"
)

func main(){
	var logger = log.GetLogger()
	//election.Elect()
	kubeClientSet := conf.InitKubeClient()
	apisixClientset := conf.InitApisixClient()
	sharedInformerFactory := api6Informers.NewSharedInformerFactory(apisixClientset, 0)
	stop := make(chan struct{})
	c := &controller.Api6Controller{
		KubeClientSet: kubeClientSet,
		Api6ClientSet: apisixClientset,
		SharedInformerFactory: sharedInformerFactory,
		Stop: stop,
	}
	// ApisixRoute
	c.ApisixRoute()
	// ApisixUpstream
	c.ApisixUpstream()
	// ApisixService
	c.ApisixService()

	go sharedInformerFactory.Start(stop)

	// endpoint informer
	coreSharedInformerFactory := coreInformers.NewSharedInformerFactory(kubeClientSet, 0)
	conf.EndpointsInformer = coreSharedInformerFactory.Core().V1().Endpoints()
	controller.Watch()
	go conf.EndpointsInformer.Informer().Run(stop)

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

