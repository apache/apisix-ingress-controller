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
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+genclient
//+genclient:nonNamespaced
//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster,shortName=acc,categories=apisix-ingress-controller
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+kubebuilder:validation:Optional

// ApisixClusterConfig is the Schema for the ApisixClusterConfig resource.
// An ApisixClusterConfig is used to identify an APISIX cluster, it's a
// ClusterScoped resource so the name is unique.
// It also contains some cluster-level configurations like monitoring.
type ApisixClusterConfig struct {
	metav1.TypeMeta   `json:",inline" yaml:",inline"`
	metav1.ObjectMeta `json:"metadata" yaml:"metadata"`

	// Spec defines the desired state of ApisixClusterConfigSpec.
	Spec   ApisixClusterConfigSpec `json:"spec" yaml:"spec"`
	Status ApisixStatus            `json:"status,omitempty" yaml:"status,omitempty"`
}

// ApisixClusterConfigSpec defines the desired state of ApisixClusterConfigSpec.
type ApisixClusterConfigSpec struct {
	// Monitoring categories all monitoring related features.
	// +optional
	Monitoring *ApisixClusterMonitoringConfig `json:"monitoring" yaml:"monitoring"`
	// Admin contains the Admin API information about APISIX cluster.
	// +optional
	Admin *ApisixClusterAdminConfig `json:"admin" yaml:"admin"`
}

// ApisixClusterMonitoringConfig categories all monitoring related features.
type ApisixClusterMonitoringConfig struct {
	// Prometheus is the config for using Prometheus in APISIX Cluster.
	// +optional
	Prometheus ApisixClusterPrometheusConfig `json:"prometheus" yaml:"prometheus"`
	// Skywalking is the config for using Skywalking in APISIX Cluster.
	// +optional
	Skywalking ApisixClusterSkywalkingConfig `json:"skywalking" yaml:"skywalking"`
}

// ApisixClusterPrometheusConfig is the config for using Prometheus in APISIX Cluster.
type ApisixClusterPrometheusConfig struct {
	// Enable means whether enable Prometheus or not.
	Enable bool `json:"enable" yaml:"enable"`
}

// ApisixClusterSkywalkingConfig is the config for using Skywalking in APISIX Cluster.
type ApisixClusterSkywalkingConfig struct {
	// Enable means whether enable Skywalking or not.
	Enable bool `json:"enable" yaml:"enable"`
	// SampleRatio means the ratio to collect
	SampleRatio resource.Quantity `json:"sampleRatio" yaml:"sampleRatio"`
}

// ApisixClusterAdminConfig is the admin config for the corresponding APISIX Cluster.
type ApisixClusterAdminConfig struct {
	// BaseURL is the base URL for the APISIX Admin API.
	// It looks like "http://apisix-admin.default.svc.cluster.local:9080/apisix/admin"
	BaseURL string `json:"baseURL" yaml:"baseURL"`
	// AdminKey is used to verify the admin API user.
	AdminKey string `json:"adminKey" yaml:"adminKey"`
	// ClientTimeout is request timeout for the APISIX Admin API client
	ClientTimeout time.Duration `json:"clientTimeout" yaml:"clientTimeout"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ApisixClusterConfigList contains a list of ApisixClusterConfig.
type ApisixClusterConfigList struct {
	metav1.TypeMeta `json:",inline" yaml:",inline"`
	metav1.ListMeta `json:"metadata" yaml:"metadata"`

	Items []ApisixClusterConfig `json:"items" yaml:"items"`
}
