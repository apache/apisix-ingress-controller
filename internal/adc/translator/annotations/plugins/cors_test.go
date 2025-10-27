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

func TestCorsHandler(t *testing.T) {
	anno := map[string]string{
		annotations.AnnotationsEnableCors:       "true",
		annotations.AnnotationsCorsAllowHeaders: "abc,def",
		annotations.AnnotationsCorsAllowOrigin:  "https://a.com",
		annotations.AnnotationsCorsAllowMethods: "GET,HEAD",
	}
	p := NewCorsHandler()
	out, err := p.Handle(annotations.NewExtractor(anno))
	assert.Nil(t, err, "checking given error")
	config := out.(*adctypes.CorsConfig)
	assert.Equal(t, "abc,def", config.AllowHeaders)
	assert.Equal(t, "https://a.com", config.AllowOrigins)
	assert.Equal(t, "GET,HEAD", config.AllowMethods)

	assert.Equal(t, "cors", p.PluginName())

	anno[annotations.AnnotationsEnableCors] = "false"
	out, err = p.Handle(annotations.NewExtractor(anno))
	assert.Nil(t, err, "checking given error")
	assert.Nil(t, out, "checking given output")
}
