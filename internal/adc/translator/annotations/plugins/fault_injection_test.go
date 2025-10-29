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

package plugins

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/internal/adc/translator/annotations"
)

func TestFaultInjectionHttpAllowMethods(t *testing.T) {
	handler := NewFaultInjectionHandler()
	assert.Equal(t, "fault-injection", handler.PluginName())

	extractor := annotations.NewExtractor(map[string]string{
		annotations.AnnotationsHttpAllowMethods: "GET,POST",
	})

	plugin, err := handler.Handle(extractor)
	assert.NoError(t, err)
	assert.NotNil(t, plugin)

	data, err := json.Marshal(plugin)
	assert.NoError(t, err)
	assert.JSONEq(t, `{"abort":{"http_status":405,"vars":[[["request_method","!","in",["GET","POST"]]]]}}`, string(data))
}

func TestFaultInjectionHttpBlockMethods(t *testing.T) {
	handler := NewFaultInjectionHandler()
	assert.Equal(t, "fault-injection", handler.PluginName())

	extractor := annotations.NewExtractor(map[string]string{
		annotations.AnnotationsHttpBlockMethods: "GET,POST",
	})

	plugin, err := handler.Handle(extractor)
	assert.NoError(t, err)
	assert.NotNil(t, plugin)

	data, err := json.Marshal(plugin)
	assert.NoError(t, err)
	assert.JSONEq(t, `{"abort":{"http_status":405,"vars":[[["request_method","in",["GET","POST"]]]]}}`, string(data))
}
