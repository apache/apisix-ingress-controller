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

package v2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ApisixGlobalRuleSpec defines the desired state of ApisixGlobalRule.
type ApisixGlobalRuleSpec struct {
	// IngressClassName is the name of an IngressClass cluster resource.
	// The controller uses this field to decide whether the resource should be managed or not.
	IngressClassName string `json:"ingressClassName,omitempty" yaml:"ingressClassName,omitempty"`
	// Plugins contains a list of ApisixRoutePlugin
	// +kubebuilder:validation:Required
	Plugins []ApisixRoutePlugin `json:"plugins" yaml:"plugins"`
}

// ApisixGlobalRuleStatus defines the observed state of ApisixGlobalRule.
type ApisixGlobalRuleStatus = ApisixStatus

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ApisixGlobalRule is the Schema for the apisixglobalrules API.
type ApisixGlobalRule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApisixGlobalRuleSpec   `json:"spec,omitempty"`
	Status ApisixGlobalRuleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ApisixGlobalRuleList contains a list of ApisixGlobalRule.
type ApisixGlobalRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ApisixGlobalRule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ApisixGlobalRule{}, &ApisixGlobalRuleList{})
}
