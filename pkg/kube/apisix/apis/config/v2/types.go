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
package v2

import (
	"encoding/json"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/apache/apisix-ingress-controller/pkg/types"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// ApisixRoute is used to define the route rules and upstreams for Apache APISIX.
type ApisixRoute struct {
	metav1.TypeMeta   `json:",inline" yaml:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Spec              ApisixRouteSpec `json:"spec,omitempty" yaml:"spec,omitempty"`
	Status            ApisixStatus    `json:"status,omitempty" yaml:"status,omitempty"`
}

// ApisixStatus is the status report for Apisix ingress Resources
type ApisixStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

// ApisixRouteSpec is the spec definition for ApisixRouteSpec.
type ApisixRouteSpec struct {
	HTTP   []ApisixRouteHTTP   `json:"http,omitempty" yaml:"http,omitempty"`
	Stream []ApisixRouteStream `json:"stream,omitempty" yaml:"stream,omitempty"`
}

// UpstreamTimeout is settings for the read, send and connect to the upstream.
type UpstreamTimeout struct {
	Connect metav1.Duration `json:"connect,omitempty" yaml:"connect,omitempty"`
	Send    metav1.Duration `json:"send,omitempty" yaml:"send,omitempty"`
	Read    metav1.Duration `json:"read,omitempty" yaml:"read,omitempty"`
}

// ApisixRouteHTTP represents a single route in for HTTP traffic.
type ApisixRouteHTTP struct {
	// The rule name, cannot be empty.
	Name string `json:"name" yaml:"name"`
	// Route priority, when multiple routes contains
	// same URI path (for path matching), route with
	// higher priority will take effect.
	Priority int                  `json:"priority,omitempty" yaml:"priority,omitempty"`
	Timeout  *UpstreamTimeout     `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Match    ApisixRouteHTTPMatch `json:"match,omitempty" yaml:"match,omitempty"`
	// Backends represents potential backends to proxy after the route
	// rule matched. When number of backends are more than one, traffic-split
	// plugin in APISIX will be used to split traffic based on the backend weight.
	Backends         []ApisixRouteHTTPBackend  `json:"backends,omitempty" yaml:"backends,omitempty"`
	Websocket        bool                      `json:"websocket" yaml:"websocket"`
	PluginConfigName string                    `json:"plugin_config_name,omitempty" yaml:"plugin_config_name,omitempty"`
	Plugins          []ApisixRouteHTTPPlugin   `json:"plugins,omitempty" yaml:"plugins,omitempty"`
	Authentication   ApisixRouteAuthentication `json:"authentication,omitempty" yaml:"authentication,omitempty"`
}

// ApisixRouteHTTPBackend represents a HTTP backend (a Kuberentes Service).
type ApisixRouteHTTPBackend struct {
	// The name (short) of the service, note cross namespace is forbidden,
	// so be sure the ApisixRoute and Service are in the same namespace.
	ServiceName string `json:"serviceName" yaml:"serviceName"`
	// The service port, could be the name or the port number.
	ServicePort intstr.IntOrString `json:"servicePort" yaml:"servicePort"`
	// The resolve granularity, can be "endpoints" or "service",
	// when set to "endpoints", the pod ips will be used; other
	// wise, the service ClusterIP or ExternalIP will be used,
	// default is endpoints.
	ResolveGranularity string `json:"resolveGranularity,omitempty" yaml:"resolveGranularity,omitempty"`
	// Weight of this backend.
	Weight *int `json:"weight" yaml:"weight"`
	// Subset specifies a subset for the target Service. The subset should be pre-defined
	// in ApisixUpstream about this service.
	Subset string `json:"subset,omitempty" yaml:"subset,omitempty"`
}

// ApisixRouteHTTPMatch represents the match condition for hitting this route.
type ApisixRouteHTTPMatch struct {
	// URI path predicates, at least one path should be
	// configured, path could be exact or prefix, for prefix path,
	// append "*" after it, for instance, "/foo*".
	Paths []string `json:"paths" yaml:"paths"`
	// HTTP request method predicates.
	Methods []string `json:"methods,omitempty" yaml:"methods,omitempty"`
	// HTTP Host predicates, host can be a wildcard domain or
	// an exact domain. For wildcard domain, only one generic
	// level is allowed, for instance, "*.foo.com" is valid but
	// "*.*.foo.com" is not.
	Hosts []string `json:"hosts,omitempty" yaml:"hosts,omitempty"`
	// Remote address predicates, items can be valid IPv4 address
	// or IPv6 address or CIDR.
	RemoteAddrs []string `json:"remoteAddrs,omitempty" yaml:"remoteAddrs,omitempty"`
	// NginxVars represents generic match predicates,
	// it uses Nginx variable systems, so any predicate
	// like headers, querystring and etc can be leveraged
	// here to match the route.
	// For instance, it can be:
	// nginxVars:
	//   - subject: "$remote_addr"
	//     op: in
	//     value:
	//       - "127.0.0.1"
	//       - "10.0.5.11"
	NginxVars []ApisixRouteHTTPMatchExpr `json:"exprs,omitempty" yaml:"exprs,omitempty"`
}

// ApisixRouteHTTPMatchExpr represents a binary route match expression .
type ApisixRouteHTTPMatchExpr struct {
	// Subject is the expression subject, it can
	// be any string composed by literals and nginx
	// vars.
	Subject ApisixRouteHTTPMatchExprSubject `json:"subject" yaml:"subject"`
	// Op is the operator.
	Op string `json:"op" yaml:"op"`
	// Set is an array type object of the expression.
	// It should be used when the Op is "in" or "not_in";
	Set []string `json:"set" yaml:"set"`
	// Value is the normal type object for the expression,
	// it should be used when the Op is not "in" and "not_in".
	// Set and Value are exclusive so only of them can be set
	// in the same time.
	Value *string `json:"value" yaml:"value"`
}

// ApisixRouteHTTPMatchExprSubject describes the route match expression subject.
type ApisixRouteHTTPMatchExprSubject struct {
	// The subject scope, can be:
	// ScopeQuery, ScopeHeader, ScopePath
	// when subject is ScopePath, Name field
	// will be ignored.
	Scope string `json:"scope" yaml:"scope"`
	// The name of subject.
	Name string `json:"name" yaml:"name"`
}

// ApisixRouteHTTPPlugin represents an APISIX plugin.
type ApisixRouteHTTPPlugin struct {
	// The plugin name.
	Name string `json:"name" yaml:"name"`
	// Whether this plugin is in use, default is true.
	Enable bool `json:"enable" yaml:"enable"`
	// Plugin configuration.
	Config ApisixRouteHTTPPluginConfig `json:"config" yaml:"config"`
}

// ApisixRouteHTTPPluginConfig is the configuration for
// any plugins.
type ApisixRouteHTTPPluginConfig map[string]interface{}

// ApisixRouteAuthentication is the authentication-related
// configuration in ApisixRoute.
type ApisixRouteAuthentication struct {
	Enable  bool                             `json:"enable" yaml:"enable"`
	Type    string                           `json:"type" yaml:"type"`
	KeyAuth ApisixRouteAuthenticationKeyAuth `json:"keyAuth,omitempty" yaml:"keyAuth,omitempty"`
	JwtAuth ApisixRouteAuthenticationJwtAuth `json:"jwtAuth,omitempty" yaml:"jwtAuth,omitempty"`
}

// ApisixRouteAuthenticationKeyAuth is the keyAuth-related
// configuration in ApisixRouteAuthentication.
type ApisixRouteAuthenticationKeyAuth struct {
	Header string `json:"header,omitempty" yaml:"header,omitempty"`
}

// ApisixRouteAuthenticationJwtAuth is the jwt auth related
// configuration in ApisixRouteAuthentication.
type ApisixRouteAuthenticationJwtAuth struct {
	Header string `json:"header,omitempty" yaml:"header,omitempty"`
	Query  string `json:"query,omitempty" yaml:"query,omitempty"`
	Cookie string `json:"cookie,omitempty" yaml:"cookie,omitempty"`
}

func (p ApisixRouteHTTPPluginConfig) DeepCopyInto(out *ApisixRouteHTTPPluginConfig) {
	b, _ := json.Marshal(&p)
	_ = json.Unmarshal(b, out)
}

func (p *ApisixRouteHTTPPluginConfig) DeepCopy() *ApisixRouteHTTPPluginConfig {
	if p == nil {
		return nil
	}
	out := new(ApisixRouteHTTPPluginConfig)
	p.DeepCopyInto(out)
	return out
}

// ApisixRouteStream is the configuration for level 4 route
type ApisixRouteStream struct {
	// The rule name, cannot be empty.
	Name     string                    `json:"name" yaml:"name"`
	Protocol string                    `json:"protocol" yaml:"protocol"`
	Match    ApisixRouteStreamMatch    `json:"match" yaml:"match"`
	Backend  ApisixRouteStreamBackend  `json:"backend" yaml:"backend"`
	Plugins  []ApisixRouteStreamPlugin `json:"plugins,omitempty" yaml:"plugins,omitempty"`
}

// ApisixRouteStreamMatch represents the match conditions of stream route.
type ApisixRouteStreamMatch struct {
	// IngressPort represents the port listening on the Ingress proxy server.
	// It should be pre-defined as APISIX doesn't support dynamic listening.
	IngressPort int32 `json:"ingressPort" yaml:"ingressPort"`
}

// ApisixRouteStreamBackend represents a TCP backend (a Kubernetes Service).
type ApisixRouteStreamBackend struct {
	// The name (short) of the service, note cross namespace is forbidden,
	// so be sure the ApisixRoute and Service are in the same namespace.
	ServiceName string `json:"serviceName" yaml:"serviceName"`
	// The service port, could be the name or the port number.
	ServicePort intstr.IntOrString `json:"servicePort" yaml:"servicePort"`
	// The resolve granularity, can be "endpoints" or "service",
	// when set to "endpoints", the pod ips will be used; other
	// wise, the service ClusterIP or ExternalIP will be used,
	// default is endpoints.
	ResolveGranularity string `json:"resolveGranularity,omitempty" yaml:"resolveGranularity,omitempty"`
	// Subset specifies a subset for the target Service. The subset should be pre-defined
	// in ApisixUpstream about this service.
	Subset string `json:"subset,omitempty" yaml:"subset,omitempty"`
}

// ApisixRouteStreamPlugin represents an APISIX strem plugin.
type ApisixRouteStreamPlugin struct {
	// The plugin name.
	Name string `json:"name" yaml:"name"`
	// Whether this plugin is in use, default is true.
	Enable bool `json:"enable" yaml:"enable"`
	// Plugin configuration.
	Config ApisixRouteStreamPluginConfig `json:"config" yaml:"config"`
}

// ApisixRouteStreamPluginConfig is the configuration for
// any stream plugins.
type ApisixRouteStreamPluginConfig map[string]interface{}

func (p ApisixRouteStreamPluginConfig) DeepCopyInto(out *ApisixRouteStreamPluginConfig) {
	b, _ := json.Marshal(&p)
	_ = json.Unmarshal(b, out)
}

func (p *ApisixRouteStreamPluginConfig) DeepCopy() *ApisixRouteStreamPluginConfig {
	if p == nil {
		return nil
	}
	out := new(ApisixRouteStreamPluginConfig)
	p.DeepCopyInto(out)
	return out
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ApisixRouteList contains a list of ApisixRoute.
type ApisixRouteList struct {
	metav1.TypeMeta `json:",inline" yaml:",inline"`
	metav1.ListMeta `json:"metadata" yaml:"metadata"`
	Items           []ApisixRoute `json:"items,omitempty" yaml:"items,omitempty"`
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status

// ApisixClusterConfig is the Schema for the ApisixClusterConfig resource.
// An ApisixClusterConfig is used to identify an APISIX cluster, it's a
// ClusterScoped resource so the name is unique.
// It also contains some cluster-level configurations like monitoring.
type ApisixClusterConfig struct {
	metav1.TypeMeta   `json:",inline" yaml:",inline"`
	metav1.ObjectMeta `json:"metadata" yaml:"metadata"`

	// Spec defines the desired state of ApisixClusterConfigSpec.
	Spec   ApisixClusterConfigSpec `json:"spec" yaml:"spec"`
	Status ApisixStatus            `json:"status,omitempty" yaml:"status,omitempty"`
}

// ApisixClusterConfigSpec defines the desired state of ApisixClusterConfigSpec.
type ApisixClusterConfigSpec struct {
	// Monitoring categories all monitoring related features.
	// +optional
	Monitoring *ApisixClusterMonitoringConfig `json:"monitoring" yaml:"monitoring"`
	// Admin contains the Admin API information about APISIX cluster.
	// +optional
	Admin *ApisixClusterAdminConfig `json:"admin" yaml:"admin"`
}

// ApisixClusterMonitoringConfig categories all monitoring related features.
type ApisixClusterMonitoringConfig struct {
	// Prometheus is the config for using Prometheus in APISIX Cluster.
	// +optional
	Prometheus ApisixClusterPrometheusConfig `json:"prometheus" yaml:"prometheus"`
	// Skywalking is the config for using Skywalking in APISIX Cluster.
	// +optional
	Skywalking ApisixClusterSkywalkingConfig `json:"skywalking" yaml:"skywalking"`
}

// ApisixClusterPrometheusConfig is the config for using Prometheus in APISIX Cluster.
type ApisixClusterPrometheusConfig struct {
	// Enable means whether enable Prometheus or not.
	Enable bool `json:"enable" yaml:"enable"`
}

// ApisixClusterSkywalkingConfig is the config for using Skywalking in APISIX Cluster.
type ApisixClusterSkywalkingConfig struct {
	// Enable means whether enable Skywalking or not.
	Enable bool `json:"enable" yaml:"enable"`
	// SampleRatio means the ratio to collect
	SampleRatio float64 `json:"sampleRatio" yaml:"sampleRatio"`
}

// ApisixClusterAdminConfig is the admin config for the corresponding APISIX Cluster.
type ApisixClusterAdminConfig struct {
	// BaseURL is the base URL for the APISIX Admin API.
	// It looks like "http://apisix-admin.default.svc.cluster.local:9080/apisix/admin"
	BaseURL string `json:"baseURL" yaml:"baseURL"`
	// AdminKey is used to verify the admin API user.
	AdminKey string `json:"adminKey" yaml:"adminKey"`
	// ClientTimeout is request timeout for the APISIX Admin API client
	ClientTimeout types.TimeDuration `json:"clientTimeout" yaml:"clientTimeout"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ApisixClusterConfigList contains a list of ApisixClusterConfig.
type ApisixClusterConfigList struct {
	metav1.TypeMeta `json:",inline" yaml:",inline"`
	metav1.ListMeta `json:"metadata" yaml:"metadata"`

	Items []ApisixClusterConfig `json:"items" yaml:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status

// ApisixConsumer is the Schema for the ApisixConsumer resource.
// An ApisixConsumer is used to identify a consumer.
type ApisixConsumer struct {
	metav1.TypeMeta   `json:",inline" yaml:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Spec              ApisixConsumerSpec `json:"spec,omitempty" yaml:"spec,omitempty"`
	Status            ApisixStatus       `json:"status,omitempty" yaml:"status,omitempty"`
}

// ApisixConsumerSpec defines the desired state of ApisixConsumer.
type ApisixConsumerSpec struct {
	AuthParameter ApisixConsumerAuthParameter `json:"authParameter" yaml:"authParameter"`
}

type ApisixConsumerAuthParameter struct {
	BasicAuth *ApisixConsumerBasicAuth `json:"basicAuth,omitempty" yaml:"basicAuth"`
	KeyAuth   *ApisixConsumerKeyAuth   `json:"keyAuth,omitempty" yaml:"keyAuth"`
	WolfRBAC  *ApisixConsumerWolfRBAC  `json:"wolfRBAC,omitempty" yaml:"wolfRBAC"`
	JwtAuth   *ApisixConsumerJwtAuth   `json:"jwtAuth,omitempty" yaml:"jwtAuth"`
	HMACAuth  *ApisixConsumerHMACAuth  `json:"hmacAuth,omitempty" yaml:"hmacAuth"`
}

// ApisixConsumerBasicAuth defines the configuration for basic auth.
type ApisixConsumerBasicAuth struct {
	SecretRef *corev1.LocalObjectReference  `json:"secretRef,omitempty" yaml:"secretRef,omitempty"`
	Value     *ApisixConsumerBasicAuthValue `json:"value,omitempty" yaml:"value,omitempty"`
}

// ApisixConsumerBasicAuthValue defines the in-place username and password configuration for basic auth.
type ApisixConsumerBasicAuthValue struct {
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
}

// ApisixConsumerKeyAuth defines the configuration for the key auth.
type ApisixConsumerKeyAuth struct {
	SecretRef *corev1.LocalObjectReference `json:"secretRef,omitempty" yaml:"secretRef,omitempty"`
	Value     *ApisixConsumerKeyAuthValue  `json:"value,omitempty" yaml:"value,omitempty"`
}

// ApisixConsumerKeyAuthValue defines the in-place configuration for basic auth.
type ApisixConsumerKeyAuthValue struct {
	Key string `json:"key" yaml:"key"`
}

// ApisixConsumerWolfRBAC defines the configuration for the wolf-rbac auth.
type ApisixConsumerWolfRBAC struct {
	SecretRef *corev1.LocalObjectReference `json:"secretRef,omitempty" yaml:"secretRef,omitempty"`
	Value     *ApisixConsumerWolfRBACValue `json:"value,omitempty" yaml:"value,omitempty"`
}

// ApisixConsumerWolfRBAC defines the in-place server and appid and header_prefix configuration for wolf-rbac auth.
type ApisixConsumerWolfRBACValue struct {
	Server       string `json:"server,omitempty" yaml:"server,omitempty"`
	Appid        string `json:"appid,omitempty" yaml:"appid,omitempty"`
	HeaderPrefix string `json:"header_prefix,omitempty" yaml:"header_prefix,omitempty"`
}

// ApisixConsumerJwtAuth defines the configuration for the jwt auth.
type ApisixConsumerJwtAuth struct {
	SecretRef *corev1.LocalObjectReference `json:"secretRef,omitempty" yaml:"secretRef,omitempty"`
	Value     *ApisixConsumerJwtAuthValue  `json:"value,omitempty" yaml:"value,omitempty"`
}

// ApisixConsumerJwtAuthValue defines the in-place configuration for jwt auth.
type ApisixConsumerJwtAuthValue struct {
	Key          string `json:"key" yaml:"key"`
	Secret       string `json:"secret,omitempty" yaml:"secret,omitempty"`
	PublicKey    string `json:"public_key,omitempty" yaml:"public_key,omitempty"`
	PrivateKey   string `json:"private_key" yaml:"private_key,omitempty"`
	Algorithm    string `json:"algorithm,omitempty" yaml:"algorithm,omitempty"`
	Exp          int64  `json:"exp,omitempty" yaml:"exp,omitempty"`
	Base64Secret bool   `json:"base64_secret,omitempty" yaml:"base64_secret,omitempty"`
}

// ApisixConsumerHMACAuth defines the configuration for the hmac auth.
type ApisixConsumerHMACAuth struct {
	SecretRef *corev1.LocalObjectReference `json:"secretRef,omitempty" yaml:"secretRef,omitempty"`
	Value     *ApisixConsumerHMACAuthValue `json:"value,omitempty" yaml:"value,omitempty"`
}

// ApisixConsumerHMACAuthValue defines the in-place configuration for hmac auth.
type ApisixConsumerHMACAuthValue struct {
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

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ApisixConsumerList contains a list of ApisixConsumer.
type ApisixConsumerList struct {
	metav1.TypeMeta `json:",inline" yaml:",inline"`
	metav1.ListMeta `json:"metadata" yaml:"metadata"`
	Items           []ApisixConsumer `json:"items,omitempty" yaml:"items,omitempty"`
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

	Spec   *ApisixUpstreamSpec `json:"spec,omitempty" yaml:"spec,omitempty"`
	Status ApisixStatus        `json:"status,omitempty" yaml:"status,omitempty"`
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
	Retries *int `json:"retries,omitempty" yaml:"retries,omitempty"`

	// Timeout settings for the read, send and connect to the upstream.
	// +optional
	Timeout *UpstreamTimeout `json:"timeout,omitempty" yaml:"timeout,omitempty"`

	// The health check configurations for the upstream.
	// +optional
	HealthCheck *HealthCheck `json:"healthCheck,omitempty" yaml:"healthCheck,omitempty"`

	// Set the client certificate when connecting to TLS upstream.
	// +optional
	TLSSecret *ApisixSecret `json:"tlsSecret,omitempty" yaml:"tlsSecret,omitempty"`

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
	HTTPCodes    []int `json:"httpCodes,omitempty" yaml:"httpCodes,omitempty"`
	HTTPFailures int   `json:"httpFailures,omitempty" yaml:"http_failures,omitempty"`
	TCPFailures  int   `json:"tcpFailures,omitempty" yaml:"tcpFailures,omitempty"`
	Timeouts     int   `json:"timeout,omitempty" yaml:"timeout,omitempty"`
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
	Status ApisixStatus `json:"status,omitempty" yaml:"status,omitempty"`
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

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status

// ApisixPluginConfig is the Schema for the ApisixPluginConfig resource.
// An ApisixPluginConfig is used to support a group of plugin configs
type ApisixPluginConfig struct {
	metav1.TypeMeta   `json:",inline" yaml:",inline"`
	metav1.ObjectMeta `json:"metadata" yaml:"metadata"`

	// Spec defines the desired state of ApisixPluginConfigSpec.
	Spec   ApisixPluginConfigSpec `json:"spec" yaml:"spec"`
	Status ApisixStatus           `json:"status,omitempty" yaml:"status,omitempty"`
}

// ApisixPluginConfigSpec defines the desired state of ApisixPluginConfigSpec.
type ApisixPluginConfigSpec struct {
	// Plugins contains a list of ApisixRouteHTTPPlugin
	// +required
	Plugins []ApisixRouteHTTPPlugin `json:"plugins" yaml:"plugins"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:generate=true

// ApisixPluginConfigList contains a list of ApisixPluginConfig.
type ApisixPluginConfigList struct {
	metav1.TypeMeta `json:",inline" yaml:",inline"`
	metav1.ListMeta `json:"metadata" yaml:"metadata"`
	Items           []ApisixPluginConfig `json:"items,omitempty" yaml:"items,omitempty"`
}
