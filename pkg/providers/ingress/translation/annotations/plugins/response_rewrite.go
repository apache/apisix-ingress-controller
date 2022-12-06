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
	"strconv"

	"github.com/apache/apisix-ingress-controller/pkg/providers/ingress/translation/annotations"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

type responseRewrite struct{}

// NewResponseRewriteHandler creates a handler to convert annotations about
// ResponseRewrite to APISIX response-rewrite plugin.
func NewResponseRewriteHandler() PluginAnnotationsHandler {
	return &responseRewrite{}
}

func (c *responseRewrite) PluginName() string {
	return "response-rewrite"
}

func (c *responseRewrite) Handle(e annotations.Extractor) (interface{}, error) {
	if !e.GetBoolAnnotation(annotations.AnnotationsEnableResponseRewrite) {
		return nil, nil
	}
	var plugin apisixv1.ResponseRewriteConfig
	// Transformation fail defaults to 0.
	plugin.StatusCode, _ = strconv.Atoi(e.GetStringAnnotation(annotations.AnnotationsResponseRewriteStatusCode))
	plugin.Body = e.GetStringAnnotation(annotations.AnnotationsResponseRewriteBody)
	plugin.BodyBase64 = e.GetBoolAnnotation(annotations.AnnotationsResponseRewriteBodyBase64)
	return &plugin, nil
}
