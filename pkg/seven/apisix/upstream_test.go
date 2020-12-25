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

func TestUpstreamsUnmarshalJSON(t *testing.T) {
	var ups Upstreams
	emptyData := `
{
	"key": "test",
	"nodes": {}
}
`
	err := json.Unmarshal([]byte(emptyData), &ups)
	assert.Nil(t, err)

	emptyData = `
{
	"key": "test",
	"nodes": {"a": "b", "c": "d"}
}
`
	err = json.Unmarshal([]byte(emptyData), &ups)
	assert.Equal(t, err.Error(), "unexpected non-empty object")

	emptyArray := `
{
	"key": "test",
	"nodes": []
}
`
	err = json.Unmarshal([]byte(emptyArray), &ups)
	assert.Nil(t, err)

	normalData := `
{
	"key": "test",
	"nodes": [
		{
			"key": "ups1",
			"value": {
				"desc": "test upstream 1",
				"type": "rr",
				"nodes": {
					"192.168.12.12": 100
				}
			}
		}
	]
}
`
	err = json.Unmarshal([]byte(normalData), &ups)
	assert.Nil(t, err)
	assert.Equal(t, ups.Key, "test")
	assert.Equal(t, len(ups.Upstreams), 1)

	key := *ups.Upstreams[0].Key
	assert.Equal(t, key, "ups1")
	desc := *ups.Upstreams[0].UpstreamNodes.Desc
	assert.Equal(t, desc, "test upstream 1")
	lb := *ups.Upstreams[0].UpstreamNodes.LBType
	assert.Equal(t, lb, "rr")

	assert.Equal(t, len(ups.Upstreams[0].UpstreamNodes.Nodes), 1)
	assert.Equal(t, ups.Upstreams[0].UpstreamNodes.Nodes["192.168.12.12"], int64(100))
}
