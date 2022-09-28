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
package configmap

import (
	"context"

	"github.com/apache/apisix-ingress-controller/pkg/config"
	providertypes "github.com/apache/apisix-ingress-controller/pkg/providers/types"
	"github.com/apache/apisix-ingress-controller/pkg/providers/utils"
	"k8s.io/client-go/tools/cache"
)

var _ Provider = (*configmapProvider)(nil)

type Provider interface {
	providertypes.Provider
}

type configmapProvider struct {
	cfg *config.Config

	configmapInformer   cache.SharedIndexInformer
	configmapController *configmapController
}

func NewProvider(common *providertypes.Common) (Provider, error) {
	p := &configmapProvider{
		cfg: common.Config,

		configmapInformer: common.ConfigMapInformer,
	}

	p.configmapController = newConfigMapController(common)

	p.configmapController.Subscription(p.cfg.PluginMetadataConfigMap)

	return p, nil
}

func (p *configmapProvider) Run(ctx context.Context) {
	e := utils.ParallelExecutor{}

	e.Add(func() {
		p.configmapController.run(ctx)
	})

	e.Wait()
}
