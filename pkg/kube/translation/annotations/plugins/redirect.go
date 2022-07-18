// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the Licensannotations.  You may obtain a copy of the License at
//
//     http://www.apachannotations.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the Licensannotations.
package plugins

import (
	"net/http"
	"strconv"

	"github.com/apache/apisix-ingress-controller/pkg/kube/translation/annotations"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

const (
	_httpToHttps      = annotations.AnnotationsPrefix + "http-to-https"
	_httpRedirect     = annotations.AnnotationsPrefix + "http-redirect"
	_httpRedirectCode = annotations.AnnotationsPrefix + "http-redirect-code"
)

type redirect struct{}

// NewRedirectHandler creates a handler to convert
// annotations about redirect control to APISIX redirect plugin.
func NewRedirectHandler() PluginHandler {
	return &redirect{}
}

func (r *redirect) PluginName() string {
	return "redirect"
}

func (r *redirect) Handle(ing *annotations.Ingress) (interface{}, error) {
	var plugin apisixv1.RedirectConfig
	plugin.HttpToHttps = annotations.GetBoolAnnotation(_httpToHttps, ing)
	plugin.URI = annotations.GetStringAnnotation(_httpRedirect, ing)
	// Transformation fail defaults to 0.
	plugin.RetCode, _ = strconv.Atoi(annotations.GetStringAnnotation(_httpRedirectCode, ing))
	// To avoid empty redirect plugin config, adding the check about the redirect.
	if plugin.HttpToHttps {
		return &plugin, nil
	}
	if plugin.URI != "" {
		// Default is http.StatusMovedPermanently, the allowed value is between http.StatusMultipleChoices and http.StatusPermanentRedirect.
		if plugin.RetCode < http.StatusMovedPermanently || plugin.RetCode > http.StatusPermanentRedirect {
			plugin.RetCode = http.StatusMovedPermanently
		}
		return &plugin, nil
	}
	return nil, nil
}
