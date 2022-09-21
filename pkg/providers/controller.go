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
package providers

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"

	"github.com/apache/apisix-ingress-controller/pkg/api"
	"github.com/apache/apisix-ingress-controller/pkg/apisix"
	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/kube"
	apisixscheme "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/clientset/versioned/scheme"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/metrics"
	apisixprovider "github.com/apache/apisix-ingress-controller/pkg/providers/apisix"
	apisixtranslation "github.com/apache/apisix-ingress-controller/pkg/providers/apisix/translation"
	"github.com/apache/apisix-ingress-controller/pkg/providers/gateway"
	ingressprovider "github.com/apache/apisix-ingress-controller/pkg/providers/ingress"
	"github.com/apache/apisix-ingress-controller/pkg/providers/k8s"
	"github.com/apache/apisix-ingress-controller/pkg/providers/k8s/namespace"
	"github.com/apache/apisix-ingress-controller/pkg/providers/k8s/pod"
	"github.com/apache/apisix-ingress-controller/pkg/providers/translation"
	providertypes "github.com/apache/apisix-ingress-controller/pkg/providers/types"
	"github.com/apache/apisix-ingress-controller/pkg/providers/utils"
)

const (
	// _component is used for event component
	_component = "ApisixIngress"
	// minimum interval for ingress sync to APISIX
	_mininumApisixResourceSyncInterval = 60 * time.Second
)

// Controller is the ingress apisix controller object.
type Controller struct {
	name             string
	namespace        string
	cfg              *config.Config
	apisix           apisix.APISIX
	apiServer        *api.Server
	MetricsCollector metrics.Collector
	kubeClient       *kube.KubeClient
	// recorder event
	recorder record.EventRecorder

	// leaderContextCancelFunc will be called when apisix-ingress-controller
	// decides to give up its leader role.
	leaderContextCancelFunc context.CancelFunc

	translator       translation.Translator
	apisixTranslator apisixtranslation.ApisixTranslator

	informers *providertypes.ListerInformer

	namespaceProvider namespace.WatchingNamespaceProvider
	podProvider       pod.Provider
	kubeProvider      k8s.Provider
	gatewayProvider   *gateway.Provider
	apisixProvider    apisixprovider.Provider
	ingressProvider   ingressprovider.Provider
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

	kubeClient, err := kube.NewKubeClient(cfg)
	if err != nil {
		return nil, err
	}

	apiSrv, err := api.NewServer(cfg)
	if err != nil {
		return nil, err
	}

	// recorder
	utilruntime.Must(apisixscheme.AddToScheme(scheme.Scheme))
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.Client.CoreV1().Events("")})

	c := &Controller{
		name:             podName,
		namespace:        podNamespace,
		cfg:              cfg,
		apiServer:        apiSrv,
		apisix:           client,
		MetricsCollector: metrics.NewPrometheusCollector(),
		kubeClient:       kubeClient,
		recorder:         eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: _component}),
	}
	return c, nil
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
	c.MetricsCollector.ResetLeader(false)

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
		Client: c.kubeClient.Client.CoordinationV1(),
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
					c.MetricsCollector.ResetLeader(false)
					// delete the old APISIX cluster, so that the cached state
					// like synchronization won't be used next time the candidate
					// becomes the leader again.
					c.apisix.DeleteCluster(c.cfg.APISIX.DefaultClusterName)
				}
			},
			OnStoppedLeading: func() {
				log.Infow("controller now is running as a candidate",
					zap.String("namespace", c.namespace),
					zap.String("pod", c.name),
				)
				c.MetricsCollector.ResetLeader(false)
				// delete the old APISIX cluster, so that the cached state
				// like synchronization won't be used next time the candidate
				// becomes the leader again.
				c.apisix.DeleteCluster(c.cfg.APISIX.DefaultClusterName)
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
	curCtx, cancel := context.WithCancel(rootCtx)
	c.leaderContextCancelFunc = cancel
	elector.Run(curCtx)
	select {
	case <-rootCtx.Done():
		return nil
	default:
		goto election
	}
}

func (c *Controller) initSharedInformers() *providertypes.ListerInformer {
	kubeFactory := c.kubeClient.NewSharedIndexInformerFactory()
	apisixFactory := c.kubeClient.NewAPISIXSharedIndexInformerFactory()

	epLister, epInformer := kube.NewEndpointListerAndInformer(kubeFactory, c.cfg.Kubernetes.WatchEndpointSlices)
	svcInformer := kubeFactory.Core().V1().Services().Informer()
	svcLister := kubeFactory.Core().V1().Services().Lister()

	var (
		apisixUpstreamInformer cache.SharedIndexInformer
	)
	switch c.cfg.Kubernetes.APIVersion {
	case config.ApisixV2beta3:
		apisixUpstreamInformer = apisixFactory.Apisix().V2beta3().ApisixUpstreams().Informer()
	case config.ApisixV2:
		apisixUpstreamInformer = apisixFactory.Apisix().V2().ApisixUpstreams().Informer()
	default:
		panic(fmt.Errorf("unsupported API version %v", c.cfg.Kubernetes.APIVersion))
	}

	apisixUpstreamLister := kube.NewApisixUpstreamLister(
		apisixFactory.Apisix().V2beta3().ApisixUpstreams().Lister(),
		apisixFactory.Apisix().V2().ApisixUpstreams().Lister(),
	)

	podInformer := kubeFactory.Core().V1().Pods().Informer()
	podLister := kubeFactory.Core().V1().Pods().Lister()

	secretInformer := kubeFactory.Core().V1().Secrets().Informer()
	secretLister := kubeFactory.Core().V1().Secrets().Lister()

	listerInformer := &providertypes.ListerInformer{
		EpLister:               epLister,
		EpInformer:             epInformer,
		SvcLister:              svcLister,
		SvcInformer:            svcInformer,
		SecretLister:           secretLister,
		SecretInformer:         secretInformer,
		PodLister:              podLister,
		PodInformer:            podInformer,
		ApisixUpstreamLister:   apisixUpstreamLister,
		ApisixUpstreamInformer: apisixUpstreamInformer,
	}

	return listerInformer
}

func (c *Controller) run(ctx context.Context) {
	log.Infow("controller tries to leading ...",
		zap.String("namespace", c.namespace),
		zap.String("pod", c.name),
	)

	var cancelFunc context.CancelFunc
	ctx, cancelFunc = context.WithCancel(ctx)
	defer cancelFunc()

	// give up leader
	defer c.leaderContextCancelFunc()

	clusterOpts := &apisix.ClusterOptions{
		Name:             c.cfg.APISIX.DefaultClusterName,
		AdminKey:         c.cfg.APISIX.DefaultClusterAdminKey,
		BaseURL:          c.cfg.APISIX.DefaultClusterBaseURL,
		MetricsCollector: c.MetricsCollector,
	}
	err := c.apisix.AddCluster(ctx, clusterOpts)
	if err != nil && err != apisix.ErrDuplicatedCluster {
		// TODO give up the leader role
		log.Errorf("failed to add default cluster: %s", err)
		return
	}

	if err := c.apisix.Cluster(c.cfg.APISIX.DefaultClusterName).HasSynced(ctx); err != nil {
		// TODO give up the leader role
		log.Errorf("failed to wait the default cluster to be ready: %s", err)

		// re-create apisix cluster, used in next c.run
		if err = c.apisix.UpdateCluster(ctx, clusterOpts); err != nil {
			log.Errorf("failed to update default cluster: %s", err)
			return
		}
		return
	}

	// Creation Phase

	c.informers = c.initSharedInformers()
	common := &providertypes.Common{
		ListerInformer:   c.informers,
		Config:           c.cfg,
		APISIX:           c.apisix,
		KubeClient:       c.kubeClient,
		MetricsCollector: c.MetricsCollector,
		Recorder:         c.recorder,
	}

	c.namespaceProvider, err = namespace.NewWatchingNamespaceProvider(ctx, c.kubeClient, c.cfg)
	if err != nil {
		ctx.Done()
		return
	}

	c.podProvider, err = pod.NewProvider(common, c.namespaceProvider)
	if err != nil {
		ctx.Done()
		return
	}

	c.translator = translation.NewTranslator(&translation.TranslatorOptions{
		APIVersion:           c.cfg.Kubernetes.APIVersion,
		EndpointLister:       c.informers.EpLister,
		ServiceLister:        c.informers.SvcLister,
		SecretLister:         c.informers.SecretLister,
		PodLister:            c.informers.PodLister,
		ApisixUpstreamLister: c.informers.ApisixUpstreamLister,
		PodProvider:          c.podProvider,
	})

	c.apisixProvider, c.apisixTranslator, err = apisixprovider.NewProvider(common, c.namespaceProvider, c.translator)
	if err != nil {
		ctx.Done()
		return
	}

	c.ingressProvider, err = ingressprovider.NewProvider(common, c.namespaceProvider, c.translator, c.apisixTranslator)
	if err != nil {
		ctx.Done()
		return
	}

	c.kubeProvider, err = k8s.NewProvider(common, c.translator, c.namespaceProvider, c.apisixProvider, c.ingressProvider)
	if err != nil {
		ctx.Done()
		return
	}

	if c.cfg.Kubernetes.EnableGatewayAPI {
		c.gatewayProvider, err = gateway.NewGatewayProvider(&gateway.ProviderOptions{
			Cfg:               c.cfg,
			APISIX:            c.apisix,
			APISIXClusterName: c.cfg.APISIX.DefaultClusterName,
			KubeTranslator:    c.translator,
			RestConfig:        nil,
			KubeClient:        c.kubeClient.Client,
			MetricsCollector:  c.MetricsCollector,
			NamespaceProvider: c.namespaceProvider,
		})
		if err != nil {
			ctx.Done()
			return
		}
	}

	// Init Phase

	if err = c.namespaceProvider.Init(ctx); err != nil {
		ctx.Done()
		return
	}
	if err = c.apisixProvider.Init(ctx); err != nil {
		ctx.Done()
		return
	}

	// Run Phase

	e := utils.ParallelExecutor{}

	e.Add(func() {
		c.checkClusterHealth(ctx, cancelFunc)
	})

	e.Add(func() {
		c.informers.Run(ctx)
	})

	e.Add(func() {
		c.namespaceProvider.Run(ctx)
	})

	e.Add(func() {
		c.kubeProvider.Run(ctx)
	})

	e.Add(func() {
		c.apisixProvider.Run(ctx)
	})

	e.Add(func() {
		c.ingressProvider.Run(ctx)
	})

	if c.cfg.Kubernetes.EnableGatewayAPI {
		e.Add(func() {
			c.gatewayProvider.Run(ctx)
		})
	}

	e.Add(func() {
		c.resourceSyncLoop(ctx, c.cfg.ApisixResourceSyncInterval.Duration)
	})
	c.MetricsCollector.ResetLeader(true)

	log.Infow("controller now is running as leader",
		zap.String("namespace", c.namespace),
		zap.String("pod", c.name),
	)

	<-ctx.Done()
	e.Wait()

	for _, execErr := range e.Errors() {
		log.Error(execErr.Error())
	}
	if len(e.Errors()) > 0 {
		log.Error("Start failed, abort...")
		cancelFunc()
	}
}

func (c *Controller) checkClusterHealth(ctx context.Context, cancelFunc context.CancelFunc) {
	defer cancelFunc()
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}

		err := c.apisix.Cluster(c.cfg.APISIX.DefaultClusterName).HealthCheck(ctx)
		if err != nil {
			// Finally failed health check, then give up leader.
			log.Warnf("failed to check health for default cluster: %s, give up leader", err)
			c.apiServer.HealthState.Lock()
			defer c.apiServer.HealthState.Unlock()

			c.apiServer.HealthState.Err = err
			return
		}
		log.Debugf("success check health for default cluster")
		c.MetricsCollector.IncrCheckClusterHealth(c.name)
	}
}

func (c *Controller) syncAllResources() {
	e := utils.ParallelExecutor{}

	e.Add(c.ingressProvider.ResourceSync)
	e.Add(c.apisixProvider.ResourceSync)

	e.Wait()
}

func (c *Controller) resourceSyncLoop(ctx context.Context, interval time.Duration) {
	// The interval shall not be less than 60 seconds.
	if interval < _mininumApisixResourceSyncInterval {
		log.Warnw("The apisix-resource-sync-interval shall not be less than 60 seconds.",
			zap.String("apisix-resource-sync-interval", interval.String()),
		)
		interval = _mininumApisixResourceSyncInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.syncAllResources()
			continue
		case <-ctx.Done():
			return
		}
	}
}
