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

package adc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/incubator4/go-resty-expr/expr"
)

const (
	TypeRoute          = "route"
	TypeService        = "service"
	TypeConsumer       = "consumer"
	TypeSSL            = "ssl"
	TypeGlobalRule     = "global_rule"
	TypePluginMetadata = "plugin_metadata"
)

type Object interface {
	GetLabels() map[string]string
}

// +k8s:deepcopy-gen=true
type Metadata struct {
	ID     string            `json:"id,omitempty" yaml:"id,omitempty"`
	Name   string            `json:"name,omitempty" yaml:"name,omitempty"`
	Desc   string            `json:"description,omitempty" yaml:"description,omitempty"`
	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

func (m *Metadata) GetLabels() map[string]string { return m.Labels }

type Resources struct {
	ConsumerGroups []*ConsumerGroup `json:"consumer_groups,omitempty" yaml:"consumer_groups,omitempty"`
	Consumers      []*Consumer      `json:"consumers,omitempty" yaml:"consumers,omitempty"`
	GlobalRules    GlobalRule       `json:"global_rules,omitempty" yaml:"global_rules,omitempty"`
	PluginMetadata PluginMetadata   `json:"plugin_metadata,omitempty" yaml:"plugin_metadata,omitempty"`
	Services       []*Service       `json:"services,omitempty" yaml:"services,omitempty"`
	SSLs           []*SSL           `json:"ssls,omitempty" yaml:"ssls,omitempty"`
}

type GlobalRule Plugins

func (g *GlobalRule) DeepCopy() GlobalRule {
	original := Plugins(*g)
	copied := original.DeepCopy()
	return GlobalRule(copied)
}

// +k8s:deepcopy-gen=true
type GlobalRuleItem struct {
	Metadata `json:",inline" yaml:",inline"`

	Plugins Plugins `json:"plugins" yaml:"plugins"`
}

type PluginMetadata Plugins

func (p *PluginMetadata) DeepCopy() PluginMetadata {
	original := Plugins(*p)
	copied := original.DeepCopy()
	return PluginMetadata(copied)
}

// +k8s:deepcopy-gen=true
type ConsumerGroup struct {
	Metadata  `json:",inline" yaml:",inline"`
	Consumers []Consumer `json:"consumers,omitempty" yaml:"consumers,omitempty"`
	Name      string     `json:"name" yaml:"name"`
	Plugins   Plugins    `json:"plugins" yaml:"plugins"`
}

// +k8s:deepcopy-gen=true
type Consumer struct {
	Metadata `json:",inline" yaml:",inline"`

	Credentials []Credential `json:"credentials,omitempty" yaml:"credentials,omitempty"`
	Plugins     Plugins      `json:"plugins,omitempty" yaml:"plugins,omitempty"`
	Username    string       `json:"username" yaml:"username"`
}

// +k8s:deepcopy-gen=true
type Credential struct {
	Metadata `json:",inline" yaml:",inline"`

	Config Plugins `json:"config" yaml:"config"`
	Type   string  `json:"type" yaml:"type"`
}

// +k8s:deepcopy-gen=true
type Service struct {
	Metadata `json:",inline" yaml:",inline"`

	Hosts           []string       `json:"hosts,omitempty" yaml:"hosts,omitempty"`
	PathPrefix      string         `json:"path_prefix,omitempty" yaml:"path_prefix,omitempty"`
	Plugins         Plugins        `json:"plugins,omitempty" yaml:"plugins,omitempty"`
	Routes          []*Route       `json:"routes,omitempty" yaml:"routes,omitempty"`
	StreamRoutes    []*StreamRoute `json:"stream_routes,omitempty" yaml:"stream_routes,omitempty"`
	StripPathPrefix *bool          `json:"strip_path_prefix,omitempty" yaml:"strip_path_prefix,omitempty"`
	Upstream        *Upstream      `json:"upstream,omitempty" yaml:"upstream,omitempty"`
	Upstreams       []*Upstream    `json:"upstreams,omitempty" yaml:"upstreams,omitempty"`
}

// +k8s:deepcopy-gen=true
type Route struct {
	Metadata `json:",inline" yaml:",inline"`

	EnableWebsocket *bool    `json:"enable_websocket,omitempty" yaml:"enable_websocket,omitempty"`
	FilterFunc      string   `json:"filter_func,omitempty" yaml:"filter_func,omitempty"`
	Hosts           []string `json:"hosts,omitempty" yaml:"hosts,omitempty"`
	Methods         []string `json:"methods,omitempty" yaml:"methods,omitempty"`
	Plugins         Plugins  `json:"plugins,omitempty" yaml:"plugins,omitempty"`
	Priority        *int64   `json:"priority,omitempty" yaml:"priority,omitempty"`
	RemoteAddrs     []string `json:"remote_addrs,omitempty" yaml:"remote_addrs,omitempty"`
	Timeout         *Timeout `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Uris            []string `json:"uris" yaml:"uris"`
	Vars            Vars     `json:"vars,omitempty" yaml:"vars,omitempty"`
}

type Timeout struct {
	Connect int `json:"connect"`
	Read    int `json:"read"`
	Send    int `json:"send"`
}

// +k8s:deepcopy-gen=true
type StreamRoute struct {
	Metadata `json:",inline" yaml:",inline"`

	Plugins    Plugins `json:"plugins,omitempty"`
	RemoteAddr string  `json:"remote_addr,omitempty"`
	ServerAddr string  `json:"server_addr,omitempty"`
	ServerPort int32   `json:"server_port,omitempty"`
	SNI        string  `json:"sni,omitempty"`
}

// +k8s:deepcopy-gen=true
type Upstream struct {
	Metadata `json:",inline" yaml:",inline"`

	HashOn       string        `json:"hash_on,omitempty" yaml:"hash_on,omitempty"`
	Key          string        `json:"key,omitempty" yaml:"key,omitempty"`
	Nodes        UpstreamNodes `json:"nodes" yaml:"nodes"`
	PassHost     string        `json:"pass_host,omitempty" yaml:"pass_host,omitempty"`
	Retries      *int64        `json:"retries,omitempty" yaml:"retries,omitempty"`
	RetryTimeout *float64      `json:"retry_timeout,omitempty" yaml:"retry_timeout,omitempty"`
	Scheme       string        `json:"scheme,omitempty" yaml:"scheme,omitempty"`
	ServiceName  string        `json:"service_name,omitempty" yaml:"service_name,omitempty"`
	Timeout      *Timeout      `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Type         UpstreamType  `json:"type,omitempty" yaml:"type,omitempty"`
	UpstreamHost string        `json:"upstream_host,omitempty" yaml:"upstream_host,omitempty"`

	Checks *UpstreamHealthCheck `json:"checks,omitempty" yaml:"checks,omitempty"`
	TLS    *ClientTLS           `json:"tls,omitempty" yaml:"tls,omitempty"`
	// for Service Discovery
	DiscoveryType string            `json:"discovery_type,omitempty" yaml:"discovery_type,omitempty"`
	DiscoveryArgs map[string]string `json:"discovery_args,omitempty" yaml:"discovery_args,omitempty"`
}

// UpstreamHealthCheck defines the active and/or passive health check for an Upstream,
// with the upstream health check feature, pods can be kicked out or joined in quickly,
// if the feedback of Kubernetes liveness/readiness probe is long.
// +k8s:deepcopy-gen=true
type UpstreamHealthCheck struct {
	Active  *UpstreamActiveHealthCheck  `json:"active" yaml:"active"`
	Passive *UpstreamPassiveHealthCheck `json:"passive,omitempty" yaml:"passive,omitempty"`
}

// ClientTLS is tls cert and key use in mTLS
// +k8s:deepcopy-gen=true
type ClientTLS struct {
	Cert string `json:"client_cert,omitempty" yaml:"client_cert,omitempty"`
	Key  string `json:"client_key,omitempty" yaml:"client_key,omitempty"`
}

// UpstreamActiveHealthCheck defines the active upstream health check configuration.
// +k8s:deepcopy-gen=true
type UpstreamActiveHealthCheck struct {
	Type               string                             `json:"type,omitempty" yaml:"type,omitempty"`
	Timeout            int                                `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Concurrency        int                                `json:"concurrency,omitempty" yaml:"concurrency,omitempty"`
	Host               string                             `json:"host,omitempty" yaml:"host,omitempty"`
	Port               int32                              `json:"port,omitempty" yaml:"port,omitempty"`
	HTTPPath           string                             `json:"http_path,omitempty" yaml:"http_path,omitempty"`
	HTTPSVerifyCert    bool                               `json:"https_verify_cert,omitempty" yaml:"https_verify_cert,omitempty"`
	HTTPRequestHeaders []string                           `json:"req_headers,omitempty" yaml:"req_headers,omitempty"`
	Healthy            UpstreamActiveHealthCheckHealthy   `json:"healthy,omitempty" yaml:"healthy,omitempty"`
	Unhealthy          UpstreamActiveHealthCheckUnhealthy `json:"unhealthy,omitempty" yaml:"unhealthy,omitempty"`
}

// UpstreamPassiveHealthCheck defines the passive health check configuration for an upstream.
// Passive health checks rely on analyzing live traffic to determine the health status of upstream nodes.
// +k8s:deepcopy-gen=true
type UpstreamPassiveHealthCheck struct {
	// Type is the passive health check type. For example: `http`.
	Type string `json:"type,omitempty" yaml:"type,omitempty"`
	// Healthy defines the conditions under which an upstream node is considered healthy.
	Healthy UpstreamPassiveHealthCheckHealthy `json:"healthy,omitempty" yaml:"healthy,omitempty"`
	// Unhealthy defines the conditions under which an upstream node is considered unhealthy.
	Unhealthy UpstreamPassiveHealthCheckUnhealthy `json:"unhealthy,omitempty" yaml:"unhealthy,omitempty"`
}

// UpstreamActiveHealthCheckHealthy defines the conditions used to actively determine whether an upstream node is healthy.
// +k8s:deepcopy-gen=true
type UpstreamActiveHealthCheckHealthy struct {
	UpstreamPassiveHealthCheckHealthy `json:",inline" yaml:",inline"`

	// Interval defines the time interval for checking targets, in seconds.
	Interval int `json:"interval,omitempty" yaml:"interval,omitempty"`
}

// UpstreamPassiveHealthCheckHealthy defines the conditions used to passively determine whether an upstream node is healthy.
// +k8s:deepcopy-gen=true
type UpstreamPassiveHealthCheckHealthy struct {
	HTTPStatuses []int `json:"http_statuses,omitempty" yaml:"http_statuses,omitempty"`
	Successes    int   `json:"successes,omitempty" yaml:"successes,omitempty"`
}

// UpstreamPassiveHealthCheckUnhealthy defines the conditions used to passively determine whether an upstream node is unhealthy.
// +k8s:deepcopy-gen=true
type UpstreamPassiveHealthCheckUnhealthy struct {
	HTTPStatuses []int `json:"http_statuses,omitempty" yaml:"http_statuses,omitempty"`
	HTTPFailures int   `json:"http_failures,omitempty" yaml:"http_failures,omitempty"`
	TCPFailures  int   `json:"tcp_failures,omitempty" yaml:"tcp_failures,omitempty"`
	Timeouts     int   `json:"timeouts,omitempty" yaml:"timeouts,omitempty"`
}

// UpstreamActiveHealthCheckHealthy defines the conditions used to actively determine whether an upstream node is unhealthy.
// +k8s:deepcopy-gen=true
type UpstreamActiveHealthCheckUnhealthy struct {
	UpstreamPassiveHealthCheckUnhealthy `json:",inline" yaml:",inline"`

	// Interval defines the time interval for checking targets, in seconds.
	Interval int `json:"interval,omitempty" yaml:"interval,omitempty"`
}

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

// TrafficSplitConfigRuleWeightedUpstream defines a weighted backend in a traffic split rule.
// This is used by the APISIX traffic-split plugin to distribute traffic
// across multiple upstreams based on weight.
// +k8s:deepcopy-gen=true
type TrafficSplitConfigRuleWeightedUpstream struct {
	// UpstreamID is the identifier of a pre-defined upstream.
	UpstreamID string `json:"upstream_id,omitempty"`

	// Upstream specifies upstream configuration.
	// If provided, it overrides UpstreamID.
	Upstream *Upstream `json:"upstream,omitempty"`

	// Weight defines the percentage of traffic routed to this upstream.
	// The final routing decision is based on relative weights.
	Weight int `json:"weight"`
}

// TLSClass defines the client TLS configuration for mutual TLS (mTLS) authentication.
// +k8s:deepcopy-gen=true
type TLSClass struct {
	// ClientCERT is the PEM-encoded client certificate.
	ClientCERT string `json:"client_cert,omitempty"`

	// ClientCERTID is the reference ID to a stored client certificate.
	ClientCERTID string `json:"client_cert_id,omitempty"`

	// ClientKey is the PEM-encoded private key for the client certificate.
	ClientKey string `json:"client_key,omitempty"`

	// Verify indicates whether the server's certificate should be verified.
	// If false, TLS verification is skipped.
	Verify *bool `json:"verify,omitempty"`
}

// +k8s:deepcopy-gen=true
type SSL struct {
	Metadata `json:",inline" yaml:",inline"`

	Certificates []Certificate `json:"certificates" yaml:"certificates"`
	Client       *ClientClass  `json:"client,omitempty" yaml:"client,omitempty"`
	Snis         []string      `json:"snis" yaml:"snis"`
	SSLProtocols []SSLProtocol `json:"ssl_protocols,omitempty" yaml:"ssl_protocols,omitempty"`
	Type         *SSLType      `json:"type,omitempty" yaml:"type,omitempty"`
}

// +k8s:deepcopy-gen=true
type Certificate struct {
	Certificate string `json:"certificate" yaml:"certificate"`
	Key         string `json:"key" yaml:"key"`
}

// +k8s:deepcopy-gen=true
type ClientClass struct {
	CA               string   `json:"ca" yaml:"ca"`
	Depth            *int64   `json:"depth,omitempty" yaml:"depth,omitempty"`
	SkipMtlsURIRegex []string `json:"skip_mtls_uri_regex,omitempty" yaml:"skip_mtls_uri_regex,omitempty"`
}

type Method string

const (
	Connect Method = "CONNECT"
	Delete  Method = "DELETE"
	Get     Method = "GET"
	Head    Method = "HEAD"
	Options Method = "OPTIONS"
	Patch   Method = "PATCH"
	Post    Method = "POST"
	Purge   Method = "PURGE"
	Put     Method = "PUT"
	Trace   Method = "TRACE"
)

type Scheme string

type UpstreamType string

const (
	Chash      UpstreamType = "chash"
	Ewma       UpstreamType = "ewma"
	LeastConn  UpstreamType = "least_conn"
	Roundrobin UpstreamType = "roundrobin"
)

type SSLProtocol string

const (
	TLSv11 SSLProtocol = "TLSv1.1"
	TLSv12 SSLProtocol = "TLSv1.2"
	TLSv13 SSLProtocol = "TLSv1.3"
)

type SSLType string

const (
	Client SSLType = "client"
	Server SSLType = "server"
)

type Plugins map[string]any

func (p *Plugins) DeepCopyInto(out *Plugins) {
	b, _ := json.Marshal(&p)
	_ = json.Unmarshal(b, out)
}

func (p Plugins) DeepCopy() Plugins {
	if p == nil {
		return nil
	}
	out := make(Plugins)
	p.DeepCopyInto(&out)
	return out
}

// UpstreamNode is the node in upstream
type UpstreamNode struct {
	Host     string `json:"host" yaml:"host"`
	Port     int    `json:"port" yaml:"port"`
	Weight   int    `json:"weight" yaml:"weight"`
	Priority int    `json:"priority,omitempty" yaml:"priority,omitempty"`
}

// UpstreamNodes is the upstream node list.
type UpstreamNodes []UpstreamNode

func mapKV2Node(key string, val float64) (*UpstreamNode, error) {
	hp := strings.Split(key, ":")
	host := hp[0]
	//  according to APISIX upstream nodes policy, port is required
	port := "80"

	if len(hp) > 2 {
		return nil, errors.New("invalid upstream node")
	} else if len(hp) == 2 {
		port = hp[1]
	}

	portInt, err := strconv.Atoi(port)
	if err != nil {
		return nil, fmt.Errorf("parse port to int fail: %s", err.Error())
	}

	node := &UpstreamNode{
		Host:   host,
		Port:   portInt,
		Weight: int(val),
	}

	return node, nil
}

// UnmarshalJSON implements json.Unmarshaler interface.
// lua-cjson doesn't distinguish empty array and table,
// and by default empty array will be encoded as '{}'.
// We have to maintain the compatibility.
func (n *UpstreamNodes) UnmarshalJSON(p []byte) error {
	var data []UpstreamNode
	if p[0] == '{' {
		value := map[string]float64{}
		if err := json.Unmarshal(p, &value); err != nil {
			return err
		}
		for k, v := range value {
			node, err := mapKV2Node(k, v)
			if err != nil {
				return err
			}
			data = append(data, *node)
		}
		*n = data
		return nil
	}
	if err := json.Unmarshal(p, &data); err != nil {
		return err
	}
	*n = data
	return nil
}

func (n Upstream) MarshalJSON() ([]byte, error) {
	type Alias Upstream
	// APISIX does not allow discovery_type and nodes to exist at the same time.
	// https://github.com/apache/apisix/blob/01b4b49eb2ba642b337f7a1fbe1894a77942910b/apisix/schema_def.lua#L501-L504
	if n.DiscoveryType != "" {
		aux := struct {
			Alias
			Nodes UpstreamNodes `json:"nodes,omitempty" yaml:"nodes,omitempty"`
		}{
			Alias: (Alias)(n),
		}
		aux.Nodes = nil
		return json.Marshal(&aux)
	}

	// By default Go serializes a nil slice as JSON null.
	// For APISIX compatibility, nil UpstreamNodes should be encoded as [] instead.
	// https://github.com/apache/apisix/blob/77dacda31277a31d6014b4970e36bae2a5c30907/apisix/schema_def.lua#L295-L338
	if n.Nodes == nil {
		n.Nodes = UpstreamNodes{}
	}
	return json.Marshal((Alias)(n))
}

func ComposeSSLName(kind, namespace, name string) string {
	p := make([]byte, 0, len(kind)+len(namespace)+len(name)+2)
	buf := bytes.NewBuffer(p)

	buf.WriteString(kind)
	buf.WriteByte('_')
	buf.WriteString(namespace)
	buf.WriteByte('_')
	buf.WriteString(name)

	return buf.String()
}

// ComposeRouteName uses namespace, name and rule name to compose
// the route name.
func ComposeRouteName(namespace, name string, rule string) string {
	// FIXME Use sync.Pool to reuse this buffer if the upstream
	// name composing code path is hot.
	p := make([]byte, 0, len(namespace)+len(name)+len(rule)+2)
	buf := bytes.NewBuffer(p)

	buf.WriteString(namespace)
	buf.WriteByte('_')
	buf.WriteString(name)
	buf.WriteByte('_')
	buf.WriteString(rule)

	return buf.String()
}

// ComposeStreamRouteName uses namespace, name and rule name to compose
// the stream_route name.
func ComposeStreamRouteName(namespace, name string, rule string, typ string) string {
	if typ == "" {
		typ = "TCP"
	}
	// FIXME Use sync.Pool to reuse this buffer if the upstream
	// name composing code path is hot.
	p := make([]byte, 0, len(namespace)+len(name)+len(rule)+len(typ)+3)
	buf := bytes.NewBuffer(p)

	buf.WriteString(namespace)
	buf.WriteByte('_')
	buf.WriteString(name)
	buf.WriteByte('_')
	buf.WriteString(rule)
	buf.WriteByte('_')
	buf.WriteString(typ)

	return buf.String()
}

func ComposeServiceNameWithRule(namespace, name string, rule string) string {
	// FIXME Use sync.Pool to reuse this buffer if the upstream
	// name composing code path is hot.
	var p []byte
	plen := len(namespace) + len(name) + 2

	p = make([]byte, 0, plen)
	buf := bytes.NewBuffer(p)
	buf.WriteString(namespace)
	buf.WriteByte('_')
	buf.WriteString(name)
	buf.WriteByte('_')
	buf.WriteString(rule)

	return buf.String()
}

func ComposeGRPCServiceNameWithRule(namespace, name string, rule string) string {
	return ComposeServicesNameWithScheme(namespace, name, rule, "grpc")
}

func ComposeServicesNameWithScheme(namespace, name string, rule string, scheme string) string {
	var p []byte
	plen := len(namespace) + len(name) + len(rule) + len(scheme) + 3

	p = make([]byte, 0, plen)
	buf := bytes.NewBuffer(p)
	buf.WriteString(namespace)
	buf.WriteByte('_')
	buf.WriteString(name)
	buf.WriteByte('_')
	buf.WriteString(rule)
	buf.WriteByte('_')
	buf.WriteString(scheme)

	return buf.String()
}

func ComposeServiceNameWithStream(namespace, name string, rule, typ string) string {
	return ComposeServicesNameWithScheme(namespace, name, rule, typ)
}

func ComposeConsumerName(namespace, name string) string {
	// FIXME Use sync.Pool to reuse this buffer if the upstream
	// name composing code path is hot.
	p := make([]byte, 0, len(namespace)+len(name)+1)
	buf := bytes.NewBuffer(p)

	buf.WriteString(namespace)
	buf.WriteByte('_')
	buf.WriteString(name)

	return buf.String()
}

// NewDefaultUpstream returns an empty Upstream with default values.
func NewDefaultService() *Service {
	return &Service{
		Metadata: Metadata{
			Labels: map[string]string{
				"managed-by": "apisix-ingress-controller",
			},
		},
		Plugins: make(Plugins),
	}
}

func NewDefaultUpstream() *Upstream {
	return &Upstream{
		Metadata: Metadata{
			Labels: map[string]string{
				"managed-by": "apisix-ingress-controller",
			},
		},
		Type: Roundrobin,
	}
}

// NewDefaultRoute returns an empty Route with default values.
func NewDefaultRoute() *Route {
	return &Route{
		Metadata: Metadata{
			Desc: "Created by apisix-ingress-controller, DO NOT modify it manually",
			Labels: map[string]string{
				"managed-by": "apisix-ingress-controller",
			},
		},
	}
}

// NewDefaultStreamRoute returns an empty StreamRoute with default values.
func NewDefaultStreamRoute() *StreamRoute {
	return &StreamRoute{
		Metadata: Metadata{
			Desc: "Created by apisix-ingress-controller, DO NOT modify it manually",
			Labels: map[string]string{
				"managed-by": "apisix-ingress-controller",
			},
		},
	}
}

const (
	PluginProxyRewrite    string = "proxy-rewrite"
	PluginRedirect        string = "redirect"
	PluginResponseRewrite string = "response-rewrite"
	PluginProxyMirror     string = "proxy-mirror"
	PluginCORS            string = "cors"
)

// RewriteConfig is the rule config for proxy-rewrite plugin.
type RewriteConfig struct {
	RewriteTarget      string   `json:"uri,omitempty" yaml:"uri,omitempty"`
	RewriteTargetRegex []string `json:"regex_uri,omitempty" yaml:"regex_uri,omitempty"`
	Headers            *Headers `json:"headers,omitempty" yaml:"headers,omitempty"`
	Host               string   `json:"host,omitempty" yaml:"host,omitempty"`
}

type Headers struct {
	Set    map[string]string `json:"set,omitempty" yaml:"set,omitempty"`
	Add    map[string]string `json:"add,omitempty" yaml:"add,omitempty"`
	Remove []string          `json:"remove,omitempty" yaml:"remove,omitempty"`
}

// ResponseRewriteConfig is the rule config for response-rewrite plugin.
type ResponseRewriteConfig struct {
	StatusCode   int                 `json:"status_code,omitempty" yaml:"status_code,omitempty"`
	Body         string              `json:"body,omitempty" yaml:"body,omitempty"`
	BodyBase64   bool                `json:"body_base64,omitempty" yaml:"body_base64,omitempty"`
	Headers      *ResponseHeaders    `json:"headers,omitempty" yaml:"headers,omitempty"`
	LuaRestyExpr []expr.Expr         `json:"vars,omitempty" yaml:"vars,omitempty"`
	Filters      []map[string]string `json:"filters,omitempty" yaml:"filters,omitempty"`
}

type FaultInjectionConfig struct {
	Abort *FaultInjectionAbortConfig `json:"abort,omitempty" yaml:"abort,omitempty"`
}

type FaultInjectionAbortConfig struct {
	HTTPStatus int           `json:"http_status" yaml:"http_status"`
	Vars       [][]expr.Expr `json:"vars,omitempty" yaml:"vars,omitempty"`
}

type ResponseHeaders struct {
	Set    map[string]string `json:"set,omitempty" yaml:"set,omitempty"`
	Add    []string          `json:"add,omitempty" yaml:"add,omitempty"`
	Remove []string          `json:"remove,omitempty" yaml:"remove,omitempty"`
}

// RequestMirror is the rule config for proxy-mirror plugin.
type RequestMirror struct {
	Host string `json:"host" yaml:"host"`
}

// RedirectConfig is the rule config for redirect plugin.
type RedirectConfig struct {
	HttpToHttps bool   `json:"http_to_https,omitempty" yaml:"http_to_https,omitempty"`
	URI         string `json:"uri,omitempty" yaml:"uri,omitempty"`
	RetCode     int    `json:"ret_code,omitempty" yaml:"ret_code,omitempty"`
}

const (
	StatusSuccess       = "success"
	StatusFailed        = "failed"
	StatusPartialFailed = "partial_failed"
)

type SyncResult struct {
	Status         string       `json:"status"`
	TotalResources int          `json:"total_resources"`
	SuccessCount   int          `json:"success_count"`
	FailedCount    int          `json:"failed_count"`
	Success        []SyncStatus `json:"success"`
	Failed         []SyncStatus `json:"failed"`
}

type SyncStatus struct {
	Event    StatusEvent     `json:"event"`
	FailedAt time.Time       `json:"failed_at,omitempty"`
	SyncedAt time.Time       `json:"synced_at,omitempty"`
	Reason   string          `json:"reason,omitempty"`
	Response ResponseDetails `json:"response,omitempty"`
}

type StatusEvent struct {
	ResourceType string `json:"resourceType"`
	Type         string `json:"type"`
	ResourceID   string `json:"resourceId"`
	ResourceName string `json:"resourceName"`
	ParentID     string `json:"parentId,omitempty"`
}

type ResponseDetails struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
}

type ResponseData struct {
	Value    map[string]any `json:"value"`
	ErrorMsg string         `json:"error_msg"`
}

// Vars represents the route match expressions of APISIX.
type Vars [][]StringOrSlice

// UnmarshalJSON implements json.Unmarshaler interface.
// lua-cjson doesn't distinguish empty array and table,
// and by default empty array will be encoded as '{}'.
// We have to maintain the compatibility.
func (vars *Vars) UnmarshalJSON(p []byte) error {
	if p[0] == '{' {
		if len(p) != 2 {
			return errors.New("unexpected non-empty object")
		}
		return nil
	}
	var data [][]StringOrSlice
	if err := json.Unmarshal(p, &data); err != nil {
		return err
	}
	*vars = data
	return nil
}

// StringOrSlice represents a string or a string slice.
// TODO Do not use interface{} to avoid the reflection overheads.
// +k8s:deepcopy-gen=true
type StringOrSlice struct {
	StrVal   string          `json:"-"`
	SliceVal []StringOrSlice `json:"-"`
}

func (s *StringOrSlice) MarshalJSON() ([]byte, error) {
	if s.SliceVal != nil {
		return json.Marshal(s.SliceVal)
	}
	return json.Marshal(s.StrVal)
}

func (s *StringOrSlice) UnmarshalJSON(p []byte) error {
	if len(p) == 0 {
		return errors.New("empty object")
	}
	if p[0] == '[' {
		return json.Unmarshal(p, &s.SliceVal)
	}
	return json.Unmarshal(p, &s.StrVal)
}

type Config struct {
	Name        string
	ServerAddrs []string
	Token       string
	TlsVerify   bool
	BackendType string
}

// MarshalJSON implements custom JSON marshaling for adcConfig
// It excludes the Token field for security reasons
func (c Config) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Name        string   `json:"name"`
		ServerAddrs []string `json:"serverAddrs"`
		TlsVerify   bool     `json:"tlsVerify"`
	}{
		Name:        c.Name,
		ServerAddrs: c.ServerAddrs,
		TlsVerify:   c.TlsVerify,
	})
}

var (
	ResolveGranularity = struct {
		Endpoint string
		Service  string
	}{
		Endpoint: "endpoint",
		Service:  "service",
	}
)

// ComposeUpstreamName uses namespace, name, ruleIndex, backendIndex, serviceName info to compose
// the upstream name.
// the resolveGranularity is not composited in the upstream name when it is endpoint.
// ref: https://github.com/apache/apisix-ingress-controller/blob/10059afe3e84b693cc61e6df7a0040890a9d16eb/pkg/types/apisix/v1/types.go#L595-L598
func ComposeUpstreamName(namespace, name, ruleIndex, backendIndex string) string {
	var p []byte
	plen := len(namespace) + len(name) + len(ruleIndex) + len(backendIndex) + 3

	p = make([]byte, 0, plen)
	buf := bytes.NewBuffer(p)
	buf.WriteString(namespace)
	buf.WriteByte('_')
	buf.WriteString(name)
	buf.WriteByte('_')
	buf.WriteString(ruleIndex)
	buf.WriteByte('_')
	buf.WriteString(backendIndex)

	return buf.String()
}

// ComposeExternalUpstreamName uses ApisixUpstream namespace, name to compose the upstream name.
func ComposeExternalUpstreamName(namespace, name string) string {
	return namespace + "_" + name
}

// ComposeUpstreamNameForBackendRef composes upstream name using kind, namespace, name and port.
func ComposeUpstreamNameForBackendRef(kind, namespace, name string, port int32) string {
	return fmt.Sprintf("%s_%s_%s_%d", kind, namespace, name, port)
}
