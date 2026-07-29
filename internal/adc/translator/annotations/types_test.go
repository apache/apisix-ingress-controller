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
)

func TestGetStringsAnnotation(t *testing.T) {
	const key = "test-key"
	testCases := []struct {
		name        string
		annotations map[string]string
		expected    []string
	}{
		{
			name:        "no spaces",
			annotations: map[string]string{key: "127.0.0.1,0.0.0.0"},
			expected:    []string{"127.0.0.1", "0.0.0.0"},
		},
		{
			name:        "spaces after comma",
			annotations: map[string]string{key: "127.0.0.1, 0.0.0.0"},
			expected:    []string{"127.0.0.1", "0.0.0.0"},
		},
		{
			name:        "tabs and surrounding spaces",
			annotations: map[string]string{key: " 127.0.0.1 ,\t0.0.0.0\t"},
			expected:    []string{"127.0.0.1", "0.0.0.0"},
		},
		{
			name:        "empty and blank elements",
			annotations: map[string]string{key: "127.0.0.1,, ,0.0.0.0,"},
			expected:    []string{"127.0.0.1", "0.0.0.0"},
		},
		{
			name:        "single value",
			annotations: map[string]string{key: "127.0.0.1"},
			expected:    []string{"127.0.0.1"},
		},
		{
			name:        "all blank elements",
			annotations: map[string]string{key: " , ,\t"},
			expected:    nil,
		},
		{
			name:        "absent annotation",
			annotations: map[string]string{},
			expected:    nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := NewExtractor(tc.annotations)
			assert.Equal(t, tc.expected, e.GetStringsAnnotation(key))
		})
	}
}
