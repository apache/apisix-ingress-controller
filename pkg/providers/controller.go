// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package providers

import (
	"context"
	"errors"
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
	networkingv1 "k8s.io/client-go/listers/networking/v1"
	networkingv1beta1 "k8s.io/client-go/listers/networking/v1beta1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"

	"github.com/apache/apisix-ingress-controller/pkg/api"
	"github.com/apache/apisix-ingress-controller/pkg/apisix"
	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/kube"
	apisixscheme "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/clientset/versioned/scheme"
	v2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/listers/config/v2"
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
	_minimumApisixResourceSyncInterval = 60 * time.Second
)

// Controller is the ingress apisix controller object.
type Controller struct {
	name             string
	namespace        string
	resourceSyncCh   chan string
	cfg              *config.Config
	apisix           apisix.APISIX
	apiServer        *api.Server
	MetricsCollector metrics.Collector
	kubeClient       *kube.KubeClient
	// recorder event
	recorder record.EventRecorder

	translator       translation.Translator
	apisixTranslator apisixtranslation.ApisixTranslator

	informers *providertypes.ListerInformer

	namespaceProvider namespace.WatchingNamespaceProvider
	podProvider       pod.Provider
	kubeProvider      k8s.Provider
	gatewayProvider   *gateway.Provider
	apisixProvider    apisixprovider.Provider
	ingressProvider   ingressprovider.Provider

	elector *leaderelection.LeaderElector
}

// NewController creates an ingress apisix controller object.
func NewController(cfg *config.Config) (*Controller, error) {
	podName := os.Getenv("POD_NAME")
	if podName == "" {
		podName = os.Getenv("HOSTNAME")
	}
	if podName == "" {
		var err error
		podName, err = os.Hostname()
		if err != nil {
			return nil, err
		}
	}
	podNamespace := os.Getenv("POD_NAMESPACE")
	if podNamespace == "" {
		podNamespace = "default"
	}
	client, err := apisix.NewClient(cfg.APISIX.AdminAPIVersion)
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
		resourceSyncCh:   make(chan string),
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
func (c *Controller) Run(ctx context.Context) error {
	rootCtx, rootCancel := context.WithCancel(ctx)
	defer rootCancel()

	go func() {
		log.Info("start api server")
		// todo: propagate context instead
		if err := c.apiServer.Run(rootCtx.Done()); err != nil {
			log.Errorf("failed to launch API Server: %s", err)
		}
	}()

	// warm up informers
	c.informers = c.initSharedInformers()

	c.MetricsCollector.ResetLeader(false)

	leaderElectionLeaseLock := &resourcelock.LeaseLock{
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

	leaderElectionConfig := leaderelection.LeaderElectionConfig{
		ReleaseOnCancel: true,
		Name:            "ingress-apisix",
		Lock:            leaderElectionLeaseLock,
		LeaseDuration:   15 * time.Second,
		RenewDeadline:   5 * time.Second,
		RetryPeriod:     2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				c.MetricsCollector.ResetLeader(true)
				log.Infow("controller now is running as leader",
					zap.String("namespace", c.namespace),
					zap.String("pod", c.name),
				)
				err := c.run(rootCtx)
				if err != nil {
					log.Errorf("controller run failed: %v", err)
				}
				rootCancel()
			},
			OnNewLeader: func(identity string) {
				log.Warnf("found a new leader %s", identity)
				if identity != leaderElectionLeaseLock.LockConfig.Identity {
					log.Infow("controller now is running as a candidate",
						zap.String("namespace", c.namespace),
						zap.String("pod", c.name),
					)
					c.MetricsCollector.ResetLeader(false)
				}
			},
			OnStoppedLeading: func() {
				c.MetricsCollector.ResetLeader(false)
				log.Infow("controller lost leader, exiting",
					zap.String("namespace", c.namespace),
					zap.String("pod", c.name),
				)
				// rootCancel might be to slow, and controllers may have bugs that cause them to not yield
				// the safest way to step down is to simply cause a pod restart
				os.Exit(0)
			},
		},
	}

	leaderElector, err := leaderelection.NewLeaderElector(leaderElectionConfig)
	if err != nil {
		panic(err)
	}
	if leaderElectionConfig.WatchDog != nil {
		leaderElectionConfig.WatchDog.SetLeaderElection(leaderElector)
	}
	// todo: this should never be necessary if only one controller runs
	c.elector = leaderElector

	leaderElector.Run(rootCtx)

	return nil
}

func (c *Controller) initSharedInformers() *providertypes.ListerInformer {
	kubeFactory := c.kubeClient.NewSharedIndexInformerFactory()
	apisixFactory := c.kubeClient.NewAPISIXSharedIndexInformerFactory()

	var (
		ingressInformer cache.SharedIndexInformer

		ingressListerV1      networkingv1.IngressLister
		ingressListerV1beta1 networkingv1beta1.IngressLister
	)

	var (
		apisixUpstreamInformer      cache.SharedIndexInformer
		apisixRouteInformer         cache.SharedIndexInformer
		apisixPluginConfigInformer  cache.SharedIndexInformer
		apisixConsumerInformer      cache.SharedIndexInformer
		apisixTlsInformer           cache.SharedIndexInformer
		apisixClusterConfigInformer cache.SharedIndexInformer
		ApisixGlobalRuleInformer    cache.SharedIndexInformer

		apisixRouteListerV2         v2.ApisixRouteLister
		apisixUpstreamListerV2      v2.ApisixUpstreamLister
		apisixTlsListerV2           v2.ApisixTlsLister
		apisixClusterConfigListerV2 v2.ApisixClusterConfigLister
		apisixConsumerListerV2      v2.ApisixConsumerLister
		apisixPluginConfigListerV2  v2.ApisixPluginConfigLister
		ApisixGlobalRuleListerV2    v2.ApisixGlobalRuleLister
	)

	switch c.cfg.Kubernetes.APIVersion {
	case config.ApisixV2:
		apisixRouteInformer = apisixFactory.Apisix().V2().ApisixRoutes().Informer()
		apisixTlsInformer = apisixFactory.Apisix().V2().ApisixTlses().Informer()
		apisixClusterConfigInformer = apisixFactory.Apisix().V2().ApisixClusterConfigs().Informer()
		apisixConsumerInformer = apisixFactory.Apisix().V2().ApisixConsumers().Informer()
		apisixPluginConfigInformer = apisixFactory.Apisix().V2().ApisixPluginConfigs().Informer()
		apisixUpstreamInformer = apisixFactory.Apisix().V2().ApisixUpstreams().Informer()
		ApisixGlobalRuleInformer = apisixFactory.Apisix().V2().ApisixGlobalRules().Informer()

		apisixRouteListerV2 = apisixFactory.Apisix().V2().ApisixRoutes().Lister()
		apisixUpstreamListerV2 = apisixFactory.Apisix().V2().ApisixUpstreams().Lister()
		apisixTlsListerV2 = apisixFactory.Apisix().V2().ApisixTlses().Lister()
		apisixClusterConfigListerV2 = apisixFactory.Apisix().V2().ApisixClusterConfigs().Lister()
		apisixConsumerListerV2 = apisixFactory.Apisix().V2().ApisixConsumers().Lister()
		apisixPluginConfigListerV2 = apisixFactory.Apisix().V2().ApisixPluginConfigs().Lister()
		ApisixGlobalRuleListerV2 = apisixFactory.Apisix().V2().ApisixGlobalRules().Lister()

	default:
		panic(fmt.Errorf("unsupported API version %v", c.cfg.Kubernetes.APIVersion))
	}

	apisixUpstreamLister := kube.NewApisixUpstreamLister(apisixUpstreamListerV2)
	apisixRouteLister := kube.NewApisixRouteLister(apisixRouteListerV2)
	apisixTlsLister := kube.NewApisixTlsLister(apisixTlsListerV2)
	apisixClusterConfigLister := kube.NewApisixClusterConfigLister(apisixClusterConfigListerV2)
	apisixConsumerLister := kube.NewApisixConsumerLister(apisixConsumerListerV2)
	apisixPluginConfigLister := kube.NewApisixPluginConfigLister(apisixPluginConfigListerV2)
	ApisixGlobalRuleLister := kube.NewApisixGlobalRuleLister(c.cfg.Kubernetes.APIVersion, ApisixGlobalRuleListerV2)

	epLister, epInformer := kube.NewEndpointListerAndInformer(kubeFactory, c.cfg.Kubernetes.WatchEndpointSlices)
	svcInformer := kubeFactory.Core().V1().Services().Informer()
	svcLister := kubeFactory.Core().V1().Services().Lister()

	podInformer := kubeFactory.Core().V1().Pods().Informer()
	podLister := kubeFactory.Core().V1().Pods().Lister()

	secretInformer := kubeFactory.Core().V1().Secrets().Informer()
	secretLister := kubeFactory.Core().V1().Secrets().Lister()

	configmapInformer := kubeFactory.Core().V1().ConfigMaps().Informer()
	configmapLister := kubeFactory.Core().V1().ConfigMaps().Lister()

	switch c.cfg.Kubernetes.IngressVersion {
	case config.IngressNetworkingV1beta1:
		ingressInformer = kubeFactory.Networking().V1beta1().Ingresses().Informer()
		ingressListerV1beta1 = kubeFactory.Networking().V1beta1().Ingresses().Lister()
	default:
		ingressInformer = kubeFactory.Networking().V1().Ingresses().Informer()
		ingressListerV1 = kubeFactory.Networking().V1().Ingresses().Lister()
	}

	ingressLister := kube.NewIngressLister(ingressListerV1, ingressListerV1beta1)

	listerInformer := &providertypes.ListerInformer{
		ApisixFactory: apisixFactory,
		KubeFactory:   kubeFactory,

		EpLister:          epLister,
		EpInformer:        epInformer,
		SvcLister:         svcLister,
		SvcInformer:       svcInformer,
		SecretLister:      secretLister,
		SecretInformer:    secretInformer,
		PodLister:         podLister,
		PodInformer:       podInformer,
		ConfigMapInformer: configmapInformer,
		ConfigMapLister:   configmapLister,
		IngressInformer:   ingressInformer,
		IngressLister:     ingressLister,

		ApisixUpstreamLister:      apisixUpstreamLister,
		ApisixRouteLister:         apisixRouteLister,
		ApisixConsumerLister:      apisixConsumerLister,
		ApisixTlsLister:           apisixTlsLister,
		ApisixPluginConfigLister:  apisixPluginConfigLister,
		ApisixClusterConfigLister: apisixClusterConfigLister,
		ApisixGlobalRuleLister:    ApisixGlobalRuleLister,

		ApisixUpstreamInformer:      apisixUpstreamInformer,
		ApisixPluginConfigInformer:  apisixPluginConfigInformer,
		ApisixRouteInformer:         apisixRouteInformer,
		ApisixClusterConfigInformer: apisixClusterConfigInformer,
		ApisixConsumerInformer:      apisixConsumerInformer,
		ApisixTlsInformer:           apisixTlsInformer,
		ApisixGlobalRuleInformer:    ApisixGlobalRuleInformer,
	}

	return listerInformer
}

func (c *Controller) run(ctx context.Context) error {
	log.Infow("controller tries to leading ...",
		zap.String("namespace", c.namespace),
		zap.String("pod", c.name),
	)

	var cancelFunc context.CancelFunc
	ctx, cancelFunc = context.WithCancel(ctx)
	defer cancelFunc()

	clusterOpts := &apisix.ClusterOptions{
		AdminAPIVersion:   c.cfg.APISIX.AdminAPIVersion,
		Name:              c.cfg.APISIX.DefaultClusterName,
		AdminKey:          c.cfg.APISIX.DefaultClusterAdminKey,
		BaseURL:           c.cfg.APISIX.DefaultClusterBaseURL,
		MetricsCollector:  c.MetricsCollector,
		SyncComparison:    c.cfg.ApisixResourceSyncComparison,
		EnableEtcdServer:  c.cfg.EtcdServer.Enabled,
		Prefix:            c.cfg.EtcdServer.Prefix,
		ListenAddress:     c.cfg.EtcdServer.ListenAddress,
		SchemaSynced:      !c.cfg.EtcdServer.Enabled,
		CacheSynced:       !c.cfg.EtcdServer.Enabled,
		SSLKeyEncryptSalt: c.cfg.EtcdServer.SSLKeyEncryptSalt,
	}

	// TODO: needs retry logic
	err := c.apisix.AddCluster(ctx, clusterOpts)
	if err != nil && err != apisix.ErrDuplicatedCluster {
		log.Errorf("failed to add default cluster: %s", err)
		return err
	}

	if err := c.apisix.Cluster(c.cfg.APISIX.DefaultClusterName).HasSynced(ctx); err != nil {
		log.Errorf("failed to wait the default cluster to be ready: %s", err)
		return err
	}

	// Creation Phase

	log.Info("creating controller")

	common := &providertypes.Common{
		ControllerNamespace: c.namespace,
		ListerInformer:      c.informers,
		Config:              c.cfg,
		APISIX:              c.apisix,
		KubeClient:          c.kubeClient,
		MetricsCollector:    c.MetricsCollector,
		Recorder:            c.recorder,
		Elector:             c.elector,
	}

	c.namespaceProvider, err = namespace.NewWatchingNamespaceProvider(ctx, c.kubeClient, c.cfg, c.resourceSyncCh)
	if err != nil {
		return err
	}

	c.podProvider, err = pod.NewProvider(common, c.namespaceProvider)
	if err != nil {
		return err
	}

	c.translator = translation.NewTranslator(&translation.TranslatorOptions{
		APIVersion:           c.cfg.Kubernetes.APIVersion,
		EndpointLister:       c.informers.EpLister,
		ServiceLister:        c.informers.SvcLister,
		SecretLister:         c.informers.SecretLister,
		PodLister:            c.informers.PodLister,
		ApisixUpstreamLister: c.informers.ApisixUpstreamLister,
		PodProvider:          c.podProvider,
		IngressClassName:     c.cfg.Kubernetes.IngressClass,
	})

	c.apisixProvider, c.apisixTranslator, err = apisixprovider.NewProvider(common, c.namespaceProvider, c.translator)
	if err != nil {
		return err
	}

	c.ingressProvider, err = ingressprovider.NewProvider(common, c.namespaceProvider, c.translator, c.apisixTranslator)
	if err != nil {
		return err
	}

	c.kubeProvider, err = k8s.NewProvider(common, c.translator, c.namespaceProvider, c.apisixProvider, c.ingressProvider)
	if err != nil {
		return err
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
			ListerInformer:    common.ListerInformer,
		})
		if err != nil {
			return err
		}
	}

	// Init Phase

	log.Info("init namespaces")

	if err = c.namespaceProvider.Init(ctx); err != nil {
		return err
	}

	log.Info("wait for resource sync")

	// Wait for resource sync
	if ok := c.informers.StartAndWaitForCacheSync(ctx); !ok {
		return errors.New("StartAndWaitForCacheSync failed")
	}

	log.Info("init providers")

	// Compare resource
	if !c.cfg.EtcdServer.Enabled {
		if err = c.apisixProvider.Init(ctx); err != nil {
			return err
		}
	}

	// Run Phase

	log.Info("try to run providers")

	e := utils.ParallelExecutor{}

	e.Add(func() {
		c.checkClusterHealth(ctx, cancelFunc)
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

	<-ctx.Done()
	e.Wait()

	for _, execErr := range e.Errors() {
		log.Error(execErr.Error())
	}
	if len(e.Errors()) > 0 {
		log.Error("Start failed, abort...")
		cancelFunc()
	}

	return nil
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
			c.apiServer.HealthState.Lock()
			c.apiServer.HealthState.Err = err
			c.apiServer.HealthState.Unlock()
			// Finally failed health check, then give up leader.
			log.Warnf("failed to check health for default cluster: %s, give up leader", err)
		} else {
			if c.apiServer.HealthState.Err != nil {
				c.apiServer.HealthState.Lock()
				c.apiServer.HealthState.Err = err
				c.apiServer.HealthState.Unlock()
			}
			log.Debugf("success check health for default cluster")
			c.MetricsCollector.IncrCheckClusterHealth(c.name)
		}
	}
}

func (c *Controller) syncResources(interval time.Duration, namespace string) {
	e := utils.ParallelExecutor{}

	e.Add(func() {
		c.ingressProvider.ResourceSync(namespace)
	})

	e.Add(func() {
		c.apisixProvider.ResourceSync(interval, namespace)
	})

	e.Wait()
}

func (c *Controller) resourceSyncLoop(ctx context.Context, interval time.Duration) {
	if interval == 0 {
		log.Info("apisix-resource-sync-interval set to 0, periodically synchronization disabled.")
		return
	}
	// The interval shall not be less than 60 seconds.
	if interval < _minimumApisixResourceSyncInterval {
		log.Warnw("The apisix-resource-sync-interval shall not be less than 60 seconds.",
			zap.String("apisix-resource-sync-interval", interval.String()),
		)
		interval = _minimumApisixResourceSyncInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case namespace := <-c.resourceSyncCh:
			c.syncResources(0, namespace)
		case <-ticker.C:
			c.syncResources(interval, "")
		case <-ctx.Done():
			return
		}
	}
}
