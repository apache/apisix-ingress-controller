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
	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/kube"
	apisixscheme "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/clientset/versioned/scheme"
	listersv1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/listers/config/v1"
	listersv2alpha1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/listers/config/v2alpha1"
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
	name              string
	namespace         string
	cfg               *config.Config
	wg                sync.WaitGroup
	watchingNamespace map[string]struct{}
	apisix            apisix.APISIX
	translator        translation.Translator
	apiServer         *api.Server
	metricsCollector  metrics.Collector
	kubeClient        *kube.KubeClient
	// recorder event
	recorder record.EventRecorder
	// this map enrolls which ApisixTls objects refer to a Kubernetes
	// Secret object.
	secretSSLMap *sync.Map

	// leaderContextCancelFunc will be called when apisix-ingress-controller
	// decides to give up its leader role.
	leaderContextCancelFunc context.CancelFunc

	// common informers and listers
	epInformer                  cache.SharedIndexInformer
	epLister                    listerscorev1.EndpointsLister
	svcInformer                 cache.SharedIndexInformer
	svcLister                   listerscorev1.ServiceLister
	ingressLister               kube.IngressLister
	ingressInformer             cache.SharedIndexInformer
	secretInformer              cache.SharedIndexInformer
	secretLister                listerscorev1.SecretLister
	apisixUpstreamInformer      cache.SharedIndexInformer
	apisixUpstreamLister        listersv1.ApisixUpstreamLister
	apisixRouteLister           kube.ApisixRouteLister
	apisixRouteInformer         cache.SharedIndexInformer
	apisixTlsLister             listersv1.ApisixTlsLister
	apisixTlsInformer           cache.SharedIndexInformer
	apisixClusterConfigLister   listersv2alpha1.ApisixClusterConfigLister
	apisixClusterConfigInformer cache.SharedIndexInformer

	// resource controllers
	endpointsController *endpointsController
	ingressController   *ingressController
	secretController    *secretController

	apisixUpstreamController      *apisixUpstreamController
	apisixRouteController         *apisixRouteController
	apisixTlsController           *apisixTlsController
	apisixClusterConfigController *apisixClusterConfigController
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

	var (
		watchingNamespace map[string]struct{}
	)
	if len(cfg.Kubernetes.AppNamespaces) > 1 || cfg.Kubernetes.AppNamespaces[0] != v1.NamespaceAll {
		watchingNamespace = make(map[string]struct{}, len(cfg.Kubernetes.AppNamespaces))
		for _, ns := range cfg.Kubernetes.AppNamespaces {
			watchingNamespace[ns] = struct{}{}
		}
	}

	// recorder
	utilruntime.Must(apisixscheme.AddToScheme(scheme.Scheme))
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.Client.CoreV1().Events("")})

	c := &Controller{
		name:              podName,
		namespace:         podNamespace,
		cfg:               cfg,
		apiServer:         apiSrv,
		apisix:            client,
		metricsCollector:  metrics.NewPrometheusCollector(podName, podNamespace),
		kubeClient:        kubeClient,
		watchingNamespace: watchingNamespace,
		secretSSLMap:      new(sync.Map),
		recorder:          eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: _component}),
	}
	return c, nil
}

func (c *Controller) initWhenStartLeading() {
	var (
		ingressInformer     cache.SharedIndexInformer
		apisixRouteInformer cache.SharedIndexInformer
	)

	kubeFactory := c.kubeClient.NewSharedIndexInformerFactory()
	apisixFactory := c.kubeClient.NewAPISIXSharedIndexInformerFactory()

	c.epLister = kubeFactory.Core().V1().Endpoints().Lister()
	c.svcLister = kubeFactory.Core().V1().Services().Lister()
	c.ingressLister = kube.NewIngressLister(
		kubeFactory.Networking().V1().Ingresses().Lister(),
		kubeFactory.Networking().V1beta1().Ingresses().Lister(),
		kubeFactory.Extensions().V1beta1().Ingresses().Lister(),
	)
	c.secretLister = kubeFactory.Core().V1().Secrets().Lister()
	c.apisixRouteLister = kube.NewApisixRouteLister(
		apisixFactory.Apisix().V1().ApisixRoutes().Lister(),
		apisixFactory.Apisix().V2alpha1().ApisixRoutes().Lister(),
	)
	c.apisixUpstreamLister = apisixFactory.Apisix().V1().ApisixUpstreams().Lister()
	c.apisixTlsLister = apisixFactory.Apisix().V1().ApisixTlses().Lister()
	c.apisixClusterConfigLister = apisixFactory.Apisix().V2alpha1().ApisixClusterConfigs().Lister()

	c.translator = translation.NewTranslator(&translation.TranslatorOptions{
		EndpointsLister:      c.epLister,
		ServiceLister:        c.svcLister,
		ApisixUpstreamLister: c.apisixUpstreamLister,
		SecretLister:         c.secretLister,
	})

	if c.cfg.Kubernetes.IngressVersion == config.IngressNetworkingV1 {
		ingressInformer = kubeFactory.Networking().V1().Ingresses().Informer()
	} else if c.cfg.Kubernetes.IngressVersion == config.IngressNetworkingV1beta1 {
		ingressInformer = kubeFactory.Networking().V1beta1().Ingresses().Informer()
	} else {
		ingressInformer = kubeFactory.Extensions().V1beta1().Ingresses().Informer()
	}
	if c.cfg.Kubernetes.ApisixRouteVersion == config.ApisixRouteV2alpha1 {
		apisixRouteInformer = apisixFactory.Apisix().V2alpha1().ApisixRoutes().Informer()
	} else {
		apisixRouteInformer = apisixFactory.Apisix().V1().ApisixRoutes().Informer()
	}

	c.epInformer = kubeFactory.Core().V1().Endpoints().Informer()
	c.svcInformer = kubeFactory.Core().V1().Services().Informer()
	c.ingressInformer = ingressInformer
	c.apisixRouteInformer = apisixRouteInformer
	c.apisixUpstreamInformer = apisixFactory.Apisix().V1().ApisixUpstreams().Informer()
	c.apisixClusterConfigInformer = apisixFactory.Apisix().V2alpha1().ApisixClusterConfigs().Informer()
	c.secretInformer = kubeFactory.Core().V1().Secrets().Informer()
	c.apisixTlsInformer = apisixFactory.Apisix().V1().ApisixTlses().Informer()

	c.endpointsController = c.newEndpointsController()
	c.apisixUpstreamController = c.newApisixUpstreamController()
	c.ingressController = c.newIngressController()
	c.apisixRouteController = c.newApisixRouteController()
	c.apisixClusterConfigController = c.newApisixClusterConfigController()
	c.apisixTlsController = c.newApisixTlsController()
	c.secretController = c.newSecretController()
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
		// Set it to false as current leaderelection implementation will report
		// "Failed to release lock: resource name may not be empty" error when
		// ReleaseOnCancel is true and the Run context is cancelled.
		ReleaseOnCancel: false,
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
		Name:     c.cfg.APISIX.DefaultClusterName,
		AdminKey: c.cfg.APISIX.DefaultClusterAdminKey,
		BaseURL:  c.cfg.APISIX.DefaultClusterBaseURL,
	}
	err := c.apisix.AddCluster(clusterOpts)
	if err != nil && err != apisix.ErrDuplicatedCluster {
		// TODO give up the leader role
		log.Errorf("failed to add default cluster: %s", err)
		return
	}

	if err := c.apisix.Cluster(c.cfg.APISIX.DefaultClusterName).HasSynced(ctx); err != nil {
		// TODO give up the leader role
		log.Errorf("failed to wait the default cluster to be ready: %s", err)

		// re-create apisix cluster, used in next c.run
		if err = c.apisix.UpdateCluster(clusterOpts); err != nil {
			log.Errorf("failed to update default cluster: %s", err)
			return
		}
		return
	}

	c.initWhenStartLeading()

	c.goAttach(func() {
		c.checkClusterHealth(ctx, cancelFunc)
	})
	c.goAttach(func() {
		c.epInformer.Run(ctx.Done())
	})
	c.goAttach(func() {
		c.svcInformer.Run(ctx.Done())
	})
	c.goAttach(func() {
		c.ingressInformer.Run(ctx.Done())
	})
	c.goAttach(func() {
		c.apisixRouteInformer.Run(ctx.Done())
	})
	c.goAttach(func() {
		c.apisixUpstreamInformer.Run(ctx.Done())
	})
	c.goAttach(func() {
		c.apisixClusterConfigInformer.Run(ctx.Done())
	})
	c.goAttach(func() {
		c.secretInformer.Run(ctx.Done())
	})
	c.goAttach(func() {
		c.apisixTlsInformer.Run(ctx.Done())
	})
	c.goAttach(func() {
		c.endpointsController.run(ctx)
	})
	c.goAttach(func() {
		c.apisixUpstreamController.run(ctx)
	})
	c.goAttach(func() {
		c.ingressController.run(ctx)
	})
	c.goAttach(func() {
		c.apisixRouteController.run(ctx)
	})
	c.goAttach(func() {
		c.apisixClusterConfigController.run(ctx)
	})
	c.goAttach(func() {
		c.apisixTlsController.run(ctx)
	})
	c.goAttach(func() {
		c.secretController.run(ctx)
	})

	c.metricsCollector.ResetLeader(true)

	log.Infow("controller now is running as leader",
		zap.String("namespace", c.namespace),
		zap.String("pod", c.name),
	)

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

func (c *Controller) checkClusterHealth(ctx context.Context, cancelFunc context.CancelFunc) {
	defer cancelFunc()
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
		}

		err := c.apisix.Cluster(c.cfg.APISIX.DefaultClusterName).HealthCheck(ctx)
		if err != nil {
			// Finally failed health check, then give up leader.
			log.Warnf("failed to check health for default cluster: %s, give up leader", err)
			return
		}
		log.Debugf("success check health for default cluster")
	}
}
