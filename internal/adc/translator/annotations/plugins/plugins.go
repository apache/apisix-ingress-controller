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
	"github.com/api7/gopkg/pkg/log"
	"go.uber.org/zap"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/adc/translator/annotations"
)

// Handler abstracts the behavior so that the apisix-ingress-controller knows
// how to parse some annotations and convert them to APISIX plugins.
type PluginAnnotationsHandler interface {
	// Handle parses the target annotation and converts it to the type-agnostic structure.
	// The return value might be nil since some features have an explicit switch, users should
	// judge whether Handle is failed by the second error value.
	Handle(annotations.Extractor) (any, error)
	// PluginName returns a string which indicates the target plugin name in APISIX.
	PluginName() string
}

var (
	_handlers = []PluginAnnotationsHandler{
		NewCorsHandler(),
	}
)

type plugins struct{}

func NewParser() annotations.IngressAnnotationsParser {
	return &plugins{}
}

func (p *plugins) Parse(e annotations.Extractor) (any, error) {
	plugins := make(adctypes.Plugins)
	for _, handler := range _handlers {
		out, err := handler.Handle(e)
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
