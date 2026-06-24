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
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
)

func TestHTTPADCExecutorCacheKey(t *testing.T) {
	e := &HTTPADCExecutor{}

	// No nonce set: cacheKey is the bare name.
	assert.Equal(t, "Gateway/ns/name", e.cacheKey("Gateway/ns/name"))

	// Nonce set: appended to the cacheKey.
	e.SetCacheKeyNonce("abc")
	assert.Equal(t, "Gateway/ns/name:abc", e.cacheKey("Gateway/ns/name"))

	// Empty nonce: ignored, falls back to the bare name.
	e.SetCacheKeyNonce("")
	assert.Equal(t, "Gateway/ns/name", e.cacheKey("Gateway/ns/name"))
}

func TestClientRefreshCacheKeyNonce(t *testing.T) {
	e := &HTTPADCExecutor{}
	c := &Client{executor: e}

	first := c.RefreshCacheKeyNonce()
	assert.NotEmpty(t, first)
	assert.Equal(t, "name:"+first, e.cacheKey("name"))

	// Re-acquiring leadership rotates the nonce to a new value.
	second := c.RefreshCacheKeyNonce()
	assert.NotEmpty(t, second)
	assert.NotEqual(t, first, second)
	assert.Equal(t, "name:"+second, e.cacheKey("name"))
}

// TestHTTPADCExecutorBuildHTTPRequestCacheKey verifies the nonce actually
// reaches the cacheKey field of the request body sent to the ADC server.
func TestHTTPADCExecutorBuildHTTPRequestCacheKey(t *testing.T) {
	e := &HTTPADCExecutor{
		serverURL: "http://127.0.0.1:3000",
		log:       logr.Discard(),
	}
	config := adctypes.Config{Name: "Gateway/ns/name"}

	cacheKeyOf := func(req *http.Request) string {
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		var parsed ADCServerRequest
		require.NoError(t, json.Unmarshal(body, &parsed))
		return parsed.Task.Opts.CacheKey
	}

	// Without a nonce the bare name is used.
	req, err := e.buildHTTPRequest(context.Background(), "http://apisix:9180", config, nil, nil, &adctypes.Resources{}, http.MethodPut, "/sync")
	require.NoError(t, err)
	assert.Equal(t, "Gateway/ns/name", cacheKeyOf(req))

	// With a nonce it is appended.
	e.SetCacheKeyNonce("abc")
	req, err = e.buildHTTPRequest(context.Background(), "http://apisix:9180", config, nil, nil, &adctypes.Resources{}, http.MethodPut, "/sync")
	require.NoError(t, err)
	assert.Equal(t, "Gateway/ns/name:abc", cacheKeyOf(req))
}
