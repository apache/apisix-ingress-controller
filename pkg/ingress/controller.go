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
package ingress

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"

	"github.com/apache/apisix-ingress-controller/pkg/api"
	"github.com/apache/apisix-ingress-controller/pkg/apisix"
	apisixcache "github.com/apache/apisix-ingress-controller/pkg/apisix/cache"
	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/ingress/gateway"
	"github.com/apache/apisix-ingress-controller/pkg/ingress/namespace"
	"github.com/apache/apisix-ingress-controller/pkg/ingress/utils"
	"github.com/apache/apisix-ingress-controller/pkg/kube"
	configv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta3"
	apisixscheme "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/clientset/versioned/scheme"
	listersv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/listers/config/v2beta3"
	"github.com/apache/apisix-ingress-controller/pkg/kube/translation"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/metrics"
	"github.com/apache/apisix-ingress-controller/pkg/types"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

const (
	// _component is used for event component
	_component = "ApisixIngress"
	// _resourceSynced is used when a resource is synced successfully
	_resourceSynced = "ResourcesSynced"
	// _messageResourceSynced is used to specify controller
	_messageResourceSynced = "%s synced successfully"
	// _resourceSyncAborted is used when a resource synced failed
	_resourceSyncAborted = "ResourceSyncAborted"
	// _messageResourceFailed is used to report error
	_messageResourceFailed = "%s synced failed, with error: %s"
)

// Controller is the ingress apisix controller object.
type Controller struct {
	name             string
	namespace        string
	cfg              *config.Config
	apisix           apisix.APISIX
	podCache         types.PodCache
	translator       translation.Translator
	apiServer        *api.Server
	MetricsCollector metrics.Collector
	kubeClient       *kube.KubeClient
	// recorder event
	recorder record.EventRecorder
	// this map enrolls which ApisixTls objects refer to a Kubernetes
	// Secret object.
	// type: Map<SecretKey, Map<ApisixTlsKey, ApisixTls>>
	// SecretKey is `namespace_name`, ApisixTlsKey is kube style meta key: `namespace/name`
	secretSSLMap *sync.Map

	// leaderContextCancelFunc will be called when apisix-ingress-controller
	// decides to give up its leader role.
	leaderContextCancelFunc context.CancelFunc

	// common informers and listers
	podInformer                 cache.SharedIndexInformer
	podLister                   listerscorev1.PodLister
	epInformer                  cache.SharedIndexInformer
	epLister                    kube.EndpointLister
	svcInformer                 cache.SharedIndexInformer
	svcLister                   listerscorev1.ServiceLister
	ingressLister               kube.IngressLister
	ingressInformer             cache.SharedIndexInformer
	secretInformer              cache.SharedIndexInformer
	secretLister                listerscorev1.SecretLister
	apisixUpstreamInformer      cache.SharedIndexInformer
	apisixUpstreamLister        listersv2beta3.ApisixUpstreamLister
	apisixRouteLister           kube.ApisixRouteLister
	apisixRouteInformer         cache.SharedIndexInformer
	apisixTlsLister             kube.ApisixTlsLister
	apisixTlsInformer           cache.SharedIndexInformer
	apisixClusterConfigLister   kube.ApisixClusterConfigLister
	apisixClusterConfigInformer cache.SharedIndexInformer
	apisixConsumerInformer      cache.SharedIndexInformer
	apisixConsumerLister        kube.ApisixConsumerLister
	apisixPluginConfigInformer  cache.SharedIndexInformer
	apisixPluginConfigLister    kube.ApisixPluginConfigLister

	// resource controllers
	podController           *podController
	endpointsController     *endpointsController
	endpointSliceController *endpointSliceController
	ingressController       *ingressController
	secretController        *secretController

	namespaceProvider namespace.WatchingProvider
	gatewayProvider   *gateway.GatewayProvider

	apisixUpstreamController      *apisixUpstreamController
	apisixRouteController         *apisixRouteController
	apisixTlsController           *apisixTlsController
	apisixClusterConfigController *apisixClusterConfigController
	apisixConsumerController      *apisixConsumerController
	apisixPluginConfigController  *apisixPluginConfigController
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
		secretSSLMap:     new(sync.Map),
		recorder:         eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: _component}),

		podCache: types.NewPodCache(),
	}
	return c, nil
}

func (c *Controller) initWhenStartLeading() {
	var (
		ingressInformer             cache.SharedIndexInformer
		apisixRouteInformer         cache.SharedIndexInformer
		apisixPluginConfigInformer  cache.SharedIndexInformer
		apisixTlsInformer           cache.SharedIndexInformer
		apisixClusterConfigInformer cache.SharedIndexInformer
		apisixConsumerInformer      cache.SharedIndexInformer
	)

	kubeFactory := c.kubeClient.NewSharedIndexInformerFactory()
	apisixFactory := c.kubeClient.NewAPISIXSharedIndexInformerFactory()

	c.podLister = kubeFactory.Core().V1().Pods().Lister()
	c.epLister, c.epInformer = kube.NewEndpointListerAndInformer(kubeFactory, c.cfg.Kubernetes.WatchEndpointSlices)
	c.svcLister = kubeFactory.Core().V1().Services().Lister()
	c.ingressLister = kube.NewIngressLister(
		kubeFactory.Networking().V1().Ingresses().Lister(),
		kubeFactory.Networking().V1beta1().Ingresses().Lister(),
		kubeFactory.Extensions().V1beta1().Ingresses().Lister(),
	)
	c.secretLister = kubeFactory.Core().V1().Secrets().Lister()
	c.apisixRouteLister = kube.NewApisixRouteLister(
		apisixFactory.Apisix().V2beta2().ApisixRoutes().Lister(),
		apisixFactory.Apisix().V2beta3().ApisixRoutes().Lister(),
		apisixFactory.Apisix().V2().ApisixRoutes().Lister(),
	)
	c.apisixUpstreamLister = apisixFactory.Apisix().V2beta3().ApisixUpstreams().Lister()
	c.apisixTlsLister = kube.NewApisixTlsLister(
		apisixFactory.Apisix().V2beta3().ApisixTlses().Lister(),
		apisixFactory.Apisix().V2().ApisixTlses().Lister(),
	)
	c.apisixClusterConfigLister = kube.NewApisixClusterConfigLister(
		apisixFactory.Apisix().V2beta3().ApisixClusterConfigs().Lister(),
		apisixFactory.Apisix().V2().ApisixClusterConfigs().Lister(),
	)
	c.apisixConsumerLister = kube.NewApisixConsumerLister(
		apisixFactory.Apisix().V2beta3().ApisixConsumers().Lister(),
		apisixFactory.Apisix().V2().ApisixConsumers().Lister(),
	)
	c.apisixPluginConfigLister = kube.NewApisixPluginConfigLister(
		apisixFactory.Apisix().V2beta3().ApisixPluginConfigs().Lister(),
		apisixFactory.Apisix().V2().ApisixPluginConfigs().Lister(),
	)

	c.translator = translation.NewTranslator(&translation.TranslatorOptions{
		PodCache:             c.podCache,
		PodLister:            c.podLister,
		EndpointLister:       c.epLister,
		ServiceLister:        c.svcLister,
		ApisixUpstreamLister: c.apisixUpstreamLister,
		SecretLister:         c.secretLister,
		UseEndpointSlices:    c.cfg.Kubernetes.WatchEndpointSlices,
	})

	if c.cfg.Kubernetes.IngressVersion == config.IngressNetworkingV1 {
		ingressInformer = kubeFactory.Networking().V1().Ingresses().Informer()
	} else if c.cfg.Kubernetes.IngressVersion == config.IngressNetworkingV1beta1 {
		ingressInformer = kubeFactory.Networking().V1beta1().Ingresses().Informer()
	} else {
		ingressInformer = kubeFactory.Extensions().V1beta1().Ingresses().Informer()
	}

	switch c.cfg.Kubernetes.ApisixRouteVersion {
	case config.ApisixRouteV2beta2:
		apisixRouteInformer = apisixFactory.Apisix().V2beta2().ApisixRoutes().Informer()
	case config.ApisixRouteV2beta3:
		apisixRouteInformer = apisixFactory.Apisix().V2beta3().ApisixRoutes().Informer()
	case config.ApisixRouteV2:
		apisixRouteInformer = apisixFactory.Apisix().V2().ApisixRoutes().Informer()
	default:
		panic(fmt.Errorf("unsupported ApisixRoute version %s", c.cfg.Kubernetes.ApisixRouteVersion))
	}

	switch c.cfg.Kubernetes.ApisixTlsVersion {
	case config.ApisixV2beta3:
		apisixTlsInformer = apisixFactory.Apisix().V2beta3().ApisixTlses().Informer()
	case config.ApisixV2:
		apisixTlsInformer = apisixFactory.Apisix().V2().ApisixTlses().Informer()
	default:
		panic(fmt.Errorf("unsupported ApisixTls version %s", c.cfg.Kubernetes.ApisixTlsVersion))
	}

	switch c.cfg.Kubernetes.ApisixClusterConfigVersion {
	case config.ApisixV2beta3:
		apisixClusterConfigInformer = apisixFactory.Apisix().V2beta3().ApisixClusterConfigs().Informer()
	case config.ApisixV2:
		apisixClusterConfigInformer = apisixFactory.Apisix().V2().ApisixClusterConfigs().Informer()
	default:
		panic(fmt.Errorf("unsupported ApisixClusterConfig version %v", c.cfg.Kubernetes.ApisixClusterConfigVersion))
	}

	switch c.cfg.Kubernetes.ApisixConsumerVersion {
	case config.ApisixRouteV2beta3:
		apisixConsumerInformer = apisixFactory.Apisix().V2beta3().ApisixConsumers().Informer()
	case config.ApisixRouteV2:
		apisixConsumerInformer = apisixFactory.Apisix().V2().ApisixConsumers().Informer()
	default:
		panic(fmt.Errorf("unsupported ApisixConsumer version %v", c.cfg.Kubernetes.ApisixConsumerVersion))
	}

	switch c.cfg.Kubernetes.ApisixPluginConfigVersion {
	case config.ApisixV2beta3:
		apisixPluginConfigInformer = apisixFactory.Apisix().V2beta3().ApisixPluginConfigs().Informer()
	case config.ApisixV2:
		apisixPluginConfigInformer = apisixFactory.Apisix().V2().ApisixPluginConfigs().Informer()
	default:
		panic(fmt.Errorf("unsupported ApisixPluginConfig version %v", c.cfg.Kubernetes.ApisixPluginConfigVersion))
	}

	c.podInformer = kubeFactory.Core().V1().Pods().Informer()
	c.svcInformer = kubeFactory.Core().V1().Services().Informer()
	c.ingressInformer = ingressInformer
	c.apisixRouteInformer = apisixRouteInformer
	c.apisixUpstreamInformer = apisixFactory.Apisix().V2beta3().ApisixUpstreams().Informer()
	c.apisixClusterConfigInformer = apisixClusterConfigInformer
	c.secretInformer = kubeFactory.Core().V1().Secrets().Informer()
	c.apisixTlsInformer = apisixTlsInformer
	c.apisixConsumerInformer = apisixConsumerInformer
	c.apisixPluginConfigInformer = apisixPluginConfigInformer

	if c.cfg.Kubernetes.WatchEndpointSlices {
		c.endpointSliceController = c.newEndpointSliceController()
	} else {
		c.endpointsController = c.newEndpointsController()
	}
	c.podController = c.newPodController()
	c.apisixUpstreamController = c.newApisixUpstreamController()
	c.ingressController = c.newIngressController()
	c.apisixRouteController = c.newApisixRouteController()
	c.apisixClusterConfigController = c.newApisixClusterConfigController()
	c.apisixTlsController = c.newApisixTlsController()
	c.secretController = c.newSecretController()
	c.apisixConsumerController = c.newApisixConsumerController()
	c.apisixPluginConfigController = c.newApisixPluginConfigController()
}

func (c *Controller) syncManifests(ctx context.Context, added, updated, deleted *utils.Manifest) error {
	return utils.SyncManifests(ctx, c.apisix, c.cfg.APISIX.DefaultClusterName, added, updated, deleted)
}

// recorderEvent recorder events for resources
func (c *Controller) recorderEvent(object runtime.Object, eventtype, reason string, err error) {
	if err != nil {
		message := fmt.Sprintf(_messageResourceFailed, _component, err.Error())
		c.recorder.Event(object, eventtype, reason, message)
	} else {
		message := fmt.Sprintf(_messageResourceSynced, _component)
		c.recorder.Event(object, eventtype, reason, message)
	}
}

// recorderEvent recorder events for resources
func (c *Controller) recorderEventS(object runtime.Object, eventtype, reason string, msg string) {
	c.recorder.Event(object, eventtype, reason, msg)
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

	c.initWhenStartLeading()

	c.namespaceProvider, err = namespace.NewWatchingProvider(ctx, c.kubeClient, c.cfg)
	if err != nil {
		ctx.Done()
		return
	}

	c.gatewayProvider, err = gateway.NewGatewayProvider(&gateway.GatewayProviderOptions{
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

	// compare resources of k8s with objects of APISIX
	if err = c.CompareResources(ctx); err != nil {
		ctx.Done()
		return
	}

	e := utils.ParallelExecutor{}

	e.Add(func() {
		c.checkClusterHealth(ctx, cancelFunc)
	})
	e.Add(func() {
		c.podInformer.Run(ctx.Done())
	})
	e.Add(func() {
		c.epInformer.Run(ctx.Done())
	})
	e.Add(func() {
		c.svcInformer.Run(ctx.Done())
	})
	e.Add(func() {
		c.ingressInformer.Run(ctx.Done())
	})
	e.Add(func() {
		c.apisixRouteInformer.Run(ctx.Done())
	})
	e.Add(func() {
		c.apisixUpstreamInformer.Run(ctx.Done())
	})
	e.Add(func() {
		c.apisixClusterConfigInformer.Run(ctx.Done())
	})
	e.Add(func() {
		c.secretInformer.Run(ctx.Done())
	})
	e.Add(func() {
		c.apisixTlsInformer.Run(ctx.Done())
	})
	e.Add(func() {
		c.apisixConsumerInformer.Run(ctx.Done())
	})
	e.Add(func() {
		c.apisixPluginConfigInformer.Run(ctx.Done())
	})
	e.Add(func() {
		c.podController.run(ctx)
	})
	e.Add(func() {
		if c.cfg.Kubernetes.WatchEndpointSlices {
			c.endpointSliceController.run(ctx)
		} else {
			c.endpointsController.run(ctx)
		}
	})

	e.Add(func() {
		c.namespaceProvider.Run(ctx)
	})

	if c.cfg.Kubernetes.EnableGatewayAPI {
		e.Add(func() {
			c.gatewayProvider.Run(ctx)
		})
	}

	e.Add(func() {
		c.apisixUpstreamController.run(ctx)
	})
	e.Add(func() {
		c.ingressController.run(ctx)
	})
	e.Add(func() {
		c.apisixRouteController.run(ctx)
	})
	e.Add(func() {
		c.apisixClusterConfigController.run(ctx)
	})
	e.Add(func() {
		c.apisixTlsController.run(ctx)
	})
	e.Add(func() {
		c.secretController.run(ctx)
	})
	e.Add(func() {
		c.apisixConsumerController.run(ctx)
	})
	e.Add(func() {
		c.apisixPluginConfigController.run(ctx)
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

// isWatchingNamespace accepts a resource key, getting the namespace part
// and checking whether the namespace is being watched.
func (c *Controller) isWatchingNamespace(key string) (ok bool) {
	return c.namespaceProvider.IsWatchingNamespace(key)
}

func (c *Controller) syncSSL(ctx context.Context, ssl *apisixv1.Ssl, event types.EventType) error {
	var (
		err error
	)
	clusterName := c.cfg.APISIX.DefaultClusterName
	if event == types.EventDelete {
		err = c.apisix.Cluster(clusterName).SSL().Delete(ctx, ssl)
	} else if event == types.EventUpdate {
		_, err = c.apisix.Cluster(clusterName).SSL().Update(ctx, ssl)
	} else {
		_, err = c.apisix.Cluster(clusterName).SSL().Create(ctx, ssl)
	}
	return err
}

func (c *Controller) syncConsumer(ctx context.Context, consumer *apisixv1.Consumer, event types.EventType) (err error) {
	clusterName := c.cfg.APISIX.DefaultClusterName
	if event == types.EventDelete {
		err = c.apisix.Cluster(clusterName).Consumer().Delete(ctx, consumer)
	} else if event == types.EventUpdate {
		_, err = c.apisix.Cluster(clusterName).Consumer().Update(ctx, consumer)
	} else {
		_, err = c.apisix.Cluster(clusterName).Consumer().Create(ctx, consumer)
	}
	return
}

func (c *Controller) syncEndpoint(ctx context.Context, ep kube.Endpoint) error {
	namespace, err := ep.Namespace()
	if err != nil {
		return err
	}
	svcName := ep.ServiceName()
	svc, err := c.svcLister.Services(namespace).Get(svcName)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Infof("service %s/%s not found", namespace, svcName)
			return nil
		}
		log.Errorf("failed to get service %s/%s: %s", namespace, svcName, err)
		return err
	}
	var subsets []configv2beta3.ApisixUpstreamSubset
	subsets = append(subsets, configv2beta3.ApisixUpstreamSubset{})
	au, err := c.apisixUpstreamLister.ApisixUpstreams(namespace).Get(svcName)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			log.Errorf("failed to get ApisixUpstream %s/%s: %s", namespace, svcName, err)
			return err
		}
	} else if au.Spec != nil && len(au.Spec.Subsets) > 0 {
		subsets = append(subsets, au.Spec.Subsets...)
	}

	clusters := c.apisix.ListClusters()
	for _, port := range svc.Spec.Ports {
		for _, subset := range subsets {
			nodes, err := c.translator.TranslateUpstreamNodes(ep, port.Port, subset.Labels)
			if err != nil {
				log.Errorw("failed to translate upstream nodes",
					zap.Error(err),
					zap.Any("endpoints", ep),
					zap.Int32("port", port.Port),
				)
			}
			name := apisixv1.ComposeUpstreamName(namespace, svcName, subset.Name, port.Port)
			for _, cluster := range clusters {
				if err := c.syncUpstreamNodesChangeToCluster(ctx, cluster, nodes, name); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (c *Controller) syncUpstreamNodesChangeToCluster(ctx context.Context, cluster apisix.Cluster, nodes apisixv1.UpstreamNodes, upsName string) error {
	upstream, err := cluster.Upstream().Get(ctx, upsName)
	if err != nil {
		if err == apisixcache.ErrNotFound {
			log.Warnw("upstream is not referenced",
				zap.String("cluster", cluster.String()),
				zap.String("upstream", upsName),
			)
			return nil
		} else {
			log.Errorw("failed to get upstream",
				zap.String("upstream", upsName),
				zap.String("cluster", cluster.String()),
				zap.Error(err),
			)
			return err
		}
	}

	upstream.Nodes = nodes

	log.Debugw("upstream binds new nodes",
		zap.Any("upstream", upstream),
		zap.String("cluster", cluster.String()),
	)

	updated := &utils.Manifest{
		Upstreams: []*apisixv1.Upstream{upstream},
	}
	return c.syncManifests(ctx, nil, updated, nil)
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
