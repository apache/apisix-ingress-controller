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
package translation

import (
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/apisix-ingress-controller/pkg/kube/translation/annotations"
	"github.com/apache/apisix-ingress-controller/pkg/kube/translation/annotations/plugins"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/imdario/mergo"
)

var (
	_handlers = map[string]annotations.IngressAnnotations{
		"Plugins": plugins.NewPluginsParser(),
	}
)

func (t *translator) translateAnnotations(meta metav1.ObjectMeta) *annotations.Ingress {
	ing := &annotations.Ingress{
		ObjectMeta: meta,
	}
	data := make(map[string]interface{})
	for name, handler := range _handlers {
		out, err := handler.Parse(ing)
		if err != nil {
			log.Warnw("failed to handle annotations",
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
