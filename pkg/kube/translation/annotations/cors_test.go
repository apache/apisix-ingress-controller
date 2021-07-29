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
package annotations

import (
	"testing"

	"github.com/stretchr/testify/assert"

	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

func TestCorsHandler(t *testing.T) {
	annotations := map[string]string{
		_enableCors:       "true",
		_corsAllowHeaders: "abc,def",
		_corsAllowOrigin:  "https://a.com",
		_corsAllowMethods: "GET,HEAD",
	}
	p := NewCorsHandler()
	out, err := p.Handle(NewExtractor(annotations))
	assert.Nil(t, err, "checking given error")
	config := out.(*apisixv1.CorsConfig)
	assert.Equal(t, config.AllowHeaders, "abc,def")
	assert.Equal(t, config.AllowOrigins, "https://a.com")
	assert.Equal(t, config.AllowMethods, "GET,HEAD")

	assert.Equal(t, p.PluginName(), "cors")

	annotations[_enableCors] = "false"
	out, err = p.Handle(NewExtractor(annotations))
	assert.Nil(t, err, "checking given error")
	assert.Nil(t, out, "checking given output")
}
