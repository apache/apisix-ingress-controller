// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
package gateway

import (
	"context"
	"fmt"
	"sync"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
	gatewayclientset "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
	gatewayexternalversions "sigs.k8s.io/gateway-api/pkg/client/informers/externalversions"
	gatewaylistersv1alpha2 "sigs.k8s.io/gateway-api/pkg/client/listers/apis/v1alpha2"
	gatewaylistersv1beta1 "sigs.k8s.io/gateway-api/pkg/client/listers/apis/v1beta1"

	"github.com/apache/apisix-ingress-controller/pkg/apisix"
	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/kube"
	"github.com/apache/apisix-ingress-controller/pkg/metrics"
	gatewaytranslation "github.com/apache/apisix-ingress-controller/pkg/providers/gateway/translation"
	"github.com/apache/apisix-ingress-controller/pkg/providers/gateway/types"
	"github.com/apache/apisix-ingress-controller/pkg/providers/k8s/namespace"
	"github.com/apache/apisix-ingress-controller/pkg/providers/translation"
	"github.com/apache/apisix-ingress-controller/pkg/providers/utils"
)

const (
	ProviderName = "GatewayAPI"
)

type Provider struct {
	name string

	gatewayClassesLock sync.RWMutex
	// key is "name" of GatewayClass
	gatewayClasses map[string]struct{}

	listenersLock sync.RWMutex
	// meta key ("ns/name") of Gateway -> section name -> ListenerConf
	listeners     map[string]map[string]*types.ListenerConf
	portListeners map[gatewayv1beta1.PortNumber]*types.ListenerConf

	*ProviderOptions
	gatewayClient gatewayclientset.Interface

	translator gatewaytranslation.Translator

	gatewayController *gatewayController
	gatewayInformer   cache.SharedIndexInformer
	gatewayLister     gatewaylistersv1beta1.GatewayLister

	gatewayClassController *gatewayClassController
	gatewayClassInformer   cache.SharedIndexInformer
	gatewayClassLister     gatewaylistersv1beta1.GatewayClassLister

	gatewayHTTPRouteController *gatewayHTTPRouteController
	gatewayHTTPRouteInformer   cache.SharedIndexInformer
	gatewayHTTPRouteLister     gatewaylistersv1beta1.HTTPRouteLister

	gatewayTLSRouteController *gatewayTLSRouteController
	gatewayTLSRouteInformer   cache.SharedIndexInformer
	gatewayTLSRouteLister     gatewaylistersv1alpha2.TLSRouteLister

	gatewayTCPRouteController *gatewayTCPRouteController
	gatewayTCPRouteInformer   cache.SharedIndexInformer
	gatewayTCPRouteLister     gatewaylistersv1alpha2.TCPRouteLister

	gatewayUDPRouteController *gatewayUDPRouteController
	gatewayUDPRouteInformer   cache.SharedIndexInformer
	gatewayUDPRouteLister     gatewaylistersv1alpha2.UDPRouteLister
}

type ProviderOptions struct {
	Cfg               *config.Config
	APISIX            apisix.APISIX
	APISIXClusterName string
	KubeTranslator    translation.Translator
	RestConfig        *rest.Config
	KubeClient        kubernetes.Interface
	MetricsCollector  metrics.Collector
	NamespaceProvider namespace.WatchingNamespaceProvider
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

		gatewayClasses: make(map[string]struct{}),

		listeners:     make(map[string]map[string]*types.ListenerConf),
		portListeners: make(map[gatewayv1beta1.PortNumber]*types.ListenerConf),

		ProviderOptions: opts,
		gatewayClient:   gatewayKubeClient,

		translator: gatewaytranslation.NewTranslator(&gatewaytranslation.TranslatorOptions{
			KubeTranslator: opts.KubeTranslator,
		}),
	}

	gatewayFactory := gatewayexternalversions.NewSharedInformerFactory(p.gatewayClient, p.Cfg.Kubernetes.ResyncInterval.Duration)

	p.gatewayLister = gatewayFactory.Gateway().V1beta1().Gateways().Lister()
	p.gatewayInformer = gatewayFactory.Gateway().V1beta1().Gateways().Informer()

	p.gatewayClassLister = gatewayFactory.Gateway().V1beta1().GatewayClasses().Lister()
	p.gatewayClassInformer = gatewayFactory.Gateway().V1beta1().GatewayClasses().Informer()

	p.gatewayHTTPRouteLister = gatewayFactory.Gateway().V1beta1().HTTPRoutes().Lister()
	p.gatewayHTTPRouteInformer = gatewayFactory.Gateway().V1beta1().HTTPRoutes().Informer()

	p.gatewayTLSRouteLister = gatewayFactory.Gateway().V1alpha2().TLSRoutes().Lister()
	p.gatewayTLSRouteInformer = gatewayFactory.Gateway().V1alpha2().TLSRoutes().Informer()

	p.gatewayTCPRouteLister = gatewayFactory.Gateway().V1alpha2().TCPRoutes().Lister()
	p.gatewayTCPRouteInformer = gatewayFactory.Gateway().V1alpha2().TCPRoutes().Informer()

	p.gatewayUDPRouteLister = gatewayFactory.Gateway().V1alpha2().UDPRoutes().Lister()
	p.gatewayUDPRouteInformer = gatewayFactory.Gateway().V1alpha2().UDPRoutes().Informer()

	p.gatewayController = newGatewayController(p)

	p.gatewayClassController, err = newGatewayClassController(p)
	if err != nil {
		return nil, err
	}

	p.gatewayHTTPRouteController = newGatewayHTTPRouteController(p)

	p.gatewayTLSRouteController = newGatewayTLSRouteController(p)
	p.gatewayUDPRouteController = newGatewayUDPRouteController(p)

	p.gatewayTCPRouteController = newGatewayTCPRouteController(p)

	return p, nil
}

func (p *Provider) Run(ctx context.Context) {
	e := utils.ParallelExecutor{}

	// Run informer
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
		p.gatewayTLSRouteInformer.Run(ctx.Done())
	})
	e.Add(func() {
		p.gatewayTCPRouteInformer.Run(ctx.Done())
	})

	// Run Controller
	e.Add(func() {
		p.gatewayUDPRouteInformer.Run(ctx.Done())
	})

	e.Add(func() {
		p.gatewayController.run(ctx)
	})
	e.Add(func() {
		p.gatewayClassController.run(ctx)
	})
	e.Add(func() {
		p.gatewayHTTPRouteController.run(ctx)
	})
	e.Add(func() {
		p.gatewayTLSRouteController.run(ctx)
	})
	e.Add(func() {
		p.gatewayTCPRouteController.run(ctx)
	})

	e.Add(func() {
		p.gatewayUDPRouteController.run(ctx)
	})

	e.Wait()
}

func (p *Provider) AddGatewayClass(name string) {
	p.gatewayClassesLock.Lock()
	defer p.gatewayClassesLock.Unlock()

	p.gatewayClasses[name] = struct{}{}
}

func (p *Provider) RemoveGatewayClass(name string) {
	p.gatewayClassesLock.Lock()
	defer p.gatewayClassesLock.Unlock()

	delete(p.gatewayClasses, name)
}

func (p *Provider) HasGatewayClass(name string) bool {
	p.gatewayClassesLock.RLock()
	defer p.gatewayClassesLock.RUnlock()

	_, ok := p.gatewayClasses[name]
	return ok
}

func (p *Provider) AddListeners(ns, name string, listeners map[string]*types.ListenerConf) error {
	p.listenersLock.Lock()
	defer p.listenersLock.Unlock()

	key := ns + "/" + name

	// Check port conflicts
	for _, listenerConf := range listeners {
		if allocated, found := p.portListeners[listenerConf.Port]; found {
			// TODO: support multi-error
			return fmt.Errorf("port %d already allocated by %s/%s section %s",
				listenerConf.Port, allocated.Namespace, allocated.Name, allocated.SectionName)
		}
	}

	previousListeners, ok := p.listeners[key]
	if ok {
		// remove previous listeners
		for _, listenerConf := range previousListeners {
			delete(p.portListeners, listenerConf.Port)
		}
	}

	// save data
	p.listeners[key] = listeners

	for _, listenerConf := range listeners {
		p.portListeners[listenerConf.Port] = listenerConf
	}

	return nil
}

func (p *Provider) RemoveListeners(ns, name string) error {

	return nil
}

func (p *Provider) FindListener(ns, name, sectionName string) (*types.ListenerConf, error) {

	return nil, nil
}
