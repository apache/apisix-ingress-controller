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
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

const (
	_enableCors       = AnnotationsPrefix + "enable-cors"
	_corsAllowOrigin  = AnnotationsPrefix + "cors-allow-origin"
	_corsAllowHeaders = AnnotationsPrefix + "cors-allow-headers"
	_corsAllowMethods = AnnotationsPrefix + "cors-allow-methods"
)

type cors struct{}

// NewCorsHandler creates a handler to convert annotations about
// CORS to APISIX cors plugin.
func NewCorsHandler() Handler {
	return &cors{}
}

func (c *cors) PluginName() string {
	return "cors"
}

func (c *cors) Handle(e Extractor) (interface{}, error) {
	if !e.GetBoolAnnotation(_enableCors) {
		return nil, nil
	}
	return &apisixv1.CorsConfig{
		AllowOrigins: e.GetStringAnnotation(_corsAllowOrigin),
		AllowMethods: e.GetStringAnnotation(_corsAllowMethods),
		AllowHeaders: e.GetStringAnnotation(_corsAllowHeaders),
	}, nil
}
