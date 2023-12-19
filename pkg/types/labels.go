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
package types

import (
	"fmt"
	"strings"
)

// Labels contains a series of labels.
type Labels map[string]string

// IsSubsetOf checks whether the current Labels is the subset of
// the passed Labels.
func (s Labels) IsSubsetOf(f Labels) bool {
	if len(s) == 0 {
		// Empty labels matches everything.
		return true
	}
	for k, v := range s {
		if vv, ok := f[k]; !ok || vv != v {
			return false
		}
	}
	return true
}

// MultiValueLabels contains a series of labels with multiple values.
type MultiValueLabels map[string][]string

func (s MultiValueLabels) BuildQuery() []string {
	query := []string{}
	for k, v := range s {
		query = append(query, fmt.Sprintf("%s in (%s)", k, strings.Join(v, ",")))
	}
	return query
}

// IsSubsetOf checks whether the current Labels is the subset of
// the passed Labels.
func (s MultiValueLabels) IsSubsetOf(f Labels) bool {
	if len(s) == 0 {
		// Empty labels matches everything.
		return true
	}
	for key, vals := range s {
		if val, ok := f[key]; !ok || !arrContains(vals, val) {
			return false
		}
	}
	return true
}

func arrContains(arr []string, ele string) bool {
	for _, e := range arr {
		if e == ele {
			return true
		}
	}
	return false
}
