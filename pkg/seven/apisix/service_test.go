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

func TestServiceUnmarshalJSON(t *testing.T) {
	var svc Services
	emptyData := `
{
	"key": "test",
	"nodes": {}
}
`
	err := json.Unmarshal([]byte(emptyData), &svc)
	assert.Nil(t, err)

	emptyData = `
{
	"key": "test",
	"nodes": {"a": "b", "c": "d"}
}
`
	err = json.Unmarshal([]byte(emptyData), &svc)
	assert.Equal(t, err.Error(), "unexpected non-empty object")

	emptyArray := `
{
	"key": "test",
	"nodes": []
}
`
	err = json.Unmarshal([]byte(emptyArray), &svc)
	assert.Nil(t, err)

	normalData := `
{
	"key": "test",
	"nodes": [
		{
			"key": "svc1",
			"value": {
				"desc": "test service 1",
				"upstream_id": "123",
				"plugins": {}
			}
		}
	]
}
`
	err = json.Unmarshal([]byte(normalData), &svc)
	assert.Nil(t, err)
	assert.Equal(t, svc.Key, "test")
	assert.Equal(t, len(svc.Services), 1)

	key := *svc.Services[0].Key
	assert.Equal(t, key, "svc1")
	desc := *svc.Services[0].ServiceValue.Desc
	assert.Equal(t, desc, "test service 1")

	upstreamId := *svc.Services[0].ServiceValue.UpstreamId
	assert.Equal(t, upstreamId, "123")
}
