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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ApisixTlsSpec defines configurations for TLS and mutual TLS.
type ApisixTlsSpec struct {
	// IngressClassName specifies which IngressClass this resource is associated with.
	// The APISIX controller only processes this resource if the class matches its own.
	IngressClassName string `json:"ingressClassName,omitempty" yaml:"ingressClassName,omitempty"`

	// Hosts lists the SNI (Server Name Indication) hostnames that this TLS configuration applies to.
	// Must contain at least one host.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Hosts []HostType `json:"hosts" yaml:"hosts,omitempty"`

	// Secret refers to the Kubernetes TLS secret containing the certificate and private key.
	// This secret must exist in the specified namespace and contain valid TLS data.
	// +kubebuilder:validation:Required
	Secret ApisixSecret `json:"secret" yaml:"secret"`

	// Client defines mutual TLS (mTLS) settings, such as the CA certificate and verification depth.
	Client *ApisixMutualTlsClientConfig `json:"client,omitempty" yaml:"client,omitempty"`
}

// ApisixTlsStatus defines the observed state of ApisixTls.
type ApisixTlsStatus = ApisixStatus

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=atls,path=apisixtlses
// +kubebuilder:printcolumn:name="SNIs",type=string,JSONPath=`.spec.hosts`
// +kubebuilder:printcolumn:name="Secret Name",type=string,JSONPath=`.spec.secret.name`
// +kubebuilder:printcolumn:name="Secret Namespace",type=string,JSONPath=`.spec.secret.namespace`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:printcolumn:name="Client CA Secret Name",type=string,JSONPath=`.spec.client.ca.name`
// +kubebuilder:printcolumn:name="Client CA Secret Namespace",type=string,JSONPath=`.spec.client.ca.namespace`

// ApisixTls defines configuration for TLS and mutual TLS (mTLS).
type ApisixTls struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// ApisixTlsSpec defines the TLS configuration.
	Spec   ApisixTlsSpec   `json:"spec,omitempty"`
	Status ApisixTlsStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ApisixTlsList contains a list of ApisixTls.
type ApisixTlsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ApisixTls `json:"items"`
}

// +kubebuilder:validation:Pattern="^\\*?[0-9a-zA-Z-.]+$"
type HostType string

// ApisixSecret describes a reference to a Kubernetes Secret, including its name and namespace.
// This is used to locate secrets such as certificates or credentials for plugins or TLS configuration.
type ApisixSecret struct {
	// Name is the name of the Kubernetes Secret.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Name string `json:"name" yaml:"name"`
	// Namespace is the namespace where the Kubernetes Secret is located.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	Namespace string `json:"namespace" yaml:"namespace"`
}

// ApisixMutualTlsClientConfig describes the mutual TLS CA and verification settings.
type ApisixMutualTlsClientConfig struct {
	// CASecret references the secret containing the CA certificate for client certificate validation.
	CASecret ApisixSecret `json:"caSecret,omitempty" yaml:"caSecret,omitempty"`
	// Depth specifies the maximum verification depth for the client certificate chain.
	Depth int `json:"depth,omitempty" yaml:"depth,omitempty"`
	// SkipMTLSUriRegex contains RegEx patterns for URIs to skip mutual TLS verification.
	SkipMTLSUriRegex []string `json:"skip_mtls_uri_regex,omitempty" yaml:"skip_mtls_uri_regex,omitempty"`
}

func init() {
	SchemeBuilder.Register(&ApisixTls{}, &ApisixTlsList{})
}
