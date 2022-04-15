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

func TestTranslatePluginConfigV2beta3(t *testing.T) {
	apc := &configv2beta3.ApisixPluginConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "apc",
			Namespace: "test-ns",
		},
		Spec: configv2beta3.ApisixPluginConfigSpec{
			Plugins: []configv2beta3.ApisixRouteHTTPPlugin{
				{
					Name:   "case1",
					Enable: true,
					Config: map[string]interface{}{
						"key-1": 1,
						"key-2": 2,
					},
				},
				{
					Name:   "case2",
					Enable: false,
					Config: map[string]interface{}{
						"key-3": 3,
						"key-4": 4,
						"key-5": 5,
					},
				},
				{
					Name:   "case3",
					Enable: true,
					Config: map[string]interface{}{
						"key-6": 6,
						"key-7": 7,
						"key-8": 8,
						"key-9": 9,
					},
				},
			},
		},
	}
	trans := &translator{}
	ctx, err := trans.TranslatePluginConfigV2beta3(apc)
	assert.NoError(t, err)
	assert.Len(t, ctx.PluginConfigs, 1)
	assert.Len(t, ctx.PluginConfigs[0].Plugins, 2)
}

func TestTranslatePluginConfigV2beta3NotStrictly(t *testing.T) {
	apc := &configv2beta3.ApisixPluginConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "apc",
			Namespace: "test-ns",
		},
		Spec: configv2beta3.ApisixPluginConfigSpec{
			Plugins: []configv2beta3.ApisixRouteHTTPPlugin{
				{
					Name:   "case1",
					Enable: true,
					Config: map[string]interface{}{
						"key-1": 1,
						"key-2": 2,
					},
				},
				{
					Name:   "case2",
					Enable: false,
					Config: map[string]interface{}{
						"key-3": 3,
						"key-4": 4,
						"key-5": 5,
					},
				},
				{
					Name:   "case3",
					Enable: true,
					Config: map[string]interface{}{
						"key-6": 6,
						"key-7": 7,
						"key-8": 8,
						"key-9": 9,
					},
				},
			},
		},
	}
	trans := &translator{}
	ctx, err := trans.TranslatePluginConfigV2beta3NotStrictly(apc)
	assert.NoError(t, err)
	assert.Len(t, ctx.PluginConfigs, 1)
	assert.Len(t, ctx.PluginConfigs[0].Plugins, 0)
}
