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

	"github.com/stretchr/testify/assert"

	v1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

func TestValidateHTTPPluginSchema(t *testing.T) {
	validator, err := NewReferenceFile("../../conf/apisix-schema.json")
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}

	plugins := v1.Plugins{
		"echo": map[string]any{
			"body": -1,
		},
	}

	valid, err := validator.ValidateHTTPPluginSchema(plugins)
	assert.False(t, valid, "expected schema to be invalid, but it was valid")
	assert.Error(t, err, "failed to validate schema")
	assert.Contains(t, err.Error(), "body: Invalid type. Expected: string, given: integer")

	plugins = v1.Plugins{
		"echo": map[string]any{
			"body": "hello world",
		},
	}

	valid, err = validator.ValidateHTTPPluginSchema(plugins)
	assert.Nil(t, err, "failed to validate schema")
	assert.True(t, valid, "expected schema to be valid, but it was invalid")

	plugins = v1.Plugins{
		"echo": map[string]any{
			"body": "456",
		},
		"key-auth": map[string]any{
			"header": 123,
		},
	}

	valid, err = validator.ValidateHTTPPluginSchema(plugins)
	assert.False(t, valid, "expected schema to be invalid, but it was valid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "header: Invalid type. Expected: string, given: integer")

	plugins = v1.Plugins{
		"non-plugin": map[string]any{
			"body": "456",
		},
	}

	valid, err = validator.ValidateHTTPPluginSchema(plugins)
	assert.False(t, valid, "expected schema to be invalid, but it was valid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown plugin [non-plugin]")
}

func TestValidateSteamPluginSchema(t *testing.T) {
	validator, err := NewReferenceFile("../../conf/apisix-schema.json")
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}

	plugins := v1.Plugins{
		"echo": map[string]any{
			"body": -1,
		},
	}

	valid, err := validator.ValidateSteamPluginSchema(plugins)
	assert.False(t, valid, "expected schema to be invalid, but it was valid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown stream plugin [echo]")

	plugins = v1.Plugins{
		"mqtt-proxy": map[string]any{
			"protocol_name": "protol-name",
		},
	}

	valid, err = validator.ValidateSteamPluginSchema(plugins)
	assert.False(t, valid, "expected schema to be invalid, but it was valid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "protocol_level is required")

	plugins = v1.Plugins{
		"mqtt-proxy": map[string]any{
			"protocol_name":  "protol-name",
			"protocol_level": 4,
		},
	}

	valid, err = validator.ValidateSteamPluginSchema(plugins)
	assert.True(t, valid, "expected schema to be valid, but it was invalid")
	assert.Nil(t, err, "failed to validate schema")
}
