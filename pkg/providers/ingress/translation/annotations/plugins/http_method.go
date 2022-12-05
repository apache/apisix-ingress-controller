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
	"net/http"

	"github.com/apache/apisix-ingress-controller/pkg/providers/ingress/translation/annotations"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"

	"github.com/incubator4/go-resty-expr/expr"
)

type HttpMethod struct{}

// NewHttpHandler creates a handler to convert annotations about
// HttpMethod to APISIX cors plugin.
func NewHttpMethodHandler() PluginAnnotationsHandler {
	return &HttpMethod{}
}

func (h HttpMethod) PluginName() string {
	return "response-rewrite"
}

func (h HttpMethod) Handle(e annotations.Extractor) (interface{}, error) {
	var plugin apisixv1.ResponseRewriteConfig

	allowMethods := e.GetStringsAnnotation(annotations.AnnotationsHttpAllowMethods)
	blockMethods := e.GetStringsAnnotation(annotations.AnnotationsHttpBlockMethods)

	plugin.StatusCode = http.StatusMethodNotAllowed

	if len(allowMethods) > 0 {
		plugin.LuaRestyExpr = []expr.Expr{
			expr.StringExpr("request_method").Not().In(
				expr.ArrayExpr(expr.ExprArrayFromStrings(allowMethods)...),
			),
		}

	} else if len(blockMethods) > 0 {
		plugin.LuaRestyExpr = []expr.Expr{
			expr.StringExpr("request_method").In(
				expr.ArrayExpr(expr.ExprArrayFromStrings(blockMethods)...),
			),
		}

	} else {
		return nil, nil
	}
	return &plugin, nil
}
