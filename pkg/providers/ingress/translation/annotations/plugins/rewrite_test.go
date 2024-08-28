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

func TestRewriteHandler(t *testing.T) {
	anno := map[string]string{
		annotations.AnnotationsRewriteTarget:              "/sample",
		annotations.AnnotationsRewriteTargetRegex:         "/sample/(.*)",
		annotations.AnnotationsRewriteTargetRegexTemplate: "/$1",
		annotations.AnnotationsRewriteHeaderAdd:           "testkey1:testval1,testkey2:testval2",
		annotations.AnnotationsRewriteHeaderRemove:        "testkey1,testkey2",
		annotations.AnnotationsRewriteHeaderSet:           "testsetkey1:testsetval1,testsetkey2:testsetval2",
	}
	p := NewRewriteHandler()
	out, err := p.Handle(annotations.NewExtractor(anno))
	assert.Nil(t, err, "checking given error")
	config := out.(*apisixv1.RewriteConfig)
	assert.Equal(t, "/sample", config.RewriteTarget)
	assert.Equal(t, []string{"/sample/(.*)", "/$1"}, config.RewriteTargetRegex)
	assert.Equal(t, "proxy-rewrite", p.PluginName())
	assert.Equal(t, map[string]string{
		"testkey1": "testval1",
		"testkey2": "testval2",
	}, config.Headers.GetAddHeaders())
	assert.Equal(t, []string{"testkey1", "testkey2"}, config.Headers.GetRemovedHeaders())
	assert.Equal(t, map[string]string{
		"testsetkey1": "testsetval1",
		"testsetkey2": "testsetval2",
	}, config.Headers.GetSetHeaders())
}
