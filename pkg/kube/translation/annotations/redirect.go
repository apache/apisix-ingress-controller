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
	"strconv"

	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

const (
	_httpToHttps           = AnnotationsPrefix + "http-to-https"
	_permanentRedirect     = AnnotationsPrefix + "permanent-redirect"
	_permanentRedirectCode = AnnotationsPrefix + "permanent-redirect-code"
)

type redirect struct{}

// NewRedirectHandler creates a handler to convert
// annotations about redirect control to APISIX redirect plugin.
func NewRedirectHandler() Handler {
	return &redirect{}
}

func (r *redirect) PluginName() string {
	return "redirect"
}

func (r *redirect) Handle(e Extractor) (interface{}, error) {
	var plugin apisixv1.RedirectConfig
	plugin.HttpToHttps = e.GetBoolAnnotation(_httpToHttps)
	plugin.URI = e.GetStringAnnotation(_permanentRedirect)
	// Transformation fail defaults to 0, the plugin will make default handling.
	plugin.RetCode, _ = strconv.Atoi(e.GetStringAnnotation(_permanentRedirectCode))
	// To avoid empty redirect plugin config, adding the check about the redirect.
	if plugin.HttpToHttps || plugin.URI != "" {
		return &plugin, nil
	}
	return nil, nil
}
