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

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDefaultConfigListenerPortMatchMode(t *testing.T) {
	cfg := NewDefaultConfig()
	assert.Equal(t, ListenerPortMatchModeAuto, cfg.ListenerPortMatchMode)
}

func TestConfigValidateListenerPortMatchMode(t *testing.T) {
	tests := []struct {
		name      string
		mode      ListenerPortMatchMode
		expectErr bool
	}{
		{
			name:      "default auto",
			mode:      ListenerPortMatchModeAuto,
			expectErr: false,
		},
		{
			name:      "explicit",
			mode:      ListenerPortMatchModeExplicit,
			expectErr: false,
		},
		{
			name:      "off",
			mode:      ListenerPortMatchModeOff,
			expectErr: false,
		},
		{
			name:      "empty mode is allowed",
			mode:      "",
			expectErr: false,
		},
		{
			name:      "invalid mode",
			mode:      "invalid",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewDefaultConfig()
			cfg.ListenerPortMatchMode = tt.mode

			err := cfg.Validate()
			if tt.expectErr {
				assert.Error(t, err)
				assert.ErrorContains(t, err, "invalid listener_port_match_mode")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
