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

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/adc/translator/annotations"
)

type responseRewrite struct{}

// NewResponseRewriteHandler creates a handler to convert annotations about
// ResponseRewrite to APISIX response-rewrite plugin.
func NewResponseRewriteHandler() PluginAnnotationsHandler {
	return &responseRewrite{}
}

func (r *responseRewrite) PluginName() string {
	return "response-rewrite"
}

func (r *responseRewrite) Handle(e annotations.Extractor) (any, error) {
	if !e.GetBoolAnnotation(annotations.AnnotationsEnableResponseRewrite) {
		return nil, nil
	}

	plugin := &adctypes.ResponseRewriteConfig{
		BodyBase64: e.GetBoolAnnotation(annotations.AnnotationsResponseRewriteBodyBase64),
		Body:       e.GetStringAnnotation(annotations.AnnotationsResponseRewriteBody),
	}

	// Parse status code, transformation fail defaults to 0
	if statusCodeStr := e.GetStringAnnotation(annotations.AnnotationsResponseRewriteStatusCode); statusCodeStr != "" {
		if statusCode, err := strconv.Atoi(statusCodeStr); err == nil {
			plugin.StatusCode = statusCode
		}
	}

	// Handle headers
	addHeaders := e.GetStringsAnnotation(annotations.AnnotationsResponseRewriteHeaderAdd)
	setHeaders := e.GetStringsAnnotation(annotations.AnnotationsResponseRewriteHeaderSet)
	removeHeaders := e.GetStringsAnnotation(annotations.AnnotationsResponseRewriteHeaderRemove)

	if len(addHeaders) > 0 || len(setHeaders) > 0 || len(removeHeaders) > 0 {
		headers := &adctypes.ResponseHeaders{
			Add:    addHeaders,
			Remove: removeHeaders,
		}

		// Convert set headers from ["key:value", ...] to map[string]string
		if len(setHeaders) > 0 {
			headers.Set = make(map[string]string)
			for _, header := range setHeaders {
				if key, value, found := parseHeaderKeyValue(header); found {
					headers.Set[key] = value
				}
			}
		}

		plugin.Headers = headers
	}

	return plugin, nil
}

// parseHeaderKeyValue parses a header string in format "key:value" and returns key, value and success flag
func parseHeaderKeyValue(header string) (string, string, bool) {
	for i := 0; i < len(header); i++ {
		if header[i] == ':' {
			return header[:i], header[i+1:], true
		}
	}
	return "", "", false
}
