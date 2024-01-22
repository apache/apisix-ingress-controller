// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package translation

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/id"
	"github.com/apache/apisix-ingress-controller/pkg/kube"
	configv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/providers/translation"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

func (t *translator) TranslateGlobalRule(agr kube.ApisixGlobalRule) (*translation.TranslateContext, error) {
	switch agr.GroupVersion() {
	case config.ApisixV2:
		return t.translateGlobalRuleV2(agr.V2())
	default:
		return nil, fmt.Errorf("translator: source group version not supported: %s", agr.GroupVersion())
	}
}

func (t *translator) translateGlobalRuleV2(config *configv2.ApisixGlobalRule) (*translation.TranslateContext, error) {
	ctx := translation.DefaultEmptyTranslateContext()
	pluginMap := make(apisixv1.Plugins)
	if len(config.Spec.Plugins) > 0 {
		for _, plugin := range config.Spec.Plugins {
			if !plugin.Enable {
				continue
			}

			if plugin.Config != nil {
				// Here, it will override same key.
				if t, ok := pluginMap[plugin.Name]; ok {
					log.Infow("TranslateGlobalRuleV2 override same plugin key",
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
	pc := apisixv1.NewDefaultGlobalRule()
	pc.ID = id.GenID(apisixv1.ComposeGlobalRuleName(config.Namespace, config.Name))
	pc.Plugins = pluginMap
	ctx.AddGlobalRule(pc)
	return ctx, nil
}

func (t *translator) GenerateGlobalRuleV2DeleteMark(config *configv2.ApisixGlobalRule) (*translation.TranslateContext, error) {
	ctx := translation.DefaultEmptyTranslateContext()
	pc := apisixv1.NewDefaultGlobalRule()
	pc.ID = id.GenID(apisixv1.ComposeGlobalRuleName(config.Namespace, config.Name))
	ctx.AddGlobalRule(pc)
	return ctx, nil
}
