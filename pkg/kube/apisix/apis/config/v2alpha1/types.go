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
package v2alpha1

import (
	"encoding/json"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/apache/apisix-ingress-controller/pkg/types"
)

const (
	// OpEqual means the equal ("==") operator in nginxVars.
	OpEqual = "Equal"
	// OpNotEqual means the not equal ("~=") operator in nginxVars.
	OpNotEqual = "NotEqual"
	// OpGreaterThan means the greater than (">") operator in nginxVars.
	OpGreaterThan = "GreaterThan"
	// OpGreaterThanEqual means the greater than (">=") operator in nginxVars.
	OpGreaterThanEqual = "GreaterThanEqual"
	// OpLessThan means the less than ("<") operator in nginxVars.
	OpLessThan = "LessThan"
	// OpLessThanEqual means the less than equal ("<=") operator in nginxVars.
	OpLessThanEqual = "LessThanEqual"
	// OpRegexMatch means the regex match ("~~") operator in nginxVars.
	OpRegexMatch = "RegexMatch"
	// OpRegexNotMatch means the regex not match ("!~~") operator in nginxVars.
	OpRegexNotMatch = "RegexNotMatch"
	// OpRegexMatchCaseInsensitive means the regex match "~*" (case insensitive mode) operator in nginxVars.
	OpRegexMatchCaseInsensitive = "RegexMatchCaseInsensitive"
	// OpRegexNotMatchCaseInsensitive means the regex not match "!~*" (case insensitive mode) operator in nginxVars.
	OpRegexNotMatchCaseInsensitive = "RegexNotMatchCaseInsensitive"
	// OpIn means the in operator ("in") in nginxVars.
	OpIn = "In"
	// OpNotIn means the not in operator ("not_in") in nginxVars.
	OpNotIn = "NotIn"

	// ScopeQuery means the route match expression subject is in the querystring.
	ScopeQuery = "Query"
	// ScopeHeader means the route match expression subject is in request headers.
	ScopeHeader = "Header"
	// ScopePath means the route match expression subject is the uri path.
	ScopePath = "Path"
	// ScopeCookie means the route match expression subject is in cookie.
	ScopeCookie = "Cookie"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// ApisixRoute is used to define the route rules and upstreams for Apache APISIX.
type ApisixRoute struct {
	metav1.TypeMeta   `json:",inline" yaml:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Spec              *ApisixRouteSpec `json:"spec,omitempty" yaml:"spec,omitempty"`
	Status            ApisixStatus     `json:"status,omitempty" yaml:"status,omitempty"`
}

// ApisixStatus is the status report for Apisix ingress Resources
type ApisixStatus struct {
	Conditions *[]metav1.Condition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

// ApisixRouteSpec is the spec definition for ApisixRouteSpec.
type ApisixRouteSpec struct {
	HTTP []*ApisixRouteHTTP `json:"http,omitempty" yaml:"http,omitempty"`
	TCP  []*ApisixRouteTCP  `json:"tcp,omitempty" yaml:"tcp,omitempty"`
}

// ApisixRouteHTTP represents a single route in for HTTP traffic.
type ApisixRouteHTTP struct {
	// The rule name, cannot be empty.
	Name string `json:"name" yaml:"name"`
	// Route priority, when multiple routes contains
	// same URI path (for path matching), route with
	// higher priority will take effect.
	Priority int                   `json:"priority,omitempty" yaml:"priority,omitempty"`
	Match    *ApisixRouteHTTPMatch `json:"match,omitempty" yaml:"match,omitempty"`
	// Deprecated: Backend will be removed in the future, use Backends instead.
	Backend *ApisixRouteHTTPBackend `json:"backend" yaml:"backend"`
	// Backends represents potential backends to proxy after the route
	// rule matched. When number of backends are more than one, traffic-split
	// plugin in APISIX will be used to split traffic based on the backend weight.
	Backends       []*ApisixRouteHTTPBackend  `json:"backends" yaml:"backends"`
	Websocket      bool                       `json:"websocket" yaml:"websocket"`
	Plugins        []*ApisixRouteHTTPPlugin   `json:"plugins,omitempty" yaml:"plugins,omitempty"`
	Authentication *ApisixRouteAuthentication `json:"authentication,omitempty" yaml:"authentication,omitempty"`
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
	ResolveGranularity string `json:"resolveGranularity" yaml:"resolveGranularity"`
	// Weight of this backend.
	Weight *int `json:"weight" yaml:"weight"`
	// Subset specifies a subset for the target Service. The subset should be pre-defined
	// in ApisixUpstream about this service.
	Subset string `json:"subset" yaml:"subset"`
}

// ApisixRouteHTTPPlugin represents an APISIX plugin.
type ApisixRouteHTTPPlugin struct {
	// The plugin name.
	Name string `json:"name" yaml:"name"`
	// Whether this plugin is in use, default is true.
	Enable bool `json:"enable" yaml:"enable"`
	// Plugin configuration.
	// TODO we may use protobuf to define it.
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
	KeyAuth ApisixRouteAuthenticationKeyAuth `json:"keyauth,omitempty" yaml:"keyauth,omitempty"`
}

// ApisixRouteAuthenticationKeyAuth is the keyAuth-related
// configuration in ApisixRouteAuthentication.
type ApisixRouteAuthenticationKeyAuth struct {
	Header string `json:"header,omitempty" yaml:"header,omitempty"`
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

// ApisixRouteTCP is the configuration for tcp route.
type ApisixRouteTCP struct {
	// The rule name, cannot be empty.
	Name    string                `json:"name" yaml:"name"`
	Match   ApisixRouteTCPMatch   `json:"match" yaml:"match"`
	Backend ApisixRouteTCPBackend `json:"backend" yaml:"backend"`
}

// ApisixRouteTCPMatch represents the match conditions of tcp route.
type ApisixRouteTCPMatch struct {
	// IngressPort represents the port listening on the Ingress proxy server.
	// It should be pre-defined as APISIX doesn't support dynamic listening.
	IngressPort int32 `json:"ingressPort" yaml:"ingressPort"`
}

// ApisixRouteTCPBackend represents a TCP backend (a Kubernetes Service).
type ApisixRouteTCPBackend struct {
	// The name (short) of the service, note cross namespace is forbidden,
	// so be sure the ApisixRoute and Service are in the same namespace.
	ServiceName string `json:"serviceName" yaml:"serviceName"`
	// The service port, could be the name or the port number.
	ServicePort intstr.IntOrString `json:"servicePort" yaml:"servicePort"`
	// The resolve granularity, can be "endpoints" or "service",
	// when set to "endpoints", the pod ips will be used; other
	// wise, the service ClusterIP or ExternalIP will be used,
	// default is endpoints.
	ResolveGranularity string `json:"resolveGranularity" yaml:"resolveGranularity"`
	// Subset specifies a subset for the target Service. The subset should be pre-defined
	// in ApisixUpstream about this service.
	Subset string `json:"subset" yaml:"subset"`
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
}

// ApisixConsumerBasicAuth defines the configuration for basic auth.
type ApisixConsumerBasicAuth struct {
	SecretRef *corev1.LocalObjectReference  `json:"secretRef,omitempty" yaml:"secretRef,omitempty"`
	Value     *ApisixConsumerBasicAuthValue `json:"value,omitempty" yaml:"value,omitempty"`
}

// ApisixConsumerBasicAuthValue defines the in-place username and password configuration for basic auth.
type ApisixConsumerBasicAuthValue struct {
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"username"`
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

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ApisixConsumerList contains a list of ApisixConsumer.
type ApisixConsumerList struct {
	metav1.TypeMeta `json:",inline" yaml:",inline"`
	metav1.ListMeta `json:"metadata" yaml:"metadata"`
	Items           []ApisixConsumer `json:"items,omitempty" yaml:"items,omitempty"`
}
