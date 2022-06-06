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
package gateway

import (
	"context"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	gatewayclientset "sigs.k8s.io/gateway-api/pkg/client/clientset/gateway/versioned"
	gatewayexternalversions "sigs.k8s.io/gateway-api/pkg/client/informers/gateway/externalversions"
	gatewaylistersv1alpha2 "sigs.k8s.io/gateway-api/pkg/client/listers/gateway/apis/v1alpha2"

	"github.com/apache/apisix-ingress-controller/pkg/apisix"
	"github.com/apache/apisix-ingress-controller/pkg/config"
	gatewaytranslation "github.com/apache/apisix-ingress-controller/pkg/ingress/gateway/translation"
	"github.com/apache/apisix-ingress-controller/pkg/ingress/namespace"
	"github.com/apache/apisix-ingress-controller/pkg/ingress/utils"
	"github.com/apache/apisix-ingress-controller/pkg/kube"
	"github.com/apache/apisix-ingress-controller/pkg/kube/translation"
	"github.com/apache/apisix-ingress-controller/pkg/metrics"
)

const (
	ProviderName = "GatewayAPI"
)

type Provider struct {
	name string

	*ProviderOptions
	gatewayClient gatewayclientset.Interface

	translator gatewaytranslation.Translator

	gatewayController *gatewayController
	gatewayInformer   cache.SharedIndexInformer
	gatewayLister     gatewaylistersv1alpha2.GatewayLister

	gatewayClassController *gatewayClassController
	gatewayClassInformer   cache.SharedIndexInformer
	gatewayClassLister     gatewaylistersv1alpha2.GatewayClassLister

	gatewayHTTPRouteController *gatewayHTTPRouteController
	gatewayHTTPRouteInformer   cache.SharedIndexInformer
	gatewayHTTPRouteLister     gatewaylistersv1alpha2.HTTPRouteLister
}

type ProviderOptions struct {
	Cfg               *config.Config
	APISIX            apisix.APISIX
	APISIXClusterName string
	KubeTranslator    translation.Translator
	RestConfig        *rest.Config
	KubeClient        kubernetes.Interface
	MetricsCollector  metrics.Collector
	NamespaceProvider namespace.WatchingProvider
}

func NewGatewayProvider(opts *ProviderOptions) (*Provider, error) {
	var err error
	if opts.RestConfig == nil {
		restConfig, err := kube.BuildRestConfig(opts.Cfg.Kubernetes.Kubeconfig, "")
		if err != nil {
			return nil, err
		}

		opts.RestConfig = restConfig
	}
	gatewayKubeClient, err := gatewayclientset.NewForConfig(opts.RestConfig)
	if err != nil {
		return nil, err
	}

	p := &Provider{
		name: ProviderName,

		ProviderOptions: opts,
		gatewayClient:   gatewayKubeClient,

		translator: gatewaytranslation.NewTranslator(&gatewaytranslation.TranslatorOptions{
			KubeTranslator: opts.KubeTranslator,
		}),
	}

	gatewayFactory := gatewayexternalversions.NewSharedInformerFactory(p.gatewayClient, p.Cfg.Kubernetes.ResyncInterval.Duration)

	p.gatewayLister = gatewayFactory.Gateway().V1alpha2().Gateways().Lister()
	p.gatewayInformer = gatewayFactory.Gateway().V1alpha2().Gateways().Informer()

	p.gatewayClassLister = gatewayFactory.Gateway().V1alpha2().GatewayClasses().Lister()
	p.gatewayClassInformer = gatewayFactory.Gateway().V1alpha2().GatewayClasses().Informer()

	p.gatewayHTTPRouteLister = gatewayFactory.Gateway().V1alpha2().HTTPRoutes().Lister()
	p.gatewayHTTPRouteInformer = gatewayFactory.Gateway().V1alpha2().HTTPRoutes().Informer()

	p.gatewayController = newGatewayController(p)

	p.gatewayClassController, err = newGatewayClassController(p)
	if err != nil {
		return nil, err
	}

	p.gatewayHTTPRouteController = newGatewayHTTPRouteController(p)

	return p, nil
}

func (p *Provider) Run(ctx context.Context) {
	e := utils.ParallelExecutor{}

	e.Add(func() {
		p.gatewayInformer.Run(ctx.Done())
	})

	e.Add(func() {
		p.gatewayClassInformer.Run(ctx.Done())
	})

	e.Add(func() {
		p.gatewayHTTPRouteInformer.Run(ctx.Done())
	})

	e.Add(func() {
		p.gatewayController.run(ctx)
	})

	e.Add(func() {
		p.gatewayHTTPRouteController.run(ctx)
	})

	e.Wait()
}
