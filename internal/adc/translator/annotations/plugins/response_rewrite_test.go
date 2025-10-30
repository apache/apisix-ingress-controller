// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package plugins

import (
	"testing"

	"github.com/stretchr/testify/assert"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/adc/translator/annotations"
)

func TestResponseRewriteHandler(t *testing.T) {
	anno := map[string]string{
		annotations.AnnotationsEnableResponseRewrite:       "true",
		annotations.AnnotationsResponseRewriteStatusCode:   "200",
		annotations.AnnotationsResponseRewriteBody:         "bar_body",
		annotations.AnnotationsResponseRewriteBodyBase64:   "false",
		annotations.AnnotationsResponseRewriteHeaderAdd:    "testkey1:testval1,testkey2:testval2",
		annotations.AnnotationsResponseRewriteHeaderRemove: "testkey1,testkey2",
		annotations.AnnotationsResponseRewriteHeaderSet:    "testkey1:testval1,testkey2:testval2",
	}
	p := NewResponseRewriteHandler()
	out, err := p.Handle(annotations.NewExtractor(anno))
	assert.Nil(t, err, "checking given error")
	config := out.(*adctypes.ResponseRewriteConfig)
	assert.Equal(t, 200, config.StatusCode)
	assert.Equal(t, "bar_body", config.Body)
	assert.Equal(t, false, config.BodyBase64)
	assert.Equal(t, "response-rewrite", p.PluginName())
	assert.Equal(t, []string{"testkey1:testval1", "testkey2:testval2"}, config.Headers.Add)
	assert.Equal(t, []string{"testkey1", "testkey2"}, config.Headers.Remove)
	assert.Equal(t, map[string]string{
		"testkey1": "testval1",
		"testkey2": "testval2",
	}, config.Headers.Set)
}

func TestResponseRewriteHandlerDisabled(t *testing.T) {
	anno := map[string]string{
		annotations.AnnotationsEnableResponseRewrite:     "false",
		annotations.AnnotationsResponseRewriteStatusCode: "400",
		annotations.AnnotationsResponseRewriteBody:       "bar_body",
	}
	p := NewResponseRewriteHandler()
	out, err := p.Handle(annotations.NewExtractor(anno))
	assert.Nil(t, err, "checking given error")
	assert.Nil(t, out, "checking given output")
}

func TestResponseRewriteHandlerBase64(t *testing.T) {
	anno := map[string]string{
		annotations.AnnotationsEnableResponseRewrite:     "true",
		annotations.AnnotationsResponseRewriteBody:       "YmFyLWJvZHk=",
		annotations.AnnotationsResponseRewriteBodyBase64: "true",
	}
	p := NewResponseRewriteHandler()
	out, err := p.Handle(annotations.NewExtractor(anno))
	assert.Nil(t, err, "checking given error")
	config := out.(*adctypes.ResponseRewriteConfig)
	assert.Equal(t, "YmFyLWJvZHk=", config.Body)
	assert.Equal(t, true, config.BodyBase64)
}

func TestResponseRewriteHandlerInvalidStatusCode(t *testing.T) {
	anno := map[string]string{
		annotations.AnnotationsEnableResponseRewrite:     "true",
		annotations.AnnotationsResponseRewriteStatusCode: "invalid",
		annotations.AnnotationsResponseRewriteBody:       "bar_body",
	}
	p := NewResponseRewriteHandler()
	out, err := p.Handle(annotations.NewExtractor(anno))
	assert.Nil(t, err, "checking given error")
	config := out.(*adctypes.ResponseRewriteConfig)
	assert.Equal(t, 0, config.StatusCode, "invalid status code should default to 0")
	assert.Equal(t, "bar_body", config.Body)
}

func TestParseHeaderKeyValue(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantKey   string
		wantValue string
		wantFound bool
	}{
		{
			name:      "valid header",
			input:     "Content-Type:application/json",
			wantKey:   "Content-Type",
			wantValue: "application/json",
			wantFound: true,
		},
		{
			name:      "header with colon in value",
			input:     "X-Custom:value:with:colons",
			wantKey:   "X-Custom",
			wantValue: "value:with:colons",
			wantFound: true,
		},
		{
			name:      "invalid header without colon",
			input:     "InvalidHeader",
			wantKey:   "",
			wantValue: "",
			wantFound: false,
		},
		{
			name:      "empty value",
			input:     "X-Empty:",
			wantKey:   "X-Empty",
			wantValue: "",
			wantFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, value, found := parseHeaderKeyValue(tt.input)
			assert.Equal(t, tt.wantKey, key)
			assert.Equal(t, tt.wantValue, value)
			assert.Equal(t, tt.wantFound, found)
		})
	}
}
