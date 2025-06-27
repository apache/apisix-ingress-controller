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

// ApisixPluginConfigSpec defines the desired state of ApisixPluginConfigSpec.
type ApisixPluginConfigSpec struct {
	// IngressClassName is the name of an IngressClass cluster resource.
	// The controller uses this field to decide whether the resource should be managed or not.
	// +kubebuilder:validation:Optional
	IngressClassName string `json:"ingressClassName,omitempty" yaml:"ingressClassName,omitempty"`
	// Plugins contain a list of ApisixRoutePlugin
	// +kubebuilder:validation:Required
	Plugins []ApisixRoutePlugin `json:"plugins" yaml:"plugins"`
}

// ApisixPluginConfigStatus defines the observed state of ApisixPluginConfig.
type ApisixPluginConfigStatus = ApisixStatus

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=apc

// ApisixPluginConfig is the Schema for the apisixpluginconfigs API.
type ApisixPluginConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApisixPluginConfigSpec   `json:"spec,omitempty"`
	Status ApisixPluginConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ApisixPluginConfigList contains a list of ApisixPluginConfig.
type ApisixPluginConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ApisixPluginConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ApisixPluginConfig{}, &ApisixPluginConfigList{})
}
