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

package annotations

import (
	"testing"

	"github.com/stretchr/testify/assert"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
)

func TestCorsParser_Parse(t *testing.T) {
	tests := []struct {
		name        string
		annotations map[string]string
		expected    *adctypes.CorsConfig
		expectNil   bool
	}{
		{
			name: "cors disabled",
			annotations: map[string]string{
				AnnotationsEnableCors: "false",
			},
			expectNil: true,
		},
		{
			name: "cors enabled with all fields",
			annotations: map[string]string{
				AnnotationsEnableCors:       "true",
				AnnotationsCorsAllowOrigin:  "https://example.com",
				AnnotationsCorsAllowMethods: "GET,POST,PUT",
				AnnotationsCorsAllowHeaders: "Content-Type,Authorization",
			},
			expected: &adctypes.CorsConfig{
				AllowOrigins: "https://example.com",
				AllowMethods: "GET,POST,PUT",
				AllowHeaders: "Content-Type,Authorization",
			},
		},
		{
			name: "cors enabled with partial fields",
			annotations: map[string]string{
				AnnotationsEnableCors:      "true",
				AnnotationsCorsAllowOrigin: "https://example.com",
			},
			expected: &adctypes.CorsConfig{
				AllowOrigins: "https://example.com",
			},
		},
		{
			name: "cors enabled without any config",
			annotations: map[string]string{
				AnnotationsEnableCors: "true",
			},
			expected: &adctypes.CorsConfig{},
		},
		{
			name:        "no cors annotation",
			annotations: map[string]string{},
			expectNil:   true,
		},
		{
			name: "cors enabled with wildcard origin",
			annotations: map[string]string{
				AnnotationsEnableCors:       "true",
				AnnotationsCorsAllowOrigin:  "*",
				AnnotationsCorsAllowMethods: "*",
				AnnotationsCorsAllowHeaders: "*",
			},
			expected: &adctypes.CorsConfig{
				AllowOrigins: "*",
				AllowMethods: "*",
				AllowHeaders: "*",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewCorsParser()
			extractor := NewExtractor(tt.annotations)

			result, err := parser.Parse(extractor)

			assert.NoError(t, err)

			if tt.expectNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				corsConfig, ok := result.(*adctypes.CorsConfig)
				assert.True(t, ok)
				assert.Equal(t, tt.expected.AllowOrigins, corsConfig.AllowOrigins)
				assert.Equal(t, tt.expected.AllowMethods, corsConfig.AllowMethods)
				assert.Equal(t, tt.expected.AllowHeaders, corsConfig.AllowHeaders)
			}
		})
	}
}
