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
	"strings"

	"github.com/apache/apisix-ingress-controller/pkg/providers/ingress/translation/annotations"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

type upstreamscheme struct{}

func NewParser() annotations.IngressAnnotationsParser {
	return &upstreamscheme{}
}

func (w *upstreamscheme) Parse(e annotations.Extractor) (interface{}, error) {
	scheme := strings.ToLower(e.GetStringAnnotation(annotations.AnnotationsUpstreamScheme))
	if scheme == "" {
		return nil, nil
	}
	_, ok := apisixv1.ValidSchemes[scheme]
	if ok {
		return scheme, nil
	}

	keys := make([]string, 0, len(apisixv1.ValidSchemes))
	for key := range apisixv1.ValidSchemes {
		keys = append(keys, key)
	}

	return nil, fmt.Errorf("scheme %s is not supported, Only { %s } are supported", scheme, strings.Join(keys, ", "))
}
