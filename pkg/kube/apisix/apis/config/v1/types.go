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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// ApisixRoute is used to define the route rules and upstreams for Apache APISIX.
// The definition closes the Kubernetes Ingress resource.
type ApisixRoute struct {
	metav1.TypeMeta   `json:",inline" yaml:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Spec              *ApisixRouteSpec `json:"spec,omitempty" yaml:"spec,omitempty"`
}

// ApisixRouteSpec is the spec definition for ApisixRouteSpec.
type ApisixRouteSpec struct {
	Rules []Rule `json:"rules,omitempty"`
}

// Rule represents a single route rule in ApisixRoute.
type Rule struct {
	Host string `json:"host,omitempty"`
	Http Http   `json:"http,omitempty"`
}

// Http represents all route rules in HTTP scope.
type Http struct {
	Paths []Path `json:"paths,omitempty"`
}

// Path defines an URI based route rule.
type Path struct {
	Path    string   `json:"path,omitempty"`
	Backend Backend  `json:"backend,omitempty"`
	Plugins []Plugin `json:"plugins,omitempty"`
}

// Backend defines an upstream, it should be an existing Kubernetes Service.
// Note the Service should be in the same namespace with ApisixRoute resource,
// i.e. cross namespacing is not allowed.
type Backend struct {
	ServiceName string `json:"serviceName,omitempty"`
	ServicePort int    `json:"servicePort,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ApisixRouteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ApisixRoute `json:"items,omitempty"`
}

// +genclient
// +genclient:noStatus

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// ApisixUpstream is used to decorate Upstream in APISIX, such as load
// balacing type.
type ApisixUpstream struct {
	metav1.TypeMeta   `json:",inline" yaml:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Spec              *ApisixUpstreamSpec `json:"spec,omitempty" yaml:"spec,omitempty"`
}

// ApisixUpstreamSpec describes the specification of Upstream in APISIX.
type ApisixUpstreamSpec struct {
	Ports []Port `json:"ports,omitempty"`
}

// Port is the port-specific configurations.
type Port struct {
	Port         int           `json:"port,omitempty"`
	LoadBalancer *LoadBalancer `json:"loadbalancer,omitempty"`
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

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ApisixUpstreamList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ApisixUpstream `json:"items,omitempty"`
}

// +genclient
// +genclient:noStatus

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// ApisixService is used to define the Service resource in APISIX, it's
// useful to use Service to put all common configurations and let Route
// to reference it.
type ApisixService struct {
	metav1.TypeMeta   `json:",inline" yaml:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Spec              *ApisixServiceSpec `json:"spec,omitempty" yaml:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ApisixServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ApisixService `json:"items,omitempty"`
}

// ApisixServiceSpec describes the ApisixService specification.
type ApisixServiceSpec struct {
	Upstream string   `json:"upstream,omitempty"`
	Port     int      `json:"port,omitempty"`
	Plugins  []Plugin `json:"plugins,omitempty"`
}

type Plugin struct {
	Name      string    `json:"name,omitempty"`
	Enable    bool      `json:"enable,omitempty"`
	Config    Config    `json:"config,omitempty"`
	ConfigSet ConfigSet `json:"config_set,omitempty"`
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
// +genclient:noStatus

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// ApisixTLS defines SSL resource in APISIX.
type ApisixTLS struct {
	metav1.TypeMeta   `json:",inline" yaml:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Spec              *ApisixTLSSpec `json:"spec,omitempty" yaml:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ApisixTLSList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ApisixTLS `json:"items,omitempty"`
}

// ApisixTLSSpec is the specification of ApisixSSL.
type ApisixTLSSpec struct {
	Hosts  []string     `json:"hosts,omitempty"`
	Secret ApisixSecret `json:"secret,omitempty"`
}

// ApisixSecret describes the Kubernetes Secret name and namespace.
type ApisixSecret struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}
