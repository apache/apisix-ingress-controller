package apisix

import (
	"context"
	"fmt"

	"k8s.io/client-go/tools/cache"

	"github.com/apache/apisix-ingress-controller/pkg/config"
	apisixtranslation "github.com/apache/apisix-ingress-controller/pkg/providers/apisix/translation"
	"github.com/apache/apisix-ingress-controller/pkg/providers/k8s"
	"github.com/apache/apisix-ingress-controller/pkg/providers/k8s/namespace"
	"github.com/apache/apisix-ingress-controller/pkg/providers/translation"
	providertypes "github.com/apache/apisix-ingress-controller/pkg/providers/types"
	"github.com/apache/apisix-ingress-controller/pkg/providers/utils"
)

const (
	ProviderName = "APISIX"
)

type apisixCommon struct {
	*providertypes.Common

	namespaceProvider namespace.WatchingNamespaceProvider
	translator        apisixtranslation.ApisixTranslator
}

var _ Provider = (*apisixProvider)(nil)

type Provider interface {
	providertypes.Provider

	ResourceSync()
}

type apisixProvider struct {
	name string

	apisixUpstreamController      *apisixUpstreamController
	apisixRouteController         *apisixRouteController
	apisixTlsController           *apisixTlsController
	apisixClusterConfigController *apisixClusterConfigController
	apisixConsumerController      *apisixConsumerController
	apisixPluginConfigController  *apisixPluginConfigController

	svcInformer    cache.SharedIndexInformer
	secretInformer cache.SharedIndexInformer

	apisixUpstreamInformer      cache.SharedIndexInformer
	apisixRouteInformer         cache.SharedIndexInformer
	apisixTlsInformer           cache.SharedIndexInformer
	apisixClusterConfigInformer cache.SharedIndexInformer
	apisixConsumerInformer      cache.SharedIndexInformer
	apisixPluginConfigInformer  cache.SharedIndexInformer
}

func NewProvider(common *providertypes.Common,
	kubeProvider k8s.Provider, namespaceProvider namespace.WatchingNamespaceProvider,
	translator translation.Translator) (Provider, apisixtranslation.ApisixTranslator, error) {
	p := &apisixProvider{
		name: ProviderName,
	}

	kubeFactory := common.KubeClient.NewSharedIndexInformerFactory()
	apisixFactory := common.KubeClient.NewAPISIXSharedIndexInformerFactory()

	svcLister := kubeFactory.Core().V1().Services().Lister()
	p.svcInformer = kubeFactory.Core().V1().Services().Informer()

	secretLister := kubeFactory.Core().V1().Secrets().Lister()
	p.secretInformer = kubeFactory.Core().V1().Secrets().Informer()

	apisixTranslator := apisixtranslation.NewApisixTranslator(svcLister, secretLister, translator)
	c := &apisixCommon{
		Common:            common,
		namespaceProvider: namespaceProvider,
		translator:        apisixTranslator,
	}

	switch c.Config.Kubernetes.APIVersion {
	case config.ApisixV2beta3:
		p.apisixUpstreamInformer = apisixFactory.Apisix().V2beta3().ApisixUpstreams().Informer()
		p.apisixRouteInformer = apisixFactory.Apisix().V2beta3().ApisixRoutes().Informer()
		p.apisixTlsInformer = apisixFactory.Apisix().V2beta3().ApisixTlses().Informer()
		p.apisixClusterConfigInformer = apisixFactory.Apisix().V2beta3().ApisixClusterConfigs().Informer()
		p.apisixConsumerInformer = apisixFactory.Apisix().V2beta3().ApisixConsumers().Informer()
		p.apisixPluginConfigInformer = apisixFactory.Apisix().V2beta3().ApisixPluginConfigs().Informer()
	case config.ApisixV2:
		p.apisixUpstreamInformer = apisixFactory.Apisix().V2().ApisixUpstreams().Informer()
		p.apisixRouteInformer = apisixFactory.Apisix().V2().ApisixRoutes().Informer()
		p.apisixTlsInformer = apisixFactory.Apisix().V2().ApisixTlses().Informer()
		p.apisixClusterConfigInformer = apisixFactory.Apisix().V2().ApisixClusterConfigs().Informer()
		p.apisixConsumerInformer = apisixFactory.Apisix().V2().ApisixConsumers().Informer()
		p.apisixPluginConfigInformer = apisixFactory.Apisix().V2().ApisixPluginConfigs().Informer()
	default:
		panic(fmt.Errorf("unsupported API version %v", c.Config.Kubernetes.APIVersion))
	}

	p.apisixUpstreamController = newApisixUpstreamController(c, p.svcInformer, svcLister, p.apisixUpstreamInformer)
	p.apisixRouteController = newApisixRouteController(c, p.svcInformer, p.apisixRouteInformer)
	p.apisixTlsController = newApisixTlsController(c, p.secretInformer, p.apisixTlsInformer)
	p.apisixClusterConfigController = newApisixClusterConfigController(c, p.apisixClusterConfigInformer)
	p.apisixConsumerController = newApisixConsumerController(c, p.apisixConsumerInformer)
	p.apisixPluginConfigController = newApisixPluginConfigController(c, p.apisixPluginConfigInformer)

	return p, apisixTranslator, nil
}

func (p *apisixProvider) Run(ctx context.Context) {
	e := utils.ParallelExecutor{}

	e.Add(func() {
		p.svcInformer.Run(ctx.Done())
	})
	e.Add(func() {
		p.secretInformer.Run(ctx.Done())
	})

	e.Add(func() {
		p.apisixUpstreamInformer.Run(ctx.Done())
	})
	e.Add(func() {
		p.apisixRouteInformer.Run(ctx.Done())
	})
	e.Add(func() {
		p.apisixTlsInformer.Run(ctx.Done())
	})
	e.Add(func() {
		p.apisixClusterConfigInformer.Run(ctx.Done())
	})
	e.Add(func() {
		p.apisixConsumerInformer.Run(ctx.Done())
	})
	e.Add(func() {
		p.apisixPluginConfigInformer.Run(ctx.Done())
	})

	e.Add(func() {
		p.apisixUpstreamController.run(ctx)
	})
	e.Add(func() {
		p.apisixRouteController.run(ctx)
	})
	e.Add(func() {
		p.apisixTlsController.run(ctx)
	})
	e.Add(func() {
		p.apisixClusterConfigController.run(ctx)
	})
	e.Add(func() {
		p.apisixConsumerController.run(ctx)
	})
	e.Add(func() {
		p.apisixPluginConfigController.run(ctx)
	})

	e.Wait()
}

func (p *apisixProvider) ResourceSync() {
	e := utils.ParallelExecutor{}

	e.Add(p.apisixUpstreamController.ResourceSync)
	e.Add(p.apisixRouteController.ResourceSync)
	e.Add(p.apisixTlsController.ResourceSync)
	e.Add(p.apisixClusterConfigController.ResourceSync)
	e.Add(p.apisixConsumerController.ResourceSync)
	e.Add(p.apisixPluginConfigController.ResourceSync)

	e.Wait()
}
