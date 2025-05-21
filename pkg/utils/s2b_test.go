// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
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
	"reflect"
	"testing"
)

func TestString2Byte(t *testing.T) {
	type args struct {
		raw string
	}
	tests := []struct {
		name  string
		args  args
		wantB []byte
	}{
		{
			name: "test-1",
			args: args{
				raw: "a",
			},
			wantB: []byte{'a'},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotB := String2Byte(tt.args.raw); !reflect.DeepEqual(gotB, tt.wantB) {
				t.Errorf("String2byte() = %v, want %v", gotB, tt.wantB)
			}
		})
	}
}
