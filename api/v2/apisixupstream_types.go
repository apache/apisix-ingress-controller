// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v2

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ApisixUpstreamSpec describes the specification of ApisixUpstream.
type ApisixUpstreamSpec struct {
	// IngressClassName is the name of an IngressClass cluster resource.
	// controller implementations use this field to know whether they should be
	// serving this ApisixUpstream resource, by a transitive connection
	// (controller -> IngressClass -> ApisixUpstream resource).
	// +kubebuilder:validation:Optional
	IngressClassName string `json:"ingressClassName,omitempty" yaml:"ingressClassName,omitempty"`
	// ExternalNodes contains external nodes the Upstream should use
	// If this field is set, the upstream will use these nodes directly without any further resolves
	// +kubebuilder:validation:Optional
	ExternalNodes []ApisixUpstreamExternalNode `json:"externalNodes,omitempty" yaml:"externalNodes,omitempty"`

	ApisixUpstreamConfig `json:",inline" yaml:",inline"`

	PortLevelSettings []PortLevelSettings `json:"portLevelSettings,omitempty" yaml:"portLevelSettings,omitempty"`
}

// ApisixUpstreamStatus defines the observed state of ApisixUpstream.
type ApisixUpstreamStatus = ApisixStatus

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ApisixUpstream is the Schema for the apisixupstreams API.
type ApisixUpstream struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

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

// ApisixUpstreamExternalNode is the external node conf
type ApisixUpstreamExternalNode struct {
	Name string                     `json:"name,omitempty" yaml:"name"`
	Type ApisixUpstreamExternalType `json:"type,omitempty" yaml:"type"`
	// +kubebuilder:validation:Optional
	Weight *int `json:"weight,omitempty" yaml:"weight"`
	// Port defines the port of the external node
	// +kubebuilder:validation:Optional
	Port *int `json:"port,omitempty" yaml:"port"`
}

// ApisixUpstreamConfig contains rich features on APISIX Upstream, for instance
// load balancer, health check, etc.
type ApisixUpstreamConfig struct {
	// LoadBalancer represents the load balancer configuration for Kubernetes Service.
	// The default strategy is round robin.
	// +kubebuilder:validation:Optional
	LoadBalancer *LoadBalancer `json:"loadbalancer,omitempty" yaml:"loadbalancer,omitempty"`
	// The scheme used to talk with the upstream.
	// Now value can be http, grpc.
	// +kubebuilder:validation:Optional
	Scheme string `json:"scheme,omitempty" yaml:"scheme,omitempty"`

	// How many times that the proxy (Apache APISIX) should do when
	// errors occur (error, timeout or bad http status codes like 500, 502).
	// +kubebuilder:validation:Optional
	Retries *int `json:"retries,omitempty" yaml:"retries,omitempty"`

	// Timeout settings for the read, send and connect to the upstream.
	// +kubebuilder:validation:Optional
	Timeout *UpstreamTimeout `json:"timeout,omitempty" yaml:"timeout,omitempty"`

	// The health check configurations for the upstream.
	// +kubebuilder:validation:Optional
	HealthCheck *HealthCheck `json:"healthCheck,omitempty" yaml:"healthCheck,omitempty"`

	// Set the client certificate when connecting to TLS upstream.
	// +kubebuilder:validation:Optional
	TLSSecret *ApisixSecret `json:"tlsSecret,omitempty" yaml:"tlsSecret,omitempty"`

	// Subsets groups the service endpoints by their labels. Usually used to differentiate
	// service versions.
	// +kubebuilder:validation:Optional
	Subsets []ApisixUpstreamSubset `json:"subsets,omitempty" yaml:"subsets,omitempty"`

	// Configures the host when the request is forwarded to the upstream.
	// Can be one of pass, node or rewrite.
	// +kubebuilder:validation:Optional
	PassHost string `json:"passHost,omitempty" yaml:"passHost,omitempty"`

	// Specifies the host of the Upstream request. This is only valid if
	// the pass_host is set to rewrite
	// +kubebuilder:validation:Optional
	UpstreamHost string `json:"upstreamHost,omitempty" yaml:"upstreamHost,omitempty"`

	// Discovery is used to configure service discovery for upstream.
	// +kubebuilder:validation:Optional
	Discovery *Discovery `json:"discovery,omitempty" yaml:"discovery,omitempty"`
}

// PortLevelSettings configures the ApisixUpstreamConfig for each individual port. It inherits
// configurations from the outer level (the whole Kubernetes Service) and overrides some of
// them if they are set on the port level.
type PortLevelSettings struct {
	ApisixUpstreamConfig `json:",inline" yaml:",inline"`

	// Port is a Kubernetes Service port, it should be already defined.
	Port int32 `json:"port" yaml:"port"`
}

// ApisixUpstreamExternalType is the external service type
type ApisixUpstreamExternalType string

// LoadBalancer describes the load balancing parameters.
type LoadBalancer struct {
	Type string `json:"type" yaml:"type"`
	// The HashOn and Key fields are required when Type is "chash".
	// HashOn represents the key fetching scope.
	HashOn string `json:"hashOn,omitempty" yaml:"hashOn,omitempty"`
	// Key represents the hash key.
	Key string `json:"key,omitempty" yaml:"key,omitempty"`
}

// HealthCheck describes the upstream health check parameters.
type HealthCheck struct {
	Active  *ActiveHealthCheck  `json:"active" yaml:"active"`
	Passive *PassiveHealthCheck `json:"passive,omitempty" yaml:"passive,omitempty"`
}

// ApisixUpstreamSubset defines a single endpoints group of one Service.
type ApisixUpstreamSubset struct {
	// Name is the name of subset.
	Name string `json:"name" yaml:"name"`
	// Labels is the label set of this subset.
	Labels map[string]string `json:"labels" yaml:"labels"`
}

// Discovery defines Service discovery related configuration.
type Discovery struct {
	ServiceName string            `json:"serviceName" yaml:"serviceName"`
	Type        string            `json:"type" yaml:"type"`
	Args        map[string]string `json:"args,omitempty" yaml:"args,omitempty"`
}

// ActiveHealthCheck defines the active kind of upstream health check.
type ActiveHealthCheck struct {
	Type           string                      `json:"type,omitempty" yaml:"type,omitempty"`
	Timeout        time.Duration               `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Concurrency    int                         `json:"concurrency,omitempty" yaml:"concurrency,omitempty"`
	Host           string                      `json:"host,omitempty" yaml:"host,omitempty"`
	Port           int32                       `json:"port,omitempty" yaml:"port,omitempty"`
	HTTPPath       string                      `json:"httpPath,omitempty" yaml:"httpPath,omitempty"`
	StrictTLS      *bool                       `json:"strictTLS,omitempty" yaml:"strictTLS,omitempty"`
	RequestHeaders []string                    `json:"requestHeaders,omitempty" yaml:"requestHeaders,omitempty"`
	Healthy        *ActiveHealthCheckHealthy   `json:"healthy,omitempty" yaml:"healthy,omitempty"`
	Unhealthy      *ActiveHealthCheckUnhealthy `json:"unhealthy,omitempty" yaml:"unhealthy,omitempty"`
}

// PassiveHealthCheck defines the conditions to judge whether
// an upstream node is healthy with the passive manager.
type PassiveHealthCheck struct {
	Type      string                       `json:"type,omitempty" yaml:"type,omitempty"`
	Healthy   *PassiveHealthCheckHealthy   `json:"healthy,omitempty" yaml:"healthy,omitempty"`
	Unhealthy *PassiveHealthCheckUnhealthy `json:"unhealthy,omitempty" yaml:"unhealthy,omitempty"`
}

// ActiveHealthCheckHealthy defines the conditions to judge whether
// an upstream node is healthy with the active manner.
type ActiveHealthCheckHealthy struct {
	PassiveHealthCheckHealthy `json:",inline" yaml:",inline"`

	Interval metav1.Duration `json:"interval,omitempty" yaml:"interval,omitempty"`
}

// ActiveHealthCheckUnhealthy defines the conditions to judge whether
// an upstream node is unhealthy with the active manager.
type ActiveHealthCheckUnhealthy struct {
	PassiveHealthCheckUnhealthy `json:",inline" yaml:",inline"`

	Interval metav1.Duration `json:"interval,omitempty" yaml:"interval,omitempty"`
}

// PassiveHealthCheckHealthy defines the conditions to judge whether
// an upstream node is healthy with the passive manner.
type PassiveHealthCheckHealthy struct {
	HTTPCodes []int `json:"httpCodes,omitempty" yaml:"httpCodes,omitempty"`
	Successes int   `json:"successes,omitempty" yaml:"successes,omitempty"`
}

// PassiveHealthCheckUnhealthy defines the conditions to judge whether
// an upstream node is unhealthy with the passive manager.
type PassiveHealthCheckUnhealthy struct {
	HTTPCodes    []int `json:"httpCodes,omitempty" yaml:"httpCodes,omitempty"`
	HTTPFailures int   `json:"httpFailures,omitempty" yaml:"http_failures,omitempty"`
	TCPFailures  int   `json:"tcpFailures,omitempty" yaml:"tcpFailures,omitempty"`
	Timeouts     int   `json:"timeout,omitempty" yaml:"timeout,omitempty"`
}

func init() {
	SchemeBuilder.Register(&ApisixUpstream{}, &ApisixUpstreamList{})
}
