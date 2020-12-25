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
package apisix

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRouteUnmarshalJSON(t *testing.T) {
	var route Routes
	emptyData := `
{
	"key": "test",
	"nodes": {}
}
`
	err := json.Unmarshal([]byte(emptyData), &route)
	assert.Nil(t, err)

	emptyData = `
{
	"key": "test",
	"nodes": {"a": "b", "c": "d"}
}
`
	err = json.Unmarshal([]byte(emptyData), &route)
	assert.Equal(t, err.Error(), "unexpected non-empty object")

	emptyArray := `
{
	"key": "test",
	"nodes": []
}
`
	err = json.Unmarshal([]byte(emptyArray), &route)
	assert.Nil(t, err)

	normalData := `
{
	"key": "test",
	"nodes": [
		{
			"key": "route 1",
			"value": {
				"desc": "test route 1",
				"upstream_id": "123",
				"service_id": "12345",
				"host": "foo.com",
				"uri": "/bar/baz",
				"methods": ["GET", "POST"]
			}
		}
	]
}
`
	err = json.Unmarshal([]byte(normalData), &route)
	assert.Nil(t, err)
	assert.Equal(t, route.Key, "test")
	assert.Equal(t, len(route.Routes), 1)

	key := *route.Routes[0].Key
	assert.Equal(t, key, "route 1")
	desc := *route.Routes[0].Value.Desc
	assert.Equal(t, desc, "test route 1")
	upstreamId := *route.Routes[0].Value.UpstreamId
	assert.Equal(t, upstreamId, "123")
	svcId := *route.Routes[0].Value.ServiceId
	assert.Equal(t, svcId, "12345")
	assert.Equal(t, *route.Routes[0].Value.Host, "foo.com")
	assert.Equal(t, *route.Routes[0].Value.Uri, "/bar/baz")
	assert.Equal(t, *route.Routes[0].Value.Methods[0], "GET")
	assert.Equal(t, *route.Routes[0].Value.Methods[1], "POST")
}

func TestRouteConvertWithoutDesc(t *testing.T) {
	upsId := "1"
	svcId := "2"
	key := "foo/bar"
	r := &Route{
		Key: &key,
		Value: Value{
			UpstreamId: &upsId,
			ServiceId:  &svcId,
			Host:       nil,
		},
	}
	_, err := r.convert("mygroup")
	assert.Nil(t, err)
}
