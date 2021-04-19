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

// CorsPlugin is the cors plugin.
type CorsPlugin struct {
	Origins string `json:"origins,omitempty"`
	Headers string `json:"headers,omitempty"`
	Methods string `json:"methods,omitempty"`
	MaxAge  int64  `json:"max_age,omitempty"`
}

var (
	_enableCors       = "k8s.apisix.apache.org/enable-cors"
	_corsAllowOrigin  = "k8s.apisix.apache.org/cors-allow-origin"
	_corsAllowHeaders = "k8s.apisix.apache.org/cors-allow-headers"
	_corsAllowMethods = "k8s.apisix.apache.org/cors-allow-methods"
)

// BuildCorsPlugin build the cors plugin config body.
func BuildCorsPlugin(annotations map[string]string) *CorsPlugin {
	enable, ok := annotations[_enableCors]
	if !ok || enable == "false" {
		return nil
	}

	var cors CorsPlugin
	if ao, ok := annotations[_corsAllowOrigin]; ok {
		cors.Origins = ao
	}
	if ah, ok := annotations[_corsAllowHeaders]; ok {
		cors.Headers = ah
	}
	if am, ok := annotations[_corsAllowMethods]; ok {
		cors.Methods = am
	}
	return &cors
}
