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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ApisixUpstreamSpec describes the desired configuration of an ApisixUpstream resource.
// It defines how traffic should be routed to backend services, including upstream node
// definitions and custom configuration.
type ApisixUpstreamSpec struct {
	// IngressClassName is the name of an IngressClass cluster resource.
	// Controller implementations use this field to determine whether they
	// should process this ApisixUpstream resource.
	// +kubebuilder:validation:Optional
	IngressClassName string `json:"ingressClassName,omitempty" yaml:"ingressClassName,omitempty"`

	// ExternalNodes defines a static list of backend nodes located outside the cluster.
	// When this field is set, the upstream will route traffic directly to these nodes
	// without DNS resolution or service discovery.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinItems=1
	ExternalNodes []ApisixUpstreamExternalNode `json:"externalNodes,omitempty" yaml:"externalNodes,omitempty"`

	// ApisixUpstreamConfig holds the core upstream configuration, such as load balancing,
	// health checks, retries, and TLS settings. This struct is inlined for simplicity.
	ApisixUpstreamConfig `json:",inline" yaml:",inline"`

	// PortLevelSettings allows fine-grained upstream configuration for specific ports,
	// useful when a backend service exposes multiple ports with different behaviors or protocols.
	PortLevelSettings []PortLevelSettings `json:"portLevelSettings,omitempty" yaml:"portLevelSettings,omitempty"`
}

// ApisixUpstreamStatus defines the observed state of ApisixUpstream.
type ApisixUpstreamStatus = ApisixStatus

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=au

// ApisixUpstream defines configuration for upstream services.
type ApisixUpstream struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// ApisixUpstreamSpec defines the upstream configuration.
	Spec   ApisixUpstreamSpec   `json:"spec,omitempty"`
	Status ApisixUpstreamStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ApisixUpstreamList contains a list of ApisixUpstream.
type ApisixUpstreamList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ApisixUpstream `json:"items"`
}

// ApisixUpstreamExternalNode defines configuration for an external upstream node.
// This allows referencing services outside the cluster.
type ApisixUpstreamExternalNode struct {
	// Name is the hostname or IP address of the external node.
	Name string `json:"name,omitempty" yaml:"name"`

	// Type indicates the kind of external node. Can be `Domain`, or `Service`.
	Type ApisixUpstreamExternalType `json:"type,omitempty" yaml:"type"`

	// Weight defines the load balancing weight of this node.
	// Higher values increase the share of traffic sent to this node.
	// +kubebuilder:validation:Optional
	Weight *int `json:"weight,omitempty" yaml:"weight"`

	// Port specifies the port number on which the external node is accepting traffic.
	// +kubebuilder:validation:Optional
	Port *int `json:"port,omitempty" yaml:"port"`
}

// ApisixUpstreamConfig defines configuration for upstream services.
type ApisixUpstreamConfig struct {
	// LoadBalancer specifies the load balancer configuration for Kubernetes Service.
	// +kubebuilder:validation:Optional
	LoadBalancer *LoadBalancer `json:"loadbalancer,omitempty" yaml:"loadbalancer,omitempty"`

	// Scheme is the protocol used to communicate with the upstream.
	// Default is `http`.
	// Can be `http`, `https`, `grpc`, or `grpcs`.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=http;https;grpc;grpcs;
	Scheme string `json:"scheme,omitempty" yaml:"scheme,omitempty"`

	// Retries defines the number of retry attempts APISIX should make when a failure occurs.
	// Failures include timeouts, network errors, or 5xx status codes.
	// +kubebuilder:validation:Optional
	Retries *int64 `json:"retries,omitempty" yaml:"retries,omitempty"`

	// Timeout specifies the connection, send, and read timeouts for upstream requests.
	// +kubebuilder:validation:Optional
	Timeout *UpstreamTimeout `json:"timeout,omitempty" yaml:"timeout,omitempty"`

	// HealthCheck defines the active and passive health check configuration for the upstream.
	// Deprecated: no longer supported in standalone mode.
	// +kubebuilder:validation:Optional
	HealthCheck *HealthCheck `json:"healthCheck,omitempty" yaml:"healthCheck,omitempty"`

	// TLSSecret references a Kubernetes Secret that contains the client certificate and key
	// for mutual TLS when connecting to the upstream.
	// +kubebuilder:validation:Optional
	TLSSecret *ApisixSecret `json:"tlsSecret,omitempty" yaml:"tlsSecret,omitempty"`

	// Subsets defines labeled subsets of service endpoints, typically used for
	// service versioning or canary deployments.
	// +kubebuilder:validation:Optional
	Subsets []ApisixUpstreamSubset `json:"subsets,omitempty" yaml:"subsets,omitempty"`

	// PassHost configures how the host header should be determined when a
	// request is forwarded to the upstream.
	// Default is `pass`.
	// Can be `pass`, `node` or `rewrite`:
	// * `pass`: preserve the original Host header
	// * `node`: use the upstream node’s host
	// * `rewrite`: set to a custom host via upstreamHost
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=pass;node;rewrite;
	PassHost string `json:"passHost,omitempty" yaml:"passHost,omitempty"`

	// UpstreamHost sets a custom Host header when passHost is set to `rewrite`.
	// +kubebuilder:validation:Optional
	UpstreamHost string `json:"upstreamHost,omitempty" yaml:"upstreamHost,omitempty"`

	// Discovery configures service discovery for the upstream.
	// Deprecated: no longer supported in standalone mode.
	// +kubebuilder:validation:Optional
	Discovery *Discovery `json:"discovery,omitempty" yaml:"discovery,omitempty"`
}

// PortLevelSettings configures the ApisixUpstreamConfig for each individual port. It inherits
// configuration from the outer level (the whole Kubernetes Service) and overrides some of
// them if they are set on the port level.
type PortLevelSettings struct {
	ApisixUpstreamConfig `json:",inline" yaml:",inline"`

	// Port is a Kubernetes Service port.
	Port int32 `json:"port" yaml:"port"`
}

// ApisixUpstreamExternalType is the external service type
type ApisixUpstreamExternalType string

// LoadBalancer defines the load balancing strategy for distributing traffic across upstream nodes.
type LoadBalancer struct {
	// Type specifies the load balancing algorithms to route traffic to the backend.
	// Default is `roundrobin`.
	// Can be `roundrobin`, `chash`, `ewma`, or `least_conn`.
	// +kubebuilder:validation:Enum=roundrobin;chash;ewma;least_conn;
	// +kubebuilder:default=roundrobin
	Type string `json:"type" yaml:"type"`
	// HashOn specified the type of field used for hashing, required when type is `chash`.
	// Default is `vars`. Can be `vars`, `header`, `cookie`, `consumer`, or `vars_combinations`.
	// +kubebuilder:validation:Enum=vars;header;cookie;consumer;vars_combinations;
	// +kubebuilder:default=vars
	HashOn string `json:"hashOn,omitempty" yaml:"hashOn,omitempty"`
	// Key is used with HashOn, generally required when type is `chash`.
	// When HashOn is `header` or `cookie`, specifies the name of the header or cookie.
	// When HashOn is `consumer`, key is not required, as the consumer name is used automatically.
	// When HashOn is `vars` or `vars_combinations`, key refers to one or a combination of
	// [built-in variables](/enterprise/reference/built-in-variables).
	Key string `json:"key,omitempty" yaml:"key,omitempty"`
}

// HealthCheck defines the health check configuration for upstream nodes.
// It includes active checks (proactively probing the nodes) and optional passive checks (monitoring based on traffic).
type HealthCheck struct {
	// Active health checks proactively send requests to upstream nodes to determine their availability.
	// +kubebuilder:validation:Required
	Active *ActiveHealthCheck `json:"active" yaml:"active"`
	// Passive health checks evaluate upstream health based on observed traffic, such as timeouts or errors.
	// +kubebuilder:validation:Optional
	Passive *PassiveHealthCheck `json:"passive,omitempty" yaml:"passive,omitempty"`
}

// ApisixUpstreamSubset defines a single endpoints group of one Service.
type ApisixUpstreamSubset struct {
	// Name is the name of subset.
	Name string `json:"name" yaml:"name"`
	// Labels is the label set of this subset.
	Labels map[string]string `json:"labels" yaml:"labels"`
}

// Discovery defines the service discovery configuration for dynamically resolving upstream nodes.
// This is used when APISIX integrates with a service registry such as Nacos, Consul, or Eureka.
type Discovery struct {
	// ServiceName is the name of the service to discover.
	ServiceName string `json:"serviceName" yaml:"serviceName"`
	// Type is the name of the service discovery provider.
	Type string `json:"type" yaml:"type"`
	// Args contains additional configuration parameters required by the discovery provider.
	// These are passed as key-value pairs.
	// +kubebuilder:validation:Optional
	Args map[string]string `json:"args,omitempty" yaml:"args,omitempty"`
}

// ActiveHealthCheck defines the active upstream health check configuration.
type ActiveHealthCheck struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=http;https;tcp;
	// Type is the health check type. Can be `http`, `https`, or `tcp`.
	Type string `json:"type,omitempty" yaml:"type,omitempty"`
	// Timeout sets health check timeout in seconds.
	Timeout time.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	// +kubebuilder:validation:Minimum=0
	// Concurrency sets the number of targets to be checked at the same time.
	Concurrency int `json:"concurrency,omitempty" yaml:"concurrency,omitempty"`
	// Host sets the upstream host.
	Host string `json:"host,omitempty" yaml:"host,omitempty"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=65535
	// Port sets the upstream port.
	Port int32 `json:"port,omitempty" yaml:"port,omitempty"`
	// HTTPPath sets the HTTP probe request path.
	HTTPPath string `json:"httpPath,omitempty" yaml:"httpPath,omitempty"`
	// StrictTLS sets whether to enforce TLS.
	StrictTLS *bool `json:"strictTLS,omitempty" yaml:"strictTLS,omitempty"`
	// RequestHeaders sets the request headers.
	RequestHeaders []string `json:"requestHeaders,omitempty" yaml:"requestHeaders,omitempty"`
	// Healthy configures the rules that define an upstream node as healthy.
	Healthy *ActiveHealthCheckHealthy `json:"healthy,omitempty" yaml:"healthy,omitempty"`
	// Unhealthy configures the rules that define an upstream node as unhealthy.
	Unhealthy *ActiveHealthCheckUnhealthy `json:"unhealthy,omitempty" yaml:"unhealthy,omitempty"`
}

// PassiveHealthCheck defines the conditions used to determine whether
// an upstream node is healthy or unhealthy based on passive observations.
// Passive health checks rely on real traffic responses instead of active probes.
type PassiveHealthCheck struct {
	// Type specifies the type of passive health check.
	// Can be `http`, `https`, or `tcp`.
	Type string `json:"type,omitempty" yaml:"type,omitempty"`
	// Healthy defines the conditions under which an upstream node is considered healthy.
	Healthy *PassiveHealthCheckHealthy `json:"healthy,omitempty" yaml:"healthy,omitempty"`
	// Unhealthy defines the conditions under which an upstream node is considered unhealthy.
	Unhealthy *PassiveHealthCheckUnhealthy `json:"unhealthy,omitempty" yaml:"unhealthy,omitempty"`
}

// UpstreamActiveHealthCheckHealthy defines the conditions used to actively determine whether an upstream node is healthy.
type ActiveHealthCheckHealthy struct {
	PassiveHealthCheckHealthy `json:",inline" yaml:",inline"`

	// Interval defines the time interval for checking targets, in seconds.
	Interval metav1.Duration `json:"interval,omitempty" yaml:"interval,omitempty"`
}

// UpstreamActiveHealthCheckHealthy defines the conditions used to actively determine whether an upstream node is unhealthy.
type ActiveHealthCheckUnhealthy struct {
	PassiveHealthCheckUnhealthy `json:",inline" yaml:",inline"`

	// Interval defines the time interval for checking targets, in seconds.
	Interval metav1.Duration `json:"interval,omitempty" yaml:"interval,omitempty"`
}

// PassiveHealthCheckHealthy defines the conditions used to passively determine whether an upstream node is healthy.
type PassiveHealthCheckHealthy struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinItems=1
	// HTTPCodes define a list of HTTP status codes that are considered healthy.
	HTTPCodes []int `json:"httpCodes,omitempty" yaml:"httpCodes,omitempty"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=254
	// Successes define the number of successful probes to define a healthy target.
	Successes int `json:"successes,omitempty" yaml:"successes,omitempty"`
}

// UpstreamPassiveHealthCheckUnhealthy defines the conditions used to passively determine whether an upstream node is unhealthy.
type PassiveHealthCheckUnhealthy struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinItems=1
	// HTTPCodes define a list of HTTP status codes that are considered unhealthy.
	HTTPCodes []int `json:"httpCodes,omitempty" yaml:"httpCodes,omitempty"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=254
	// HTTPFailures define the number of HTTP failures to define an unhealthy target.
	HTTPFailures int `json:"httpFailures,omitempty" yaml:"http_failures,omitempty"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=254
	// TCPFailures define the number of TCP failures to define an unhealthy target.
	TCPFailures int `json:"tcpFailures,omitempty" yaml:"tcpFailures,omitempty"`
	// Timeout sets health check timeout in seconds.
	Timeouts int `json:"timeout,omitempty" yaml:"timeout,omitempty"`
}

func init() {
	SchemeBuilder.Register(&ApisixUpstream{}, &ApisixUpstreamList{})
}
