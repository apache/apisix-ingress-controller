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
	"net/http"
	"strconv"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/adc/translator/annotations"
)

type redirect struct{}

// NewRedirectHandler creates a handler to convert
// annotations about redirect control to APISIX redirect plugin.
func NewRedirectHandler() PluginAnnotationsHandler {
	return &redirect{}
}

func (r *redirect) PluginName() string {
	return "redirect"
}

func (r *redirect) Handle(e annotations.Extractor) (any, error) {
	var plugin adctypes.RedirectConfig
	plugin.HttpToHttps = e.GetBoolAnnotation(annotations.AnnotationsHttpToHttps)
	// To avoid empty redirect plugin config, adding the check about the redirect.
	if plugin.HttpToHttps {
		return &plugin, nil
	}
	if uri := e.GetStringAnnotation(annotations.AnnotationsHttpRedirect); uri != "" {
		// Transformation fail defaults to 0.
		plugin.RetCode, _ = strconv.Atoi(e.GetStringAnnotation(annotations.AnnotationsHttpRedirectCode))
		plugin.URI = uri
		// Default is http.StatusMovedPermanently, the allowed value is between http.StatusMultipleChoices and http.StatusPermanentRedirect.
		if plugin.RetCode < http.StatusMovedPermanently || plugin.RetCode > http.StatusPermanentRedirect {
			plugin.RetCode = http.StatusMovedPermanently
		}
		return &plugin, nil
	}
	return nil, nil
}
