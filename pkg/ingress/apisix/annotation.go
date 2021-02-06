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
package apisix

import (
	"strconv"

	seven "github.com/apache/apisix-ingress-controller/pkg/seven/apisix"
	apisix "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

// BuildAnnotation return plugins and group
func BuildAnnotation(annotations map[string]string) (apisix.Plugins, string) {
	plugins := make(apisix.Plugins)
	cors := &CorsYaml{}
	// ingress.class
	group := ""
	for k, v := range annotations {
		switch {
		case k == SSLREDIRECT:
			if b, err := strconv.ParseBool(v); err == nil && b {
				// todo add ssl-redirect plugin
			}
		case k == WHITELIST:
			ipRestriction := seven.BuildIpRestriction(&v, nil)
			plugins["ip-restriction"] = ipRestriction
		case k == ENABLE_CORS:
			cors.SetEnable(v)
		case k == CORS_ALLOW_ORIGIN:
			cors.SetOrigin(v)
		case k == CORS_ALLOW_HEADERS:
			cors.SetHeaders(v)
		case k == CORS_ALLOW_METHODS:
			cors.SetMethods(v)
		case k == INGRESS_CLASS:
			group = v
		default:
			// do nothing
		}
	}
	// build CORS plugin
	if cors.Enable {
		plugins["aispeech-cors"] = cors.Build()
	}
	return plugins, group
}
