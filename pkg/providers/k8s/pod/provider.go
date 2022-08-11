// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
//
package pod

import (
	"context"

	"k8s.io/client-go/tools/cache"

	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/providers/k8s/namespace"
	providertypes "github.com/apache/apisix-ingress-controller/pkg/providers/types"
	"github.com/apache/apisix-ingress-controller/pkg/providers/utils"
	"github.com/apache/apisix-ingress-controller/pkg/types"
)

var _ Provider = (*podProvider)(nil)

type Provider interface {
	providertypes.Provider

	GetPodCache() types.PodCache
}

type podProvider struct {
	cfg *config.Config

	podInformer   cache.SharedIndexInformer
	podController *podController
}

func NewProvider(common *providertypes.Common, namespaceProvider namespace.WatchingNamespaceProvider) (Provider, error) {
	p := &podProvider{
		cfg: common.Config,

		podInformer: common.PodInformer,
	}

	p.podController = newPodController(common, namespaceProvider, p.podInformer)

	return p, nil
}

func (p *podProvider) Run(ctx context.Context) {
	e := utils.ParallelExecutor{}

	e.Add(func() {
		p.podInformer.Run(ctx.Done())
	})

	e.Add(func() {
		p.podController.run(ctx)
	})

	e.Wait()
}

func (p *podProvider) GetPodCache() types.PodCache {
	return p.podController.podCache
}
