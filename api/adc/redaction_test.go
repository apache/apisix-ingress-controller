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

package adc

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// bufferLogger builds a logger identical to the production one (zapr + zap
// console encoder) but writing into buf, so we can assert on real log output.
func bufferLogger(buf *bytes.Buffer) logr.Logger {
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
		zapcore.AddSync(buf),
		zapcore.DebugLevel,
	)
	return zapr.NewLogger(zap.New(core))
}

const secretPluginValue = "SUPER-SECRET-KAFKA-PASSWORD"

func secretPluginMap() map[string]any {
	return map[string]any{
		"kafka-logger": map[string]any{"sasl_config": map[string]any{"password": secretPluginValue}},
		"http-logger":  map[string]any{"uri": "http://logs.example"},
	}
}

// Plugin config is arbitrary user JSON that routinely carries credentials, so
// no plugin map may reach the log sink whole.
func TestPluginMapsMarshalLogEmitNamesOnly(t *testing.T) {
	for name, value := range map[string]any{
		"plugins":        Plugins(secretPluginMap()),
		"globalRules":    GlobalRule(secretPluginMap()),
		"pluginMetadata": PluginMetadata(secretPluginMap()),
	} {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			bufferLogger(&buf).V(1).Info("site", name, value)
			out := buf.String()

			assert.NotContains(t, out, secretPluginValue, "plugin config leaked into logs")
			assert.Contains(t, out, "kafka-logger", "plugin name should survive for debugging")
			assert.Contains(t, out, "http-logger")
		})
	}
}

// MarshalLog must not change what goes on the wire to the data plane.
func TestPluginMapsMarshalJSONUnaffected(t *testing.T) {
	b, err := json.Marshal(Plugins(secretPluginMap()))
	require.NoError(t, err)
	assert.Contains(t, string(b), secretPluginValue)
}
