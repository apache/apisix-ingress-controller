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

func TestNginxVars(t *testing.T) {
	tr := &translator{}
	value1 := "text/plain"
	value2 := "gzip"
	value3 := "13"
	value4 := ".*\\.php"
	ngxVars := []configv2alpha1.ApisixRouteHTTPMatchNginxVar{
		{
			Subject: "http_content_type",
			Op:      configv2alpha1.OpEqual,
			Value:   &value1,
		},
		{
			Subject: "http_content_encoding",
			Op:      configv2alpha1.OpNotEqual,
			Value:   &value2,
		},
		{
			Subject: "arg_id",
			Op:      configv2alpha1.OpGreaterThan,
			Value:   &value3,
		},
		{
			Subject: "arg_id",
			Op:      configv2alpha1.OpLessThan,
			Value:   &value3,
		},
		{
			Subject: "arg_id",
			Op:      configv2alpha1.OpRegexMatch,
			Value:   &value4,
		},
		{
			Subject: "arg_id",
			Op:      configv2alpha1.OpRegexMatchCaseInsensitive,
			Value:   &value4,
		},
		{
			Subject: "arg_id",
			Op:      configv2alpha1.OpRegexNotMatch,
			Value:   &value4,
		},
		{
			Subject: "arg_id",
			Op:      configv2alpha1.OpRegexNotMatchCaseInsensitive,
			Value:   &value4,
		},
		{
			Subject: "remote_addr",
			Op:      configv2alpha1.OpIn,
			Set: []string{
				"10.0.5.3",
				"10.0.5.4",
			},
		},
		{
			Subject: "remote_addr",
			Op:      configv2alpha1.OpNotIn,
			Set: []string{
				"10.0.5.6",
			},
		},
	}
	vars, err := tr.translateNginxVars(ngxVars)
	assert.Nil(t, err)
	assert.Len(t, vars, 10)

	assert.Len(t, vars[0], 3)
	assert.Equal(t, vars[0][0].StrVal, "http_content_type")
	assert.Equal(t, vars[0][1].StrVal, "==")
	assert.Equal(t, vars[0][2].StrVal, "text/plain")

	assert.Len(t, vars[1], 3)
	assert.Equal(t, vars[1][0].StrVal, "http_content_encoding")
	assert.Equal(t, vars[1][1].StrVal, "~=")
	assert.Equal(t, vars[1][2].StrVal, "gzip")

	assert.Len(t, vars[2], 3)
	assert.Equal(t, vars[2][0].StrVal, "arg_id")
	assert.Equal(t, vars[2][1].StrVal, ">")
	assert.Equal(t, vars[2][2].StrVal, "13")

	assert.Len(t, vars[3], 3)
	assert.Equal(t, vars[3][0].StrVal, "arg_id")
	assert.Equal(t, vars[3][1].StrVal, "<")
	assert.Equal(t, vars[3][2].StrVal, "13")

	assert.Len(t, vars[4], 3)
	assert.Equal(t, vars[4][0].StrVal, "arg_id")
	assert.Equal(t, vars[4][1].StrVal, "~~")
	assert.Equal(t, vars[4][2].StrVal, ".*\\.php")

	assert.Len(t, vars[5], 3)
	assert.Equal(t, vars[5][0].StrVal, "arg_id")
	assert.Equal(t, vars[5][1].StrVal, "~*")
	assert.Equal(t, vars[5][2].StrVal, ".*\\.php")

	assert.Len(t, vars[6], 4)
	assert.Equal(t, vars[6][0].StrVal, "arg_id")
	assert.Equal(t, vars[6][1].StrVal, "!")
	assert.Equal(t, vars[6][2].StrVal, "~~")
	assert.Equal(t, vars[6][3].StrVal, ".*\\.php")

	assert.Len(t, vars[7], 4)
	assert.Equal(t, vars[7][0].StrVal, "arg_id")
	assert.Equal(t, vars[7][1].StrVal, "!")
	assert.Equal(t, vars[7][2].StrVal, "~*")
	assert.Equal(t, vars[7][3].StrVal, ".*\\.php")

	assert.Len(t, vars[8], 3)
	assert.Equal(t, vars[8][0].StrVal, "remote_addr")
	assert.Equal(t, vars[8][1].StrVal, "in")
	assert.Equal(t, vars[8][2].SliceVal, []string{"10.0.5.3", "10.0.5.4"})

	assert.Len(t, vars[9], 4)
	assert.Equal(t, vars[9][0].StrVal, "remote_addr")
	assert.Equal(t, vars[9][1].StrVal, "!")
	assert.Equal(t, vars[9][2].StrVal, "in")
	assert.Equal(t, vars[9][3].SliceVal, []string{"10.0.5.6"})
}
