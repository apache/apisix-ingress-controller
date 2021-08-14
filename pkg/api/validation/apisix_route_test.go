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
	"context"
	"fmt"
	"testing"

	"github.com/apache/apisix-ingress-controller/pkg/apisix"
	"github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2alpha1"
	v1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

type fakeSchemaClient struct {
	schema map[string]string
}

func (c fakeSchemaClient) GetPluginSchema(ctx context.Context, name string) (*v1.Schema, error) {
	if s, ok := c.schema[name]; ok {
		return &v1.Schema{
			Name:    name,
			Content: s,
		}, nil
	}
	return nil, fmt.Errorf("can't find the plugin schema")
}

func newFakeSchemaClient() apisix.Schema {
	testData := map[string]string{
		"api-breaker": `{"required":["break_response_code"],"$comment":"this is a mark for our injected plugin schema","type":"object","properties":{"healthy":{"properties":{"successes":{"minimum":1,"type":"integer","default":3},"http_statuses":{"items":{"minimum":200,"type":"integer","maximum":499},"uniqueItems":true,"type":"array","minItems":1,"default":[200]}},"type":"object","default":{"successes":3,"http_statuses":[200]}},"break_response_code":{"minimum":200,"type":"integer","maximum":599},"max_breaker_sec":{"minimum":3,"type":"integer","default":300},"unhealthy":{"properties":{"failures":{"minimum":1,"type":"integer","default":3},"http_statuses":{"items":{"minimum":500,"type":"integer","maximum":599},"uniqueItems":true,"type":"array","minItems":1,"default":[500]}},"type":"object","default":{"failures":3,"http_statuses":[500]}},"disable":{"type":"boolean"}}}`,
	}
	return fakeSchemaClient{
		schema: testData,
	}
}

func Test_validatePlugin(t *testing.T) {
	tests := []struct {
		name         string
		pluginName   string
		pluginConfig interface{}
		wantValid    bool
	}{
		{
			name:       "validating is successes",
			pluginName: "api-breaker",
			pluginConfig: v2alpha1.ApisixRouteHTTPPluginConfig{
				"break_response_code": 200,
			},
			wantValid: true,
		},
		{
			name:       "validating is failed due to missing required fields",
			pluginName: "api-breaker",
			pluginConfig: v2alpha1.ApisixRouteHTTPPluginConfig{
				"max_breaker_sec": 60,
			},
			wantValid: false,
		},
		{
			name:       "validating is failed due to invalid break_response_code",
			pluginName: "api-breaker",
			pluginConfig: v2alpha1.ApisixRouteHTTPPluginConfig{
				"break_response_code": 100,
			},
			wantValid: false,
		},
		{
			name:       "validating is failed due to invalid max_breaker_sec",
			pluginName: "api-breaker",
			pluginConfig: v2alpha1.ApisixRouteHTTPPluginConfig{
				"break_response_code": 200,
				"max_breaker_sec":     2,
			},
			wantValid: false,
		},
		{
			name:       "unknown plugin name",
			pluginName: "Not-A-Plugin",
			pluginConfig: v2alpha1.ApisixRouteHTTPPluginConfig{
				"break_response_code": 200,
				"max_breaker_sec":     2,
			},
			wantValid: false,
		},
	}

	fakeClient := newFakeSchemaClient()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValid, _, _ := validatePlugin(fakeClient, tt.pluginName, tt.pluginConfig)
			if gotValid != tt.wantValid {
				t.Errorf("validatePlugin() gotValid = %v, want %v", gotValid, tt.wantValid)
			}
		})
	}
}
