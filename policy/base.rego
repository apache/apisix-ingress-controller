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
# and apisix.apache.org/v2beta1 - use apisix.apache.org/v2beta2 instead
_deny = msg {
	input.kind == "ApisixRoute"
	apis := ["apisix.apache.org/v1", "apisix.apache.org/v2alpha1", "apisix.apache.org/v2beta1"]
	input.apiVersion == apis[_]
	msg := sprintf("%s/%s: API %s has been deprecated, use apisix.apache.org/v2beta2 instead.", [input.kind, input.metadata.name, input.apiVersion])
}

# From apisix.apache.org/v2beta2 the ApisixRoute's spec.http.backend field has been removed
_deny = msg {
	input.apiVersion == "apisix.apache.org/v2beta2"
	input.kind == "ApisixRoute"
	some i
	input.spec.http[i].backend
	msg := sprintf("%s/%s: %s field http.backend has been removed, use http.backends instead.", [input.kind, input.metadata.name, input.spec.http[i].name])
}
