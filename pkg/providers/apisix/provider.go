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
package apisix

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/apache/apisix-ingress-controller/pkg/config"
	apisixtranslation "github.com/apache/apisix-ingress-controller/pkg/providers/apisix/translation"
	"github.com/apache/apisix-ingress-controller/pkg/providers/k8s/namespace"
	"github.com/apache/apisix-ingress-controller/pkg/providers/translation"
	providertypes "github.com/apache/apisix-ingress-controller/pkg/providers/types"
	"github.com/apache/apisix-ingress-controller/pkg/providers/utils"
	"github.com/apache/apisix-ingress-controller/pkg/types"
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
	ResourceSync(interval time.Duration, namespace string)
	NotifyServiceAdd(key string)
	NotifyApisixUpstreamChange(key string)

	SyncSecretChange(ctx context.Context, ev *types.Event, secret *corev1.Secret, secretMapKey string)
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
	apisixGlobalRuleController    *apisixGlobalRuleController
}

func NewProvider(common *providertypes.Common, namespaceProvider namespace.WatchingNamespaceProvider,
	translator translation.Translator) (Provider, apisixtranslation.ApisixTranslator, error) {
	p := &apisixProvider{
		name:              ProviderName,
		common:            common,
		namespaceProvider: namespaceProvider,
	}

	p.apisixTranslator = apisixtranslation.NewApisixTranslator(&apisixtranslation.TranslatorOptions{
		Apisix:               common.APISIX,
		ClusterName:          common.Config.APISIX.DefaultClusterName,
		IngressClassName:     common.Config.Kubernetes.IngressClass,
		ServiceLister:        common.SvcLister,
		ApisixUpstreamLister: common.ApisixUpstreamLister,
		SecretLister:         common.SecretLister,
	}, translator)
	c := &apisixCommon{
		Common:            common,
		namespaceProvider: namespaceProvider,
		translator:        p.apisixTranslator,
	}

	p.apisixUpstreamController = newApisixUpstreamController(c, p.NotifyApisixUpstreamChange)
	p.apisixRouteController = newApisixRouteController(c)
	p.apisixTlsController = newApisixTlsController(c)
	p.apisixClusterConfigController = newApisixClusterConfigController(c)
	p.apisixConsumerController = newApisixConsumerController(c)
	p.apisixPluginConfigController = newApisixPluginConfigController(c)
	if p.common.Kubernetes.APIVersion == config.ApisixV2 {
		p.apisixGlobalRuleController = newApisixGlobalRuleController(c)
	}

	return p, p.apisixTranslator, nil
}

func (p *apisixProvider) Run(ctx context.Context) {
	e := utils.ParallelExecutor{}

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
	if p.common.Kubernetes.APIVersion == config.ApisixV2 {
		e.Add(func() {
			p.apisixGlobalRuleController.run(ctx)
		})
	}

	e.Wait()
}

func (p *apisixProvider) ResourceSync(interval time.Duration, namespace string) {
	e := utils.ParallelExecutor{}

	e.Add(func() {
		p.apisixUpstreamController.ResourceSync(interval, namespace)
	})
	e.Add(func() {
		p.apisixRouteController.ResourceSync(interval, namespace)
	})
	e.Add(func() {
		p.apisixTlsController.ResourceSync(interval, namespace)
	})
	e.Add(func() {
		p.apisixClusterConfigController.ResourceSync(interval)
	})
	e.Add(func() {
		p.apisixConsumerController.ResourceSync(interval, namespace)
	})
	e.Add(func() {
		p.apisixPluginConfigController.ResourceSync(interval, namespace)
	})
	if p.common.Kubernetes.APIVersion == config.ApisixV2 {
		e.Add(func() {
			p.apisixGlobalRuleController.ResourceSync(interval, namespace)
		})
	}

	e.Wait()
}

func (p *apisixProvider) NotifyServiceAdd(key string) {
	p.apisixRouteController.NotifyServiceAdd(key)
}

func (p *apisixProvider) NotifyApisixUpstreamChange(key string) {
	p.apisixRouteController.NotifyApisixUpstreamChange(key)
}

func (p *apisixProvider) SyncSecretChange(ctx context.Context, ev *types.Event, secret *corev1.Secret, secretMapKey string) {
	p.apisixTlsController.SyncSecretChange(ctx, ev, secret, secretMapKey)
}
