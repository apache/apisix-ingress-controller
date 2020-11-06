package controller

import (
	"github.com/golang/glog"
	clientSet "github.com/gxthrj/apisix-ingress-types/pkg/client/clientset/versioned"
	"github.com/gxthrj/apisix-ingress-types/pkg/client/informers/externalversions"
	"github.com/api7/ingress-controller/log"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/informers"
)

var logger = log.GetLogger()

// recover any exception
func recoverException() {
	if err := recover(); err != nil {
		glog.Error(err)
	}
}

type Api6Controller struct {
	KubeClientSet         kubernetes.Interface
	Api6ClientSet    clientSet.Interface
	SharedInformerFactory externalversions.SharedInformerFactory
	CoreSharedInformerFactory informers.SharedInformerFactory
	Stop                  chan struct{}
}

func (api6 *Api6Controller) ApisixRoute() {
	arc := BuildApisixRouteController(
		api6.KubeClientSet,
		api6.Api6ClientSet,
		api6.SharedInformerFactory.Apisix().V1().ApisixRoutes())
	arc.Run(api6.Stop)
}

func (api6 *Api6Controller) ApisixUpstream() {
	auc := BuildApisixUpstreamController(
		api6.KubeClientSet,
		api6.Api6ClientSet,
		api6.SharedInformerFactory.Apisix().V1().ApisixUpstreams())
	auc.Run(api6.Stop)
}

func (api6 *Api6Controller) ApisixService() {
	auc := BuildApisixServiceController(
		api6.KubeClientSet,
		api6.Api6ClientSet,
		api6.SharedInformerFactory.Apisix().V1().ApisixServices())
	auc.Run(api6.Stop)
}

func (api6 *Api6Controller) Endpoint() {
	auc := BuildEndpointController(api6.KubeClientSet)
	//conf.EndpointsInformer)
	auc.Run(api6.Stop)
}