#
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
package main

deny[msg] {
	input.apiVersion == "v1"
	input.kind == "List"
	obj := input.items[_]
	msg := _deny with input as obj
}

deny[msg] {
	input.apiVersion != "v1"
	input.kind != "List"
	msg := _deny
}

# Base on https://github.com/apache/apisix-ingress-controller/blob/master/CHANGELOG.md#130
# ApisixRoute under apisix.apache.org/v1, apisix.apache.org/v2alpha1
# and apisix.apache.org/v2beta1 - use apisix.apache.org/v2beta3 instead
_deny = msg {
	input.kind == "ApisixRoute"
	apis := ["apisix.apache.org/v1", "apisix.apache.org/v2alpha1", "apisix.apache.org/v2beta1", "apisix.apache.org/v2beta2"]
	input.apiVersion == apis[_]
	msg := sprintf("%s/%s: API %s has been deprecated, use apisix.apache.org/v2beta3 instead.", [input.kind, input.metadata.name, input.apiVersion])
}

# From apisix.apache.org/v2beta3 the ApisixRoute's spec.http.backend field has been removed
_deny = msg {
	input.apiVersion == "apisix.apache.org/v2beta3"
	input.kind == "ApisixRoute"
	some i
	input.spec.http[i].backend
	msg := sprintf("%s/%s: %s field http.backend has been removed, use http.backends instead.", [input.kind, input.metadata.name, input.spec.http[i].name])
}
