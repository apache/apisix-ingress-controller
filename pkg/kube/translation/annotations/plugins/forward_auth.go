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
	"github.com/apache/apisix-ingress-controller/pkg/kube/translation/annotations"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

const (
	_forwardAuthURI             = annotations.AnnotationsPrefix + "auth-uri"
	_forwardAuthSSLVerify       = annotations.AnnotationsPrefix + "auth-ssl-verify"
	_forwardAuthRequestHeaders  = annotations.AnnotationsPrefix + "auth-request-headers"
	_forwardAuthUpstreamHeaders = annotations.AnnotationsPrefix + "auth-upstream-headers"
	_forwardAuthClientHeaders   = annotations.AnnotationsPrefix + "auth-client-headers"
)

type forwardAuth struct{}

// NewForwardAuthHandler creates a handler to convert
// annotations about forward authentication to APISIX forward-auth plugin.
func NewForwardAuthHandler() PluginHandler {
	return &forwardAuth{}
}

func (i *forwardAuth) PluginName() string {
	return "forward-auth"
}

func (i *forwardAuth) Handle(ing *annotations.Ingress) (interface{}, error) {
	uri := annotations.GetStringAnnotation(_forwardAuthURI, ing)
	sslVerify := true
	if annotations.GetStringAnnotation(_forwardAuthSSLVerify, ing) == "false" {
		sslVerify = false
	}
	if len(uri) > 0 {
		return &apisixv1.ForwardAuthConfig{
			URI:             uri,
			SSLVerify:       sslVerify,
			RequestHeaders:  annotations.GetStringsAnnotation(_forwardAuthRequestHeaders, ing),
			UpstreamHeaders: annotations.GetStringsAnnotation(_forwardAuthUpstreamHeaders, ing),
			ClientHeaders:   annotations.GetStringsAnnotation(_forwardAuthClientHeaders, ing),
		}, nil
	}

	return nil, nil
}
