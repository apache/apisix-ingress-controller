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
package endpoint

import (
	"context"

	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/providers/k8s/namespace"
	"github.com/apache/apisix-ingress-controller/pkg/providers/translation"
	providertypes "github.com/apache/apisix-ingress-controller/pkg/providers/types"
	"github.com/apache/apisix-ingress-controller/pkg/providers/utils"
)

var _ Provider = (*endpointProvider)(nil)

type Provider interface {
	providertypes.Provider
}

type endpointProvider struct {
	cfg *config.Config

	endpointsController     *endpointsController
	endpointSliceController *endpointSliceController
}

func NewProvider(common *providertypes.Common, translator translation.Translator, namespaceProvider namespace.WatchingNamespaceProvider) (Provider, error) {
	p := &endpointProvider{
		cfg: common.Config,
	}

	base := &baseEndpointController{
		Common:     common,
		translator: translator,

		svcLister:            common.SvcLister,
		apisixUpstreamLister: common.ApisixUpstreamLister,
	}

	if common.Kubernetes.WatchEndpointSlices {
		p.endpointSliceController = newEndpointSliceController(base, namespaceProvider)
	} else {
		p.endpointsController = newEndpointsController(base, namespaceProvider)
	}

	return p, nil
}

func (p *endpointProvider) Run(ctx context.Context) {
	e := utils.ParallelExecutor{}

	e.Add(func() {
		if p.cfg.Kubernetes.WatchEndpointSlices {
			p.endpointSliceController.run(ctx)
		} else {
			p.endpointsController.run(ctx)
		}
	})

	e.Wait()
}
