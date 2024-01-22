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
package translation

import (
	"github.com/imdario/mergo"
	"go.uber.org/zap"

	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/providers/ingress/translation/annotations"
	"github.com/apache/apisix-ingress-controller/pkg/providers/ingress/translation/annotations/pluginconfig"
	"github.com/apache/apisix-ingress-controller/pkg/providers/ingress/translation/annotations/plugins"
	"github.com/apache/apisix-ingress-controller/pkg/providers/ingress/translation/annotations/regex"
	"github.com/apache/apisix-ingress-controller/pkg/providers/ingress/translation/annotations/servicenamespace"
	"github.com/apache/apisix-ingress-controller/pkg/providers/ingress/translation/annotations/upstream"
	"github.com/apache/apisix-ingress-controller/pkg/providers/ingress/translation/annotations/websocket"
	apisix "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

// Structure extracted by Ingress Resource
type Ingress struct {
	Plugins          apisix.Plugins
	UseRegex         bool
	EnableWebSocket  bool
	PluginConfigName string
	ServiceNamespace string
	Upstream         upstream.Upstream
}

var (
	_parsers = map[string]annotations.IngressAnnotationsParser{
		"Plugins":          plugins.NewParser(),
		"UseRegex":         regex.NewParser(),
		"EnableWebSocket":  websocket.NewParser(),
		"PluginConfigName": pluginconfig.NewParser(),
		"ServiceNamespace": servicenamespace.NewParser(),
		"Upstream":         upstream.NewParser(),
	}
)

func (t *translator) TranslateAnnotations(anno map[string]string) *Ingress {
	ing := &Ingress{}
	extractor := annotations.NewExtractor(anno)
	data := make(map[string]interface{})
	for name, parser := range _parsers {
		out, err := parser.Parse(extractor)
		if err != nil {
			log.Warnw("failed to parse annotations",
				zap.Error(err),
			)
			continue
		}
		if out != nil {
			data[name] = out
		}
	}
	err := mergo.MapWithOverwrite(ing, data)
	if err != nil {
		log.Errorw("unexpected error merging extracted annotations", zap.Error(err))
	}
	return ing
}
