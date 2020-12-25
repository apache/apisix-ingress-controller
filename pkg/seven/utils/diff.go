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
package utils

import (
	"encoding/json"
	"github.com/golang/glog"
	"github.com/yudai/gojsondiff"
)

var (
	differ = gojsondiff.New()
)

func HasDiff(a, b interface{}) (bool, error) {
	aJSON, err := json.Marshal(a)
	if err != nil {
		return false, err
	}
	bJSON, err := json.Marshal(b)
	if err != nil {
		return false, err
	}
	if d, err := differ.Compare(aJSON, bJSON); err != nil {
		return false, err
	} else {
		glog.V(2).Info(d.Deltas())
		return d.Modified(), nil
	}
}

func Diff(a, b interface{}) (gojsondiff.Diff, error) {
	aJSON, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	bJSON, err := json.Marshal(b)
	if err != nil {
		return nil, err
	}
	if d, err := differ.Compare(aJSON, bJSON); err != nil {
		return nil, err
	} else {
		return d, nil
	}
}
