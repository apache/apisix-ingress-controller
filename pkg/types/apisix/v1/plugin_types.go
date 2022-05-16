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
package v1

// TrafficSplitConfig is the config of traffic-split plugin.
// +k8s:deepcopy-gen=true
type TrafficSplitConfig struct {
	Rules []TrafficSplitConfigRule `json:"rules"`
}

// TrafficSplitConfigRule is the rule config in traffic-split plugin config.
// +k8s:deepcopy-gen=true
type TrafficSplitConfigRule struct {
	WeightedUpstreams []TrafficSplitConfigRuleWeightedUpstream `json:"weighted_upstreams"`
}

// TrafficSplitConfigRuleWeightedUpstream is the weighted upstream config in
// the traffic split plugin rule.
// +k8s:deepcopy-gen=true
type TrafficSplitConfigRuleWeightedUpstream struct {
	UpstreamID string `json:"upstream_id,omitempty"`
	Weight     int    `json:"weight"`
}

// IPRestrictConfig is the rule config for ip-restriction plugin.
// +k8s:deepcopy-gen=true
type IPRestrictConfig struct {
	Allowlist []string `json:"whitelist,omitempty"`
	Blocklist []string `json:"blacklist,omitempty"`
}

// CorsConfig is the rule config for cors plugin.
// +k8s:deepcopy-gen=true
type CorsConfig struct {
	AllowOrigins string `json:"allow_origins,omitempty"`
	AllowMethods string `json:"allow_methods,omitempty"`
	AllowHeaders string `json:"allow_headers,omitempty"`
}

// KeyAuthConsumerConfig is the rule config for key-auth plugin
// used in Consumer object.
// +k8s:deepcopy-gen=true
type KeyAuthConsumerConfig struct {
	Key string `json:"key"`
}

// KeyAuthRouteConfig is the rule config for key-auth plugin
// used in Route object.
type KeyAuthRouteConfig struct {
	Header string `json:"header,omitempty"`
}

// BasicAuthConsumerConfig is the rule config for basic-auth plugin
// used in Consumer object.
// +k8s:deepcopy-gen=true
type BasicAuthConsumerConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// JwtAuthConsumerConfig is the rule config for jwt-auth plugin
// used in Consumer object.
// +k8s:deepcopy-gen=true
type JwtAuthConsumerConfig struct {
	Key          string `json:"key" yaml:"key"`
	Secret       string `json:"secret,omitempty" yaml:"secret,omitempty"`
	PublicKey    string `json:"public_key,omitempty" yaml:"public_key,omitempty"`
	PrivateKey   string `json:"private_key" yaml:"private_key,omitempty"`
	Algorithm    string `json:"algorithm,omitempty" yaml:"algorithm,omitempty"`
	Exp          int64  `json:"exp,omitempty" yaml:"exp,omitempty"`
	Base64Secret bool   `json:"base64_secret,omitempty" yaml:"base64_secret,omitempty"`
}

// BasicAuthRouteConfig is the rule config for basic-auth plugin
// used in Route object.
// +k8s:deepcopy-gen=true
type BasicAuthRouteConfig struct{}

// RewriteConfig is the rule config for proxy-rewrite plugin.
// +k8s:deepcopy-gen=true
type RewriteConfig struct {
	RewriteTarget      string   `json:"uri,omitempty"`
	RewriteTargetRegex []string `json:"regex_uri,omitempty"`
}

// RedirectConfig is the rule config for redirect plugin.
// +k8s:deepcopy-gen=true
type RedirectConfig struct {
	HttpToHttps bool   `json:"http_to_https,omitempty"`
	URI         string `json:"uri,omitempty"`
	RetCode     int    `json:"ret_code,omitempty"`
}

// ForwardAuthConfig is the rule config for forward-auth plugin.
// +k8s:deepcopy-gen=true
type ForwardAuthConfig struct {
	URI             string   `json:"uri"`
	SSLVerify       bool     `json:"ssl_verify"`
	RequestHeaders  []string `json:"request_headers,omitempty"`
	UpstreamHeaders []string `json:"upstream_headers,omitempty"`
	ClientHeaders   []string `json:"client_headers,omitempty"`
}

// BasicAuthConfig is the rule config for basic-auth plugin.
// +k8s:deepcopy-gen=true
type BasicAuthConfig struct {
}

// KeyAuthConfig is the rule config for key-auth plugin.
// +k8s:deepcopy-gen=true
type KeyAuthConfig struct {
}
