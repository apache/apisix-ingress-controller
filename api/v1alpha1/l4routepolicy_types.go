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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

// L4RoutePolicySpec defines the desired state of L4RoutePolicy.
type L4RoutePolicySpec struct {
	// TargetRefs identifies the L4 route resources (TCPRoute, UDPRoute, or TLSRoute)
	// to which this policy applies. Only same-namespace targets are supported.
	//
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	// +kubebuilder:validation:XValidation:rule="self.all(r, r.kind == 'TCPRoute' || r.kind == 'UDPRoute' || r.kind == 'TLSRoute')",message="targetRefs kind must be TCPRoute, UDPRoute, or TLSRoute"
	// +kubebuilder:validation:XValidation:rule="self.all(r, r.group == 'gateway.networking.k8s.io')",message="targetRefs group must be gateway.networking.k8s.io"
	TargetRefs []gatewayv1alpha2.LocalPolicyTargetReferenceWithSectionName `json:"targetRefs"`

	// Plugins is the list of APISIX stream plugins to attach to the targeted L4 routes.
	// Plugin names should be valid APISIX stream plugin names (e.g., limit-conn, ip-restriction).
	//
	// +optional
	Plugins []Plugin `json:"plugins,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// L4RoutePolicy defines plugin configuration for Gateway API L4 routes (TCPRoute, UDPRoute, TLSRoute).
// It follows the Gateway API Policy Attachment pattern and attaches APISIX stream plugins
// to the targeted L4 route resources.
type L4RoutePolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of L4RoutePolicy.
	Spec   L4RoutePolicySpec `json:"spec,omitempty"`
	Status PolicyStatus      `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// L4RoutePolicyList contains a list of L4RoutePolicy.
type L4RoutePolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []L4RoutePolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&L4RoutePolicy{}, &L4RoutePolicyList{})
}
