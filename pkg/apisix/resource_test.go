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

func TestItemUnmarshalJSON(t *testing.T) {
	var items node
	emptyData := `
{
	"key": "test",
	"nodes": {}
}
`
	err := json.Unmarshal([]byte(emptyData), &items)
	assert.Nil(t, err)

	emptyData = `
{
	"key": "test",
	"nodes": {"a": "b", "c": "d"}
}
`
	err = json.Unmarshal([]byte(emptyData), &items)
	assert.Equal(t, err.Error(), "unexpected non-empty object")

	emptyArray := `
{
	"key": "test",
	"nodes": []
}
`
	err = json.Unmarshal([]byte(emptyArray), &items)
	assert.Nil(t, err)
}

func TestItemConvertRoute(t *testing.T) {
	item := &item{
		Key: "/apisix/routes/001",
		Value: json.RawMessage(`
			{
				"upstream_id": "13",
				"service_id": "14",
				"host": "foo.com",
				"uri": "/shop/133/details",
				"desc": "unknown",
				"methods": ["GET", "POST"]
			}
		`),
	}

	r, err := item.route("qa")
	assert.Nil(t, err)
	assert.Equal(t, r.UpstreamId, "13")
	assert.Equal(t, r.ServiceId, "14")
	assert.Equal(t, r.Host, "foo.com")
	assert.Equal(t, r.Path, "/shop/133/details")
	assert.Equal(t, r.Methods[0], "GET")
	assert.Equal(t, r.Methods[1], "POST")
	assert.Equal(t, r.Name, "unknown")
	assert.Equal(t, r.FullName, "qa_unknown")
}
