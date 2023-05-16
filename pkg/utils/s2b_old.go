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

//go:build !go1.20
// +build !go1.20

package utils

import (
	"reflect"
	"unsafe"
)

// String2Byte converts a string to a byte slice without memory allocation.
//
// Note since Go 1.20, the reflect.StringHeader and reflect.SliceHeader types
// will be depreciated and not recommended to be used.
func String2Byte(raw string) (b []byte) {

	strHeader := (*reflect.StringHeader)(unsafe.Pointer(&raw))
	bHeader := (*reflect.SliceHeader)(unsafe.Pointer(&b))

	bHeader.Data = strHeader.Data
	bHeader.Cap = strHeader.Len
	bHeader.Len = strHeader.Len
	return b
}
