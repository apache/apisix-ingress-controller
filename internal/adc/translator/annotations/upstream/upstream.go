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

	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/adc/translator/annotations"
)

func NewParser() annotations.IngressAnnotationsParser {
	return &Upstream{}
}

type Upstream struct {
	Scheme         string
	Retries        int
	TimeoutRead    int
	TimeoutConnect int
	TimeoutSend    int
}

var validSchemes = map[string]struct{}{
	apiv2.SchemeHTTP:  {},
	apiv2.SchemeHTTPS: {},
	apiv2.SchemeGRPC:  {},
	apiv2.SchemeGRPCS: {},
}

func (u Upstream) Parse(e annotations.Extractor) (any, error) {
	if scheme := strings.ToLower(e.GetStringAnnotation(annotations.AnnotationsUpstreamScheme)); scheme != "" {
		if _, ok := validSchemes[scheme]; ok {
			u.Scheme = scheme
		} else {
			return nil, fmt.Errorf("invalid upstream scheme: %s", scheme)
		}
	}

	if retry := e.GetStringAnnotation(annotations.AnnotationsUpstreamRetry); retry != "" {
		t, err := strconv.Atoi(retry)
		if err != nil {
			return nil, fmt.Errorf("could not parse retry as an integer: %s", err.Error())
		}
		u.Retries = t
	}

	if timeoutConnect := strings.TrimSuffix(e.GetStringAnnotation(annotations.AnnotationsUpstreamTimeoutConnect), "s"); timeoutConnect != "" {
		t, err := strconv.Atoi(timeoutConnect)
		if err != nil {
			return nil, fmt.Errorf("could not parse timeout as an integer: %s", err.Error())
		}
		u.TimeoutConnect = t
	}

	if timeoutRead := strings.TrimSuffix(e.GetStringAnnotation(annotations.AnnotationsUpstreamTimeoutRead), "s"); timeoutRead != "" {
		t, err := strconv.Atoi(timeoutRead)
		if err != nil {
			return nil, fmt.Errorf("could not parse timeout as an integer: %s", err.Error())
		}
		u.TimeoutRead = t
	}

	if timeoutSend := strings.TrimSuffix(e.GetStringAnnotation(annotations.AnnotationsUpstreamTimeoutSend), "s"); timeoutSend != "" {
		t, err := strconv.Atoi(timeoutSend)
		if err != nil {
			return nil, fmt.Errorf("could not parse timeout as an integer: %s", err.Error())
		}
		u.TimeoutSend = t
	}

	return u, nil
}
