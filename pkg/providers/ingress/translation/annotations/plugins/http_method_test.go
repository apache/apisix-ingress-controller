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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/pkg/providers/ingress/translation/annotations"
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"

	"github.com/incubator4/go-resty-expr/expr"
)

// annotations:
//
//	k8s.apisix.apache.org/allow-http-methods: GET,POST
func TestAnnotationsHttpAllowMethod(t *testing.T) {
	anno := map[string]string{
		annotations.AnnotationsHttpAllowMethods: "GET,POST",
	}
	p := NewHttpMethodHandler()
	out, err := p.Handle(annotations.NewExtractor(anno))
	assert.Nil(t, err, "checking given error")
	config := out.(*apisixv1.ResponseRewriteConfig)

	assert.Equal(t, http.StatusMethodNotAllowed, config.StatusCode)
	assert.Equal(t, []expr.Expr{
		expr.StringExpr("request_method").Not().In(expr.ArrayExpr(
			expr.StringExpr("GET"),
			expr.StringExpr("POST"),
		)),
	}, config.LuaRestyExpr)
}

// annotations:
//
//	k8s.apisix.apache.org/block-http-methods: GET,PUT
func TestAnnotationsHttpBlockMethod(t *testing.T) {
	anno := map[string]string{
		annotations.AnnotationsHttpBlockMethods: "GET,PUT",
	}
	p := NewHttpMethodHandler()
	out, err := p.Handle(annotations.NewExtractor(anno))
	assert.Nil(t, err, "checking given error")
	config := out.(*apisixv1.ResponseRewriteConfig)

	assert.Equal(t, 405, config.StatusCode)
	assert.Equal(t, []expr.Expr{
		expr.StringExpr("request_method").In(expr.ArrayExpr(
			expr.StringExpr("GET"),
			expr.StringExpr("PUT"),
		)),
	}, config.LuaRestyExpr)
}

// annotations:
//
//	k8s.apisix.apache.org/allow-http-methods: GET
//	k8s.apisix.apache.org/block-http-methods: POST,PUT
//
// Only allow methods would be accepted, block methods would be ignored.
func TestAnnotationsHttpBothMethod(t *testing.T) {
	anno := map[string]string{
		annotations.AnnotationsHttpAllowMethods: "GET",
		annotations.AnnotationsHttpBlockMethods: "POST,PUT",
	}
	p := NewHttpMethodHandler()
	out, err := p.Handle(annotations.NewExtractor(anno))
	assert.Nil(t, err, "checking given error")
	config := out.(*apisixv1.ResponseRewriteConfig)

	assert.Equal(t, http.StatusMethodNotAllowed, config.StatusCode)
	assert.Equal(t, []expr.Expr{
		expr.StringExpr("request_method").Not().In(expr.ArrayExpr(
			expr.StringExpr("GET"),
		)),
	}, config.LuaRestyExpr)
}
