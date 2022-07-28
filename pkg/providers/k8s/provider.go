package k8s

import (
	"context"

	"github.com/pkg/errors"

	"github.com/apache/apisix-ingress-controller/pkg/providers/k8s/endpoint"
	"github.com/apache/apisix-ingress-controller/pkg/providers/k8s/namespace"
	"github.com/apache/apisix-ingress-controller/pkg/providers/translation"
	providertypes "github.com/apache/apisix-ingress-controller/pkg/providers/types"
	"github.com/apache/apisix-ingress-controller/pkg/providers/utils"
)

var _ Provider = (*k8sProvider)(nil)

type Provider interface {
	providertypes.Provider
}

type k8sProvider struct {
	secretController *secretController

	endpoint endpoint.Provider
}

func NewProvider(common *providertypes.Common, translator translation.Translator, namespaceProvider namespace.WatchingNamespaceProvider) (Provider, error) {
	var err error
	provider := &k8sProvider{}

	provider.endpoint, err = endpoint.NewProvider(common, translator, namespaceProvider)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init endpoint provider")
	}

	provider.secretController = newSecretController()

	return provider, nil
}

func (p *k8sProvider) Run(ctx context.Context) {
	e := utils.ParallelExecutor{}

	e.Add(func() {
		p.secretController.run(ctx)
	})

	e.Wait()
}
