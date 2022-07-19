// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package translation

import (
	"testing"

	"github.com/apache/apisix-ingress-controller/pkg/kube/translation/annotations"
	apisix "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
	"github.com/stretchr/testify/assert"
)

func TestAnnotationsPlugins(t *testing.T) {
	anno := map[string]string{
		annotations.AnnotationsAuthType: "basicAuth",
	}

	ingress := (&translator{}).translateAnnotations(anno)
	assert.Len(t, ingress.Plugins, 1)
	assert.Equal(t, apisix.Plugins{
		"basic-auth": &apisix.BasicAuthConfig{},
	}, ingress.Plugins)

	anno[annotations.AnnotationsEnableCsrf] = "true"
	anno[annotations.AnnotationsCsrfKey] = "csrf-key"
	ingress = (&translator{}).translateAnnotations(anno)
	assert.Len(t, ingress.Plugins, 2)
	assert.Equal(t, apisix.Plugins{
		"basic-auth": &apisix.BasicAuthConfig{},
		"csrf": &apisix.CSRFConfig{
			Key: "csrf-key",
		},
	}, ingress.Plugins)
}

func TestAnnotationsPluginConfigName(t *testing.T) {
	anno := map[string]string{
		annotations.AnnotationsPluginConfigName: "plugin-config-echo",
	}

	ingress := (&translator{}).translateAnnotations(anno)
	assert.Equal(t, "plugin-config-echo", ingress.PluginConfigName)
}

func TestAnnotationsEnableWebSocket(t *testing.T) {
	anno := map[string]string{
		annotations.AnnotationsEnableWebSocket: "true",
	}

	ingress := (&translator{}).translateAnnotations(anno)
	assert.Equal(t, true, ingress.EnableWebSocket)
}

func TestAnnotationsUseRegex(t *testing.T) {
	anno := map[string]string{
		annotations.AnnotationsUseRegex: "true",
	}

	ingress := (&translator{}).translateAnnotations(anno)
	assert.Equal(t, true, ingress.UseRegex)
}
