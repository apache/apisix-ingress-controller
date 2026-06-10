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

	"github.com/stretchr/testify/assert"
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
