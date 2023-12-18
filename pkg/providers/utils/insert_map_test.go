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
package utils

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestInsertKeyInMap(t *testing.T) {
	type testCase struct {
		key    string
		value  interface{}
		dest   string
		merged string
	}
	testCases := []testCase{{
		dest: `{
			"a":1,
			"b":{
				"c":{
					"d":"e"
				},
				"f":"g"
			}
		}`,
		key:   `b.c`,
		value: 2,
		merged: `{
			"a":1,
			"b":{
				"c":2,
				"f":"g"
			}
		}`,
	}, {
		dest: `{
			"a":1,
			"b":{
				"c": 2,
				"f":"g"
			}
		}`,
		key: `b.c`,
		value: map[string]string{
			"d": "e",
		},
		merged: `{
			"a":1,
			"b":{
				"c":{
					"d":"e"
				},
				"f":"g"
			}
		}`,
	}, {
		dest: `{
			"a":1,
			"b":{
				"c": 2,
				"f":"g"
			}
		}`,
		key: `b.c.d`,
		value: map[string]string{
			"x": "y",
		},
		merged: `{
			"a":1,
			"b":{
				"c":{
					"d":{
						"x":"y"
					}
				},
				"f":"g"
			}
		}`,
	}, {
		dest: `{
			"a":1,
			"b":"old"
		}
		`,
		key:   "b",
		value: "new",
		merged: `{
			"a":1,
			"b":"new"
		}`,
	}}

	for _, t0 := range testCases {
		destMap := make(map[string]interface{})
		err := json.Unmarshal([]byte(t0.dest), &destMap)
		if err != nil {
			t.Fatal(err)
		}
		out := make(map[string]interface{})
		err = json.Unmarshal([]byte(t0.merged), &out)
		if err != nil {
			t.Fatal(err)
		}
		outB, err := json.MarshalIndent(out, " ", "")
		if err != nil {
			t.Fatal(err)
		}

		InsertKeyInMap(t0.key, t0.value, destMap)
		fmt.Println(destMap)
		merged, err := json.MarshalIndent(destMap, " ", "")
		if err != nil {
			t.Fatal(err)
		}
		if string(outB) != string(merged) {
			t.Errorf("Expected \n%s\n but got \n%s\n", string(outB), string(merged))
		}
	}
}
