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
	"encoding/json"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2alpha1"
)

// +genclient
// +genclient:noStatus

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// ApisixRoute is used to define the route rules and upstreams for Apache APISIX.
// The definition closes the Kubernetes Ingress resource.
// +kubebuilder:resource:shortName=ar
// +kubebuilder:pruning:PreserveUnknownFields
type ApisixRoute struct {
	metav1.TypeMeta   `json:",inline" yaml:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Spec              *ApisixRouteSpec `json:"spec,omitempty" yaml:"spec,omitempty"`
}

// ApisixRouteSpec is the spec definition for ApisixRouteSpec.
type ApisixRouteSpec struct {
	Rules []Rule `json:"rules,omitempty" yaml:"rules,omitempty"`
}

// Rule represents a single route rule in ApisixRoute.
type Rule struct {
	Host string `json:"host,omitempty" yaml:"host,omitempty"`
	Http Http   `json:"http,omitempty" yaml:"http,omitempty"`
}

// Http represents all route rules in HTTP scope.
type Http struct {
	Paths []Path `json:"paths,omitempty" yaml:"paths,omitempty"`
}

// Path defines an URI based route rule.
type Path struct {
	Path    string   `json:"path,omitempty" yaml:"path,omitempty"`
	Backend Backend  `json:"backend,omitempty" yaml:"backend,omitempty"`
	Plugins []Plugin `json:"plugins,omitempty" yaml:"plugins,omitempty"`
}

// Backend defines an upstream, it should be an existing Kubernetes Service.
// Note the Service should be in the same namespace with ApisixRoute resource,
// i.e. cross namespacing is not allowed.
type Backend struct {
	ServiceName string `json:"serviceName,omitempty" yaml:"serviceName,omitempty"`
	ServicePort int    `json:"servicePort,omitempty" yaml:"servicePort,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ApisixRouteList struct {
	metav1.TypeMeta `json:",inline" yaml:",inline"`
	metav1.ListMeta `json:"metadata" yaml:"metadata"`
	Items           []ApisixRoute `json:"items,omitempty" yaml:"items,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// ApisixUpstream is a decorator for Kubernetes Service, it arms the Service
// with rich features like health check, retry policies, load balancer and others.
// It's designed to have same name with the Kubernetes Service and can be customized
// for individual port.
type ApisixUpstream struct {
	metav1.TypeMeta   `json:",inline" yaml:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	Spec   *ApisixUpstreamSpec   `json:"spec,omitempty" yaml:"spec,omitempty"`
	Status v2alpha1.ApisixStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

// ApisixUpstreamSpec describes the specification of ApisixUpstream.
type ApisixUpstreamSpec struct {
	ApisixUpstreamConfig `json:",inline" yaml:",inline"`

	PortLevelSettings []PortLevelSettings `json:"portLevelSettings,omitempty" yaml:"portLevelSettings,omitempty"`
}

// ApisixUpstreamConfig contains rich features on APISIX Upstream, for instance
// load balancer, health check and etc.
type ApisixUpstreamConfig struct {
	// LoadBalancer represents the load balancer configuration for Kubernetes Service.
	// The default strategy is round robin.
	// +optional
	LoadBalancer *LoadBalancer `json:"loadbalancer,omitempty" yaml:"loadbalancer,omitempty"`
	// The scheme used to talk with the upstream.
	// Now value can be http, grpc.
	// +optional
	Scheme string `json:"scheme,omitempty" yaml:"scheme,omitempty"`

	// How many times that the proxy (Apache APISIX) should do when
	// errors occur (error, timeout or bad http status codes like 500, 502).
	// +optional
	Retries int `json:"retries,omitempty" yaml:"retries,omitempty"`

	// Timeout settings for the read, send and connect to the upstream.
	// +optional
	Timeout *UpstreamTimeout `json:"timeout,omitempty" yaml:"timeout,omitempty"`

	// The health check configurations for the upstream.
	// +optional
	HealthCheck *HealthCheck `json:"healthCheck,omitempty" yaml:"healthCheck,omitempty"`
	// Subsets groups the service endpoints by their labels. Usually used to differentiate
	// service versions.
	// +optional
	Subsets []ApisixUpstreamSubset `json:"subsets,omitempty" yaml:"subsets,omitempty"`
}

// ApisixUpstreamSubset defines a single endpoints group of one Service.
type ApisixUpstreamSubset struct {
	// Name is the name of subset.
	Name string `json:"name" yaml:"name"`
	// Labels is the label set of this subset.
	Labels map[string]string `json:"labels" yaml:"labels"`
}

// UpstreamTimeout is settings for the read, send and connect to the upstream.
type UpstreamTimeout struct {
	Connect metav1.Duration `json:"connect,omitempty" yaml:"connect,omitempty"`
	Send    metav1.Duration `json:"send,omitempty" yaml:"send,omitempty"`
	Read    metav1.Duration `json:"read,omitempty" yaml:"read,omitempty"`
}

// PortLevelSettings configures the ApisixUpstreamConfig for each individual port. It inherits
// configurations from the outer level (the whole Kubernetes Service) and overrides some of
// them if they are set on the port level.
type PortLevelSettings struct {
	ApisixUpstreamConfig `json:",inline" yaml:",inline"`

	// Port is a Kubernetes Service port, it should be already defined.
	Port int32 `json:"port" yaml:"port"`
}

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
	HTTPCodes    []int         `json:"httpCodes,omitempty" yaml:"httpCodes,omitempty"`
	HTTPFailures int           `json:"httpFailures,omitempty" yaml:"http_failures,omitempty"`
	TCPFailures  int           `json:"tcpFailures,omitempty" yaml:"tcpFailures,omitempty"`
	Timeout      time.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ApisixUpstreamList struct {
	metav1.TypeMeta `json:",inline" yaml:",inline"`
	metav1.ListMeta `json:"metadata" yaml:"metadata"`
	Items           []ApisixUpstream `json:"items,omitempty" yaml:"items,omitempty"`
}

type Plugin struct {
	Name      string    `json:"name,omitempty" yaml:"name,omitempty"`
	Enable    bool      `json:"enable,omitempty" yaml:"enable,omitempty"`
	Config    Config    `json:"config,omitempty" yaml:"config,omitempty"`
	ConfigSet ConfigSet `json:"config_set,omitempty" yaml:"config_set,omitempty"`
}

type ConfigSet []interface{}

func (p ConfigSet) DeepCopyInto(out *ConfigSet) {
	b, _ := json.Marshal(&p)
	_ = json.Unmarshal(b, out)
}

func (p *ConfigSet) DeepCopy() *ConfigSet {
	if p == nil {
		return nil
	}
	out := new(ConfigSet)
	p.DeepCopyInto(out)
	return out
}

type Config map[string]interface{}

func (p Config) DeepCopyInto(out *Config) {
	b, _ := json.Marshal(&p)
	_ = json.Unmarshal(b, out)
}

func (p *Config) DeepCopy() *Config {
	if p == nil {
		return nil
	}
	out := new(Config)
	p.DeepCopyInto(out)
	return out
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:shortName=atls
// +kubebuilder:subresource:status
// ApisixTls defines SSL resource in APISIX.
type ApisixTls struct {
	metav1.TypeMeta   `json:",inline" yaml:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Spec              *ApisixTlsSpec `json:"spec,omitempty" yaml:"spec,omitempty"`
	// +optional
	Status v2alpha1.ApisixStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:printcolumn:name="SNIs",type=string,JSONPath=`.spec.hosts`
// +kubebuilder:printcolumn:name="Secret Name",type=string,JSONPath=`.spec.secret.name`
// +kubebuilder:printcolumn:name="Secret Namespace",type=string,JSONPath=`.spec.secret.namespace`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:printcolumn:name="Client CA Secret Name",type=string,JSONPath=`.spec.client.ca.name`
// +kubebuilder:printcolumn:name="Client CA Secret Namespace",type=string,JSONPath=`.spec.client.ca.namespace`
type ApisixTlsList struct {
	metav1.TypeMeta `json:",inline" yaml:",inline"`
	metav1.ListMeta `json:"metadata" yaml:"metadata"`
	Items           []ApisixTls `json:"items,omitempty" yaml:"items,omitempty"`
}

// +kubebuilder:validation:Pattern="^\\*?[0-9a-zA-Z-.]+$"
type HostType string

// ApisixTlsSpec is the specification of ApisixSSL.
type ApisixTlsSpec struct {
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Hosts []HostType `json:"hosts" yaml:"hosts,omitempty"`
	// +required
	// +kubebuilder:validation:Required
	Secret ApisixSecret `json:"secret" yaml:"secret"`
	// +optional
	Client *ApisixMutualTlsClientConfig `json:"client,omitempty" yaml:"client,omitempty"`
}

// ApisixSecret describes the Kubernetes Secret name and namespace.
type ApisixSecret struct {
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Name string `json:"name" yaml:"name"`
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Namespace string `json:"namespace" yaml:"namespace"`
}

// ApisixMutualTlsClientConfig describes the mutual TLS CA and verify depth
type ApisixMutualTlsClientConfig struct {
	CASecret ApisixSecret `json:"caSecret,omitempty" yaml:"caSecret,omitempty"`
	Depth    int          `json:"depth,omitempty" yaml:"depth,omitempty"`
}
