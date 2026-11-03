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

package v2_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBackendTrafficPolicyTimeoutSchema(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	schema := loadCRDSchemaVersion(t, filepath.Join(filepath.Dir(thisFile), "..", "..",
		"config", "crd", "bases", "apisix.apache.org_backendtrafficpolicies.yaml"), "v1alpha1")
	for _, field := range []string{"connect", "read", "send"} {
		for _, value := range []string{"0s", "60s", "500ms", "1.5s", "1m30s", "-500ms", "", "invalid", "9223372036854775808ns"} {
			t.Run(field+"/"+value, func(t *testing.T) {
				policy := map[string]any{"spec": map[string]any{
					"targetRefs": []any{map[string]any{"group": "", "kind": "Service", "name": "backend"}},
					"timeout":    map[string]any{field: value},
				}}
				err := schema.Validate(t, policy)
				switch value {
				case "-500ms", "", "invalid", "9223372036854775808ns":
					require.Error(t, err)
				default:
					require.NoError(t, err)
				}
			})
		}
	}
}
