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
	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/provider"
)

// TranslateApisixGlobalRule translates ApisixGlobalRule to APISIX GlobalRule
func (t *Translator) TranslateApisixGlobalRule(tctx *provider.TranslateContext, obj *apiv2.ApisixGlobalRule) (*TranslateResult, error) {
	t.Log.V(1).Info("translating ApisixGlobalRule", "namespace", obj.Namespace, "name", obj.Name)

	// Create global rule plugins
	plugins := make(adctypes.Plugins)

	// Translate each plugin from the spec
	for _, plugin := range obj.Spec.Plugins {
		// Check if plugin is enabled (default to true if not specified)
		if !plugin.Enable {
			continue
		}

		pluginConfig := t.buildPluginConfig(plugin, obj.Namespace, tctx.Secrets)
		plugins[plugin.Name] = pluginConfig
	}

	return &TranslateResult{
		GlobalRules: adctypes.GlobalRule(plugins),
	}, nil
}
