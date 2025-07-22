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

package v1alpha1

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// Consumer defines configuration for a consumer.
type Consumer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// ConsumerSpec defines configuration for a consumer, including consumer name,
	// authentication credentials, and plugin settings.
	Spec   ConsumerSpec   `json:"spec,omitempty"`
	Status ConsumerStatus `json:"status,omitempty"`
}

type ConsumerStatus struct {
	Status `json:",inline"`
}
type ConsumerSpec struct {
	// GatewayRef specifies the gateway details.
	GatewayRef GatewayRef `json:"gatewayRef,omitempty"`
	// Credentials specifies the credential details of a consumer.
	Credentials []Credential `json:"credentials,omitempty"`
	// Plugins define the plugins associated with a consumer.
	Plugins []Plugin `json:"plugins,omitempty"`
}

type GatewayRef struct {
	// Name is the name of the gateway.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// Kind is the type of Kubernetes object. Default is `Gateway`.
	// +kubebuilder:default=Gateway
	Kind *string `json:"kind,omitempty"`
	// Group is the API group the resource belongs to. Default is `gateway.networking.k8s.io`.
	// +kubebuilder:default=gateway.networking.k8s.io
	Group *string `json:"group,omitempty"`
	// Namespace is namespace of the resource.
	Namespace *string `json:"namespace,omitempty"`
}

type Credential struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=jwt-auth;basic-auth;key-auth;hmac-auth;
	// Type specifies the type of authentication to configure credentials for.
	// Can be `jwt-auth`, `basic-auth`, `key-auth`, or `hmac-auth`.
	Type string `json:"type"`
	// Config specifies the credential details for authentication.
	Config apiextensionsv1.JSON `json:"config,omitempty"`
	// SecretRef references to the Secret that contains the credentials.
	SecretRef *SecretReference `json:"secretRef,omitempty"`
	// Name is the name of the credential.
	Name string `json:"name,omitempty"`
}

type SecretReference struct {
	// Name is the name of the secret.
	Name string `json:"name"`
	// Namespace is the namespace of the secret.
	Namespace *string `json:"namespace,omitempty"`
}

// +kubebuilder:object:root=true
type ConsumerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Consumer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Consumer{}, &ConsumerList{})
}
