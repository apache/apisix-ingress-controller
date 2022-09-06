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

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

//+genclient
//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
//+kubebuilder:resource:shortName=apc,categories=apisix-ingress-controller
//+kubebuilder:subresource:status
//+kubebuilder:validation:Optional

// ApisixPluginConfig is the Schema for the ApisixPluginConfig resource.
// An ApisixPluginConfig is used to support a group of plugin configs
type ApisixPluginConfig struct {
	metav1.TypeMeta   `json:",inline" yaml:",inline"`
	metav1.ObjectMeta `json:"metadata" yaml:"metadata"`

	// Spec defines the desired state of ApisixPluginConfigSpec.
	Spec   ApisixPluginConfigSpec `json:"spec" yaml:"spec"`
	Status ApisixStatus           `json:"status,omitempty" yaml:"status,omitempty"`
}

// ApisixPluginConfigSpec defines the desired state of ApisixPluginConfigSpec.
type ApisixPluginConfigSpec struct {
	// Plugins contains a list of ApisixRoutePlugin
	// +required
	Plugins []ApisixRoutePlugin `json:"plugins" yaml:"plugins"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:generate=true

// ApisixPluginConfigList contains a list of ApisixPluginConfig.
type ApisixPluginConfigList struct {
	metav1.TypeMeta `json:",inline" yaml:",inline"`
	metav1.ListMeta `json:"metadata" yaml:"metadata"`
	Items           []ApisixPluginConfig `json:"items,omitempty" yaml:"items,omitempty"`
}
