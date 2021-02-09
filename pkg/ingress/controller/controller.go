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
	"context"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"

	"github.com/apache/apisix-ingress-controller/pkg/api"
	"github.com/apache/apisix-ingress-controller/pkg/apisix"
	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/kube"
	clientset "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/clientset/versioned"
	crdclientset "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/clientset/versioned"
	"github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/informers/externalversions"
	listersv1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/listers/config/v1"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/metrics"
	"github.com/apache/apisix-ingress-controller/pkg/seven/conf"
)

// recover any exception
func recoverException() {
	if err := recover(); err != nil {
		log.Error(err)
	}
}

// Controller is the ingress apisix controller object.
type Controller struct {
	name               string
	namespace          string
	cfg                *config.Config
	wg                 sync.WaitGroup
	watchingNamespace  map[string]struct{}
	apisix             apisix.APISIX
	translator         kube.Translator
	apiServer          *api.Server
	clientset          kubernetes.Interface
	crdClientset       crdclientset.Interface
	metricsCollector   metrics.Collector
	crdController      *Api6Controller
	crdInformerFactory externalversions.SharedInformerFactory

	// common informers and listers
	epInformer             cache.SharedIndexInformer
	epLister               listerscorev1.EndpointsLister
	svcInformer            cache.SharedIndexInformer
	svcLister              listerscorev1.ServiceLister
	ingressLister          kube.IngressLister
	ingressV1Informer      cache.SharedIndexInformer
	ingressV1beta1Informer cache.SharedIndexInformer
	apisixUpstreamInformer cache.SharedIndexInformer
	apisixUpstreamLister   listersv1.ApisixUpstreamLister

	// resource conrollers
	endpointsController      *endpointsController
	ingressController        *ingressController
	apisixUpstreamController *apisixUpstreamController
}

// NewController creates an ingress apisix controller object.
func NewController(cfg *config.Config) (*Controller, error) {
	podName := os.Getenv("POD_NAME")
	podNamespace := os.Getenv("POD_NAMESPACE")
	if podNamespace == "" {
		podNamespace = "default"
	}
	client, err := apisix.NewClient()
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
	kube.EndpointsInformer = kube.CoreSharedInformerFactory.Core().V1().Endpoints()

	ingressLister := kube.NewIngressLister(kube.CoreSharedInformerFactory.Networking().V1().Ingresses().Lister(),
		kube.CoreSharedInformerFactory.Networking().V1beta1().Ingresses().Lister())

	c := &Controller{
		name:               podName,
		namespace:          podNamespace,
		cfg:                cfg,
		apiServer:          apiSrv,
		apisix:             client,
		metricsCollector:   metrics.NewPrometheusCollector(podName, podNamespace),
		clientset:          kube.GetKubeClient(),
		crdClientset:       crdClientset,
		crdInformerFactory: sharedInformerFactory,
		watchingNamespace:  watchingNamespace,

		epInformer:             kube.CoreSharedInformerFactory.Core().V1().Endpoints().Informer(),
		epLister:               kube.CoreSharedInformerFactory.Core().V1().Endpoints().Lister(),
		svcInformer:            kube.CoreSharedInformerFactory.Core().V1().Services().Informer(),
		svcLister:              kube.CoreSharedInformerFactory.Core().V1().Services().Lister(),
		ingressV1Informer:      kube.CoreSharedInformerFactory.Networking().V1().Ingresses().Informer(),
		ingressV1beta1Informer: kube.CoreSharedInformerFactory.Networking().V1beta1().Ingresses().Informer(),
		ingressLister:          ingressLister,
		apisixUpstreamInformer: sharedInformerFactory.Apisix().V1().ApisixUpstreams().Informer(),
		apisixUpstreamLister:   sharedInformerFactory.Apisix().V1().ApisixUpstreams().Lister(),
	}
	c.translator = kube.NewTranslator(&kube.TranslatorOptions{
		EndpointsLister:      c.epLister,
		ServiceLister:        c.svcLister,
		ApisixUpstreamLister: c.apisixUpstreamLister,
	})

	c.endpointsController = c.newEndpointsController()
	c.apisixUpstreamController = c.newApisixUpstreamController()
	c.ingressController = c.newIngressController()

	return c, nil
}

func (c *Controller) goAttach(handler func()) {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		handler()
	}()
}

// Eventf implements the resourcelock.EventRecorder interface.
func (c *Controller) Eventf(_ runtime.Object, eventType string, reason string, message string, _ ...interface{}) {
	log.Infow(reason, zap.String("message", message), zap.String("event_type", eventType))
}

// Run launches the controller.
func (c *Controller) Run(stop chan struct{}) error {
	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()
	go func() {
		<-stop
		rootCancel()
	}()
	c.metricsCollector.ResetLeader(false)

	go func() {
		if err := c.apiServer.Run(rootCtx.Done()); err != nil {
			log.Errorf("failed to launch API Server: %s", err)
		}
	}()

	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Namespace: c.namespace,
			Name:      c.cfg.Kubernetes.ElectionID,
		},
		Client: c.clientset.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity:      c.name,
			EventRecorder: c,
		},
	}
	cfg := leaderelection.LeaderElectionConfig{
		Lock:          lock,
		LeaseDuration: 15 * time.Second,
		RenewDeadline: 5 * time.Second,
		RetryPeriod:   2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: c.run,
			OnNewLeader: func(identity string) {
				log.Warnf("found a new leader %s", identity)
				if identity != c.name {
					log.Infow("controller now is running as a candidate",
						zap.String("namespace", c.namespace),
						zap.String("pod", c.name),
					)
				}
			},
			OnStoppedLeading: func() {
				log.Infow("controller now is running as a candidate",
					zap.String("namespace", c.namespace),
					zap.String("pod", c.name),
				)
				c.metricsCollector.ResetLeader(false)
			},
		},
		ReleaseOnCancel: true,
		Name:            "ingress-apisix",
	}

	elector, err := leaderelection.NewLeaderElector(cfg)
	if err != nil {
		log.Errorf("failed to create leader elector: %s", err.Error())
		return err
	}

election:
	elector.Run(rootCtx)
	select {
	case <-rootCtx.Done():
		return nil
	default:
		goto election
	}
}

func (c *Controller) run(ctx context.Context) {
	log.Infow("controller now is running as leader",
		zap.String("namespace", c.namespace),
		zap.String("pod", c.name),
	)
	c.metricsCollector.ResetLeader(true)

	err := c.apisix.AddCluster(&apisix.ClusterOptions{
		Name:     "",
		AdminKey: c.cfg.APISIX.AdminKey,
		BaseURL:  c.cfg.APISIX.BaseURL,
	})
	if err != nil {
		// TODO give up the leader role.
		log.Errorf("failed to add default cluster: %s", err)
		return
	}

	if err := c.apisix.Cluster("").HasSynced(ctx); err != nil {
		// TODO give up the leader role.
		log.Errorf("failed to wait the default cluster to be ready: %s", err)
		return
	}

	c.goAttach(func() {
		c.epInformer.Run(ctx.Done())
	})
	c.goAttach(func() {
		c.svcInformer.Run(ctx.Done())
	})
	c.goAttach(func() {
		c.ingressV1Informer.Run(ctx.Done())
	})
	c.goAttach(func() {
		c.ingressV1beta1Informer.Run(ctx.Done())
	})
	c.goAttach(func() {
		c.endpointsController.run(ctx)
	})
	c.goAttach(func() {
		c.apisixUpstreamController.run(ctx)
	})

	ac := &Api6Controller{
		KubeClientSet:             c.clientset,
		Api6ClientSet:             c.crdClientset,
		SharedInformerFactory:     c.crdInformerFactory,
		CoreSharedInformerFactory: kube.CoreSharedInformerFactory,
		Stop:                      ctx.Done(),
	}

	// ApisixRoute
	ac.ApisixRoute(c)
	// ApisixTLS
	ac.ApisixTLS(c)

	c.goAttach(func() {
		ac.SharedInformerFactory.Start(ctx.Done())
	})

	<-ctx.Done()
	c.wg.Wait()
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
	Api6ClientSet             clientset.Interface
	SharedInformerFactory     externalversions.SharedInformerFactory
	CoreSharedInformerFactory informers.SharedInformerFactory
	Stop                      <-chan struct{}
}

func (api6 *Api6Controller) ApisixRoute(controller *Controller) {
	arc := BuildApisixRouteController(
		api6.KubeClientSet,
		api6.Api6ClientSet,
		api6.SharedInformerFactory.Apisix().V1().ApisixRoutes(),
		controller)
	if err := arc.Run(api6.Stop); err != nil {
		log.Errorf("failed to run ApisixRouteController: %s", err)
	}
}

func (api6 *Api6Controller) ApisixTLS(controller *Controller) {
	atc := BuildApisixTlsController(
		api6.KubeClientSet,
		api6.Api6ClientSet,
		api6.SharedInformerFactory.Apisix().V1().ApisixTlses(),
		controller)
	if err := atc.Run(api6.Stop); err != nil {
		log.Errorf("failed to run ApisixTlsController: %s", err)
	}
}
