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
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

// HTTPRoutePolicySpec defines the desired state of HTTPRoutePolicy.
type HTTPRoutePolicySpec struct {
	// TargetRef identifies an API object (i.e. HTTPRoute, Ingress) to apply HTTPRoutePolicy to.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	TargetRefs []gatewayv1alpha2.LocalPolicyTargetReferenceWithSectionName `json:"targetRefs"`
	// Priority sets the priority for route. A higher value sets a higher priority in route matching.
	Priority *int64 `json:"priority,omitempty" yaml:"priority,omitempty"`
	// Vars sets the request matching conditions.
	Vars []apiextensionsv1.JSON `json:"vars,omitempty" yaml:"vars,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// HTTPRoutePolicy is the Schema for the httproutepolicies API.
type HTTPRoutePolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// HTTPRoutePolicySpec defines the desired state and configuration of a HTTPRoutePolicy,
	// including route priority and request matching conditions.
	Spec   HTTPRoutePolicySpec `json:"spec,omitempty"`
	Status PolicyStatus        `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// HTTPRoutePolicyList contains a list of HTTPRoutePolicy.
type HTTPRoutePolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HTTPRoutePolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HTTPRoutePolicy{}, &HTTPRoutePolicyList{})
}
