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

	// helper function
	isLastLabel := func(labels *[]string, idx int) bool {
		if labels == &lLabels {
			return idx == lLength-1
		}
		if labels == &rLabels {
			return idx == rLength-1
		}
		return false
	}

	// reverse matching, hostname seq like ["moc", "xisipa", "ssergni"]
	for i := 0; i < lLength || i < rLength; i++ {
		if isLastLabel(&lLabels, i) {
			if lLabels[i] == "*" {
				// wildcard prefix matched
				break
			}
			if !isLastLabel(&rLabels, i) {
				// "ingress.apisix.com" "apisix.com" mismatched.
				return false
			}
		}
		if isLastLabel(&rLabels, i) {
			if rLabels[i] == "*" {
				break
			}
			if !isLastLabel(&lLabels, i) {
				return false
			}
		}

		if lLabels[i] != rLabels[i] {
			return false
		}
	}
	return true
}
