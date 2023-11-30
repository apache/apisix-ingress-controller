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
package ingress

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	apisixtranslation "github.com/apache/apisix-ingress-controller/pkg/providers/apisix/translation"
	ingresstranslation "github.com/apache/apisix-ingress-controller/pkg/providers/ingress/translation"
	"github.com/apache/apisix-ingress-controller/pkg/providers/k8s/namespace"
	"github.com/apache/apisix-ingress-controller/pkg/providers/translation"
	providertypes "github.com/apache/apisix-ingress-controller/pkg/providers/types"
	"github.com/apache/apisix-ingress-controller/pkg/providers/utils"
	"github.com/apache/apisix-ingress-controller/pkg/types"
)

const (
	ProviderName = "Ingress"
)

type ingressCommon struct {
	*providertypes.Common

	namespaceProvider namespace.WatchingNamespaceProvider
	translator        ingresstranslation.IngressTranslator
}

var _ Provider = (*ingressProvider)(nil)

type Provider interface {
	providertypes.Provider

	ResourceSync(namespace string)

	SyncSecretChange(ctx context.Context, ev *types.Event, secret *corev1.Secret, secretMapKey string)
}

type ingressProvider struct {
	name string

	ingressController *ingressController
}

func NewProvider(common *providertypes.Common, namespaceProvider namespace.WatchingNamespaceProvider,
	translator translation.Translator, apisixTranslator apisixtranslation.ApisixTranslator) (Provider, error) {
	p := &ingressProvider{
		name: ProviderName,
	}

	c := &ingressCommon{
		Common:            common,
		namespaceProvider: namespaceProvider,
		translator: ingresstranslation.NewIngressTranslator(&ingresstranslation.TranslatorOptions{
			Apisix:        common.APISIX,
			ClusterName:   common.Config.APISIX.DefaultClusterName,
			ServiceLister: common.SvcLister,
		}, translator, apisixTranslator),
	}

	p.ingressController = newIngressController(c)

	return p, nil
}

func (p *ingressProvider) Run(ctx context.Context) {
	e := utils.ParallelExecutor{}

	e.Add(func() {
		p.ingressController.run(ctx)
	})

	e.Wait()
}

func (p *ingressProvider) ResourceSync(namespace string) {
	e := utils.ParallelExecutor{}

	e.Add(
		func() {
			p.ingressController.ResourceSync(namespace)
		})

	e.Wait()
}

func (p *ingressProvider) SyncSecretChange(ctx context.Context, ev *types.Event, secret *corev1.Secret, secretMapKey string) {
	p.ingressController.SyncSecretChange(ctx, ev, secret, secretMapKey)
}
