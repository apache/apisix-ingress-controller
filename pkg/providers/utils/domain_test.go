// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHostnameMatch(t *testing.T) {
	normalHostname := "sub.sample.com"
	validHostnames := []string{"", "sub.sample.com", "*.sample.com"}
	invalidHostnames := []string{"a.sample.com", "sample.com", "a.sub.sample.com"}
	for _, v := range validHostnames {
		assert.True(t, IsHostnameMatch(normalHostname, v))
	}
	for _, i := range invalidHostnames {
		assert.False(t, IsHostnameMatch(normalHostname, i))
	}

	wildcardHostname := "*.sample.com"
	validHostnames = []string{"", "sub.sample.com", "a.sub.sample.com", "*.sample.com"}
	invalidHostnames = []string{"sample.com", "*.example.com"}
	for _, v := range validHostnames {
		assert.True(t, IsHostnameMatch(wildcardHostname, v))
	}
	for _, i := range invalidHostnames {
		assert.False(t, IsHostnameMatch(wildcardHostname, i))
	}
}
