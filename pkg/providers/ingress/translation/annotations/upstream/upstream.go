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
package upstream

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/apache/apisix-ingress-controller/pkg/providers/ingress/translation/annotations"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

func NewParser() annotations.IngressAnnotationsParser {
	return &Upstream{}
}

type Upstream struct {
	Scheme         string
	Retry          int
	TimeoutRead    int
	TimeoutConnect int
	TimeoutSend    int
}

func (u *Upstream) Parse(e annotations.Extractor) (interface{}, error) {
	scheme := strings.ToLower(e.GetStringAnnotation(annotations.AnnotationsUpstreamScheme))
	if scheme != "" {
		_, ok := apisixv1.ValidSchemes[scheme]
		if !ok {
			keys := make([]string, 0, len(apisixv1.ValidSchemes))
			for key := range apisixv1.ValidSchemes {
				keys = append(keys, key)
			}
			return nil, fmt.Errorf("scheme %s is not supported, Only { %s } are supported", scheme, strings.Join(keys, ", "))
		}
		u.Scheme = scheme
	}

	retry := e.GetStringAnnotation(annotations.AnnotationsUpstreamRetry)
	if retry != "" {
		t, err := strconv.Atoi(retry)
		if err != nil {
			return nil, fmt.Errorf("could not parse retry as an integer: %s", err.Error())
		}
		u.Retry = t
	}

	timeoutConnect := strings.TrimSuffix(e.GetStringAnnotation(annotations.AnnotationsUpstreamTimeoutConnect), "s")
	if timeoutConnect != "" {
		t, err := strconv.Atoi(timeoutConnect)
		if err != nil {
			return nil, fmt.Errorf("could not parse timeout as an integer: %s", err.Error())
		}
		u.TimeoutConnect = t
	}

	timeoutRead := strings.TrimSuffix(e.GetStringAnnotation(annotations.AnnotationsUpstreamTimeoutRead), "s")
	if timeoutRead != "" {
		t, err := strconv.Atoi(timeoutRead)
		if err != nil {
			return nil, fmt.Errorf("could not parse timeout as an integer: %s", err.Error())
		}
		u.TimeoutRead = t
	}

	timeoutSend := strings.TrimSuffix(e.GetStringAnnotation(annotations.AnnotationsUpstreamTimeoutSend), "s")
	if timeoutSend != "" {
		t, err := strconv.Atoi(timeoutSend)
		if err != nil {
			return nil, fmt.Errorf("could not parse timeout as an integer: %s", err.Error())
		}
		u.TimeoutSend = t
	}

	return *u, nil
}
