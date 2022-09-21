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
package plugins

import (
	"github.com/apache/apisix-ingress-controller/pkg/providers/ingress/translation/annotations"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

type cors struct{}

// NewCorsHandler creates a handler to convert annotations about
// CORS to APISIX cors plugin.
func NewCorsHandler() PluginAnnotationsHandler {
	return &cors{}
}

func (c *cors) PluginName() string {
	return "cors"
}

func (c *cors) Handle(e annotations.Extractor) (interface{}, error) {
	if !e.GetBoolAnnotation(annotations.AnnotationsEnableCors) {
		return nil, nil
	}
	return &apisixv1.CorsConfig{
		AllowOrigins: e.GetStringAnnotation(annotations.AnnotationsCorsAllowOrigin),
		AllowMethods: e.GetStringAnnotation(annotations.AnnotationsCorsAllowMethods),
		AllowHeaders: e.GetStringAnnotation(annotations.AnnotationsCorsAllowHeaders),
	}, nil
}
