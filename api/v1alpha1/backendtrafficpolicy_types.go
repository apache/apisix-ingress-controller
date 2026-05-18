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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// BackendTrafficPolicy defines configuration for traffic handling policies applied to backend services.
type BackendTrafficPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// BackendTrafficPolicySpec defines traffic handling policies applied to backend services,
	// such as load balancing strategy, connection settings, and failover behavior.
	Spec   BackendTrafficPolicySpec `json:"spec,omitempty"`
	Status PolicyStatus             `json:"status,omitempty"`
}

type BackendTrafficPolicySpec struct {
	// TargetRef identifies an API object to apply policy to.
	// Currently, Backends (i.e. Service, ServiceImport, or any
	// implementation-specific backendRef) are the only valid API
	// target references.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	TargetRefs []BackendPolicyTargetReferenceWithSectionName `json:"targetRefs"`
	// LoadBalancer represents the load balancer configuration for Kubernetes Service.
	// The default strategy is round robin.
	LoadBalancer *LoadBalancer `json:"loadbalancer,omitempty" yaml:"loadbalancer,omitempty"`
	// Scheme is the protocol used to communicate with the upstream.
	// Default is `http`.
	// Can be `http`, `https`, `grpc`, or `grpcs`.
	// +kubebuilder:validation:Enum=http;https;grpc;grpcs;
	// +kubebuilder:default=http
	Scheme string `json:"scheme,omitempty" yaml:"scheme,omitempty"`

	// Retries specify the number of times the gateway should retry sending
	// requests when errors such as timeouts or 502 errors occur.
	// +optional
	Retries *int `json:"retries,omitempty" yaml:"retries,omitempty"`

	// Timeout sets the read, send, and connect timeouts to the upstream.
	Timeout *Timeout `json:"timeout,omitempty" yaml:"timeout,omitempty"`

	// PassHost configures how the host header should be determined when a
	// request is forwarded to the upstream.
	// Default is `pass`. Can be `pass`, `node` or `rewrite`:
	// * `pass`: preserve the original Host header
	// * `node`: use the upstream node’s host
	// * `rewrite`: set to a custom host via `upstreamHost`
	//
	// +kubebuilder:validation:Enum=pass;node;rewrite;
	// +kubebuilder:default=pass
	PassHost string `json:"passHost,omitempty" yaml:"passHost,omitempty"`

	// UpstreamHost specifies the host of the Upstream request. Used only if
	// passHost is set to `rewrite`.
	Host Hostname `json:"upstreamHost,omitempty" yaml:"upstreamHost,omitempty"`

	// HealthCheck defines active and passive health check configuration for
	// the upstream backends. When configured, APISIX will probe backends
	// (active) or monitor live traffic (passive) to detect and bypass
	// unhealthy nodes.
	// +optional
	HealthCheck *HealthCheck `json:"healthCheck,omitempty" yaml:"healthCheck,omitempty"`
}

// LoadBalancer describes the load balancing parameters.
// +kubebuilder:validation:XValidation:rule="!(has(self.key) && self.type != 'chash')"
type LoadBalancer struct {
	// Type specifies the load balancing algorithms to route traffic to the backend.
	// Default is `roundrobin`.
	// Can be `roundrobin`, `chash`, `ewma`, or `least_conn`.
	// +kubebuilder:validation:Enum=roundrobin;chash;ewma;least_conn;
	// +kubebuilder:default=roundrobin
	// +kubebuilder:validation:Required
	Type string `json:"type" yaml:"type"`
	// HashOn specified the type of field used for hashing, required when type is `chash`.
	// Default is `vars`.
	// Can be `vars`, `header`, `cookie`, `consumer`, or `vars_combinations`.
	// +kubebuilder:validation:Enum=vars;header;cookie;consumer;vars_combinations;
	// +kubebuilder:default=vars
	HashOn string `json:"hashOn,omitempty" yaml:"hashOn,omitempty"`
	// Key is used with HashOn, generally required when type is `chash`.
	// When HashOn is `header` or `cookie`, specifies the name of the header or cookie.
	// When HashOn is `consumer`, key is not required, as the consumer name is used automatically.
	// When HashOn is `vars` or `vars_combinations`, key refers to one or a combination of
	// [APISIX variable](https://apisix.apache.org/docs/apisix/apisix-variable/).
	Key string `json:"key,omitempty" yaml:"key,omitempty"`
}

type Timeout struct {
	// Connection timeout. Default is `60s`.
	// +kubebuilder:default="60s"
	// +kubebuilder:validation:Pattern=`^[0-9]+s$`
	// +kubebuilder:validation:Type=string
	Connect metav1.Duration `json:"connect,omitempty" yaml:"connect,omitempty"`
	// Send timeout. Default is `60s`.
	// +kubebuilder:default="60s"
	// +kubebuilder:validation:Pattern=`^[0-9]+s$`
	// +kubebuilder:validation:Type=string
	Send metav1.Duration `json:"send,omitempty" yaml:"send,omitempty"`
	// Read timeout. Default is `60s`.
	// +kubebuilder:default="60s"
	// +kubebuilder:validation:Pattern=`^[0-9]+s$`
	// +kubebuilder:validation:Type=string
	Read metav1.Duration `json:"read,omitempty" yaml:"read,omitempty"`
}

// +kubebuilder:object:root=true
type BackendTrafficPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BackendTrafficPolicy `json:"items"`
}

// HealthCheck defines the active and passive health check configuration for upstream nodes.
type HealthCheck struct {
	// Active health checks proactively send requests to upstream nodes to determine their availability.
	// +kubebuilder:validation:Required
	Active *ActiveHealthCheck `json:"active" yaml:"active"`
	// Passive health checks evaluate upstream health based on observed traffic (timeouts, errors).
	// +kubebuilder:validation:Optional
	Passive *PassiveHealthCheck `json:"passive,omitempty" yaml:"passive,omitempty"`
}

// ActiveHealthCheck defines the active upstream health check configuration.
type ActiveHealthCheck struct {
	// Type is the health check type. Can be `http`, `https`, or `tcp`.
	// +kubebuilder:validation:Enum=http;https;tcp;
	// +kubebuilder:default=http
	// +optional
	Type string `json:"type,omitempty" yaml:"type,omitempty"`

	// Timeout sets health check timeout.
	// +optional
	Timeout metav1.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty"`

	// Concurrency sets the number of targets to be checked at the same time.
	// +kubebuilder:validation:Minimum=0
	// +optional
	Concurrency int `json:"concurrency,omitempty" yaml:"concurrency,omitempty"`

	// Host sets the upstream host used in the health check request.
	// +optional
	Host string `json:"host,omitempty" yaml:"host,omitempty"`

	// Port sets the port on the upstream node to probe.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +optional
	Port int32 `json:"port,omitempty" yaml:"port,omitempty"`

	// HTTPPath sets the HTTP path for the probe request.
	// +optional
	HTTPPath string `json:"httpPath,omitempty" yaml:"httpPath,omitempty"`

	// StrictTLS controls whether TLS certificate validation is enforced.
	// +optional
	StrictTLS *bool `json:"strictTLS,omitempty" yaml:"strictTLS,omitempty"`

	// RequestHeaders sets additional HTTP request headers for the probe.
	// +optional
	RequestHeaders []string `json:"requestHeaders,omitempty" yaml:"requestHeaders,omitempty"`

	// Healthy configures the thresholds for marking a node healthy.
	// +optional
	Healthy *ActiveHealthCheckHealthy `json:"healthy,omitempty" yaml:"healthy,omitempty"`

	// Unhealthy configures the thresholds for marking a node unhealthy.
	// +optional
	Unhealthy *ActiveHealthCheckUnhealthy `json:"unhealthy,omitempty" yaml:"unhealthy,omitempty"`
}

// PassiveHealthCheck defines passive health check configuration based on observed traffic.
type PassiveHealthCheck struct {
	// Type is the passive health check type. Can be `http`, `https`, or `tcp`.
	// +kubebuilder:validation:Enum=http;https;tcp;
	// +kubebuilder:default=http
	// +optional
	Type string `json:"type,omitempty" yaml:"type,omitempty"`

	// Healthy defines conditions under which a node is considered healthy.
	// +optional
	Healthy *PassiveHealthCheckHealthy `json:"healthy,omitempty" yaml:"healthy,omitempty"`

	// Unhealthy defines conditions under which a node is considered unhealthy.
	// +optional
	Unhealthy *PassiveHealthCheckUnhealthy `json:"unhealthy,omitempty" yaml:"unhealthy,omitempty"`
}

// ActiveHealthCheckHealthy defines thresholds for actively marking an upstream node healthy.
type ActiveHealthCheckHealthy struct {
	PassiveHealthCheckHealthy `json:",inline" yaml:",inline"`

	// Interval defines the time between health check probes.
	// Minimum is 1s.
	Interval metav1.Duration `json:"interval,omitempty" yaml:"interval,omitempty"`
}

// ActiveHealthCheckUnhealthy defines thresholds for actively marking an upstream node unhealthy.
type ActiveHealthCheckUnhealthy struct {
	PassiveHealthCheckUnhealthy `json:",inline" yaml:",inline"`

	// Interval defines the time between health check probes.
	// Minimum is 1s.
	Interval metav1.Duration `json:"interval,omitempty" yaml:"interval,omitempty"`
}

// PassiveHealthCheckHealthy defines conditions for passively marking a node healthy.
type PassiveHealthCheckHealthy struct {
	// HTTPCodes is the list of HTTP status codes considered healthy.
	// +kubebuilder:validation:MinItems=1
	// +optional
	HTTPCodes []int `json:"httpCodes,omitempty" yaml:"httpCodes,omitempty"`

	// Successes is the number of consecutive successful responses required to mark a node healthy.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=254
	// +optional
	Successes int `json:"successes,omitempty" yaml:"successes,omitempty"`
}

// PassiveHealthCheckUnhealthy defines conditions for passively marking a node unhealthy.
type PassiveHealthCheckUnhealthy struct {
	// HTTPCodes is the list of HTTP status codes considered unhealthy.
	// +kubebuilder:validation:MinItems=1
	// +optional
	HTTPCodes []int `json:"httpCodes,omitempty" yaml:"httpCodes,omitempty"`

	// HTTPFailures is the number of HTTP failures to mark a node unhealthy.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=254
	// +optional
	HTTPFailures int `json:"httpFailures,omitempty" yaml:"httpFailures,omitempty"`

	// TCPFailures is the number of TCP failures to mark a node unhealthy.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=254
	// +optional
	TCPFailures int `json:"tcpFailures,omitempty" yaml:"tcpFailures,omitempty"`

	// Timeouts is the number of timeouts to mark a node unhealthy.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=254
	// +optional
	Timeouts int `json:"timeouts,omitempty" yaml:"timeouts,omitempty"`
}

func init() {
	SchemeBuilder.Register(&BackendTrafficPolicy{}, &BackendTrafficPolicyList{})
}
