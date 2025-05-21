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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
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
	// Can be one of `http`, `https`, `grpc`, or `grpcs`.
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
	// Default is `pass`.
	// Can be one of `pass`, `node` or `rewrite`.
	//
	// +kubebuilder:validation:Enum=pass;node;rewrite;
	// +kubebuilder:default=pass
	PassHost string `json:"passHost,omitempty" yaml:"passHost,omitempty"`

	// UpstreamHost specifies the host of the Upstream request. Used only if
	// passHost is set to `rewrite`.
	Host Hostname `json:"upstreamHost,omitempty" yaml:"upstreamHost,omitempty"`
}

// LoadBalancer describes the load balancing parameters.
// +kubebuilder:validation:XValidation:rule="!(has(self.key) && self.type != 'chash')"
type LoadBalancer struct {
	// Type specifies the load balancing algorithms.
	// Default is `roundrobin`.
	// Can be one of `roundrobin`, `chash`, `ewma`, or `least_conn`.
	// +kubebuilder:validation:Enum=roundrobin;chash;ewma;least_conn;
	// +kubebuilder:default=roundrobin
	// +kubebuilder:validation:Required
	Type string `json:"type" yaml:"type"`
	// HashOn specified the type of field used for hashing, required when Type is `chash`.
	// Default is `vars`.
	// Can be one of `vars`, `header`, `cookie`, `consumer`, or `vars_combinations`.
	// +kubebuilder:validation:Enum=vars;header;cookie;consumer;vars_combinations;
	// +kubebuilder:default=vars
	HashOn string `json:"hashOn,omitempty" yaml:"hashOn,omitempty"`
	// Key is used with HashOn, generally required when Type is `chash`.
	// When HashOn is `header` or `cookie`, specifies the name of the header or cookie.
	// When HashOn is `consumer`, key is not required, as the consumer name is used automatically.
	// When HashOn is `vars` or `vars_combinations`, key refers to one or a combination of
	// [built-in variables](/enterprise/reference/built-in-variables).
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

func init() {
	SchemeBuilder.Register(&BackendTrafficPolicy{}, &BackendTrafficPolicyList{})
}
