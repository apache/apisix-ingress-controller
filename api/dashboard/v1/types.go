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

package v1

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"database/sql/driver"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
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

	// DefaultUpstreamTimeout represents the default connect,
	// read and send timeout (in seconds) with upstreams.
	DefaultUpstreamTimeout = 60

	// PassHostPass represents pass option for pass_host Upstream settings.
	PassHostPass = "pass"
	// PassHostPass represents node option for pass_host Upstream settings.
	PassHostNode = "node"
	// PassHostPass represents rewrite option for pass_host Upstream settings.
	PassHostRewrite = "rewrite"
)

var ValidSchemes map[string]struct{} = map[string]struct{}{
	SchemeHTTP:  {},
	SchemeHTTPS: {},
	SchemeGRPC:  {},
	SchemeGRPCS: {},
}

// Metadata contains all meta information about resources.
// +k8s:deepcopy-gen=true
type Metadata struct {
	ID     string            `json:"id,omitempty" yaml:"id,omitempty"`
	Name   string            `json:"name,omitempty" yaml:"name,omitempty"`
	Desc   string            `json:"desc,omitempty" yaml:"desc,omitempty"`
	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

func (m *Metadata) GetID() string {
	return m.ID
}

func (m *Metadata) GetName() string {
	return m.Name
}

func (m *Metadata) GetLabels() map[string]string {
	return m.Labels
}

// Upstream is the apisix upstream definition.
// +k8s:deepcopy-gen=true
type Upstream struct {
	Metadata `json:",inline" yaml:",inline"`

	Type         string               `json:"type,omitempty" yaml:"type,omitempty"`
	HashOn       string               `json:"hash_on,omitempty" yaml:"hash_on,omitempty"`
	Key          string               `json:"key,omitempty" yaml:"key,omitempty"`
	Checks       *UpstreamHealthCheck `json:"checks,omitempty" yaml:"checks,omitempty"`
	Nodes        UpstreamNodes        `json:"nodes" yaml:"nodes"`
	Scheme       string               `json:"scheme,omitempty" yaml:"scheme,omitempty"`
	Retries      *int                 `json:"retries,omitempty" yaml:"retries,omitempty"`
	Timeout      *UpstreamTimeout     `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	TLS          *ClientTLS           `json:"tls,omitempty" yaml:"tls,omitempty"`
	PassHost     string               `json:"pass_host,omitempty" yaml:"pass_host,omitempty"`
	UpstreamHost string               `json:"upstream_host,omitempty" yaml:"upstream_host,omitempty"`

	// for Service Discovery
	ServiceName   string            `json:"service_name,omitempty" yaml:"service_name,omitempty"`
	DiscoveryType string            `json:"discovery_type,omitempty" yaml:"discovery_type,omitempty"`
	DiscoveryArgs map[string]string `json:"discovery_args,omitempty" yaml:"discovery_args,omitempty"`
}

type ServiceType string

const (
	ServiceTypeHTTP   ServiceType = "http"
	ServiceTypeStream ServiceType = "stream"
)

// Upstream is the apisix upstream definition.
// +k8s:deepcopy-gen=true
type Service struct {
	Metadata `json:",inline" yaml:",inline"`
	Type     ServiceType `json:"type,omitempty" yaml:"type,omitempty"`
	Upstream *Upstream   `json:"upstream,omitempty" yaml:"upstream,omitempty"`
	Hosts    []string    `json:"hosts,omitempty" yaml:"hosts,omitempty"`
	Plugins  Plugins     `json:"plugins,omitempty" yaml:"plugins,omitempty"`
}

// Route apisix route object
// +k8s:deepcopy-gen=true
type Route struct {
	Metadata        `json:",inline" yaml:",inline"`
	Host            string           `json:"host,omitempty" yaml:"host,omitempty"`
	Hosts           []string         `json:"hosts,omitempty" yaml:"hosts,omitempty"`
	Uri             string           `json:"uri,omitempty" yaml:"uri,omitempty"`
	Priority        int              `json:"priority,omitempty" yaml:"priority,omitempty"`
	Timeout         *UpstreamTimeout `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Vars            Vars             `json:"vars,omitempty" yaml:"vars,omitempty"`
	Paths           []string         `json:"paths,omitempty" yaml:"paths,omitempty"`
	Methods         []string         `json:"methods,omitempty" yaml:"methods,omitempty"`
	EnableWebsocket bool             `json:"enable_websocket,omitempty" yaml:"enable_websocket,omitempty"`
	RemoteAddrs     []string         `json:"remote_addrs,omitempty" yaml:"remote_addrs,omitempty"`
	ServiceID       string           `json:"service_id,omitempty" yaml:"service_id,omitempty"`
	Plugins         Plugins          `json:"plugins,omitempty" yaml:"plugins,omitempty"`
	PluginConfigId  string           `json:"plugin_config_id,omitempty" yaml:"plugin_config_id,omitempty"`
	FilterFunc      string           `json:"filter_func,omitempty" yaml:"filter_func,omitempty"`
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
	StrVal   string   `json:"-"`
	SliceVal []string `json:"-"`
}

func (s *StringOrSlice) MarshalJSON() ([]byte, error) {
	var (
		p   []byte
		err error
	)
	if s.SliceVal != nil {
		p, err = json.Marshal(s.SliceVal)
	} else {
		p, err = json.Marshal(s.StrVal)
	}
	return p, err
}

func (s *StringOrSlice) UnmarshalJSON(p []byte) error {
	var err error

	if len(p) == 0 {
		return errors.New("empty object")
	}
	if p[0] == '[' {
		err = json.Unmarshal(p, &s.SliceVal)
	} else {
		err = json.Unmarshal(p, &s.StrVal)
	}
	return err
}

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

// ClientTLS is tls cert and key use in mTLS
type ClientTLS struct {
	Cert string `json:"client_cert,omitempty" yaml:"client_cert,omitempty"`
	Key  string `json:"client_key,omitempty" yaml:"client_key,omitempty"`
}

// UpstreamTimeout represents the timeout settings on Upstream.
type UpstreamTimeout struct {
	// Connect is the connect timeout
	Connect int `json:"connect" yaml:"connect"`
	// Send is the send timeout
	Send int `json:"send" yaml:"send"`
	// Read is the read timeout
	Read int `json:"read" yaml:"read"`
}

// UpstreamNodes is the upstream node list.
type UpstreamNodes []UpstreamNode

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

// MarshalJSON is used to implement custom json.MarshalJSON
func (up Upstream) MarshalJSON() ([]byte, error) {

	if up.DiscoveryType != "" {
		return json.Marshal(&struct {
			Metadata `json:",inline" yaml:",inline"`

			Type   string               `json:"type,omitempty" yaml:"type,omitempty"`
			HashOn string               `json:"hash_on,omitempty" yaml:"hash_on,omitempty"`
			Key    string               `json:"key,omitempty" yaml:"key,omitempty"`
			Checks *UpstreamHealthCheck `json:"checks,omitempty" yaml:"checks,omitempty"`
			// Nodes   UpstreamNodes        `json:"nodes" yaml:"nodes"`
			Scheme       string           `json:"scheme,omitempty" yaml:"scheme,omitempty"`
			Retries      *int             `json:"retries,omitempty" yaml:"retries,omitempty"`
			Timeout      *UpstreamTimeout `json:"timeout,omitempty" yaml:"timeout,omitempty"`
			HostPass     string           `json:"pass_host,omitempty" yaml:"pass_host,omitempty"`
			UpstreamHost string           `json:"upstream_host,omitempty" yaml:"upstream_host,omitempty"`
			TLS          *ClientTLS       `json:"tls,omitempty" yaml:"tls,omitempty"`

			// for Service Discovery
			ServiceName   string            `json:"service_name,omitempty" yaml:"service_name,omitempty"`
			DiscoveryType string            `json:"discovery_type,omitempty" yaml:"discovery_type,omitempty"`
			DiscoveryArgs map[string]string `json:"discovery_args,omitempty" yaml:"discovery_args,omitempty"`
		}{
			Metadata: up.Metadata,

			Type:   up.Type,
			HashOn: up.HashOn,
			Key:    up.Key,
			Checks: up.Checks,
			// Nodes:   up.Nodes,
			Scheme:       up.Scheme,
			Retries:      up.Retries,
			Timeout:      up.Timeout,
			HostPass:     up.PassHost,
			UpstreamHost: up.UpstreamHost,
			TLS:          up.TLS,

			ServiceName:   up.ServiceName,
			DiscoveryType: up.DiscoveryType,
			DiscoveryArgs: up.DiscoveryArgs,
		})
	} else {
		return json.Marshal(&struct {
			Metadata `json:",inline" yaml:",inline"`

			Type         string               `json:"type,omitempty" yaml:"type,omitempty"`
			HashOn       string               `json:"hash_on,omitempty" yaml:"hash_on,omitempty"`
			Key          string               `json:"key,omitempty" yaml:"key,omitempty"`
			Checks       *UpstreamHealthCheck `json:"checks,omitempty" yaml:"checks,omitempty"`
			Nodes        UpstreamNodes        `json:"nodes" yaml:"nodes"`
			Scheme       string               `json:"scheme,omitempty" yaml:"scheme,omitempty"`
			Retries      *int                 `json:"retries,omitempty" yaml:"retries,omitempty"`
			Timeout      *UpstreamTimeout     `json:"timeout,omitempty" yaml:"timeout,omitempty"`
			HostPass     string               `json:"pass_host,omitempty" yaml:"pass_host,omitempty"`
			UpstreamHost string               `json:"upstream_host,omitempty" yaml:"upstream_host,omitempty"`
			TLS          *ClientTLS           `json:"tls,omitempty" yaml:"tls,omitempty"`

			// for Service Discovery
			// ServiceName   string            `json:"service_name,omitempty" yaml:"service_name,omitempty"`
			// DiscoveryType string            `json:"discovery_type,omitempty" yaml:"discovery_type,omitempty"`
			// DiscoveryArgs map[string]string `json:"discovery_args,omitempty" yaml:"discovery_args,omitempty"`
		}{
			Metadata: up.Metadata,

			Type:         up.Type,
			HashOn:       up.HashOn,
			Key:          up.Key,
			Checks:       up.Checks,
			Nodes:        up.Nodes,
			Scheme:       up.Scheme,
			Retries:      up.Retries,
			Timeout:      up.Timeout,
			HostPass:     up.PassHost,
			UpstreamHost: up.UpstreamHost,
			TLS:          up.TLS,

			// ServiceName:   up.ServiceName,
			// DiscoveryType: up.DiscoveryType,
			// DiscoveryArgs: up.DiscoveryArgs,
		})
	}

}

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

// UpstreamNode is the node in upstream
// +k8s:deepcopy-gen=true
type UpstreamNode struct {
	Host   string `json:"host,omitempty" yaml:"host,omitempty"`
	Port   int    `json:"port,omitempty" yaml:"port,omitempty"`
	Weight int    `json:"weight,omitempty" yaml:"weight,omitempty"`
}

// UpstreamHealthCheck defines the active and/or passive health check for an Upstream,
// with the upstream health check feature, pods can be kicked out or joined in quickly,
// if the feedback of Kubernetes liveness/readiness probe is long.
// +k8s:deepcopy-gen=true
type UpstreamHealthCheck struct {
	Active  *UpstreamActiveHealthCheck  `json:"active" yaml:"active"`
	Passive *UpstreamPassiveHealthCheck `json:"passive,omitempty" yaml:"passive,omitempty"`
}

// UpstreamActiveHealthCheck defines the active upstream health check.
// +k8s:deepcopy-gen=true
type UpstreamActiveHealthCheck struct {
	Type               string                             `json:"type,omitempty" yaml:"type,omitempty"`
	Timeout            int                                `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Concurrency        int                                `json:"concurrency,omitempty" yaml:"concurrency,omitempty"`
	Host               string                             `json:"host,omitempty" yaml:"host,omitempty"`
	Port               int32                              `json:"port,omitempty" yaml:"port,omitempty"`
	HTTPPath           string                             `json:"http_path,omitempty" yaml:"http_path,omitempty"`
	HTTPSVerifyCert    bool                               `json:"https_verify_certificate,omitempty" yaml:"https_verify_certificate,omitempty"`
	HTTPRequestHeaders []string                           `json:"req_headers,omitempty" yaml:"req_headers,omitempty"`
	Healthy            UpstreamActiveHealthCheckHealthy   `json:"healthy,omitempty" yaml:"healthy,omitempty"`
	Unhealthy          UpstreamActiveHealthCheckUnhealthy `json:"unhealthy,omitempty" yaml:"unhealthy,omitempty"`
}

// UpstreamPassiveHealthCheck defines the passive upstream health check.
// +k8s:deepcopy-gen=true
type UpstreamPassiveHealthCheck struct {
	Type      string                              `json:"type,omitempty" yaml:"type,omitempty"`
	Healthy   UpstreamPassiveHealthCheckHealthy   `json:"healthy,omitempty" yaml:"healthy,omitempty"`
	Unhealthy UpstreamPassiveHealthCheckUnhealthy `json:"unhealthy,omitempty" yaml:"unhealthy,omitempty"`
}

// UpstreamActiveHealthCheckHealthy defines the conditions used to actively determine whether an upstream node is healthy.
// +k8s:deepcopy-gen=true
type UpstreamActiveHealthCheckHealthy struct {
	UpstreamPassiveHealthCheckHealthy `json:",inline" yaml:",inline"`

	// Interval defines the time interval for checking targets, in seconds.
	Interval int `json:"interval,omitempty" yaml:"interval,omitempty"`
}

// UpstreamPassiveHealthCheckHealthy defines the conditions used to passively determine whether an upstream node is unhealthy.
// +k8s:deepcopy-gen=true
type UpstreamPassiveHealthCheckHealthy struct {
	HTTPStatuses []int `json:"http_statuses,omitempty" yaml:"http_statuses,omitempty"`
	Successes    int   `json:"successes,omitempty" yaml:"successes,omitempty"`
}

// UpstreamActiveHealthCheckUnhealthy defines the conditions used to actively determine whether an upstream node is unhealthy.
// +k8s:deepcopy-gen=true
type UpstreamActiveHealthCheckUnhealthy struct {
	UpstreamPassiveHealthCheckUnhealthy `json:",inline" yaml:",inline"`

	// Interval defines the time interval for checking targets, in seconds.
	Interval int `json:"interval,omitempty" yaml:"interval,omitempty"`
}

// UpstreamPassiveHealthCheckUnhealthy defines the conditions used to passively determine whether an upstream node is unhealthy.
// +k8s:deepcopy-gen=true
type UpstreamPassiveHealthCheckUnhealthy struct {
	HTTPStatuses []int `json:"http_statuses,omitempty" yaml:"http_statuses,omitempty"`
	HTTPFailures int   `json:"http_failures,omitempty" yaml:"http_failures,omitempty"`
	TCPFailures  int   `json:"tcp_failures,omitempty" yaml:"tcp_failures,omitempty"`
	Timeouts     int   `json:"timeouts,omitempty" yaml:"timeouts,omitempty"`
}

// Ssl apisix ssl object
// +k8s:deepcopy-gen=true
type Ssl struct {
	ID     string                 `json:"id,omitempty" yaml:"id,omitempty"`
	Snis   []string               `json:"snis,omitempty" yaml:"snis,omitempty"`
	Cert   string                 `json:"cert,omitempty" yaml:"cert,omitempty"`
	Key    string                 `json:"key,omitempty" yaml:"key,omitempty"`
	Status int                    `json:"status,omitempty" yaml:"status,omitempty"`
	Labels map[string]string      `json:"labels,omitempty" yaml:"labels,omitempty"`
	Client *MutualTLSClientConfig `json:"client,omitempty" yaml:"client,omitempty"`
}

// MutualTLSClientConfig apisix SSL client field
// +k8s:deepcopy-gen=true
type MutualTLSClientConfig struct {
	CA               string   `json:"ca,omitempty" yaml:"ca,omitempty"`
	Depth            int      `json:"depth,omitempty" yaml:"depth,omitempty"`
	SkipMTLSUriRegex []string `json:"skip_mtls_uri_regex,omitempty" yaml:"skip_mtls_uri_regex, omitempty"`
}

// StreamRoute represents the stream_route object in APISIX.
// +k8s:deepcopy-gen=true
type StreamRoute struct {
	// TODO metadata should use Metadata type
	ID         string            `json:"id,omitempty" yaml:"id,omitempty"`
	Name       string            `json:"name,omitempty" yaml:"name,omitempty"`
	Desc       string            `json:"desc,omitempty" yaml:"desc,omitempty"`
	Labels     map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	ServerPort int32             `json:"server_port,omitempty" yaml:"server_port,omitempty"`
	SNI        string            `json:"sni,omitempty" yaml:"sni,omitempty"`
	ServiceID  string            `json:"service_id,omitempty" yaml:"service_id,omitempty"`
	Plugins    Plugins           `json:"plugins,omitempty" yaml:"plugins,omitempty"`
}

// GlobalRule represents the global_rule object in APISIX.
// +k8s:deepcopy-gen=true
type GlobalRule struct {
	ID      string  `json:"id" yaml:"id"`
	Plugins Plugins `json:"plugins" yaml:"plugins"`
}

// Consumer represents the consumer object in APISIX.
// +k8s:deepcopy-gen=true
type Consumer struct {
	Username string            `json:"username" yaml:"username"`
	Desc     string            `json:"desc,omitempty" yaml:"desc,omitempty"`
	Labels   map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Plugins  Plugins           `json:"plugins,omitempty" yaml:"plugins,omitempty"`
}

// PluginConfig apisix plugin object
// +k8s:deepcopy-gen=true
type PluginConfig struct {
	Metadata `json:",inline" yaml:",inline"`
	Plugins  Plugins `json:"plugins" yaml:"plugins"`
}

type PluginMetadata struct {
	Name     string
	Metadata map[string]any
}

// NewDefaultUpstream returns an empty Upstream with default values.
func NewDefaultService() *Service {
	return &Service{
		Metadata: Metadata{
			Labels: map[string]string{
				"managed-by": "api7-ingress-controller",
			},
		},
		Plugins: make(Plugins),
	}
}

// NewDefaultRoute returns an empty Route with default values.
func NewDefaultRoute() *Route {
	return &Route{
		Metadata: Metadata{
			Desc: "Created by api7-ingress-controller, DO NOT modify it manually",
			Labels: map[string]string{
				"managed-by": "api7-ingress-controller",
			},
		},
	}
}

func NewDefaultUpstream() *Upstream {
	return &Upstream{
		Type:   LbRoundRobin,
		Key:    "",
		Nodes:  make(UpstreamNodes, 0),
		Scheme: SchemeHTTP,
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
		Desc: "Created by api7-ingress-controller, DO NOT modify it manually",
		Labels: map[string]string{
			"managed-by": "api7-ingress-controller",
		},
	}
}

// NewDefaultConsumer returns an empty Consumer with default values.
func NewDefaultConsumer() *Consumer {
	return &Consumer{
		Desc: "Created by api7-ingress-controller, DO NOT modify it manually",
		Labels: map[string]string{
			"managed-by": "api7-ingress-controller",
		},
	}
}

// NewDefaultPluginConfig returns an empty PluginConfig with default values.
func NewDefaultPluginConfig() *PluginConfig {
	return &PluginConfig{
		Metadata: Metadata{
			Desc: "Created by api7-ingress-controller, DO NOT modify it manually",
			Labels: map[string]string{
				"managed-by": "api7-ingress-controller",
			},
		},
		Plugins: make(Plugins),
	}
}

// NewDefaultGlobalRule returns an empty PluginConfig with default values.
func NewDefaultGlobalRule() *GlobalRule {
	return &GlobalRule{
		Plugins: make(Plugins),
	}
}

// ComposeUpstreamName uses namespace, name, subset (optional), port, resolveGranularity info to compose
// the upstream name.
// the resolveGranularity is not composited in the upstream name when it is endpoint.
func ComposeUpstreamName(namespace, name string, port int32) string {
	pstr := strconv.Itoa(int(port))
	// FIXME Use sync.Pool to reuse this buffer if the upstream
	// name composing code path is hot.
	var p []byte
	plen := len(namespace) + len(name) + len(pstr) + 2

	p = make([]byte, 0, plen)
	buf := bytes.NewBuffer(p)
	buf.WriteString(namespace)
	buf.WriteByte('_')
	buf.WriteString(name)
	buf.WriteByte('_')
	buf.WriteString(pstr)

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

func ComposeUpstreamNameWithRule(namespace, name string, rule string) string {
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

// ComposeExternalUpstreamName uses ApisixUpstream namespace, name to compose the upstream name.
func ComposeExternalUpstreamName(namespace, name string) string {
	return namespace + "_" + name
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
func ComposeStreamRouteName(namespace, name string, rule string) string {
	// FIXME Use sync.Pool to reuse this buffer if the upstream
	// name composing code path is hot.
	p := make([]byte, 0, len(namespace)+len(name)+len(rule)+6)
	buf := bytes.NewBuffer(p)

	buf.WriteString(namespace)
	buf.WriteByte('_')
	buf.WriteString(name)
	buf.WriteByte('_')
	buf.WriteString(rule)
	buf.WriteString("_tcp")

	return buf.String()
}

// ComposeConsumerName uses namespace and name of ApisixConsumer to compose
// the Consumer name.
func ComposeConsumerName(namespace, name string) string {
	p := make([]byte, 0, len(namespace)+len(name)+1)
	buf := bytes.NewBuffer(p)

	// TODO If APISIX modifies the consumer name schema, we can drop this.
	buf.WriteString(strings.ReplaceAll(namespace, "-", "_"))
	buf.WriteString("_")
	buf.WriteString(strings.ReplaceAll(name, "-", "_"))

	return buf.String()
}

// ComposePluginConfigName uses namespace, name to compose
// the plugin_config name.
func ComposePluginConfigName(namespace, name string) string {
	// FIXME Use sync.Pool to reuse this buffer if the upstream
	// name composing code path is hot.
	p := make([]byte, 0, len(namespace)+len(name)+1)
	buf := bytes.NewBuffer(p)

	buf.WriteString(namespace)
	buf.WriteByte('_')
	buf.WriteString(name)

	return buf.String()
}

// ComposeGlobalRuleName uses namespace, name to compose
// the global_rule name.
func ComposeGlobalRuleName(namespace, name string) string {
	// FIXME Use sync.Pool to reuse this buffer if the upstream
	// name composing code path is hot.
	p := make([]byte, 0, len(namespace)+len(name)+1)
	buf := bytes.NewBuffer(p)

	buf.WriteString(namespace)
	buf.WriteByte('_')
	buf.WriteString(name)

	return buf.String()
}

// Schema represents the schema of APISIX objects.
type Schema struct {
	Name    string `json:"name,omitempty" yaml:"name,omitempty"`
	Content string `json:"content,omitempty" yaml:"content,omitempty"`
}

func (s *Schema) DeepCopyInto(out *Schema) {
	b, _ := json.Marshal(&s)
	_ = json.Unmarshal(b, out)
}

func (s *Schema) DeepCopy() *Schema {
	if s == nil {
		return nil
	}
	out := new(Schema)
	s.DeepCopyInto(out)
	return out
}

type GatewayGroup struct {
	ID                       string                  `json:"id" gorm:"column:id; primaryKey; size:255;"`
	ShortID                  string                  `json:"short_id" gorm:"column:short_id; size:255; uniqueIndex:UQE_gateway_group_short_id;"`
	Name                     string                  `json:"name" gorm:"name; size:255;"`
	OrgID                    string                  `json:"-" gorm:"org_id; size:255; index:gateway_group_org_id;"`
	Type                     string                  `json:"type" gorm:"column:type; size:255; default:api7_gateway;"`
	Description              string                  `json:"description" gorm:"description; type:text;"`
	Labels                   map[string]string       `json:"labels,omitempty" gorm:"serializer:json; column:labels; type:text;"`
	Config                   GatewayGroupBasicConfig `json:"config" gorm:"serializer:json; column:config; type:text;"`
	RunningConfigID          string                  `json:"-" gorm:"column:running_config_id; size:255;"`
	ConfigVersion            int64                   `json:"-" gorm:"column:config_version;"`
	AdminKeySalt             string                  `json:"-" gorm:"column:admin_key_salt; size:255;"`
	EncryptedAdminKey        string                  `json:"-" gorm:"column:encrypted_admin_key; size:255;"`
	EnforceServicePublishing bool                    `json:"enforce_service_publishing" gorm:"column:enforce_service_publishing; default:false;"`
}

func (g *GatewayGroup) GetKeyPrefix() string {
	return fmt.Sprintf("/gateway_groups/%s", g.ShortID)
}

func (g *GatewayGroup) GetKeyPrefixEnd() string {
	return clientv3.GetPrefixRangeEnd(g.GetKeyPrefix())
}

type GatewayGroupBasicConfig struct {
	ImageTag string `json:"image_tag,omitempty"`
}

func (GatewayGroup) TableName() string {
	return "gateway_group"
}

type GatewayGroupAdminKey struct {
	Key string `json:"key" mask:"fixed"`
}

type CertificateType string

const (
	CertificateTypeEndpoint     CertificateType = "Endpoint"
	CertificateTypeIntermediate CertificateType = "Intermediate"
	CertificateTypeRoot         CertificateType = "Root"
)

type AesEncrypt string

var AESKeyring = "b2zanhtrq35f6j3m"

func PKCS7Unpadding(plantText []byte) []byte {
	length := len(plantText)
	if length == 0 {
		return plantText
	}
	padding := int(plantText[length-1])
	return plantText[:(length - padding)]
}

func FAesDecrypt(encrypted string, keyring string) (string, error) {
	if len(encrypted) == 0 {
		return "", nil
	}
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher([]byte(keyring))
	if err != nil {
		return "", err
	}
	if len(ciphertext)%aes.BlockSize != 0 {
		return "", errors.New("block size cant be zero")
	}
	iv := []byte(keyring)[:aes.BlockSize]
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(ciphertext, ciphertext)

	return string(PKCS7Unpadding(ciphertext)), nil
}

func (aesEncrypt AesEncrypt) Value() (driver.Value, error) {
	return FAesDecrypt(string(aesEncrypt), AESKeyring)
}

func (aesEncrypt *AesEncrypt) Scan(value any) error {
	var str string
	switch v := value.(type) {
	case string: // for postgres
		str = v
	case []uint8: // for mysql
		str = string(v)
	default:
		return fmt.Errorf("invalid type scan from database driver: %T", value)
	}
	res, err := FAesDecrypt(str, AESKeyring)
	if err == nil {
		*aesEncrypt = AesEncrypt(res)
	}
	return err
}

type BaseCertificate struct {
	ID          string          `json:"id" gorm:"primaryKey; column:id; size:255;"`
	Certificate string          `json:"certificate" gorm:"column:certificate; type:text;"`
	PrivateKey  AesEncrypt      `json:"private_key" gorm:"column:private_key; type:text;" mask:"fixed"`
	Expiry      Time            `json:"expiry" gorm:"column:expiry"`
	CreatedAt   Time            `json:"-" gorm:"column:created_at;autoCreateTime; <-:create;"`
	UpdatedAt   Time            `json:"-" gorm:"column:updated_at;autoUpdateTime"`
	Type        CertificateType `json:"-" gorm:"column:type;"`
}

type DataplaneCertificate struct {
	*BaseCertificate

	GatewayGroupID string `json:"gateway_group_id" gorm:"column:gateway_group_id;size:255;"`
	CACertificate  string `json:"ca_certificate" gorm:"column:ca_certificate;type:text;"`
}

func (DataplaneCertificate) TableName() string {
	return "dataplane_certificate"
}

type Time time.Time

func (t *Time) UnmarshalJSON(data []byte) error {
	ts, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		return err
	}

	*t = Time(time.Unix(ts, 0))
	return nil
}

func (t Time) MarshalJSON() ([]byte, error) {
	ts := (time.Time)(t).Unix()
	return []byte(strconv.FormatInt(ts, 10)), nil
}

func (t Time) String() string {
	return strconv.FormatInt(time.Time(t).Unix(), 10)
}

func (t *Time) Unix() int64 {
	return time.Time(*t).Unix()
}

func (t *Time) Scan(src any) error {
	switch s := src.(type) {
	case time.Time:
		*t = Time(s)
	default:
		return fmt.Errorf("invalid time type from database driver: %T", src)
	}
	return nil
}

func (t Time) Value() (driver.Value, error) {
	return time.Time(t), nil
}
