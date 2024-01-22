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
	"strings"
)

// InsertKeyInMap takes a dot separated string and recursively goes inside the destination
// to fill the value
func InsertKeyInMap(key string, value interface{}, dest map[string]interface{}) {
	if key == "" {
		return
	}
	keys := strings.SplitN(key, ".", 2)
	//base condition. the length of keys will be atleast 1
	if len(keys) < 2 {
		dest[keys[0]] = value
		return
	}

	ikey := keys[0]
	restKey := keys[1]
	if dest[ikey] == nil {
		dest[ikey] = make(map[string]interface{})
	}
	newDest, ok := dest[ikey].(map[string]interface{})
	if !ok {
		newDest = make(map[string]interface{})
		dest[ikey] = newDest
	}
	InsertKeyInMap(restKey, value, newDest)
}
