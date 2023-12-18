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
package namespace

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/kube"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	provider "github.com/apache/apisix-ingress-controller/pkg/providers/types"
	"github.com/apache/apisix-ingress-controller/pkg/providers/utils"
	"github.com/apache/apisix-ingress-controller/pkg/types"
)

type WatchingNamespaceProvider interface {
	provider.Provider

	Init(ctx context.Context) error

	IsWatchingNamespace(key string) bool
	WatchingNamespaces() []string
}

type watchingProvider struct {
	kube *kube.KubeClient
	cfg  *config.Config

	watchingNamespaces *sync.Map
	watchingLabels     types.Labels

	namespaceInformer cache.SharedIndexInformer
	namespaceLister   listerscorev1.NamespaceLister

	controller *namespaceController

	enableLabelsWatching bool
}

func NewWatchingNamespaceProvider(ctx context.Context, kube *kube.KubeClient, cfg *config.Config, syncCh chan string) (WatchingNamespaceProvider, error) {
	c := &watchingProvider{
		kube: kube,
		cfg:  cfg,

		watchingNamespaces: new(sync.Map),
		watchingLabels:     make(map[string]string),

		enableLabelsWatching: false,
	}

	if len(cfg.Kubernetes.NamespaceSelector) == 0 {
		return c, nil
	}

	// support namespace label-selector
	c.enableLabelsWatching = true
	for _, selector := range cfg.Kubernetes.NamespaceSelector {
		labelSlice := strings.Split(selector, "=")
		if len(labelSlice) != 2 {
			return nil, fmt.Errorf("bad namespace-selector format: %s, expected namespace-selector format: xxx=xxx", selector)
		}
		c.watchingLabels[labelSlice[0]] = labelSlice[1]
	}

	kubeFactory := kube.NewSharedIndexInformerFactory()
	c.namespaceInformer = kubeFactory.Core().V1().Namespaces().Informer()
	c.namespaceLister = kubeFactory.Core().V1().Namespaces().Lister()

	c.controller = newNamespaceController(c, syncCh)

	return c, nil
}

func (c *watchingProvider) Init(ctx context.Context) error {
	err := c.initWatchingNamespacesByLabels(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (c *watchingProvider) initWatchingNamespacesByLabels(ctx context.Context) error {
	labelSelector := metav1.LabelSelector{MatchLabels: c.watchingLabels}
	opts := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}
	namespaces, err := c.kube.Client.CoreV1().Namespaces().List(ctx, opts)
	if err != nil {
		return err
	}
	var nss []string

	for _, ns := range namespaces.Items {
		nss = append(nss, ns.Name)
		c.watchingNamespaces.Store(ns.Name, struct{}{})
	}
	log.Infow("label selector watching namespaces", zap.Strings("namespaces", nss))
	return nil
}

func (c *watchingProvider) Run(ctx context.Context) {
	if !c.enableLabelsWatching {
		return
	}

	e := utils.ParallelExecutor{}
	e.Add(func() {
		c.namespaceInformer.Run(ctx.Done())
	})
	e.Add(func() {
		c.controller.run(ctx)
	})

	e.Wait()
}

func (c *watchingProvider) WatchingNamespaces() []string {
	var keys []string
	if c.enableLabelsWatching {
		c.watchingNamespaces.Range(func(key, _ interface{}) bool {
			keys = append(keys, key.(string))
			return true
		})
	} else {
		namespaces, err := c.kube.Client.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
		if err != nil {
			log.Warnw("Namespace list get failed",
				zap.Error(err),
			)
			return nil
		}
		for _, ns := range namespaces.Items {
			keys = append(keys, ns.Name)
		}
	}
	return keys
}

// IsWatchingNamespace accepts a resource key, getting the namespace part
// and checking whether the namespace is being watched.
func (c *watchingProvider) IsWatchingNamespace(key string) (ok bool) {
	if !c.enableLabelsWatching {
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
	_, ok = c.watchingNamespaces.Load(ns)
	return
}
