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
package plugins

import (
	"github.com/apache/apisix-ingress-controller/pkg/kube/translation/annotations"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	apisix "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
	"go.uber.org/zap"
)

const ()

var (
	_handlers = []PluginHandler{
		NewCorsHandler(),
		NewIPRestrictionHandler(),
		NewRewriteHandler(),
		NewRedirectHandler(),
		NewForwardAuthHandler(),
		NewBasicAuthHandler(),
		NewKeyAuthHandler(),
		NewCSRFHandler(),
	}
)

type PluginHandler interface {
	// Handle parses the target annotation and converts it to the type-agnostic structure.
	// The return value might be nil since some features have an explicit switch, users should
	// judge whether Handle is failed by the second error value.
	Handle(*annotations.Ingress) (interface{}, error)
	// PluginName returns a string which indicates the target plugin name in APISIX.
	PluginName() string
}

type plugins struct{}

func NewPluginsParser() annotations.IngressAnnotations {
	return &plugins{}
}

func (p *plugins) Parse(ing *annotations.Ingress) (interface{}, error) {
	plugins := make(apisix.Plugins)
	for _, handler := range _handlers {
		out, err := handler.Handle(ing)
		if err != nil {
			log.Warnw("failed to handle annotations",
				zap.Error(err),
			)
			continue
		}
		if out != nil {
			plugins[handler.PluginName()] = out
		}
	}
	return plugins, nil
}
