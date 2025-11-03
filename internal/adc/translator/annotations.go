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

package translator

import (
	"errors"
	"fmt"

	"github.com/imdario/mergo"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/adc/translator/annotations"
	"github.com/apache/apisix-ingress-controller/internal/adc/translator/annotations/pluginconfig"
	"github.com/apache/apisix-ingress-controller/internal/adc/translator/annotations/plugins"
	"github.com/apache/apisix-ingress-controller/internal/adc/translator/annotations/regex"
	"github.com/apache/apisix-ingress-controller/internal/adc/translator/annotations/servicenamespace"
	"github.com/apache/apisix-ingress-controller/internal/adc/translator/annotations/upstream"
	"github.com/apache/apisix-ingress-controller/internal/adc/translator/annotations/websocket"
)

// Structure extracted by Ingress Resource
type IngressConfig struct {
	Upstream         upstream.Upstream
	Plugins          adctypes.Plugins
	EnableWebsocket  bool
	ServiceNamespace string
	PluginConfigName string
	UseRegex         bool
}

var ingressAnnotationParsers = map[string]annotations.IngressAnnotationsParser{
	"upstream":         upstream.NewParser(),
	"plugins":          plugins.NewParser(),
	"EnableWebsocket":  websocket.NewParser(),
	"PluginConfigName": pluginconfig.NewParser(),
	"ServiceNamespace": servicenamespace.NewParser(),
	"UseRegex":         regex.NewParser(),
}

func (t *Translator) TranslateIngressAnnotations(anno map[string]string) *IngressConfig {
	if len(anno) == 0 {
		return nil
	}
	ing := &IngressConfig{}
	if err := translateAnnotations(anno, ing); err != nil {
		t.Log.Error(err, "failed to translate ingress annotations", "annotations", anno)
	}
	return ing
}

func translateAnnotations(anno map[string]string, dst any) error {
	extractor := annotations.NewExtractor(anno)
	data := make(map[string]any)
	var errs []error

	for name, parser := range ingressAnnotationParsers {
		out, err := parser.Parse(extractor)
		if err != nil {
			errs = append(errs, fmt.Errorf("parse %s: %w", name, err))
			continue
		}
		if out != nil {
			data[name] = out
		}
	}

	if err := mergo.MapWithOverwrite(dst, data); err != nil {
		errs = append(errs, fmt.Errorf("merge: %w", err))
	}
	return errors.Join(errs...)
}
