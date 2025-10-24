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

package annotations

import (
	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
)

type corsParser struct{}

// NewCorsParser creates a parser to convert CORS annotations
// to APISIX cors plugin configuration.
func NewCorsParser() IngressAnnotationsParser {
	return &corsParser{}
}

func (c *corsParser) Parse(e Extractor) (any, error) {
	if !e.GetBoolAnnotation(AnnotationsEnableCors) {
		return nil, nil
	}

	corsConfig := &adctypes.CorsConfig{
		AllowOrigins: e.GetStringAnnotation(AnnotationsCorsAllowOrigin),
		AllowMethods: e.GetStringAnnotation(AnnotationsCorsAllowMethods),
		AllowHeaders: e.GetStringAnnotation(AnnotationsCorsAllowHeaders),
	}

	// Return nil if all fields are empty (only enable-cors is set)
	if corsConfig.AllowOrigins == "" && corsConfig.AllowMethods == "" && corsConfig.AllowHeaders == "" {
		// Use default CORS config with just enable flag
		return corsConfig, nil
	}

	return corsConfig, nil
}
