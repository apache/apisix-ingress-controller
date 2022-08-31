// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
//
package apisix

import (
	"context"
	"fmt"
	"sync"

	"k8s.io/client-go/tools/cache"

	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/kube"
	apisixtranslation "github.com/apache/apisix-ingress-controller/pkg/providers/apisix/translation"
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

	Init(ctx context.Context) error
	ResourceSync()
	NotifyServiceAdd(key string)

	GetSslFromSecretKey(string) *sync.Map
}

type apisixProvider struct {
	name              string
	common            *providertypes.Common
	namespaceProvider namespace.WatchingNamespaceProvider

	apisixTranslator              apisixtranslation.ApisixTranslator
	apisixUpstreamController      *apisixUpstreamController
	apisixRouteController         *apisixRouteController
	apisixTlsController           *apisixTlsController
	apisixClusterConfigController *apisixClusterConfigController
	apisixConsumerController      *apisixConsumerController
	apisixPluginConfigController  *apisixPluginConfigController

	apisixRouteInformer         cache.SharedIndexInformer
	apisixClusterConfigInformer cache.SharedIndexInformer
	apisixConsumerInformer      cache.SharedIndexInformer
	apisixPluginConfigInformer  cache.SharedIndexInformer
}

func NewProvider(common *providertypes.Common, namespaceProvider namespace.WatchingNamespaceProvider,
	translator translation.Translator) (Provider, apisixtranslation.ApisixTranslator, error) {
	p := &apisixProvider{
		name:              ProviderName,
		common:            common,
		namespaceProvider: namespaceProvider,
	}

	apisixFactory := common.KubeClient.NewAPISIXSharedIndexInformerFactory()

	p.apisixTranslator = apisixtranslation.NewApisixTranslator(&apisixtranslation.TranslatorOptions{
		Apisix:      common.APISIX,
		ClusterName: common.Config.APISIX.DefaultClusterName,

		ApisixUpstreamLister: common.ApisixUpstreamLister,
		ServiceLister:        common.SvcLister,
		SecretLister:         common.SecretLister,
	}, translator)
	c := &apisixCommon{
		Common:            common,
		namespaceProvider: namespaceProvider,
		translator:        p.apisixTranslator,
	}

	switch c.Config.Kubernetes.APIVersion {
	case config.ApisixV2beta3:
		p.apisixRouteInformer = apisixFactory.Apisix().V2beta3().ApisixRoutes().Informer()
		p.apisixClusterConfigInformer = apisixFactory.Apisix().V2beta3().ApisixClusterConfigs().Informer()
		p.apisixConsumerInformer = apisixFactory.Apisix().V2beta3().ApisixConsumers().Informer()
		p.apisixPluginConfigInformer = apisixFactory.Apisix().V2beta3().ApisixPluginConfigs().Informer()

	case config.ApisixV2:
		p.apisixRouteInformer = apisixFactory.Apisix().V2().ApisixRoutes().Informer()
		p.apisixClusterConfigInformer = apisixFactory.Apisix().V2().ApisixClusterConfigs().Informer()
		p.apisixConsumerInformer = apisixFactory.Apisix().V2().ApisixConsumers().Informer()
		p.apisixPluginConfigInformer = apisixFactory.Apisix().V2().ApisixPluginConfigs().Informer()
	default:
		panic(fmt.Errorf("unsupported API version %v", c.Config.Kubernetes.APIVersion))
	}

	apisixRouteLister := kube.NewApisixRouteLister(
		apisixFactory.Apisix().V2beta2().ApisixRoutes().Lister(),
		apisixFactory.Apisix().V2beta3().ApisixRoutes().Lister(),
		apisixFactory.Apisix().V2().ApisixRoutes().Lister(),
	)
	apisixClusterConfigLister := kube.NewApisixClusterConfigLister(
		apisixFactory.Apisix().V2beta3().ApisixClusterConfigs().Lister(),
		apisixFactory.Apisix().V2().ApisixClusterConfigs().Lister(),
	)
	apisixConsumerLister := kube.NewApisixConsumerLister(
		apisixFactory.Apisix().V2beta3().ApisixConsumers().Lister(),
		apisixFactory.Apisix().V2().ApisixConsumers().Lister(),
	)
	apisixPluginConfigLister := kube.NewApisixPluginConfigLister(
		apisixFactory.Apisix().V2beta3().ApisixPluginConfigs().Lister(),
		apisixFactory.Apisix().V2().ApisixPluginConfigs().Lister(),
	)

	p.apisixUpstreamController = newApisixUpstreamController(c, p.NotifyApisixUpstreamAdd)
	p.apisixRouteController = newApisixRouteController(c, p.apisixRouteInformer, apisixRouteLister)
	p.apisixTlsController = newApisixTlsController(c)
	p.apisixClusterConfigController = newApisixClusterConfigController(c, p.apisixClusterConfigInformer, apisixClusterConfigLister)
	p.apisixConsumerController = newApisixConsumerController(c, p.apisixConsumerInformer, apisixConsumerLister)
	p.apisixPluginConfigController = newApisixPluginConfigController(c, p.apisixPluginConfigInformer, apisixPluginConfigLister)

	return p, p.apisixTranslator, nil
}

func (p *apisixProvider) Run(ctx context.Context) {
	e := utils.ParallelExecutor{}

	e.Add(func() {
		p.apisixRouteInformer.Run(ctx.Done())
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

func (p *apisixProvider) NotifyServiceAdd(key string) {
	p.apisixRouteController.NotifyServiceAdd(key)
}

func (p *apisixProvider) NotifyApisixUpstreamAdd(key string) {
	p.apisixRouteController.NotifyApisixUpstreamAdd(key)
}

func (p *apisixProvider) GetSslFromSecretKey(secretMapKey string) *sync.Map {
	ssls, ok := p.apisixTlsController.secretSSLMap.Load(secretMapKey)
	if !ok {
		// This secret is not concerned.
		return nil
	}
	sslMap := ssls.(*sync.Map)
	return sslMap
}
