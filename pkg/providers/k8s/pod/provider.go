package pod

import (
	"context"

	"k8s.io/client-go/tools/cache"

	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/providers/k8s/namespace"
	providertypes "github.com/apache/apisix-ingress-controller/pkg/providers/types"
	"github.com/apache/apisix-ingress-controller/pkg/providers/utils"
	"github.com/apache/apisix-ingress-controller/pkg/types"
)

var _ Provider = (*podProvider)(nil)

type Provider interface {
	providertypes.Provider

	GetPodCache() types.PodCache
}

type podProvider struct {
	cfg *config.Config

	podInformer   cache.SharedIndexInformer
	podController *podController
}

func NewProvider(common *providertypes.Common, namespaceProvider namespace.WatchingNamespaceProvider) (Provider, error) {
	p := &podProvider{
		cfg: common.Config,
	}

	kubeFactory := common.KubeClient.NewSharedIndexInformerFactory()

	p.podInformer = kubeFactory.Core().V1().Pods().Informer()
	p.podController = newPodController(common, namespaceProvider, p.podInformer)

	return p, nil
}

func (p *podProvider) Run(ctx context.Context) {
	e := utils.ParallelExecutor{}

	e.Add(func() {
		p.podInformer.Run(ctx.Done())
	})

	e.Add(func() {
		p.podController.run(ctx)
	})

	e.Wait()
}

func (p *podProvider) GetPodCache() types.PodCache {
	return p.podController.podCache
}
