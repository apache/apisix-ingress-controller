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
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

const (
	_enableCsrf = AnnotationsPrefix + "enable-csrf"
	_csrfKey    = AnnotationsPrefix + "csrf-key"
)

type csrf struct{}

// NewCSRFHandler creates a handler to convert annotations about
// CSRF to APISIX csrf plugin.
func NewCSRFHandler() Handler {
	return &csrf{}
}

func (c *csrf) PluginName() string {
	return "csrf"
}

func (c *csrf) Handle(e Extractor) (interface{}, error) {
	if !e.GetBoolAnnotation(_enableCsrf) {
		return nil, nil
	}
	var plugin apisixv1.CSRFConfig
	plugin.Key = e.GetStringAnnotation(_csrfKey)
	if plugin.Key != "" {
		return &plugin, nil
	}
	return nil, nil
}
