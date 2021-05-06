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
	configv2alpha1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2alpha1"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

type prometheusPluginConfig struct{}

type skywalkingPluginConfig struct {
	SampleRatio float64 `json:"sample_ratio,omitempty"`
}

func (t *translator) TranslateClusterConfig(acc *configv2alpha1.ApisixClusterConfig) (*apisixv1.GlobalRule, error) {
	globalRule := &apisixv1.GlobalRule{
		ID:      id.GenID(acc.Name),
		Plugins: make(apisixv1.Plugins),
	}

	if acc.Spec.Monitoring != nil {
		if acc.Spec.Monitoring.Prometheus.Enable {
			globalRule.Plugins["prometheus"] = &prometheusPluginConfig{}
		}
		if acc.Spec.Monitoring.Skywalking.Enable {
			globalRule.Plugins["skywalking"] = &skywalkingPluginConfig{
				SampleRatio: acc.Spec.Monitoring.Skywalking.SampleRatio,
			}
		}
	}

	return globalRule, nil
}
