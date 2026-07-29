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
	"bytes"
	"testing"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/types"
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

const (
	secretTLSKey   = "-----BEGIN-PRIVATE-KEY-SUPER-SECRET-----"
	secretCredKey  = "SUPER-SECRET-KEYAUTH-KEY"
	secretAdminKey = "SUPER-SECRET-ADMINKEY"
)

func secretResources() *adctypes.Resources {
	return &adctypes.Resources{
		SSLs: []*adctypes.SSL{{
			Metadata:     adctypes.Metadata{Name: "ssl-1"},
			Certificates: []adctypes.Certificate{{Certificate: "cert", Key: secretTLSKey}},
		}},
		Consumers: []*adctypes.Consumer{{
			Username: "alice",
			Plugins:  adctypes.Plugins{"key-auth": map[string]any{"key": secretCredKey}},
		}},
	}
}

// FINDING-016/037: logging a Task must not leak TLS private keys or consumer
// credentials, while still emitting identity for debugging.
func TestTaskMarshalLogRedactsSecrets(t *testing.T) {
	var buf bytes.Buffer
	log := bufferLogger(&buf)

	task := Task{
		Key:  types.NamespacedNameKind{Namespace: "ns", Name: "route-1", Kind: "ApisixRoute"},
		Name: "ns/route-1",
		Configs: map[types.NamespacedNameKind]adctypes.Config{
			{}: {Name: "gw", Token: secretAdminKey, ServerAddrs: []string{"http://x"}},
		},
		ResourceTypes: []string{"ssl", "consumer"},
		Resources:     secretResources(),
	}
	log.Error(assert.AnError, "store insert failed", "args", task)

	out := buf.String()
	assert.NotContains(t, out, secretTLSKey, "TLS private key leaked")
	assert.NotContains(t, out, secretCredKey, "consumer credential leaked")
	assert.NotContains(t, out, secretAdminKey, "admin key leaked")
	assert.Contains(t, out, "route-1", "identity should still be logged")
}

// FINDING-023: logging the ADC request body must redact the AdminKey token and
// the secret-bearing config, without altering what is sent to the server.
func TestADCServerRequestMarshalLogRedactsToken(t *testing.T) {
	var buf bytes.Buffer
	log := bufferLogger(&buf)

	reqBody := ADCServerRequest{
		Task: ADCServerTask{
			Opts: ADCServerOpts{
				Backend:  "apisix",
				Server:   []string{"http://x"},
				Token:    secretAdminKey,
				CacheKey: "gw",
			},
			Config: *secretResources(),
		},
	}
	log.V(1).Info("prepared request body", "body", reqBody)

	out := buf.String()
	assert.NotContains(t, out, secretAdminKey, "admin key leaked")
	assert.NotContains(t, out, secretTLSKey, "TLS private key leaked")
	assert.NotContains(t, out, secretCredKey, "consumer credential leaked")
	assert.Contains(t, out, "[REDACTED]", "token should be redacted")

	// The real wire payload must be untouched.
	require.Equal(t, secretAdminKey, reqBody.Task.Opts.Token)
}
