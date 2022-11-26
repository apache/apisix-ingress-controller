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
package upstreamscheme

import (
	"fmt"

	"github.com/apache/apisix-ingress-controller/pkg/providers/ingress/translation/annotations"
)

type upstreamscheme struct{}

func NewParser() annotations.IngressAnnotationsParser {
	return &upstreamscheme{}
}

// string in slice
func inSchemes(a string) bool {
	list := []string{"http", "https", "grpc", "grpcs"}

	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func (w *upstreamscheme) Parse(e annotations.Extractor) (interface{}, error) {
	scheme := e.GetStringAnnotation(annotations.AnnotationsUpstreamScheme)

	if inSchemes(scheme) {
		return scheme, nil
	}

	return nil, fmt.Errorf("scheme %s is not supported, Only { http, https, grpc, grpcs } are supported", scheme)
}
