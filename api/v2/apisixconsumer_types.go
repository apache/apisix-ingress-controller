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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ApisixConsumerSpec defines the desired state of ApisixConsumer.
type ApisixConsumerSpec struct {
	// IngressClassName is the name of an IngressClass cluster resource.
	// The controller uses this field to decide whether the resource should be managed.
	IngressClassName string `json:"ingressClassName,omitempty" yaml:"ingressClassName,omitempty"`

	// AuthParameter defines the authentication credentials and configuration for this consumer.
	AuthParameter ApisixConsumerAuthParameter `json:"authParameter" yaml:"authParameter"`
}

// ApisixConsumerStatus defines the observed state of ApisixConsumer.
type ApisixConsumerStatus = ApisixStatus

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=ac

// ApisixConsumer defines configuration of a consumer and their authentication details.
type ApisixConsumer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// ApisixConsumerSpec defines the consumer authentication configuration.
	Spec   ApisixConsumerSpec   `json:"spec,omitempty"`
	Status ApisixConsumerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ApisixConsumerList contains a list of ApisixConsumer.
type ApisixConsumerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ApisixConsumer `json:"items"`
}

type ApisixConsumerAuthParameter struct {
	// BasicAuth configures the basic authentication details.
	BasicAuth *ApisixConsumerBasicAuth `json:"basicAuth,omitempty" yaml:"basicAuth"`
	// KeyAuth configures the key authentication details.
	KeyAuth *ApisixConsumerKeyAuth `json:"keyAuth,omitempty" yaml:"keyAuth"`
	// WolfRBAC configures the Wolf RBAC authentication details.
	WolfRBAC *ApisixConsumerWolfRBAC `json:"wolfRBAC,omitempty" yaml:"wolfRBAC"`
	// JwtAuth configures the JWT authentication details.
	JwtAuth *ApisixConsumerJwtAuth `json:"jwtAuth,omitempty" yaml:"jwtAuth"`
	// HMACAuth configures the HMAC authentication details.
	HMACAuth *ApisixConsumerHMACAuth `json:"hmacAuth,omitempty" yaml:"hmacAuth"`
	// LDAPAuth configures the LDAP authentication details.
	LDAPAuth *ApisixConsumerLDAPAuth `json:"ldapAuth,omitempty" yaml:"ldapAuth"`
}

// ApisixConsumerBasicAuth defines configuration for basic authentication.
type ApisixConsumerBasicAuth struct {
	// SecretRef references a Kubernetes Secret containing the basic authentication credentials.
	SecretRef *corev1.LocalObjectReference `json:"secretRef,omitempty" yaml:"secretRef,omitempty"`
	// Value specifies the basic authentication credentials.
	Value *ApisixConsumerBasicAuthValue `json:"value,omitempty" yaml:"value,omitempty"`
}

// ApisixConsumerBasicAuthValue defines the username and password configuration for basic authentication.
type ApisixConsumerBasicAuthValue struct {
	// Username is the basic authentication username.
	Username string `json:"username" yaml:"username"`
	// Password is the basic authentication password.
	Password string `json:"password" yaml:"password"`
}

// ApisixConsumerKeyAuth defines configuration for the key auth.
type ApisixConsumerKeyAuth struct {
	// SecretRef references a Kubernetes Secret containing the key authentication credentials.
	SecretRef *corev1.LocalObjectReference `json:"secretRef,omitempty" yaml:"secretRef,omitempty"`
	// Value specifies the key authentication credentials.
	Value *ApisixConsumerKeyAuthValue `json:"value,omitempty" yaml:"value,omitempty"`
}

// ApisixConsumerKeyAuthValue defines configuration for key authentication.
type ApisixConsumerKeyAuthValue struct {
	// Key is the credential used for key authentication.
	Key string `json:"key" yaml:"key"`
}

// ApisixConsumerWolfRBAC defines configuration for the Wolf RBAC authentication.
type ApisixConsumerWolfRBAC struct {
	// SecretRef references a Kubernetes Secret containing the Wolf RBAC token.
	SecretRef *corev1.LocalObjectReference `json:"secretRef,omitempty" yaml:"secretRef,omitempty"`
	// Value specifies the Wolf RBAC token.
	Value *ApisixConsumerWolfRBACValue `json:"value,omitempty" yaml:"value,omitempty"`
}

// ApisixConsumerWolfRBACValue defines configuration for Wolf RBAC authentication.
type ApisixConsumerWolfRBACValue struct {
	// Server is the URL of the Wolf RBAC server.
	Server string `json:"server,omitempty" yaml:"server,omitempty"`
	// Appid is the application identifier used when communicating with the Wolf RBAC server.
	Appid string `json:"appid,omitempty" yaml:"appid,omitempty"`
	// HeaderPrefix is the prefix added to request headers for RBAC enforcement.
	HeaderPrefix string `json:"header_prefix,omitempty" yaml:"header_prefix,omitempty"`
}

// ApisixConsumerJwtAuth defines configuration for JWT authentication.
type ApisixConsumerJwtAuth struct {
	// SecretRef references a Kubernetes Secret containing JWT authentication credentials.
	SecretRef *corev1.LocalObjectReference `json:"secretRef,omitempty" yaml:"secretRef,omitempty"`
	// Value specifies JWT authentication credentials.
	Value *ApisixConsumerJwtAuthValue `json:"value,omitempty" yaml:"value,omitempty"`
}

// ApisixConsumerJwtAuthValue defines configuration for JWT authentication.
type ApisixConsumerJwtAuthValue struct {
	// Key is the unique identifier for the JWT credential.
	Key string `json:"key" yaml:"key"`
	// Secret is the shared secret used to sign the JWT (for symmetric algorithms).
	Secret string `json:"secret,omitempty" yaml:"secret,omitempty"`
	// PublicKey is the public key used to verify JWT signatures (for asymmetric algorithms).
	PublicKey string `json:"public_key,omitempty" yaml:"public_key,omitempty"`
	// PrivateKey is the private key used to sign the JWT (for asymmetric algorithms).
	PrivateKey string `json:"private_key" yaml:"private_key,omitempty"`
	// Algorithm specifies the signing algorithm.
	// Can be `HS256`, `HS384`, `HS512`, `RS256`, `RS384`, `RS512`, `ES256`, `ES384`, `ES512`, `PS256`, `PS384`, `PS512`, or `EdDSA`.
	// Currently APISIX only supports `HS256`, `HS512`, `RS256`, and `ES256`. API7 Enterprise supports all algorithms.
	Algorithm string `json:"algorithm,omitempty" yaml:"algorithm,omitempty"`
	// Exp is the token expiration period in seconds.
	Exp int64 `json:"exp,omitempty" yaml:"exp,omitempty"`
	// Base64Secret indicates whether the secret is base64-encoded.
	Base64Secret bool `json:"base64_secret,omitempty" yaml:"base64_secret,omitempty"`
	// LifetimeGracePeriod is the allowed clock skew in seconds for token expiration.
	LifetimeGracePeriod int64 `json:"lifetime_grace_period,omitempty" yaml:"lifetime_grace_period,omitempty"`
}

// ApisixConsumerHMACAuth defines configuration for the HMAC authentication.
type ApisixConsumerHMACAuth struct {
	// SecretRef references a Kubernetes Secret containing the HMAC credentials.
	SecretRef *corev1.LocalObjectReference `json:"secretRef,omitempty" yaml:"secretRef,omitempty"`
	// Value specifies HMAC authentication credentials.
	Value *ApisixConsumerHMACAuthValue `json:"value,omitempty" yaml:"value,omitempty"`
}

// ApisixConsumerHMACAuthValue defines configuration for HMAC authentication.
type ApisixConsumerHMACAuthValue struct {
	// AccessKey is the identifier used to look up the HMAC secret.
	AccessKey string `json:"access_key" yaml:"access_key"`
	// SecretKey is the HMAC secret used to sign the request.
	SecretKey string `json:"secret_key" yaml:"secret_key"`
	// Algorithm specifies the hashing algorithm (e.g., "hmac-sha256").
	Algorithm string `json:"algorithm,omitempty" yaml:"algorithm,omitempty"`
	// ClockSkew is the allowed time difference (in seconds) between client and server clocks.
	ClockSkew int64 `json:"clock_skew,omitempty" yaml:"clock_skew,omitempty"`
	// SignedHeaders lists the headers that must be included in the signature.
	SignedHeaders []string `json:"signed_headers,omitempty" yaml:"signed_headers,omitempty"`
	// KeepHeaders determines whether the HMAC signature headers are preserved after verification.
	KeepHeaders bool `json:"keep_headers,omitempty" yaml:"keep_headers,omitempty"`
	// EncodeURIParams indicates whether URI parameters are encoded when calculating the signature.
	EncodeURIParams bool `json:"encode_uri_params,omitempty" yaml:"encode_uri_params,omitempty"`
	// ValidateRequestBody enables HMAC validation of the request body.
	ValidateRequestBody bool `json:"validate_request_body,omitempty" yaml:"validate_request_body,omitempty"`
	// MaxReqBody sets the maximum size (in bytes) of the request body that can be validated.
	MaxReqBody int64 `json:"max_req_body,omitempty" yaml:"max_req_body,omitempty"`
}

// ApisixConsumerLDAPAuth defines configuration for the LDAP authentication.
type ApisixConsumerLDAPAuth struct {
	// SecretRef references a Kubernetes Secret containing the LDAP credentials.
	SecretRef *corev1.LocalObjectReference `json:"secretRef" yaml:"secret"`
	// Value specifies LDAP authentication credentials.
	Value *ApisixConsumerLDAPAuthValue `json:"value,omitempty" yaml:"value,omitempty"`
}

// ApisixConsumerLDAPAuthValue defines configuration for LDAP authentication.
type ApisixConsumerLDAPAuthValue struct {
	// UserDN is the distinguished name (DN) of the LDAP user.
	UserDN string `json:"user_dn" yaml:"user_dn"`
}

func init() {
	SchemeBuilder.Register(&ApisixConsumer{}, &ApisixConsumerList{})
}
