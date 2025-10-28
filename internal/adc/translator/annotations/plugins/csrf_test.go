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

func TestCSRFHandler(t *testing.T) {
	anno := map[string]string{
		annotations.AnnotationsEnableCsrf: "true",
		annotations.AnnotationsCsrfKey:    "my-secret-key",
	}
	p := NewCSRFHandler()
	out, err := p.Handle(annotations.NewExtractor(anno))
	assert.Nil(t, err, "checking given error")
	config := out.(*adctypes.CSRFConfig)
	assert.Equal(t, "my-secret-key", config.Key)

	assert.Equal(t, "csrf", p.PluginName())

	// Test with enable-csrf set to false
	anno[annotations.AnnotationsEnableCsrf] = "false"
	out, err = p.Handle(annotations.NewExtractor(anno))
	assert.Nil(t, err, "checking given error")
	assert.Nil(t, out, "checking given output")

	// Test with enable-csrf true but no key
	anno[annotations.AnnotationsEnableCsrf] = "true"
	delete(anno, annotations.AnnotationsCsrfKey)
	out, err = p.Handle(annotations.NewExtractor(anno))
	assert.Nil(t, err, "checking given error")
	assert.Nil(t, out, "checking given output when key is missing")
}
