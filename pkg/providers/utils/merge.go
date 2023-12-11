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

// MergeMaps will iterate recursively in src map and copy the fields over to dest
func MergeMaps(src, dest map[string]interface{}) {
	for key, val := range src {
		//If destination map already has this key then recursively
		//call merge with src[key] and dest[key]
		if dest[key] != nil {
			switch v := val.(type) {
			case map[string]interface{}:
				destMap, ok := dest[key].(map[string]interface{})
				if !ok {
					destMap = make(map[string]interface{})
				}
				MergeMaps(v, destMap)
			default:
				dest[key] = src[key]
			}
		}
	}
}
