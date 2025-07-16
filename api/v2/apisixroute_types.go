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
	"strings"

	"github.com/pkg/errors"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/apache/apisix-ingress-controller/api/adc"
)

// ApisixRouteSpec is the spec definition for ApisixRoute.
// It defines routing rules for both HTTP and stream traffic.
type ApisixRouteSpec struct {
	// IngressClassName is the name of the IngressClass this route belongs to.
	// It allows multiple controllers to watch and reconcile different routes.
	IngressClassName string `json:"ingressClassName,omitempty" yaml:"ingressClassName,omitempty"`
	// HTTP defines a list of HTTP route rules.
	// Each rule specifies conditions to match HTTP requests and how to forward them.
	HTTP []ApisixRouteHTTP `json:"http,omitempty" yaml:"http,omitempty"`
	// Stream defines a list of stream route rules.
	// Each rule specifies conditions to match TCP/UDP traffic and how to forward them.
	Stream []ApisixRouteStream `json:"stream,omitempty" yaml:"stream,omitempty"`
}

// ApisixRouteStatus defines the observed state of ApisixRoute.
type ApisixRouteStatus = ApisixStatus

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=ar
// +kubebuilder:printcolumn:name="Hosts",type="string",JSONPath=".spec.http[].match.hosts",description="HTTP Hosts",priority=0
// +kubebuilder:printcolumn:name="URIs",type="string",JSONPath=".spec.http[].match.paths",description="HTTP Paths",priority=0
// +kubebuilder:printcolumn:name="Target Service (HTTP)",type="string",JSONPath=".spec.http[].backends[].serviceName",description="Backend Service for HTTP",priority=1
// +kubebuilder:printcolumn:name="Ingress Port (TCP)",type="integer",JSONPath=".spec.tcp[].match.ingressPort",description="TCP Ingress Port",priority=1
// +kubebuilder:printcolumn:name="Target Service (TCP)",type="string",JSONPath=".spec.tcp[].match.backend.serviceName",description="Backend Service for TCP",priority=1
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Creation time",priority=0

// ApisixRoute is defines configuration for HTTP and stream routes.
type ApisixRoute struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// ApisixRouteSpec defines HTTP and stream route configuration.
	Spec   ApisixRouteSpec   `json:"spec,omitempty"`
	Status ApisixRouteStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ApisixRouteList contains a list of ApisixRoute.
type ApisixRouteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ApisixRoute `json:"items"`
}

// ApisixRouteHTTP represents a single HTTP route configuration.
type ApisixRouteHTTP struct {
	// Name is the unique rule name and cannot be empty.
	Name string `json:"name" yaml:"name"`
	// Priority defines the route priority when multiple routes share the same URI path.
	// Higher values mean higher priority in route matching.
	Priority int `json:"priority,omitempty" yaml:"priority,omitempty"`
	// Timeout specifies upstream timeout settings.
	Timeout *UpstreamTimeout `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	// Match defines the HTTP request matching criteria.
	Match ApisixRouteHTTPMatch `json:"match,omitempty" yaml:"match,omitempty"`
	// Backends lists potential backend services to proxy requests to.
	// If more than one backend is specified, the `traffic-split` plugin is used
	// to distribute traffic according to backend weights.
	Backends []ApisixRouteHTTPBackend `json:"backends,omitempty" yaml:"backends,omitempty"`
	// Upstreams references ApisixUpstream CRDs.
	Upstreams []ApisixRouteUpstreamReference `json:"upstreams,omitempty" yaml:"upstreams,omitempty"`

	// Websocket enables or disables websocket support for this route.
	// +kubebuilder:validation:Optional
	Websocket bool `json:"websocket" yaml:"websocket"`
	// PluginConfigName specifies the name of the plugin config to apply.
	PluginConfigName string `json:"plugin_config_name,omitempty" yaml:"plugin_config_name,omitempty"`
	// PluginConfigNamespace specifies the namespace of the plugin config.
	// Defaults to the namespace of the ApisixRoute if not set.
	PluginConfigNamespace string `json:"plugin_config_namespace,omitempty" yaml:"plugin_config_namespace,omitempty"`
	// Plugins lists additional plugins applied to this route.
	Plugins []ApisixRoutePlugin `json:"plugins,omitempty" yaml:"plugins,omitempty"`
	// Authentication holds authentication-related configuration for this route.
	Authentication ApisixRouteAuthentication `json:"authentication,omitempty" yaml:"authentication,omitempty"`
}

// ApisixRouteStream defines the configuration for a Layer 4 (TCP/UDP) route.
type ApisixRouteStream struct {
	// Name is a unique identifier for the route. This field must not be empty.
	Name string `json:"name" yaml:"name"`
	// Protocol specifies the L4 protocol to match. Can be `tcp` or `udp`.
	Protocol string `json:"protocol" yaml:"protocol"`
	// Match defines the criteria used to match incoming TCP or UDP connections.
	Match ApisixRouteStreamMatch `json:"match" yaml:"match"`
	// Backend specifies the destination service to which traffic should be forwarded.
	Backend ApisixRouteStreamBackend `json:"backend" yaml:"backend"`
	// Plugins defines a list of plugins to apply to this route.
	Plugins []ApisixRoutePlugin `json:"plugins,omitempty" yaml:"plugins,omitempty"`
}

// UpstreamTimeout defines timeout settings for connecting, sending, and reading from the upstream.
type UpstreamTimeout struct {
	// Connect timeout for establishing a connection to the upstream.
	Connect metav1.Duration `json:"connect,omitempty" yaml:"connect,omitempty"`
	// Send timeout for sending data to the upstream.
	Send metav1.Duration `json:"send,omitempty" yaml:"send,omitempty"`
	// Read timeout for reading data from the upstream.
	Read metav1.Duration `json:"read,omitempty" yaml:"read,omitempty"`
}

// ApisixRouteHTTPMatch defines the conditions used to match incoming HTTP requests.
type ApisixRouteHTTPMatch struct {
	// Paths is a list of URI path patterns to match.
	// At least one path must be specified.
	// Supports exact matches and prefix matches.
	// For prefix matches, append `*` to the path, such as `/foo*`.
	Paths []string `json:"paths" yaml:"paths"`

	// Methods specifies the HTTP methods to match.
	Methods []string `json:"methods,omitempty" yaml:"methods,omitempty"`

	// Hosts specifies Host header values to match.
	// Supports exact and wildcard domains.
	// Only one level of wildcard is allowed (e.g., `*.example.com` is valid,
	// but `*.*.example.com` is not).
	Hosts []string `json:"hosts,omitempty" yaml:"hosts,omitempty"`

	// RemoteAddrs is a list of source IP addresses or CIDR ranges to match.
	// Supports both IPv4 and IPv6 formats.
	RemoteAddrs []string `json:"remoteAddrs,omitempty" yaml:"remoteAddrs,omitempty"`

	// NginxVars defines match conditions based on Nginx variables.
	NginxVars ApisixRouteHTTPMatchExprs `json:"exprs,omitempty" yaml:"exprs,omitempty"`

	// FilterFunc is a user-defined function for advanced request filtering.
	// The function can use Nginx variables through the `vars` parameter.
	// This field is supported in APISIX but not in API7 Enterprise.
	FilterFunc string `json:"filter_func,omitempty" yaml:"filter_func,omitempty"`
}

// ApisixRoutePlugin represents an APISIX plugin.
type ApisixRoutePlugin struct {
	// The plugin name.
	Name string `json:"name" yaml:"name"`
	// Whether this plugin is in use, default is true.
	// +kubebuilder:default=true
	Enable bool `json:"enable" yaml:"enable"`
	// Plugin configuration.
	// +kubebuilder:validation:Optional
	Config apiextensionsv1.JSON `json:"config" yaml:"config"`
	// Plugin configuration secretRef.
	// +kubebuilder:validation:Optional
	SecretRef string `json:"secretRef" yaml:"secretRef"`
}

// ApisixRouteHTTPBackend represents an HTTP backend (Kubernetes Service).
type ApisixRouteHTTPBackend struct {
	// ServiceName is the name of the Kubernetes Service.
	// Cross-namespace references are not supported—ensure the ApisixRoute
	// and the Service are in the same namespace.
	ServiceName string `json:"serviceName" yaml:"serviceName"`
	// ServicePort is the port of the Kubernetes Service.
	// This can be either the port name or port number.
	ServicePort intstr.IntOrString `json:"servicePort" yaml:"servicePort"`
	// ResolveGranularity determines how the backend service is resolved.
	// Valid values are `endpoints` and `service`. When set to `endpoints`,
	// individual pod IPs will be used; otherwise, the Service's ClusterIP or ExternalIP is used.
	// The default is `endpoints`.
	ResolveGranularity string `json:"resolveGranularity,omitempty" yaml:"resolveGranularity,omitempty"`
	// Weight specifies the relative traffic weight for this backend.
	// +kubebuilder:validation:Optional
	Weight *int `json:"weight" yaml:"weight"`
	// Subset specifies a named subset of the target Service.
	// The subset must be pre-defined in the corresponding ApisixUpstream resource.
	Subset string `json:"subset,omitempty" yaml:"subset,omitempty"`
}

// ApisixRouteUpstreamReference references an ApisixUpstream CRD to be used as a backend.
// It can be used in traffic-splitting scenarios or to select a specific upstream configuration.
type ApisixRouteUpstreamReference struct {
	// Name is the name of the ApisixUpstream resource.
	Name string `json:"name,omitempty" yaml:"name"`
	// Weight is the weight assigned to this upstream.
	// +kubebuilder:validation:Optional
	Weight *int `json:"weight,omitempty" yaml:"weight"`
}

// ApisixRouteAuthentication represents authentication-related configuration in ApisixRoute.
type ApisixRouteAuthentication struct {
	// Enable toggles authentication on or off.
	Enable bool `json:"enable" yaml:"enable"`
	// Type specifies the authentication type.
	Type string `json:"type" yaml:"type"`
	// KeyAuth defines configuration for key authentication.
	KeyAuth ApisixRouteAuthenticationKeyAuth `json:"keyAuth,omitempty" yaml:"keyAuth,omitempty"`
	// JwtAuth defines configuration for JWT authentication.
	JwtAuth ApisixRouteAuthenticationJwtAuth `json:"jwtAuth,omitempty" yaml:"jwtAuth,omitempty"`
	// LDAPAuth defines configuration for LDAP authentication.
	LDAPAuth ApisixRouteAuthenticationLDAPAuth `json:"ldapAuth,omitempty" yaml:"ldapAuth,omitempty"`
}

// ApisixRouteStreamMatch represents the matching conditions for a stream route.
type ApisixRouteStreamMatch struct {
	// IngressPort is the port on which the APISIX Ingress proxy server listens.
	// This must be a statically configured port, as APISIX does not support dynamic port binding.
	IngressPort int32 `json:"ingressPort" yaml:"ingressPort"`
	// Host is the destination host address used to match the incoming TCP/UDP traffic.
	Host string `json:"host,omitempty" yaml:"host,omitempty"`
}

// ApisixRouteStreamBackend represents the backend service for a TCP or UDP stream route.
type ApisixRouteStreamBackend struct {
	// ServiceName is the name of the Kubernetes Service.
	// Cross-namespace references are not supported—ensure the ApisixRoute
	// and the Service are in the same namespace.
	ServiceName string `json:"serviceName" yaml:"serviceName"`
	// ServicePort is the port of the Kubernetes Service.
	// This can be either the port name or port number.
	ServicePort intstr.IntOrString `json:"servicePort" yaml:"servicePort"`
	// ResolveGranularity determines how the backend service is resolved.
	// Valid values are `endpoints` and `service`. When set to `endpoints`,
	// individual pod IPs will be used; otherwise, the Service's ClusterIP or ExternalIP is used.
	// The default is `endpoints`.
	ResolveGranularity string `json:"resolveGranularity,omitempty" yaml:"resolveGranularity,omitempty"`
	// Subset specifies a named subset of the target Service.
	// The subset must be pre-defined in the corresponding ApisixUpstream resource.
	Subset string `json:"subset,omitempty" yaml:"subset,omitempty"`
}

// ApisixRouteHTTPMatchExpr represents a binary expression used to match requests based on Nginx variables.
type ApisixRouteHTTPMatchExpr struct {
	// Subject defines the left-hand side of the expression.
	// It can be any [built-in variable](/apisix/reference/built-in-variables) or string literal.
	Subject ApisixRouteHTTPMatchExprSubject `json:"subject" yaml:"subject"`

	// Op specifies the operator used in the expression.
	// Can be `Equal`, `NotEqual`, `GreaterThan`, `GreaterThanEqual`, `LessThan`, `LessThanEqual`, `RegexMatch`,
	// `RegexNotMatch`, `RegexMatchCaseInsensitive`, `RegexNotMatchCaseInsensitive`, `In`, or `NotIn`.
	Op string `json:"op" yaml:"op"`

	// Set provides a list of acceptable values for the expression.
	// This should be used when Op is `In` or `NotIn`.
	// +kubebuilder:validation:Optional
	Set []string `json:"set" yaml:"set"`

	// Value defines a single value to compare against the subject.
	// This should be used when Op is not `In` or `NotIn`.
	// Set and Value are mutually exclusive—only one should be set at a time.
	// +kubebuilder:validation:Optional
	Value *string `json:"value" yaml:"value"`
}

type ApisixRouteHTTPMatchExprs []ApisixRouteHTTPMatchExpr

func (exprs ApisixRouteHTTPMatchExprs) ToVars() (result adc.Vars, err error) {
	for _, expr := range exprs {
		if expr.Subject.Name == "" && expr.Subject.Scope != ScopePath {
			return result, errors.New("empty subject.name")
		}

		// process key
		var (
			subj string
			this adc.StringOrSlice
		)
		switch expr.Subject.Scope {
		case ScopeQuery:
			subj = "arg_" + expr.Subject.Name
		case ScopeHeader:
			subj = "http_" + strings.ReplaceAll(strings.ToLower(expr.Subject.Name), "-", "_")
		case ScopeCookie:
			subj = "cookie_" + expr.Subject.Name
		case ScopePath:
			subj = "uri"
		case ScopeVariable:
			subj = expr.Subject.Name
		default:
			return result, errors.New("invalid http match expr: subject.scope should be one of [query, header, cookie, path, variable]")
		}
		this.SliceVal = append(this.SliceVal, adc.StringOrSlice{StrVal: subj})

		// process operator
		var (
			op string
		)
		switch expr.Op {
		case OpEqual:
			op = "=="
		case OpGreaterThan:
			op = ">"
		case OpGreaterThanEqual:
			op = ">="
		case OpIn:
			op = "in"
		case OpLessThan:
			op = "<"
		case OpLessThanEqual:
			op = "<="
		case OpNotEqual:
			op = "~="
		case OpNotIn:
			op = "in"
		case OpRegexMatch:
			op = "~~"
		case OpRegexMatchCaseInsensitive:
			op = "~*"
		case OpRegexNotMatch:
			op = "~~"
		case OpRegexNotMatchCaseInsensitive:
			op = "~*"
		default:
			return result, errors.New("unknown operator")
		}
		if expr.Op == OpNotIn || expr.Op == OpRegexNotMatch || expr.Op == OpRegexNotMatchCaseInsensitive {
			this.SliceVal = append(this.SliceVal, adc.StringOrSlice{StrVal: "!"})
		}
		this.SliceVal = append(this.SliceVal, adc.StringOrSlice{StrVal: op})

		// process value
		switch expr.Op {
		case OpIn, OpNotIn:
			if expr.Set == nil {
				return result, errors.New("empty set value")
			}
			var value adc.StringOrSlice
			for _, item := range expr.Set {
				value.SliceVal = append(value.SliceVal, adc.StringOrSlice{StrVal: item})
			}
			this.SliceVal = append(this.SliceVal, value)
		default:
			if expr.Value == nil {
				return result, errors.New("empty value")
			}
			this.SliceVal = append(this.SliceVal, adc.StringOrSlice{StrVal: *expr.Value})
		}

		// append to result
		result = append(result, this.SliceVal)
	}

	return result, nil
}

// ApisixRoutePluginConfig is the configuration for
// any plugins.
type ApisixRoutePluginConfig map[string]apiextensionsv1.JSON

// ApisixRouteAuthenticationKeyAuth defines key authentication configuration in ApisixRouteAuthentication.
type ApisixRouteAuthenticationKeyAuth struct {
	// Header specifies the HTTP header name to look for the key authentication token.
	Header string `json:"header,omitempty" yaml:"header,omitempty"`
}

// ApisixRouteAuthenticationJwtAuth defines JWT authentication configuration in ApisixRouteAuthentication.
type ApisixRouteAuthenticationJwtAuth struct {
	// Header specifies the HTTP header name to look for the JWT token.
	Header string `json:"header,omitempty" yaml:"header,omitempty"`
	// Query specifies the URL query parameter name to look for the JWT token.
	Query string `json:"query,omitempty" yaml:"query,omitempty"`
	// Cookie specifies the cookie name to look for the JWT token.
	Cookie string `json:"cookie,omitempty" yaml:"cookie,omitempty"`
}

// ApisixRouteAuthenticationLDAPAuth defines LDAP authentication configuration in ApisixRouteAuthentication.
type ApisixRouteAuthenticationLDAPAuth struct {
	// BaseDN is the base distinguished name (DN) for LDAP searches.
	BaseDN string `json:"base_dn,omitempty" yaml:"base_dn,omitempty"`
	// LDAPURI is the URI of the LDAP server.
	LDAPURI string `json:"ldap_uri,omitempty" yaml:"ldap_uri,omitempty"`
	// UseTLS indicates whether to use TLS for the LDAP connection.
	UseTLS bool `json:"use_tls,omitempty" yaml:"use_tls,omitempty"`
	// UID is the user identifier attribute in LDAP.
	UID string `json:"uid,omitempty" yaml:"uid,omitempty"`
}

// ApisixRouteHTTPMatchExprSubject describes the subject of a route matching expression.
type ApisixRouteHTTPMatchExprSubject struct {
	// Scope specifies the subject scope and can be `Header`, `Query`, or `Path`.
	// When Scope is `Path`, Name will be ignored.
	Scope string `json:"scope" yaml:"scope"`
	// Name is the name of the header or query parameter.
	Name string `json:"name" yaml:"name"`
}

func init() {
	SchemeBuilder.Register(&ApisixRoute{}, &ApisixRouteList{})
}
