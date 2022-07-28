package k8s

import (
	"context"
	"github.com/apache/apisix-ingress-controller/pkg/providers/translation"

	"github.com/pkg/errors"

	"github.com/apache/apisix-ingress-controller/pkg/providers/k8s/endpoint"
	"github.com/apache/apisix-ingress-controller/pkg/providers/namespace"
	providertypes "github.com/apache/apisix-ingress-controller/pkg/providers/types"
	"github.com/apache/apisix-ingress-controller/pkg/providers/utils"
	"github.com/apache/apisix-ingress-controller/pkg/types"
)

var _ Provider = (*k8sProvider)(nil)

type Provider interface {
	providertypes.Provider

	GetPodCache() types.PodCache
}

type k8sProvider struct {
	podController    *podController
	secretController *secretController

	endpoint *endpoint.Controller
}

func NewProvider(common *providertypes.Common, translator translation.Translator, namespaceProvider namespace.WatchingNamespaceProvider) (Provider, error) {
	var err error
	provider := &k8sProvider{}

	provider.endpoint, err = endpoint.NewController(common, translator, namespaceProvider)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init endpoint provider")
	}

	kubeFactory := common.KubeClient.NewSharedIndexInformerFactory()
	podInformer := kubeFactory.Core().V1().Pods().Informer()

	provider.podController = newPodController(common, namespaceProvider, podInformer)
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
