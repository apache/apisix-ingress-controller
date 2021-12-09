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
	"github.com/apache/apisix-ingress-controller/pkg/id"
	configv2beta2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta2"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

func (t *translator) translatePluginConfig(namespace, name string, plugins apisixv1.Plugins) (*apisixv1.PluginConfig, error) {
	pc, err := t.TranslatePluginConfig(namespace, name, plugins)
	if err != nil {
		return nil, err
	}
	pc.Name = apisixv1.ComposePluginConfigName(namespace, name)
	pc.ID = id.GenID(pc.Name)
	return pc, nil
}

func (t *translator) TranslatePluginConfig(namespace, name string, plugins apisixv1.Plugins) (*apisixv1.PluginConfig, error) {
	pc := apisixv1.NewDefaultPluginConfig()
	pc.Plugins = plugins
	return pc, nil
}

// TranslatePluginConfigV2beta2 temporarily
func (t *translator) TranslatePluginConfigV2beta2(apc *configv2beta2.ApisixPluginConfig) (*TranslateContext, error) {
	ctx := defaultEmptyTranslateContext()
	if err := t.translatePluginConfigV2beta2(ctx, apc); err != nil {
		return nil, err
	}
	return ctx, nil
}

// translatePluginConfigV2beta2 temporarily
func (t *translator) translatePluginConfigV2beta2(ctx *TranslateContext, apc *configv2beta2.ApisixPluginConfig) error {
	//pluginMap := make(apisixv1.Plugins)
	// add route plugins
	//for _, plugin := range apc.Spec.Plugins {
	//
	//	if !plugin.Enable {
	//		continue
	//	}
	//	if plugin.Config != nil {
	//		pluginMap[plugin.Name] = plugin.Config
	//	} else {
	//		pluginMap[plugin.Name] = make(map[string]interface{})
	//	}
	//}
	return nil
}

// TranslatePluginConfigV2beta2NotStrictly temporarily
func (t *translator) TranslatePluginConfigV2beta2NotStrictly(*configv2beta2.ApisixPluginConfig) (*TranslateContext, error) {
	ctx := defaultEmptyTranslateContext()
	return ctx, nil
}
