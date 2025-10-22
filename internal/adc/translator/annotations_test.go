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

package translator

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/internal/adc/translator/annotations"
)

type mockParser struct {
	output any
	err    error
}

func (m *mockParser) Parse(extractor annotations.Extractor) (any, error) {
	return m.output, m.err
}

func TestTranslateAnnotations(t *testing.T) {
	tests := []struct {
		name      string
		anno      map[string]string
		parsers   map[string]annotations.IngressAnnotationsParser
		expected  any
		expectErr bool
	}{
		{
			name: "successful parsing",
			anno: map[string]string{"key1": "value1"},
			parsers: map[string]annotations.IngressAnnotationsParser{
				"key1": &mockParser{output: "parsedValue1", err: nil},
			},
			expected:  map[string]any{"key1": "parsedValue1"},
			expectErr: false,
		},
		{
			name: "parsing with error",
			anno: map[string]string{"key1": "value1"},
			parsers: map[string]annotations.IngressAnnotationsParser{
				"key1": &mockParser{output: nil, err: errors.New("parse error")},
			},
			expected:  map[string]any{},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up mock parsers
			for key, parser := range tt.parsers {
				ingressAnnotationParsers[key] = parser
			}

			dst := make(map[string]any)
			err := translateAnnotations(tt.anno, &dst)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expected, dst)

			// Clean up mock parsers
			for key := range tt.parsers {
				delete(ingressAnnotationParsers, key)
			}
		})
	}
}
