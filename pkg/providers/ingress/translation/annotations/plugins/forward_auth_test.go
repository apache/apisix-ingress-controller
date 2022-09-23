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

	"github.com/apache/apisix-ingress-controller/pkg/providers/ingress/translation/annotations"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

func TestForwardAuthHandler(t *testing.T) {
	anno := map[string]string{
		annotations.AnnotationsForwardAuthURI:             "http://127.0.0.1:9080",
		annotations.AnnotationsForwardAuthRequestHeaders:  "Authorization",
		annotations.AnnotationsForwardAuthClientHeaders:   "Location",
		annotations.AnnotationsForwardAuthUpstreamHeaders: "X-User-ID",
	}
	p := NewForwardAuthHandler()
	out, err := p.Handle(annotations.NewExtractor(anno))
	assert.Nil(t, err, "checking given error")
	config := out.(*apisixv1.ForwardAuthConfig)
	assert.Equal(t, "http://127.0.0.1:9080", config.URI)
	assert.Equal(t, []string{"Authorization"}, config.RequestHeaders)
	assert.Equal(t, []string{"Location"}, config.ClientHeaders)
	assert.Equal(t, []string{"X-User-ID"}, config.UpstreamHeaders)
	assert.Equal(t, true, config.SSLVerify)
	assert.Equal(t, "forward-auth", p.PluginName())

	anno[annotations.AnnotationsForwardAuthSSLVerify] = "false"
	out, err = p.Handle(annotations.NewExtractor(anno))
	assert.Nil(t, err, "checking given error")
	config = out.(*apisixv1.ForwardAuthConfig)
	assert.Equal(t, false, config.SSLVerify)

	anno[annotations.AnnotationsForwardAuthURI] = ""
	out, err = p.Handle(annotations.NewExtractor(anno))
	assert.Nil(t, err, "checking given error")
	assert.Nil(t, out, "checking given output")
}
