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
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/utils/ptr"
)

// The default is off: APISIX matches server_port against the port it accepted the
// connection on, which is not the port the Gateway listener declares, so injecting
// a predicate by default would 404 every route on a Service that maps 80 to 9080.
func TestNewDefaultConfigListenerPortMatchMode(t *testing.T) {
	cfg := NewDefaultConfig()
	assert.Equal(t, ListenerPortMatchModeOff, cfg.ListenerPortMatchMode)
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

func TestConfigValidateNamespaceSelector(t *testing.T) {
	tests := []struct {
		name      string
		selector  []string
		expectErr bool
	}{
		{name: "unset", selector: nil},
		{name: "1.x default", selector: []string{""}},
		{name: "equality", selector: []string{"team=a"}},
		{name: "set based", selector: []string{"env in (prod,staging),!legacy", "team=a"}},
		{name: "invalid", selector: []string{"team in a"}, expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewDefaultConfig()
			cfg.NamespaceSelector = tt.selector

			err := cfg.Validate()
			if tt.expectErr {
				assert.ErrorContains(t, err, "invalid namespace_selector")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseNamespaceSelector(t *testing.T) {
	nsLabels := labels.Set{"version": "v1", "env": "prod"}

	tests := []struct {
		name    string
		entries []string
		// nil means the selector is disabled.
		matches *bool
	}{
		// Cases ported from TestMultiValueLabelsIsSubsetOf of 1.x.
		{name: "no entry", entries: nil},
		{name: "1.x default", entries: []string{""}},
		{name: "single value", entries: []string{"env=prod"}, matches: ptr.To(true)},
		{name: "values on one key are ORed", entries: []string{"env=qa", "env=prod"}, matches: ptr.To(true)},
		{name: "value mismatch", entries: []string{"env=qa"}, matches: ptr.To(false)},
		{name: "missing key", entries: []string{"env3=not"}, matches: ptr.To(false)},
		// Entries on different keys are ANDed.
		{name: "all keys match", entries: []string{"env=prod", "version=v1"}, matches: ptr.To(true)},
		{name: "one key mismatches", entries: []string{"env=prod", "version=v2"}, matches: ptr.To(false)},
		{name: "empty entry is ignored", entries: []string{"env=qa", ""}, matches: ptr.To(false)},
		// Full selector syntax on top of 1.x.
		{name: "in merges with equality", entries: []string{"env in (qa)", "env==prod"}, matches: ptr.To(true)},
		{name: "not equal", entries: []string{"env=prod", "version!=v1"}, matches: ptr.To(false)},
		{name: "does not exist", entries: []string{"!legacy"}, matches: ptr.To(true)},
		// Only separate entries are merged, one entry keeps the standard semantics.
		{name: "one entry is not merged", entries: []string{"env=qa,env=prod"}, matches: ptr.To(false)},
		{name: "one entry with several keys", entries: []string{"env=prod,version=v1"}, matches: ptr.To(true)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector, err := ParseNamespaceSelector(tt.entries)
			require.NoError(t, err)
			if tt.matches == nil {
				assert.Nil(t, selector)
				return
			}
			require.NotNil(t, selector)
			assert.Equal(t, *tt.matches, selector.Matches(nsLabels), selector.String())
		})
	}
}
