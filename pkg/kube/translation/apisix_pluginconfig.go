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
package translation

import (
	"go.uber.org/zap"

	"github.com/apache/apisix-ingress-controller/pkg/id"
	configv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	configv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta3"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

func (t *translator) TranslatePluginConfigV2beta3(config *configv2beta3.ApisixPluginConfig) (*TranslateContext, error) {
	ctx := DefaultEmptyTranslateContext()
	pluginMap := make(apisixv1.Plugins)
	if len(config.Spec.Plugins) > 0 {
		for _, plugin := range config.Spec.Plugins {
			if !plugin.Enable {
				continue
			}
			if plugin.Config != nil {
				// Here, it will override same key.
				if t, ok := pluginMap[plugin.Name]; ok {
					log.Infow("TranslatePluginConfigV2beta3 override same plugin key",
						zap.String("key", plugin.Name),
						zap.Any("old", t),
						zap.Any("new", plugin.Config),
					)
				}
				pluginMap[plugin.Name] = plugin.Config
			} else {
				pluginMap[plugin.Name] = make(map[string]interface{})
			}
		}
	}
	pc := apisixv1.NewDefaultPluginConfig()
	pc.Name = apisixv1.ComposePluginConfigName(config.Namespace, config.Name)
	pc.ID = id.GenID(pc.Name)
	pc.Plugins = pluginMap
	ctx.AddPluginConfig(pc)
	return ctx, nil
}

func (t *translator) TranslatePluginConfigV2beta3NotStrictly(config *configv2beta3.ApisixPluginConfig) (*TranslateContext, error) {
	ctx := DefaultEmptyTranslateContext()
	pc := apisixv1.NewDefaultPluginConfig()
	pc.Name = apisixv1.ComposePluginConfigName(config.Namespace, config.Name)
	pc.ID = id.GenID(pc.Name)
	ctx.AddPluginConfig(pc)
	return ctx, nil
}

func (t *translator) TranslatePluginConfigV2(config *configv2.ApisixPluginConfig) (*TranslateContext, error) {
	ctx := defaultEmptyTranslateContext()
	pluginMap := make(apisixv1.Plugins)
	if len(config.Spec.Plugins) > 0 {
		for _, plugin := range config.Spec.Plugins {
			if !plugin.Enable {
				continue
			}
			if plugin.Config != nil {
				// Here, it will override same key.
				if t, ok := pluginMap[plugin.Name]; ok {
					log.Infow("TranslatePluginConfigV2 override same plugin key",
						zap.String("key", plugin.Name),
						zap.Any("old", t),
						zap.Any("new", plugin.Config),
					)
				}
				pluginMap[plugin.Name] = plugin.Config
			} else {
				pluginMap[plugin.Name] = make(map[string]interface{})
			}
		}
	}
	pc := apisixv1.NewDefaultPluginConfig()
	pc.Name = apisixv1.ComposePluginConfigName(config.Namespace, config.Name)
	pc.ID = id.GenID(pc.Name)
	pc.Plugins = pluginMap
	ctx.addPluginConfig(pc)
	return ctx, nil
}

func (t *translator) TranslatePluginConfigV2NotStrictly(config *configv2.ApisixPluginConfig) (*TranslateContext, error) {
	ctx := defaultEmptyTranslateContext()
	pc := apisixv1.NewDefaultPluginConfig()
	pc.Name = apisixv1.ComposePluginConfigName(config.Namespace, config.Name)
	pc.ID = id.GenID(pc.Name)
	ctx.addPluginConfig(pc)
	return ctx, nil
}
