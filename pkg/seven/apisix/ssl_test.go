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
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSslUnmarshalJSON(t *testing.T) {
	var sslList SslList
	emptyData := `
{
	"key": "test",
	"nodes": {}
}
`
	err := json.Unmarshal([]byte(emptyData), &sslList)
	assert.Nil(t, err)

	notEmptyObject := `
{
	"key": "test",
	"nodes": {"a": "b", "c": "d"}
}
`
	err = json.Unmarshal([]byte(notEmptyObject), &sslList)
	assert.Equal(t, err.Error(), "unexpected non-empty object")

	emptyArray := `
{
	"key": "test",
	"nodes": []
}
`
	err = json.Unmarshal([]byte(emptyArray), &sslList)
	assert.Nil(t, err)

	normalData := `
{
	"key": "test",
	"nodes": [
		{
			"key": "ssl id",
			"value": {
				"snis": ["test.apisix.org"],
				"cert": "root",
				"key": "123456",
				"status": 1
			}
		}
	]
}
`
	err = json.Unmarshal([]byte(normalData), &sslList)
	assert.Nil(t, err)
	assert.Equal(t, len(sslList.SslNodes), 1)

	key := *sslList.SslNodes[0].Key
	assert.Equal(t, key, "ssl id")
	cert := *sslList.SslNodes[0].Ssl.Cert
	assert.Equal(t, cert, "root")
	sslKey := *sslList.SslNodes[0].Ssl.Key
	assert.Equal(t, sslKey, "123456")
	sni := *sslList.SslNodes[0].Ssl.Snis[0]
	assert.Equal(t, sni, "test.apisix.org")
}
