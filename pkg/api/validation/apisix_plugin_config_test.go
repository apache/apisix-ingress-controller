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

package validation

import (
	"testing"

	v2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
)

func TestValidateApisixRoutePlugins(t *testing.T) {
	validPlugins := []v2.ApisixRoutePlugin{
		{
			Name: "plugin1",
			Config: v2.ApisixRoutePluginConfig{
				"param1": "value1",
				"param2": 123,
			},
		},
		{
			Name: "plugin2",
			Config: v2.ApisixRoutePluginConfig{
				"param1": "value2",
			},
		},
	}

	invalidPlugins := []v2.ApisixRoutePlugin{
		{
			Name:   "",
			Config: v2.ApisixRoutePluginConfig{},
		},
		{
			Name: "plugin1",
			Config: v2.ApisixRoutePluginConfig{
				"param1": "value1",
				"param2": "invalid",
			},
		},
	}

	tests := []struct {
		name        string
		plugins     []v2.ApisixRoutePlugin
		expectValid bool
		expectError bool
	}{
		{
			name:        "Valid ApisixRoutePlugin objects",
			plugins:     validPlugins,
			expectValid: true,
			expectError: false,
		},
		{
			name: "Invalid ApisixRoutePlugin objects with empty name and config",
			plugins: []v2.ApisixRoutePlugin{
				{Name: "", Config: v2.ApisixRoutePluginConfig{}},
			},
			expectValid: false,
			expectError: false,
		},
		{
			name:        "Invalid ApisixRoutePlugin objects with invalid config value",
			plugins:     invalidPlugins,
			expectValid: false,
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotValid, err := ValidateApisixRoutePlugins(tc.plugins)
			if gotValid != tc.expectValid {
				t.Errorf("ValidateApisixRoutePlugins() gotValid = %v, expect %v", gotValid, tc.expectValid)
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
