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
	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/adc/translator/annotations"
)

type forwardAuth struct{}

// NewForwardAuthHandler creates a handler to convert
// annotations about forward authentication to APISIX forward-auth plugin.
func NewForwardAuthHandler() PluginAnnotationsHandler {
	return &forwardAuth{}
}

func (i *forwardAuth) PluginName() string {
	return "forward-auth"
}

func (i *forwardAuth) Handle(e annotations.Extractor) (any, error) {
	uri := e.GetStringAnnotation(annotations.AnnotationsForwardAuthURI)
	sslVerify := e.GetStringAnnotation(annotations.AnnotationsForwardAuthSSLVerify) != annotations.FalseString
	if len(uri) > 0 {
		return &adctypes.ForwardAuthConfig{
			URI:             uri,
			SSLVerify:       sslVerify,
			RequestHeaders:  e.GetStringsAnnotation(annotations.AnnotationsForwardAuthRequestHeaders),
			UpstreamHeaders: e.GetStringsAnnotation(annotations.AnnotationsForwardAuthUpstreamHeaders),
			ClientHeaders:   e.GetStringsAnnotation(annotations.AnnotationsForwardAuthClientHeaders),
		}, nil
	}

	return nil, nil
}
