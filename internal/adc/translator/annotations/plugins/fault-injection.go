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

	"github.com/incubator4/go-resty-expr/expr"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/adc/translator/annotations"
)

type FaultInjection struct{}

// FaultInjection to APISIX fault-injection plugin.
func NewFaultInjectionHandler() PluginAnnotationsHandler {
	return &FaultInjection{}
}

func (h FaultInjection) PluginName() string {
	return "fault-injection"
}

func (f FaultInjection) Handle(e annotations.Extractor) (any, error) {
	var plugin adctypes.FaultInjectionConfig

	allowMethods := e.GetStringsAnnotation(annotations.AnnotationsHttpAllowMethods)
	blockMethods := e.GetStringsAnnotation(annotations.AnnotationsHttpBlockMethods)
	if len(allowMethods) == 0 && len(blockMethods) == 0 {
		return nil, nil
	}
	abort := &adctypes.FaultInjectionAbortConfig{
		HTTPStatus: http.StatusMethodNotAllowed,
	}
	if len(allowMethods) > 0 {
		abort.Vars = [][]expr.Expr{{
			expr.StringExpr("request_method").Not().In(
				expr.ArrayExpr(expr.ExprArrayFromStrings(allowMethods)...),
			),
		}}
	} else {
		abort.Vars = [][]expr.Expr{{
			expr.StringExpr("request_method").In(
				expr.ArrayExpr(expr.ExprArrayFromStrings(blockMethods)...),
			),
		}}
	}
	plugin.Abort = abort
	return &plugin, nil
}
