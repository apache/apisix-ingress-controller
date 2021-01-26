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

import "encoding/json"

// Route apisix route object
// +k8s:deepcopy-gen=true
type Route struct {
	ID              string   `json:"id,omitempty" yaml:"id,omitempty"`
	Group           string   `json:"group,omitempty" yaml:"group,omitempty"`
	FullName        string   `json:"full_name,omitempty" yaml:"full_name,omitempty"`
	ResourceVersion string   `json:"resource_version,omitempty" yaml:"resource_version,omitempty"`
	Host            string   `json:"host,omitempty" yaml:"host,omitempty"`
	Path            string   `json:"path,omitempty" yaml:"path,omitempty"`
	Name            string   `json:"name,omitempty" yaml:"name,omitempty"`
	Methods         []string `json:"methods,omitempty" yaml:"methods,omitempty"`
	ServiceId       string   `json:"service_id,omitempty" yaml:"service_id,omitempty"`
	ServiceName     string   `json:"service_name,omitempty" yaml:"service_name,omitempty"`
	UpstreamId      string   `json:"upstream_id,omitempty" yaml:"upstream_id,omitempty"`
	UpstreamName    string   `json:"upstream_name,omitempty" yaml:"upstream_name,omitempty"`
	Plugins         Plugins  `json:"plugins,omitempty" yaml:"plugins,omitempty"`
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

// Upstream apisix upstream
// +k8s:deepcopy-gen=true
type Upstream struct {
	ID              string               `json:"id,omitempty" yaml:"id,omitempty"`
	FullName        string               `json:"full_name,omitempty" yaml:"full_name,omitempty"`
	Group           string               `json:"group,omitempty" yaml:"group,omitempty"`
	ResourceVersion string               `json:"resource_version,omitempty" yaml:"resource_version,omitempty"`
	Name            string               `json:"name,omitempty" yaml:"name,omitempty"`
	Type            string               `json:"type,omitempty" yaml:"type,omitempty"`
	HashOn          string               `json:"hash_on,omitemtpy" yaml:"hash_on,omitempty"`
	Key             string               `json:"key,omitempty" yaml:"key,omitempty"`
	Nodes           []Node               `json:"nodes,omitempty" yaml:"nodes,omitempty"`
	Checks          *UpstreamHealthCheck `json:"checks,omitempty" yaml:"checks,omitempty"`
	FromKind        string               `json:"from_kind,omitempty" yaml:"from_kind,omitempty"`
}

// UpstreamHealthCheck describes the health check of Upstream.
// Users must configure the active health check while it's optional to configure
// the passive one.
// +k8s:deepcopy-gen=true
type UpstreamHealthCheck struct {
	Active  *UpstreamActiveHealthCheck  `json:"active,omitempty" yaml:"active,omitempty"`
	Passive *UpstreamPassiveHealthCheck `json:"passive,omitempty" yaml:"passive,omitempty"`
}

// UpstreamActiveHealthCheck describes the active health check.
// +k8s:deepcopy-gen=true
type UpstreamActiveHealthCheck struct {
	Type           string             `json:"type,omitempty" yaml:"type,omitempty"`
	Timeout        int                `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Concurrency    int                `json:"concurrency,omitempty" yaml:"concurrency,omitempty"`
	Host           string             `json:"host,omitempty" yaml:"host,omitempty"`
	Port           int                `json:"port,omitempty" yaml:"port,omitempty"`
	HttpPath       string             `json:"http_path,omitempty" yaml:"http_path,omitempty"`
	HttpVerifyCert bool               `json:"http_verify_certificate,omitempty" yaml:"http_verify_certificate,omitempty"`
	RequestHeaders []string           `json:"req_headers,omitempty" yaml:"req_headers,omitempty"`
	Healthy        *UpstreamHealthy   `json:"healthy,omitempty" yaml:"healthy,omitempty"`
	Unhealthy      *UpstreamUnhealthy `json:"unhealthy,omitempty" yaml:"unhealthy,omitempty"`
}

// UpstreamActiveHealthCheck describes the passive health check.
// +k8s:deepcopy-gen=true
type UpstreamPassiveHealthCheck struct {
	Type      string             `json:"type,omitempty" yaml:"type,omitempty"`
	Healthy   *UpstreamHealthy   `json:"healthy,omitempty" yaml:"healthy,omitempty"`
	Unhealthy *UpstreamUnhealthy `json:"unhealthy,omitempty" yaml:"unhealthy,omitempty"`
}

// UpstreamHealthy describes the conditions that health checker used to
// decide whether an Upstream node is healthy.
// +k8s:deepcopy-gen=true
type UpstreamHealthy struct {
	Interval     int   `json:"interval,omitempty" yaml:"interval,omitempty"`
	HttpStatuses []int `json:"http_statuses,omitempty" yaml:"http_statuses,omitempty"`
	Successes    int   `json:"successes,omitempty" yaml:"successes,omitempty"`
}

// UpstreamHealthy describes the conditions that health checker used to
// decide whether an Upstream node is unhealthy.
// +k8s:deepcopy-gen=true
type UpstreamUnhealthy struct {
	Interval     int   `json:"interval,omitempty"`
	HttpStatuses []int `json:"http_statuses,omitempty" yaml:"http_statuses,omitempty"`
	HttpFailures int   `json:"http_failures,omitempty" yaml:"http_failures,omitempty"`
	TcpFailures  int   `json:"tcp_failures,omitempty" yaml:"tcp_failures,omitempty"`
	Timeout      int   `json:"timeout,omitempty" yaml:"timeout,omitempty"`
}

// Node the node in upstream
// +k8s:deepcopy-gen=true
type Node struct {
	IP     string `json:"ip,omitempty" yaml:"ip,omitempty"`
	Port   int    `json:"port,omitempty" yaml:"port,omitempty"`
	Weight int    `json:"weight,omitempty" yaml:"weight,omitempty"`
}

// Ssl apisix ssl object
// +k8s:deepcopy-gen=true
type Ssl struct {
	ID     string   `json:"id,omitempty" yaml:"id,omitempty"`
	Snis   []string `json:"snis,omitempty" yaml:"snis,omitempty"`
	Cert   string   `json:"cert,omitempty" yaml:"cert,omitempty"`
	Key    string   `json:"key,omitempty" yaml:"key,omitempty"`
	Status int      `json:"status,omitempty" yaml:"status,omitempty"`
	Group  string   `json:"group,omitempty" yaml:"group,omitempty"`
}
