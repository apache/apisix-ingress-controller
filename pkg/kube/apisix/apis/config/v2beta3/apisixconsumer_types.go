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
package v2beta3

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+genclient
//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
//+kubebuilder:object:root=true
//+kubebuilder:resource:shortName=ac,categories=apisix-ingress-controller
//+kubebuilder:subresource:status
//+kubebuilder:validation:Optional

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
