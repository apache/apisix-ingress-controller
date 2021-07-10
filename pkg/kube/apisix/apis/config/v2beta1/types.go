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
package v2beta1

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2alpha1"
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

// ApisixRouteHTTP represents a single route in for HTTP traffic.
type ApisixRouteHTTP struct {
	// The rule name, cannot be empty.
	Name string `json:"name" yaml:"name"`
	// Route priority, when multiple routes contains
	// same URI path (for path matching), route with
	// higher priority will take effect.
	Priority int                  `json:"priority,omitempty" yaml:"priority,omitempty"`
	Match    ApisixRouteHTTPMatch `json:"match,omitempty" yaml:"match,omitempty"`
	// Deprecated: Backend will be removed in the future, use Backends instead.
	Backend v2alpha1.ApisixRouteHTTPBackend `json:"backend" yaml:"backend"`
	// Backends represents potential backends to proxy after the route
	// rule matched. When number of backends are more than one, traffic-split
	// plugin in APISIX will be used to split traffic based on the backend weight.
	Backends       []v2alpha1.ApisixRouteHTTPBackend `json:"backends" yaml:"backends"`
	Websocket      bool                              `json:"websocket" yaml:"websocket"`
	Plugins        []ApisixRouteHTTPPlugin           `json:"plugins,omitempty" yaml:"plugins,omitempty"`
	Authentication ApisixRouteAuthentication         `json:"authentication,omitempty" yaml:"authentication,omitempty"`
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
	NginxVars []v2alpha1.ApisixRouteHTTPMatchExpr `json:"exprs,omitempty" yaml:"exprs,omitempty"`
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

// ApisixRouteStream is the configuration for level 4 route
type ApisixRouteStream struct {
	// The rule name, cannot be empty.
	Name     string                   `json:"name" yaml:"name"`
	Protocol string                   `json:"protocol" yaml:"protocol"`
	Match    ApisixRouteStreamMatch   `json:"match" yaml:"match"`
	Backend  ApisixRouteStreamBackend `json:"backend" yaml:"backend"`
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
