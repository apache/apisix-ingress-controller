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

import "regexp"

var hostDef = "^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$"
var hostDefRegex = regexp.MustCompile(hostDef)

// MatchHostDef checks that host matches host's shcema
// ref to : https://github.com/apache/apisix/blob/c5fc10d9355a0c177a7532f01c77745ff0639a7f/apisix/schema_def.lua#L40
// ref to : https://github.com/kubernetes/kubernetes/blob/976a940f4a4e84fe814583848f97b9aafcdb083f/staging/src/k8s.io/apimachinery/pkg/util/validation/validation.go#L205
// They define regex differently, but k8s's dns is more accurate
func MatchHostDef(host string) bool {
	return hostDefRegex.MatchString(host)
}
