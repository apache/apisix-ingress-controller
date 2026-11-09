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
	"errors"
	"fmt"

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
	handlers = []PluginAnnotationsHandler{
		NewRedirectHandler(),
		NewRewriteHandler(),
		NewCorsHandler(),
		NewCSRFHandler(),
		NewFaultInjectionHandler(),
		NewBasicAuthHandler(),
		NewKeyAuthHandler(),
		NewResponseRewriteHandler(),
		NewIPRestrictionHandler(),
		NewForwardAuthHandler(),
	}
)

type plugins struct{}

func NewParser() annotations.IngressAnnotationsParser {
	return &plugins{}
}

// Parse returns the plugins that parsed cleanly together with the errors of the
// ones that did not. A handler error used to be logged and dropped here, which
// hid a requested plugin from the admission webhook and from the operator.
func (p *plugins) Parse(e annotations.Extractor) (any, error) {
	plugins := make(adctypes.Plugins)
	var errs []error
	for _, handler := range handlers {
		out, err := handler.Handle(e)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", handler.PluginName(), err))
			continue
		}
		if out != nil {
			plugins[handler.PluginName()] = out
		}
	}
	if len(plugins) > 0 {
		return plugins, errors.Join(errs...)
	}
	return nil, errors.Join(errs...)
}
