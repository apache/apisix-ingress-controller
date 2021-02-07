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
package state

import (
	"testing"

	"github.com/apache/apisix-ingress-controller/pkg/seven/utils"
)

type school struct {
	*province
	Name    string `json:"name"`
	Address string `json:"address"`
}

type province struct {
	Location string `json:"location"`
}

func Test_diff(t *testing.T) {
	//p1 := &province{Location: "jiangsu"}
	p2 := &province{Location: "zh"}
	s1 := &school{Name: "hello", Address: "this is a address"}
	s2 := &school{Name: "hello", Address: "this is a address", province: p2}
	t.Log(s1)
	t.Log(s2)
	if d, err := utils.Diff(s1, s2); err != nil {
		t.Log(err.Error())
	} else {
		//t.Logf("s1 vs s2 hasDiff ? %v", d)
		t.Log(d)
		for _, delta := range d.Deltas() {
			t.Log(delta.Similarity())
		}

	}

}
