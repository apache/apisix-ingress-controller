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

package translator

import (
	"encoding/json"

	"github.com/api7/gopkg/pkg/log"
	"go.uber.org/zap"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/provider"
)

// TranslateApisixGlobalRule translates ApisixGlobalRule to APISIX GlobalRule
func (t *Translator) TranslateApisixGlobalRule(tctx *provider.TranslateContext, obj *apiv2.ApisixGlobalRule) (*TranslateResult, error) {
	log.Debugw("translating ApisixGlobalRule",
		zap.String("namespace", obj.Namespace),
		zap.String("name", obj.Name),
	)

	// Create global rule plugins
	plugins := make(adctypes.Plugins)

	// Translate each plugin from the spec
	for _, plugin := range obj.Spec.Plugins {
		// Check if plugin is enabled (default to true if not specified)
		if !plugin.Enable {
			continue
		}

		pluginConfig := make(map[string]any)
		if len(plugin.Config.Raw) > 0 {
			if err := json.Unmarshal(plugin.Config.Raw, &pluginConfig); err != nil {
				log.Errorw("failed to unmarshal plugin config", zap.String("plugin", plugin.Name), zap.Error(err))
				continue
			}
		}
		plugins[plugin.Name] = pluginConfig
	}

	return &TranslateResult{
		GlobalRules: adctypes.GlobalRule(plugins),
	}, nil
}
