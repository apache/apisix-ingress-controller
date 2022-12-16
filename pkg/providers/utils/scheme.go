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
	apisixv1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

var schemeToPortMaps = map[string]int{
	apisixv1.SchemeHTTP:  80,
	apisixv1.SchemeHTTPS: 443,
	apisixv1.SchemeGRPC:  80,
	apisixv1.SchemeGRPCS: 443,
}

// SchemeToPort scheme converts to the default port
// ref https://github.com/apache/apisix/blob/c5fc10d9355a0c177a7532f01c77745ff0639a7f/apisix/upstream.lua#L167-L172
func SchemeToPort(schema string) int {
	if val, ok := schemeToPortMaps[schema]; ok {
		return val
	}
	return 80
}
