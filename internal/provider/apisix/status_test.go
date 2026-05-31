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

package apisix

import (
	"testing"

	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	"github.com/apache/apisix-ingress-controller/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestBuildErrorMessage(t *testing.T) {
	tests := []struct {
		name        string
		labels      map[string]string
		originalErr string
		want        string
	}{
		{
			name: "with all labels",
			labels: map[string]string{
				label.LabelKind:      "ApisixRoute",
				label.LabelNamespace: "default",
				label.LabelName:      "test-route",
			},
			originalErr: "connection refused",
			want:        "connection refused (resource: ApisixRoute/default/test-route)",
		},
		{
			name: "with only kind and namespace",
			labels: map[string]string{
				label.LabelKind:      "ApisixConsumer",
				label.LabelNamespace: "production",
			},
			originalErr: "timeout",
			want:        "timeout (resource: ApisixConsumer/production/)",
		},
		{
			name: "with only kind and name",
			labels: map[string]string{
				label.LabelKind: "ApisixTls",
				label.LabelName: "my-tls",
			},
			originalErr: "certificate expired",
			want:        "certificate expired (resource: ApisixTls//my-tls)",
		},
		{
			name: "with empty namespace and name",
			labels: map[string]string{
				label.LabelKind: "ApisixRoute",
			},
			originalErr: "timeout",
			want:        "timeout",
		},
		{
			name:        "with empty labels",
			labels:      map[string]string{},
			originalErr: "unknown error",
			want:        "unknown error",
		},
		{
			name:        "with nil labels",
			labels:      nil,
			originalErr: "nil labels error",
			want:        "nil labels error",
		},
		{
			name: "with complex error message",
			labels: map[string]string{
				label.LabelKind:      "ApisixGlobalRule",
				label.LabelNamespace: "test-ns",
				label.LabelName:      "global-rule-1",
			},
			originalErr: "ServerAddr: http://apisix:9180, Err: connection refused",
			want:        "ServerAddr: http://apisix:9180, Err: connection refused (resource: ApisixGlobalRule/test-ns/global-rule-1)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildErrorMessage(tt.labels, tt.originalErr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAddResourceToStatusUpdateMap(t *testing.T) {
	tests := []struct {
		name        string
		labels      map[string]string
		msg         string
		initialMap  map[types.NamespacedNameKind][]string
		expectedMap map[types.NamespacedNameKind][]string
	}{
		{
			name: "add new resource status",
			labels: map[string]string{
				label.LabelKind:      "ApisixRoute",
				label.LabelNamespace: "default",
				label.LabelName:      "test-route",
			},
			msg:        "connection refused",
			initialMap: map[types.NamespacedNameKind][]string{},
			expectedMap: map[types.NamespacedNameKind][]string{
				{
					Kind:      "ApisixRoute",
					Namespace: "default",
					Name:      "test-route",
				}: {"connection refused"},
			},
		},
		{
			name: "append to existing resource status",
			labels: map[string]string{
				label.LabelKind:      "ApisixRoute",
				label.LabelNamespace: "default",
				label.LabelName:      "test-route",
			},
			msg: "timeout error",
			initialMap: map[types.NamespacedNameKind][]string{
				{
					Kind:      "ApisixRoute",
					Namespace: "default",
					Name:      "test-route",
				}: {"previous error"},
			},
			expectedMap: map[types.NamespacedNameKind][]string{
				{
					Kind:      "ApisixRoute",
					Namespace: "default",
					Name:      "test-route",
				}: {"previous error", "timeout error"},
			},
		},
		{
			name: "add different resources",
			labels: map[string]string{
				label.LabelKind:      "ApisixConsumer",
				label.LabelNamespace: "production",
				label.LabelName:      "consumer-1",
			},
			msg: "auth failed",
			initialMap: map[types.NamespacedNameKind][]string{
				{
					Kind:      "ApisixRoute",
					Namespace: "default",
					Name:      "test-route",
				}: {"route error"},
			},
			expectedMap: map[types.NamespacedNameKind][]string{
				{
					Kind:      "ApisixRoute",
					Namespace: "default",
					Name:      "test-route",
				}: {"route error"},
				{
					Kind:      "ApisixConsumer",
					Namespace: "production",
					Name:      "consumer-1",
				}: {"auth failed"},
			},
		},
		{
			name: "handle empty labels gracefully",
			labels: map[string]string{
				label.LabelKind:      "",
				label.LabelNamespace: "",
				label.LabelName:      "",
			},
			msg:        "error with empty labels",
			initialMap: map[types.NamespacedNameKind][]string{},
			expectedMap: map[types.NamespacedNameKind][]string{
				{
					Kind:      "",
					Namespace: "",
					Name:      "",
				}: {"error with empty labels"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &apisixProvider{}
			statusUpdateMap := tt.initialMap
			d.addResourceToStatusUpdateMap(tt.labels, tt.msg, statusUpdateMap)
			assert.Equal(t, tt.expectedMap, statusUpdateMap)
		})
	}
}
