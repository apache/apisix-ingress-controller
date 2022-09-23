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
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

const (
	_forwardAuthURI             = AnnotationsPrefix + "auth-uri"
	_forwardAuthSSLVerify       = AnnotationsPrefix + "auth-ssl-verify"
	_forwardAuthRequestHeaders  = AnnotationsPrefix + "auth-request-headers"
	_forwardAuthUpstreamHeaders = AnnotationsPrefix + "auth-upstream-headers"
	_forwardAuthClientHeaders   = AnnotationsPrefix + "auth-client-headers"
)

type forwardAuth struct{}

// NewForwardAuthHandler creates a handler to convert
// annotations about forward authentication to APISIX forward-auth plugin.
func NewForwardAuthHandler() Handler {
	return &forwardAuth{}
}

func (i *forwardAuth) PluginName() string {
	return "forward-auth"
}

func (i *forwardAuth) Handle(e Extractor) (interface{}, error) {
	uri := e.GetStringAnnotation(_forwardAuthURI)
	sslVerify := true
	if e.GetStringAnnotation(_forwardAuthSSLVerify) == "false" {
		sslVerify = false
	}
	if len(uri) > 0 {
		return &apisixv1.ForwardAuthConfig{
			URI:             uri,
			SSLVerify:       sslVerify,
			RequestHeaders:  e.GetStringsAnnotation(_forwardAuthRequestHeaders),
			UpstreamHeaders: e.GetStringsAnnotation(_forwardAuthUpstreamHeaders),
			ClientHeaders:   e.GetStringsAnnotation(_forwardAuthClientHeaders),
		}, nil
	}

	return nil, nil
}
