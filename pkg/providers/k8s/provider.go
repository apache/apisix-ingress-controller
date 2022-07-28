package k8s

import (
	"context"
	"github.com/apache/apisix-ingress-controller/pkg/kube"
	"github.com/apache/apisix-ingress-controller/pkg/providers/k8s/endpoint"
	"github.com/apache/apisix-ingress-controller/pkg/providers/namespace"
	providertypes "github.com/apache/apisix-ingress-controller/pkg/providers/types"
	"github.com/apache/apisix-ingress-controller/pkg/providers/utils"
	"github.com/apache/apisix-ingress-controller/pkg/types"
	"github.com/pkg/errors"
)

var _ Provider = (*k8sProvider)(nil)

type Provider interface {
	providertypes.Provider

	GetPodCache() types.PodCache
}

type k8sProvider struct {
	cfg *providertypes.CommonConfig

	podController    *podController
	secretController *secretController

	endpoint  endpoint.Provider
	namespace namespace.WatchingNamespaceProvider
}

func NewProvider(ctx context.Context, kube *kube.KubeClient, cfg *providertypes.CommonConfig) (Provider, error) {
	var err error
	provider := &k8sProvider{
		cfg: cfg,
	}

	provider.endpoint, err = endpoint.NewProvider(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init endpoint provider")
	}
	provider.podController = newPodController()
	provider.secretController = newSecretController()

	return provider, nil
}

func (p *k8sProvider) Run(ctx context.Context) {
	e := utils.ParallelExecutor{}

	e.Add(func() {
		p.podController.run(ctx)
	})
	e.Add(func() {
		p.secretController.run(ctx)
	})

	e.Wait()
}

func (p *k8sProvider) GetPodCache() types.PodCache {
	return p.podController.podCache
}
