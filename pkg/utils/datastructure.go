// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package utils

import (
	"strings"
)

// InsertKeyInMap takes a dot separated string and recursively goes inside the destination
// to fill the value
func InsertKeyInMap(key string, value any, dest map[string]any) {
	if key == "" {
		return
	}
	keys := strings.SplitN(key, ".", 2)
	// base condition. the length of keys will be atleast 1
	if len(keys) < 2 {
		dest[keys[0]] = value
		return
	}

	ikey := keys[0]
	restKey := keys[1]
	if dest[ikey] == nil {
		dest[ikey] = make(map[string]any)
	}
	newDest, ok := dest[ikey].(map[string]any)
	if !ok {
		newDest = make(map[string]any)
		dest[ikey] = newDest
	}
	InsertKeyInMap(restKey, value, newDest)
}

func DedupComparable[T comparable](s []T) []T {
	var keys = make(map[T]struct{})
	var results []T
	for _, item := range s {
		if _, ok := keys[item]; !ok {
			keys[item] = struct{}{}
			results = append(results, item)
		}
	}
	return results
}

func AppendFunc[T any](s []T, keep func(v T) bool, values ...T) []T {
	for _, v := range values {
		if keep(v) {
			s = append(s, v)
		}
	}
	return s
}

func Filter[T any](s []T, keep func(v T) bool) []T {
	return AppendFunc(make([]T, 0), keep, s...)
}
