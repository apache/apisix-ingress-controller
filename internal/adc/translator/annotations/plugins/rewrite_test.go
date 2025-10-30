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

func TestRewriteHandler(t *testing.T) {
	t.Run("rewrite target", func(t *testing.T) {
		anno := map[string]string{
			annotations.AnnotationsRewriteTarget: "/new-path",
		}
		p := NewRewriteHandler()
		out, err := p.Handle(annotations.NewExtractor(anno))
		assert.Nil(t, err, "checking given error")
		assert.NotNil(t, out, "checking given output")
		config := out.(*adctypes.RewriteConfig)
		assert.Equal(t, "/new-path", config.RewriteTarget)
		assert.Nil(t, config.RewriteTargetRegex)
		assert.Equal(t, "proxy-rewrite", p.PluginName())
	})

	t.Run("rewrite target with regex", func(t *testing.T) {
		anno := map[string]string{
			annotations.AnnotationsRewriteTargetRegex:         "/sample/(.*)",
			annotations.AnnotationsRewriteTargetRegexTemplate: "/$1",
		}
		p := NewRewriteHandler()
		out, err := p.Handle(annotations.NewExtractor(anno))
		assert.Nil(t, err, "checking given error")
		assert.NotNil(t, out, "checking given output")
		config := out.(*adctypes.RewriteConfig)
		assert.Equal(t, "", config.RewriteTarget)
		assert.NotNil(t, config.RewriteTargetRegex)
		assert.Equal(t, []string{"/sample/(.*)", "/$1"}, config.RewriteTargetRegex)
	})

	t.Run("invalid regex", func(t *testing.T) {
		anno := map[string]string{
			annotations.AnnotationsRewriteTargetRegex:         "[invalid(regex",
			annotations.AnnotationsRewriteTargetRegexTemplate: "/$1",
		}
		p := NewRewriteHandler()
		out, err := p.Handle(annotations.NewExtractor(anno))
		assert.NotNil(t, err, "checking given error")
		assert.Nil(t, out, "checking given output")
	})

	t.Run("no annotations", func(t *testing.T) {
		anno := map[string]string{}
		p := NewRewriteHandler()
		out, err := p.Handle(annotations.NewExtractor(anno))
		assert.Nil(t, err, "checking given error")
		assert.Nil(t, out, "checking given output")
	})

	t.Run("only regex without template", func(t *testing.T) {
		anno := map[string]string{
			annotations.AnnotationsRewriteTargetRegex: "/sample/(.*)",
		}
		p := NewRewriteHandler()
		out, err := p.Handle(annotations.NewExtractor(anno))
		assert.Nil(t, err, "checking given error")
		assert.NotNil(t, out, "checking given output")
		config := out.(*adctypes.RewriteConfig)
		assert.Nil(t, config.RewriteTargetRegex, "regex should not be set without template")
	})
}
