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

import (
	"bytes"
	"encoding/json"
	"strconv"
)

const (
	// HashOnVars means the hash scope is variable.
	HashOnVars = "vars"
	// HashVarsCombination means the hash scope is the
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
)

// Metadata contains all meta information about resources.
type Metadata struct {
	ID              string `json:"id,omitempty" yaml:"id,omitempty"`
	FullName        string `json:"full_name,omitempty" yaml:"full_name,omitempty"`
	Name            string `json:"name,omitempty" yaml:"name,omitempty"`
	ResourceVersion string `json:"resource_version,omitempty" yaml:"resource_version,omitempty"`
	Group           string `json:"group,omitempty" yaml:"group,omitempty"`
}

// Route apisix route object
// +k8s:deepcopy-gen=true
type Route struct {
	Metadata `json:",inline" yaml:",inline"`

	Host         string   `json:"host,omitempty" yaml:"host,omitempty"`
	Path         string   `json:"path,omitempty" yaml:"path,omitempty"`
	Methods      []string `json:"methods,omitempty" yaml:"methods,omitempty"`
	ServiceId    string   `json:"service_id,omitempty" yaml:"service_id,omitempty"`
	ServiceName  string   `json:"service_name,omitempty" yaml:"service_name,omitempty"`
	UpstreamId   string   `json:"upstream_id,omitempty" yaml:"upstream_id,omitempty"`
	UpstreamName string   `json:"upstream_name,omitempty" yaml:"upstream_name,omitempty"`
	Plugins      Plugins  `json:"plugins,omitempty" yaml:"plugins,omitempty"`
}

type Plugins map[string]interface{}

func (p *Plugins) DeepCopyInto(out *Plugins) {
	b, _ := json.Marshal(&p)
	_ = json.Unmarshal(b, out)
}

func (p *Plugins) DeepCopy() *Plugins {
	if p == nil {
		return nil
	}
	out := new(Plugins)
	p.DeepCopyInto(out)
	return out
}

// Service apisix service
// +k8s:deepcopy-gen=true
type Service struct {
	ID              string  `json:"id,omitempty" yaml:"id,omitempty"`
	FullName        string  `json:"full_name,omitempty" yaml:"full_name,omitempty"`
	Group           string  `json:"group,omitempty" yaml:"group,omitempty"`
	ResourceVersion string  `json:"resource_version,omitempty" yaml:"resource_version,omitempty"`
	Name            string  `json:"name,omitempty" yaml:"name,omitempty"`
	UpstreamId      string  `json:"upstream_id,omitempty" yaml:"upstream_id,omitempty"`
	UpstreamName    string  `json:"upstream_name,omitempty" yaml:"upstream_name,omitempty"`
	Plugins         Plugins `json:"plugins,omitempty" yaml:"plugins,omitempty"`
	FromKind        string  `json:"from_kind,omitempty" yaml:"from_kind,omitempty"`
}

// Upstream is the apisix upstream definition.
// +k8s:deepcopy-gen=true
type Upstream struct {
	Metadata `json:",inline" yaml:",inline"`

	Type     string         `json:"type,omitempty" yaml:"type,omitempty"`
	HashOn   string         `json:"hash_on,omitemtpy" yaml:"hash_on,omitempty"`
	Key      string         `json:"key,omitempty" yaml:"key,omitempty"`
	Nodes    []UpstreamNode `json:"nodes,omitempty" yaml:"nodes,omitempty"`
	FromKind string         `json:"from_kind,omitempty" yaml:"from_kind,omitempty"`
	Scheme   string         `json:"scheme,omitempty" yaml:"scheme,omitempty"`
}

// Node the node in upstream
// +k8s:deepcopy-gen=true
type UpstreamNode struct {
	IP     string `json:"ip,omitempty" yaml:"ip,omitempty"`
	Port   int    `json:"port,omitempty" yaml:"port,omitempty"`
	Weight int    `json:"weight,omitempty" yaml:"weight,omitempty"`
}

// Ssl apisix ssl object
// +k8s:deepcopy-gen=true
type Ssl struct {
	ID       string   `json:"id,omitempty" yaml:"id,omitempty"`
	FullName string   `json:"full_name,omitempty" yaml:"full_name,omitempty"`
	Snis     []string `json:"snis,omitempty" yaml:"snis,omitempty"`
	Cert     string   `json:"cert,omitempty" yaml:"cert,omitempty"`
	Key      string   `json:"key,omitempty" yaml:"key,omitempty"`
	Status   int      `json:"status,omitempty" yaml:"status,omitempty"`
	Group    string   `json:"group,omitempty" yaml:"group,omitempty"`
}

// NewDefaultUpstream returns an empty Upstream with default values.
func NewDefaultUpstream() *Upstream {
	return &Upstream{
		Type:     LbRoundRobin,
		Key:      "",
		Nodes:    nil,
		FromKind: "",
		Scheme:   SchemeHTTP,
	}
}

// ComposeUpstreamName uses namespace, name and port info to compose
// the upstream name.
func ComposeUpstreamName(namespace, name string, port int32) string {
	pstr := strconv.Itoa(int(port))
	// FIXME Use sync.Pool to reuse this buffer if the upstream
	// name composing code path is hot.
	p := make([]byte, 0, len(namespace)+len(name)+len(pstr)+2)
	buf := bytes.NewBuffer(p)

	buf.WriteString(namespace)
	buf.WriteByte('_')
	buf.WriteString(name)
	buf.WriteByte('_')
	buf.WriteString(pstr)

	return buf.String()
}
