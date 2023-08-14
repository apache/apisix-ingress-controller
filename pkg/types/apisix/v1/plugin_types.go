// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package v1

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/incubator4/go-resty-expr/expr"
)

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

// CSRfConfig is the rule config for csrf plugin.
// +k8s:deepcopy-gen=true
type CSRFConfig struct {
	Key string `json:"key"`
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
	Key                 string `json:"key" yaml:"key"`
	Secret              string `json:"secret,omitempty" yaml:"secret,omitempty"`
	PublicKey           string `json:"public_key,omitempty" yaml:"public_key,omitempty"`
	PrivateKey          string `json:"private_key" yaml:"private_key,omitempty"`
	Algorithm           string `json:"algorithm,omitempty" yaml:"algorithm,omitempty"`
	Exp                 int64  `json:"exp,omitempty" yaml:"exp,omitempty"`
	Base64Secret        bool   `json:"base64_secret,omitempty" yaml:"base64_secret,omitempty"`
	LifetimeGracePeriod int64  `json:"lifetime_grace_period,omitempty" yaml:"lifetime_grace_period,omitempty"`
}

// HMACAuthConsumerConfig is the rule config for hmac-auth plugin
// used in Consumer object.
// +k8s:deepcopy-gen=true
type HMACAuthConsumerConfig struct {
	AccessKey           string   `json:"access_key" yaml:"access_key"`
	SecretKey           string   `json:"secret_key" yaml:"secret_key"`
	Algorithm           string   `json:"algorithm,omitempty" yaml:"algorithm,omitempty"`
	ClockSkew           int64    `json:"clock_skew,omitempty" yaml:"clock_skew,omitempty"`
	SignedHeaders       []string `json:"signed_headers,omitempty" yaml:"signed_headers,omitempty"`
	KeepHeaders         bool     `json:"keep_headers,omitempty" yaml:"keep_headers,omitempty"`
	EncodeURIParams     bool     `json:"encode_uri_params,omitempty" yaml:"encode_uri_params,omitempty"`
	ValidateRequestBody bool     `json:"validate_request_body,omitempty" yaml:"validate_request_body,omitempty"`
	MaxReqBody          int64    `json:"max_req_body,omitempty" yaml:"max_req_body,omitempty"`
}

// LDAPAuthConsumerConfig is the rule config for ldap-auth plugin
// used in Consumer object.
// +k8s:deepcopy-gen=true
type LDAPAuthConsumerConfig struct {
	UserDN string `json:"user_dn"`
}

// BasicAuthRouteConfig is the rule config for basic-auth plugin
// used in Route object.
// +k8s:deepcopy-gen=true
type BasicAuthRouteConfig struct{}

// WolfRBACConsumerConfig is the rule config for wolf-rbac plugin
// used in Consumer object.
// +k8s:deepcopy-gen=true
type WolfRBACConsumerConfig struct {
	Server       string `json:"server,omitempty"`
	Appid        string `json:"appid,omitempty"`
	HeaderPrefix string `json:"header_prefix,omitempty"`
}

// RewriteConfig is the rule config for proxy-rewrite plugin.
// +k8s:deepcopy-gen=true
type RewriteConfig struct {
	RewriteTarget      string   `json:"uri,omitempty"`
	RewriteTargetRegex []string `json:"regex_uri,omitempty"`
	Headers            Headers  `json:"headers,omitempty"`
}

// ResponseRewriteConfig is the rule config for response-rewrite plugin.
// +k8s:deepcopy-gen=true
type ResponseRewriteConfig struct {
	StatusCode   int                 `json:"status_code,omitempty"`
	Body         string              `json:"body,omitempty"`
	BodyBase64   bool                `json:"body_base64,omitempty"`
	Headers      Headers             `json:"headers,omitempty"`
	LuaRestyExpr []expr.Expr         `json:"vars,omitempty"`
	Filters      []map[string]string `json:"filters,omitempty"`
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

// RequestMirror is the rule config for proxy-mirror plugin.
// +k8s:deepcopy-gen=true
type RequestMirror struct {
	Host string `json:"host"`
}

type Headers map[string]any

func (p *Headers) DeepCopyInto(out *Headers) {
	b, _ := json.Marshal(&p)
	_ = json.Unmarshal(b, out)
}

func (p *Headers) DeepCopy() *Headers {
	if p == nil {
		return nil
	}
	out := new(Headers)
	p.DeepCopyInto(out)
	return out
}

func (p *Headers) Add(headersToAdd []string) {
	if p == nil {
		return
	}
	if headersToAdd != nil {
		addedHeader := make([]string, 0)
		for _, h := range headersToAdd {
			kv := strings.Split(h, ":")
			if len(kv) < 2 {
				continue
			}
			addedHeader = append(addedHeader, fmt.Sprintf("%s:%s", kv[0], kv[1]))
		}
		(*p)["add"] = addedHeader
	}
}

func (p *Headers) GetAddedHeaders() []string {
	if p == nil || (*p)["add"] == nil {
		return nil
	}
	addedheaders, ok := (*p)["add"].([]string)
	if ok {
		return addedheaders
	}
	return nil
}

func (p *Headers) Set(headersToSet []string) {
	if p == nil {
		return
	}
	if headersToSet != nil {
		setHeaders := make(map[string]string, 0)
		for _, h := range headersToSet {
			kv := strings.Split(h, ":")
			if len(kv) < 2 {
				continue
			}
			setHeaders[kv[0]] = kv[1]
		}
		(*p)["set"] = setHeaders
	}
}

func (p *Headers) GetSetHeaders() map[string]string {
	if p == nil || (*p)["set"] == nil {
		return nil
	}
	addedheaders, ok := (*p)["set"].(map[string]string)
	if ok {
		return addedheaders
	}
	return nil
}

func (p *Headers) Remove(headersToRemove []string) {
	if p == nil {
		return
	}
	if headersToRemove != nil {
		removeHeaders := make([]string, 0)
		removeHeaders = append(removeHeaders, headersToRemove...)
		(*p)["remove"] = removeHeaders
	}
}

func (p *Headers) GetRemovedHeaders() []string {
	if p == nil || (*p)["remove"] == nil {
		return nil
	}
	removedHeaders, ok := (*p)["remove"].([]string)
	if ok {
		return removedHeaders
	}
	return nil
}
