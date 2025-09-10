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

package v2

import (
	"time"

	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type (
	// ApisixRouteConditionType is a type of condition for a route.
	ApisixRouteConditionType = gatewayv1.RouteConditionType
	// ApisixRouteConditionReason is a reason for a route condition.
	ApisixRouteConditionReason = gatewayv1.RouteConditionReason
)

const (
	ConditionTypeAccepted      ApisixRouteConditionType   = gatewayv1.RouteConditionAccepted
	ConditionReasonAccepted    ApisixRouteConditionReason = gatewayv1.RouteReasonAccepted
	ConditionReasonInvalidSpec ApisixRouteConditionReason = "InvalidSpec"
	ConditionReasonSyncFailed  ApisixRouteConditionReason = "SyncFailed"
)

const (
	// DefaultUpstreamTimeout represents the default connect,
	// read and send timeout (in seconds) with upstreams.
	DefaultUpstreamTimeout = 60 * time.Second

	DefaultWeight = 100
)

const (
	ResolveGranularityService  = "service"
	ResolveGranularityEndpoint = "endpoint"
)

const (
	// OpEqual means the equal ("==") operator in nginxVars.
	OpEqual = "Equal"
	// OpNotEqual means the not equal ("~=") operator in nginxVars.
	OpNotEqual = "NotEqual"
	// OpGreaterThan means the greater than (">") operator in nginxVars.
	OpGreaterThan = "GreaterThan"
	// OpGreaterThanEqual means the greater than (">=") operator in nginxVars.
	OpGreaterThanEqual = "GreaterThanEqual"
	// OpLessThan means the less than ("<") operator in nginxVars.
	OpLessThan = "LessThan"
	// OpLessThanEqual means the less than equal ("<=") operator in nginxVars.
	OpLessThanEqual = "LessThanEqual"
	// OpRegexMatch means the regex match ("~~") operator in nginxVars.
	OpRegexMatch = "RegexMatch"
	// OpRegexNotMatch means the regex not match ("!~~") operator in nginxVars.
	OpRegexNotMatch = "RegexNotMatch"
	// OpRegexMatchCaseInsensitive means the regex match "~*" (case insensitive mode) operator in nginxVars.
	OpRegexMatchCaseInsensitive = "RegexMatchCaseInsensitive"
	// OpRegexNotMatchCaseInsensitive means the regex not match "!~*" (case insensitive mode) operator in nginxVars.
	OpRegexNotMatchCaseInsensitive = "RegexNotMatchCaseInsensitive"
	// OpIn means the in operator ("in") in nginxVars.
	OpIn = "In"
	// OpNotIn means the not in operator ("NotIn") in nginxVars.
	OpNotIn = "NotIn"

	// ScopeQuery means the route match expression subject is in the querystring.
	ScopeQuery = "Query"
	// ScopeHeader means the route match expression subject is in request headers.
	ScopeHeader = "Header"
	// ScopePath means the route match expression subject is the uri path.
	ScopePath = "Path"
	// ScopeCookie means the route match expression subject is in cookie.
	ScopeCookie = "Cookie"
	// ScopeVariable means the route match expression subject is in variable.
	ScopeVariable = "Variable"
)

const (
	// SchemeHTTP represents the HTTP protocol.
	SchemeHTTP = "http"
	// SchemeGRPC represents the GRPC protocol.
	SchemeGRPC = "grpc"
	// SchemeHTTPS represents the HTTPS protocol.
	SchemeHTTPS = "https"
	// SchemeGRPCS represents the GRPCS protocol.
	SchemeGRPCS = "grpcs"
	// SchemeTCP represents the TCP protocol.
	SchemeTCP = "tcp"
	// SchemeUDP represents the UDP protocol.
	SchemeUDP = "udp"
)

const (
	// HashOnVars means the hash scope is variable.
	HashOnVars = "vars"
	// HashOnVarsCombination means the hash scope is the
	// variable combination.
	HashOnVarsCombination = "vars_combinations"
	// HashOnHeader means the hash scope is HTTP request
	// headers.
	HashOnHeader = "header"
	// HashOnCookie means the hash scope is HTTP Cookie.
	HashOnCookie = "cookie"
	// HashOnConsumer means the hash scope is APISIX consumer.
	HashOnConsumer = "consumer"

	// LbRoundRobin is the round robin load balancer.
	LbRoundRobin = "roundrobin"
	// LbConsistentHash is the consistent hash load balancer.
	LbConsistentHash = "chash"
	// LbEwma is the ewma load balancer.
	LbEwma = "ewma"
	// LbLeaseConn is the least connection load balancer.
	LbLeastConn = "least_conn"
)

const (
	// PassHostPass represents pass option for pass_host Upstream settings.
	PassHostPass = "pass"
	// PassHostPass represents node option for pass_host Upstream settings.
	PassHostNode = "node"
	// PassHostPass represents rewrite option for pass_host Upstream settings.
	PassHostRewrite = "rewrite"
)

const (
	// ExternalTypeDomain type is a domain
	// +k8s:deepcopy-gen=false
	ExternalTypeDomain ApisixUpstreamExternalType = "Domain"

	// ExternalTypeService type is a K8s ExternalName service
	// +k8s:deepcopy-gen=false
	ExternalTypeService ApisixUpstreamExternalType = "Service"
)

const (
	// HealthCheckHTTP represents the HTTP kind health check.
	HealthCheckHTTP = "http"
	// HealthCheckHTTPS represents the HTTPS kind health check.
	HealthCheckHTTPS = "https"
	// HealthCheckTCP represents the TCP kind health check.
	HealthCheckTCP = "tcp"

	// HealthCheckMaxConsecutiveNumber is the max number for
	// the consecutive success/failure in upstream health check.
	HealthCheckMaxConsecutiveNumber = 254
	// ActiveHealthCheckMinInterval is the minimum interval for
	// the active health check.
	ActiveHealthCheckMinInterval = time.Second
)

var schemeToPortMaps = map[string]int{
	SchemeHTTP:  80,
	SchemeHTTPS: 443,
	SchemeGRPC:  80,
	SchemeGRPCS: 443,
}

// SchemeToPort scheme converts to the default port
// ref https://github.com/apache/apisix/blob/c5fc10d9355a0c177a7532f01c77745ff0639a7f/apisix/upstream.lua#L167-L172
func SchemeToPort(schema string) int {
	if val, ok := schemeToPortMaps[schema]; ok {
		return val
	}
	return 80
}
