// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
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

// IsHostnameMatch follow GatewayAPI specification to match listenr and route hostname.
// FYI: https://github.com/kubernetes-sigs/gateway-api/blob/a596211672a5aed54881862dc87c8c1cad9c7bd8/apis/v1beta1/gateway_types.go#L154
func IsHostnameMatch(listener, route string) bool {
	if listener == "" || route == "" {
		return true
	}

	// reverse hostname from "ingress.apisix.com" to "moc.xisipa.ssergni" for matching
	l := ReverseString(listener)
	r := ReverseString(route)

	lLabels := strings.Split(l, ".")
	rLabels := strings.Split(r, ".")
	lLength := len(lLabels)
	rLength := len(rLabels)

	// reverse matching, hostname seq like ["moc", "xisipa", "ssergni"]
	for i, j := 0, 0; i < lLength && j < rLength; i, j = i+1, j+1 {
		if i == (lLength - 1) {
			// wildcard prefix matched
			if lLabels[i] == "*" {
				break
			}

			// "ingress.apisix.com" "apisix.com" mismatched.
			if j != (rLength - 1) {
				return false
			}
		}

		if j == (rLength - 1) {
			if rLabels[j] == "*" {
				break
			}

			if i != (lLength - 1) {
				return false
			}
		}

		if lLabels[i] != rLabels[j] {
			return false
		}
	}

	return true
}
