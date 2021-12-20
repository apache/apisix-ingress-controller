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
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta3"
)

func TestTranslateApisixPluginConfig(t *testing.T) {
	apc := &configv2beta3.ApisixPluginConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "apc",
			Namespace: "test-ns",
		},
		Spec: configv2beta3.ApisixPluginConfigSpec{
			Plugins: []configv2beta3.ApisixRouteHTTPPluginConfig{
				{
					"key-1": 123,
					"key-2": map[string][]string{
						"whitelist": {
							"127.0.0.0/24",
							"113.74.26.106",
						},
					},
					"key-3": map[string]int{
						"count":         2,
						"time_window":   60,
						"rejected_code": 503,
					},
				},
				{
					"key-1": 123456,
					"key-2": map[string][]string{
						"whitelist": {
							"1.1.1.1",
						},
					},
					"key-4": map[string]int{
						"count":         5,
						"time_window":   60,
						"rejected_code": 503,
					},
				},
			},
		},
	}
	trans := &translator{}
	pluginConfig, err := trans.TranslateApisixPluginConfig(apc)
	assert.NoError(t, err)
	assert.Len(t, pluginConfig.Plugins, 4)
	assert.Equal(t, apc.Spec.Plugins[1]["key-1"], pluginConfig.Plugins["key-1"])
	assert.Equal(t, apc.Spec.Plugins[1]["key-2"], pluginConfig.Plugins["key-2"])
	assert.Equal(t, apc.Spec.Plugins[0]["key-3"], pluginConfig.Plugins["key-3"])
	assert.Equal(t, apc.Spec.Plugins[1]["key-4"], pluginConfig.Plugins["key-4"])
}
