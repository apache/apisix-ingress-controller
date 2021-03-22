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
	"testing"

	"github.com/stretchr/testify/assert"

	configv2alpha1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2alpha1"
)

func TestRouteMatchExpr(t *testing.T) {
	tr := &translator{}
	value1 := "text/plain"
	value2 := "gzip"
	value3 := "13"
	value4 := ".*\\.php"
	exprs := []configv2alpha1.ApisixRouteHTTPMatchExpr{
		{
			Subject: configv2alpha1.ApisixRouteHTTPMatchExprSubject{
				Scope: configv2alpha1.ScopeHeader,
				Name:  "Content-Type",
			},
			Op:    configv2alpha1.OpEqual,
			Value: &value1,
		},
		{
			Subject: configv2alpha1.ApisixRouteHTTPMatchExprSubject{
				Scope: configv2alpha1.ScopeHeader,
				Name:  "Content-Encoding",
			},
			Op:    configv2alpha1.OpNotEqual,
			Value: &value2,
		},
		{
			Subject: configv2alpha1.ApisixRouteHTTPMatchExprSubject{
				Scope: configv2alpha1.ScopeQuery,
				Name:  "ID",
			},
			Op:    configv2alpha1.OpGreaterThan,
			Value: &value3,
		},
		{
			Subject: configv2alpha1.ApisixRouteHTTPMatchExprSubject{
				Scope: configv2alpha1.ScopeQuery,
				Name:  "ID",
			},
			Op:    configv2alpha1.OpLessThan,
			Value: &value3,
		},
		{
			Subject: configv2alpha1.ApisixRouteHTTPMatchExprSubject{
				Scope: configv2alpha1.ScopeQuery,
				Name:  "ID",
			},
			Op:    configv2alpha1.OpRegexMatch,
			Value: &value4,
		},
		{
			Subject: configv2alpha1.ApisixRouteHTTPMatchExprSubject{
				Scope: configv2alpha1.ScopeQuery,
				Name:  "ID",
			},
			Op:    configv2alpha1.OpRegexMatchCaseInsensitive,
			Value: &value4,
		},
		{
			Subject: configv2alpha1.ApisixRouteHTTPMatchExprSubject{
				Scope: configv2alpha1.ScopeQuery,
				Name:  "ID",
			},
			Op:    configv2alpha1.OpRegexNotMatch,
			Value: &value4,
		},
		{
			Subject: configv2alpha1.ApisixRouteHTTPMatchExprSubject{
				Scope: configv2alpha1.ScopeQuery,
				Name:  "ID",
			},
			Op:    configv2alpha1.OpRegexNotMatchCaseInsensitive,
			Value: &value4,
		},
		{
			Subject: configv2alpha1.ApisixRouteHTTPMatchExprSubject{
				Scope: configv2alpha1.ScopeCookie,
				Name:  "domain",
			},
			Op: configv2alpha1.OpIn,
			Set: []string{
				"a.com",
				"b.com",
			},
		},
	}
	results, err := tr.translateRouteMatchExprs(exprs)
	assert.Nil(t, err)
	assert.Len(t, results, 9)

	assert.Len(t, results[0], 3)
	assert.Equal(t, results[0][0].StrVal, "http_content_type")
	assert.Equal(t, results[0][1].StrVal, "==")
	assert.Equal(t, results[0][2].StrVal, "text/plain")

	assert.Len(t, results[1], 3)
	assert.Equal(t, results[1][0].StrVal, "http_content_encoding")
	assert.Equal(t, results[1][1].StrVal, "~=")
	assert.Equal(t, results[1][2].StrVal, "gzip")

	assert.Len(t, results[2], 3)
	assert.Equal(t, results[2][0].StrVal, "arg_id")
	assert.Equal(t, results[2][1].StrVal, ">")
	assert.Equal(t, results[2][2].StrVal, "13")

	assert.Len(t, results[3], 3)
	assert.Equal(t, results[3][0].StrVal, "arg_id")
	assert.Equal(t, results[3][1].StrVal, "<")
	assert.Equal(t, results[3][2].StrVal, "13")

	assert.Len(t, results[4], 3)
	assert.Equal(t, results[4][0].StrVal, "arg_id")
	assert.Equal(t, results[4][1].StrVal, "~~")
	assert.Equal(t, results[4][2].StrVal, ".*\\.php")

	assert.Len(t, results[5], 3)
	assert.Equal(t, results[5][0].StrVal, "arg_id")
	assert.Equal(t, results[5][1].StrVal, "~*")
	assert.Equal(t, results[5][2].StrVal, ".*\\.php")

	assert.Len(t, results[6], 4)
	assert.Equal(t, results[6][0].StrVal, "arg_id")
	assert.Equal(t, results[6][1].StrVal, "!")
	assert.Equal(t, results[6][2].StrVal, "~~")
	assert.Equal(t, results[6][3].StrVal, ".*\\.php")

	assert.Len(t, results[7], 4)
	assert.Equal(t, results[7][0].StrVal, "arg_id")
	assert.Equal(t, results[7][1].StrVal, "!")
	assert.Equal(t, results[7][2].StrVal, "~*")
	assert.Equal(t, results[7][3].StrVal, ".*\\.php")

	assert.Len(t, results[8], 3)
	assert.Equal(t, results[8][0].StrVal, "cookie_domain")
	assert.Equal(t, results[8][1].StrVal, "in")
	assert.Equal(t, results[8][2].SliceVal, []string{"a.com", "b.com"})
}
