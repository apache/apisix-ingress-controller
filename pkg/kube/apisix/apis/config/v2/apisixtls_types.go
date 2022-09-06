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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+genclient
//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
//+kubebuilder:resource:shortName=atls
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+kubebuilder:validation:Optional

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
