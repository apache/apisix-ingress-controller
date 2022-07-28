package endpoint

import (
	"context"

	"k8s.io/client-go/tools/cache"

	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/kube"
	"github.com/apache/apisix-ingress-controller/pkg/providers/k8s/namespace"
	"github.com/apache/apisix-ingress-controller/pkg/providers/translation"
	providertypes "github.com/apache/apisix-ingress-controller/pkg/providers/types"
	"github.com/apache/apisix-ingress-controller/pkg/providers/utils"
)

var _ Provider = (*endpointProvider)(nil)

type Provider interface {
	providertypes.Provider
}

type endpointProvider struct {
	cfg *config.Config

	epInformer              cache.SharedIndexInformer
	epLister                kube.EndpointLister
	endpointsController     *endpointsController
	endpointSliceController *endpointSliceController
}

func NewProvider(common *providertypes.Common, translator translation.Translator, namespaceProvider namespace.WatchingNamespaceProvider) (Provider, error) {
	p := &endpointProvider{
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

	p.epLister, p.epInformer = kube.NewEndpointListerAndInformer(kubeFactory, common.Config.Kubernetes.WatchEndpointSlices)
	if common.Kubernetes.WatchEndpointSlices {
		p.endpointSliceController = newEndpointSliceController(base, namespaceProvider, p.epInformer, p.epLister)
	} else {
		p.endpointsController = newEndpointsController(base, namespaceProvider, p.epInformer, p.epLister)
	}

	return p, nil
}

func (p *endpointProvider) Run(ctx context.Context) {
	e := utils.ParallelExecutor{}

	e.Add(func() {
		p.epInformer.Run(ctx.Done())
	})

	e.Add(func() {
		if p.cfg.Kubernetes.WatchEndpointSlices {
			p.endpointSliceController.run(ctx)
		} else {
			p.endpointsController.run(ctx)
		}
	})

	e.Wait()
}
