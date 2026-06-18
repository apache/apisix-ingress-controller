// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package client

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildADCExecuteArgs(t *testing.T) {
	tests := []struct {
		name         string
		filePath     string
		labels       map[string]string
		includeTypes []string
		excludeTypes []string
		wantContains []string
		wantAbsent   []string
	}{
		{
			name:         "no filters",
			filePath:     "/tmp/sync.json",
			wantContains: []string{"sync", "-f", "/tmp/sync.json"},
			wantAbsent:   []string{"--label-selector", "--include-resource-type", "--exclude-resource-type"},
		},
		{
			name:         "with label selector",
			filePath:     "/tmp/sync.json",
			labels:       map[string]string{"app": "apisix"},
			wantContains: []string{"--label-selector", "app=apisix"},
			wantAbsent:   []string{"--include-resource-type", "--exclude-resource-type"},
		},
		{
			name:         "with include types",
			filePath:     "/tmp/sync.json",
			includeTypes: []string{"Consumer"},
			wantContains: []string{"--include-resource-type", "Consumer"},
			wantAbsent:   []string{"--exclude-resource-type"},
		},
		{
			name:         "with exclude types",
			filePath:     "/tmp/sync.json",
			excludeTypes: []string{"Consumer"},
			wantContains: []string{"--exclude-resource-type", "Consumer"},
			wantAbsent:   []string{"--include-resource-type"},
		},
		{
			name:         "with multiple exclude types",
			filePath:     "/tmp/sync.json",
			excludeTypes: []string{"Consumer", "ConsumerGroup"},
			wantContains: []string{"--exclude-resource-type", "Consumer", "ConsumerGroup"},
		},
		{
			name:         "with include and exclude types",
			filePath:     "/tmp/sync.json",
			includeTypes: []string{"Consumer"},
			excludeTypes: []string{"ConsumerGroup"},
			wantContains: []string{"--include-resource-type", "Consumer", "--exclude-resource-type", "ConsumerGroup"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := BuildADCExecuteArgs(tt.filePath, tt.labels, tt.includeTypes, tt.excludeTypes)
			for _, want := range tt.wantContains {
				require.Contains(t, args, want)
			}
			for _, absent := range tt.wantAbsent {
				require.NotContains(t, args, absent)
			}
		})
	}
}
