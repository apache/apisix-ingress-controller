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
	_enableCsrf = annotations.AnnotationsPrefix + "enable-csrf"
	_csrfKey    = annotations.AnnotationsPrefix + "csrf-key"
)

type csrf struct{}

// NewCSRFHandler creates a handler to convert annotations about
// CSRF to APISIX csrf plugin.
func NewCSRFHandler() PluginHandler {
	return &csrf{}
}

func (c *csrf) PluginName() string {
	return "csrf"
}

func (c *csrf) Handle(ing *annotations.Ingress) (interface{}, error) {
	if !annotations.GetBoolAnnotation(_enableCsrf, ing) {
		return nil, nil
	}
	var plugin apisixv1.CSRFConfig
	plugin.Key = annotations.GetStringAnnotation(_csrfKey, ing)
	if plugin.Key != "" {
		return &plugin, nil
	}
	return nil, nil
}
