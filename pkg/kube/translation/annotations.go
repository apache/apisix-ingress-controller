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
	seven "github.com/apache/apisix-ingress-controller/pkg/seven/apisix"
	apisix "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

const (
	_whitelist        = "k8s.apisix.apache.org/whitelist-source-range"
	_enableCors       = "k8s.apisix.apache.org/enable-cors"
	_corsAllowOrigin  = "k8s.apisix.apache.org/cors-allow-origin"
	_corsAllowHeaders = "k8s.apisix.apache.org/cors-allow-headers"
	_corsAllowMethods = "k8s.apisix.apache.org/cors-allow-methods"
)

type cors struct {
	enable       bool
	allowOrigin  string
	allowHeaders string
	allowMethods string
}

func (t *translator) TranslateAnnotations(annotations map[string]string) apisix.Plugins {
	var c cors
	plugins := make(apisix.Plugins)
	for k, v := range annotations {
		switch {
		case k == _whitelist:
			ipRestriction := seven.BuildIpRestriction(&v, nil)
			plugins["ip-restriction"] = ipRestriction
		case k == _enableCors:
			if v == "true" {
				c.enable = true
			}
		case k == _corsAllowOrigin:
			c.allowOrigin = v
		case k == _corsAllowHeaders:
			c.allowHeaders = v
		case k == _corsAllowMethods:
			c.allowMethods = v
		}
	}
	if c.enable {
		maxAge := int64(3600)
		plugins["aispeech-cors"] = seven.BuildCors(true, &c.allowOrigin, &c.allowHeaders,
			&c.allowMethods, &maxAge)
	}
	return plugins
}
