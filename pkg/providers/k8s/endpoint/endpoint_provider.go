package endpoint

import (
	"context"
	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/kube"
	"github.com/apache/apisix-ingress-controller/pkg/providers/namespace"
	"github.com/apache/apisix-ingress-controller/pkg/providers/translation"
	providertypes "github.com/apache/apisix-ingress-controller/pkg/providers/types"
	"github.com/apache/apisix-ingress-controller/pkg/providers/utils"
)

type Controller struct {
	cfg *config.Config

	endpointsController     *endpointsController
	endpointSliceController *endpointSliceController
}

func NewController(common *providertypes.Common, translator translation.Translator, namespaceProvider namespace.WatchingNamespaceProvider) (*Controller, error) {
	p := &Controller{
		cfg: common.Config,
	}

	base := &baseEndpointController{
		Common:     common,
		translator: translator,
	}

	kubeFactory := common.KubeClient.NewSharedIndexInformerFactory()
	apisixFactory := common.KubeClient.NewAPISIXSharedIndexInformerFactory()

	base.svcLister = kubeFactory.Core().V1().Services().Lister()
	base.apisixUpstreamLister = kube.NewApisixUpstreamLister(
		apisixFactory.Apisix().V2beta3().ApisixUpstreams().Lister(),
		apisixFactory.Apisix().V2().ApisixUpstreams().Lister(),
	)

	epLister, epInformer := kube.NewEndpointListerAndInformer(kubeFactory, common.Config.Kubernetes.WatchEndpointSlices)
	if common.Kubernetes.WatchEndpointSlices {
		p.endpointSliceController = newEndpointSliceController(base, namespaceProvider, epInformer, epLister)
	} else {
		p.endpointsController = newEndpointsController(base, namespaceProvider, epInformer, epLister)
	}

	return p, nil
}

func (p *Controller) Run(ctx context.Context) {
	e := utils.ParallelExecutor{}

	e.Add(func() {
		if p.cfg.Kubernetes.WatchEndpointSlices {
			p.endpointSliceController.run(ctx)
		} else {
			p.endpointsController.run(ctx)
		}
	})

	e.Wait()
}
