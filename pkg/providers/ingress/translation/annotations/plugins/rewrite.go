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
	"regexp"

	"github.com/apache/apisix-ingress-controller/pkg/providers/ingress/translation/annotations"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

type rewrite struct{}

// NewRewriteHandler creates a handler to convert
// annotations about request rewrite control to APISIX proxy-rewrite plugin.
func NewRewriteHandler() PluginAnnotationsHandler {
	return &rewrite{}
}

func (i *rewrite) PluginName() string {
	return "proxy-rewrite"
}

func (i *rewrite) Handle(e annotations.Extractor) (interface{}, error) {
	var plugin apisixv1.RewriteConfig
	rewriteTarget := e.GetStringAnnotation(annotations.AnnotationsRewriteTarget)
	rewriteTargetRegex := e.GetStringAnnotation(annotations.AnnotationsRewriteTargetRegex)
	rewriteTemplate := e.GetStringAnnotation(annotations.AnnotationsRewriteTargetRegexTemplate)

	headers := apisixv1.RewriteConfigHeaders{
		Headers: make(apisixv1.Headers),
	}
	headers.Add(e.GetStringsAnnotation(annotations.AnnotationsRewriteHeaderAdd))
	headers.Set(e.GetStringsAnnotation(annotations.AnnotationsRewriteHeaderSet))
	headers.Remove(e.GetStringsAnnotation(annotations.AnnotationsRewriteHeaderRemove))

	if rewriteTarget != "" || rewriteTargetRegex != "" || rewriteTemplate != "" || len(headers.Headers) > 0 {
		plugin.RewriteTarget = rewriteTarget
		plugin.Headers = headers
		if rewriteTargetRegex != "" && rewriteTemplate != "" {
			_, err := regexp.Compile(rewriteTargetRegex)
			if err != nil {
				return nil, err
			}
			plugin.RewriteTargetRegex = []string{rewriteTargetRegex, rewriteTemplate}
		}
		return &plugin, nil
	}
	return nil, nil
}
