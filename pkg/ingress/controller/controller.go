// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package controller

import (
	"os"
	"sync"

	v1 "k8s.io/api/core/v1"

	"k8s.io/client-go/tools/cache"

	"github.com/api7/ingress-controller/pkg/apisix"

	clientSet "github.com/gxthrj/apisix-ingress-types/pkg/client/clientset/versioned"
	crdclientset "github.com/gxthrj/apisix-ingress-types/pkg/client/clientset/versioned"
	"github.com/gxthrj/apisix-ingress-types/pkg/client/informers/externalversions"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	"github.com/api7/ingress-controller/pkg/api"
	"github.com/api7/ingress-controller/pkg/config"
	"github.com/api7/ingress-controller/pkg/kube"
	"github.com/api7/ingress-controller/pkg/log"
	"github.com/api7/ingress-controller/pkg/metrics"
	"github.com/api7/ingress-controller/pkg/seven/conf"
)

// recover any exception
func recoverException() {
	if err := recover(); err != nil {
		log.Error(err)
	}
}

// Controller is the ingress apisix controller object.
type Controller struct {
	wg                 sync.WaitGroup
	watchingNamespace  map[string]struct{}
	apiServer          *api.Server
	clientset          kubernetes.Interface
	crdClientset       crdclientset.Interface
	metricsCollector   metrics.Collector
	crdController      *Api6Controller
	crdInformerFactory externalversions.SharedInformerFactory
}

// NewController creates an ingress apisix controller object.
func NewController(cfg *config.Config) (*Controller, error) {
	podName := os.Getenv("POD_NAME")
	podNamespace := os.Getenv("POD_NAMESPACE")
	if podNamespace == "" {
		podNamespace = "default"
	}

	client, err := apisix.NewForOptions(&apisix.ClusterOptions{
		Name:     "",
		AdminKey: cfg.APISIX.AdminKey,
		BaseURL:  cfg.APISIX.BaseURL,
	})
	if err != nil {
		return nil, err
	}
	conf.SetAPISIXClient(client)

	if err := kube.InitInformer(cfg); err != nil {
		return nil, err
	}

	apiSrv, err := api.NewServer(cfg)
	if err != nil {
		return nil, err
	}

	crdClientset := kube.GetApisixClient()
	sharedInformerFactory := externalversions.NewSharedInformerFactory(crdClientset, cfg.Kubernetes.ResyncInterval.Duration)

	var watchingNamespace map[string]struct{}
	if len(cfg.Kubernetes.AppNamespaces) > 1 || cfg.Kubernetes.AppNamespaces[0] != v1.NamespaceAll {
		watchingNamespace = make(map[string]struct{}, len(cfg.Kubernetes.AppNamespaces))
		for _, ns := range cfg.Kubernetes.AppNamespaces {
			watchingNamespace[ns] = struct{}{}
		}
	}

	c := &Controller{
		apiServer:          apiSrv,
		metricsCollector:   metrics.NewPrometheusCollector(podName, podNamespace),
		clientset:          kube.GetKubeClient(),
		crdClientset:       crdClientset,
		crdInformerFactory: sharedInformerFactory,
		watchingNamespace:  watchingNamespace,
	}

	return c, nil
}

func (c *Controller) goAttach(handler func()) {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		handler()
	}()
}

// Run launches the controller.
func (c *Controller) Run(stop chan struct{}) error {
	// TODO leader election.
	c.metricsCollector.ResetLeader(true)
	log.Info("controller run as leader")

	ac := &Api6Controller{
		KubeClientSet:             c.clientset,
		Api6ClientSet:             c.crdClientset,
		SharedInformerFactory:     c.crdInformerFactory,
		CoreSharedInformerFactory: kube.CoreSharedInformerFactory,
		Stop:                      stop,
	}
	epInformer := ac.CoreSharedInformerFactory.Core().V1().Endpoints()
	kube.EndpointsInformer = epInformer
	// endpoint
	ac.Endpoint(c)
	c.goAttach(func() {
		ac.CoreSharedInformerFactory.Start(stop)
	})
	c.goAttach(func() {
		if err := c.apiServer.Run(stop); err != nil {
			log.Errorf("failed to launch API Server: %s", err)
		}
	})

	// ApisixRoute
	ac.ApisixRoute(c)
	// ApisixUpstream
	ac.ApisixUpstream(c)
	// ApisixService
	ac.ApisixService(c)
	// ApisixTLS
	ac.ApisixTLS(c)

	c.goAttach(func() {
		ac.SharedInformerFactory.Start(stop)
	})

	<-stop
	c.wg.Wait()
	return nil
}

// namespaceWatching accepts a resource key, getting the namespace part
// and checking whether the namespace is being watched.
func (c *Controller) namespaceWatching(key string) (ok bool) {
	if c.watchingNamespace == nil {
		ok = true
		return
	}
	ns, _, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		// Ignore resource with invalid key.
		ok = false
		log.Warnf("resource %s was ignored since: %s", key, err)
		return
	}
	_, ok = c.watchingNamespace[ns]
	return
}

type Api6Controller struct {
	KubeClientSet             kubernetes.Interface
	Api6ClientSet             clientSet.Interface
	SharedInformerFactory     externalversions.SharedInformerFactory
	CoreSharedInformerFactory informers.SharedInformerFactory
	Stop                      chan struct{}
}

func (api6 *Api6Controller) ApisixRoute(controller *Controller) {
	arc := BuildApisixRouteController(
		api6.KubeClientSet,
		api6.Api6ClientSet,
		api6.SharedInformerFactory.Apisix().V1().ApisixRoutes(),
		controller)
	arc.Run(api6.Stop)
}

func (api6 *Api6Controller) ApisixUpstream(controller *Controller) {
	auc := BuildApisixUpstreamController(
		api6.KubeClientSet,
		api6.Api6ClientSet,
		api6.SharedInformerFactory.Apisix().V1().ApisixUpstreams(),
		controller)
	auc.Run(api6.Stop)
}

func (api6 *Api6Controller) ApisixService(controller *Controller) {
	auc := BuildApisixServiceController(
		api6.KubeClientSet,
		api6.Api6ClientSet,
		api6.SharedInformerFactory.Apisix().V1().ApisixServices(),
		controller)
	auc.Run(api6.Stop)
}

func (api6 *Api6Controller) ApisixTLS(controller *Controller) {
	auc := BuildApisixTlsController(
		api6.KubeClientSet,
		api6.Api6ClientSet,
		api6.SharedInformerFactory.Apisix().V1().ApisixTlses(),
		controller)
	auc.Run(api6.Stop)
}

func (api6 *Api6Controller) Endpoint(controller *Controller) {
	auc := BuildEndpointController(api6.KubeClientSet, controller)
	//conf.EndpointsInformer)
	auc.Run(api6.Stop)
}
