package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2alpha1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status

// ApisixPluginConfig is the Schema for the ApisixPluginConfig resource.
// An ApisixPluginConfig is used to support a group of plugin configs
type ApisixPluginConfig struct {
	metav1.TypeMeta   `json:",inline" yaml:",inline"`
	metav1.ObjectMeta `json:"metadata" yaml:"metadata"`

	// Spec defines the desired state of ApisixPluginConfigSpec.
	Spec   ApisixPluginConfigSpec `json:"spec" yaml:"spec"`
	Status v2alpha1.ApisixStatus  `json:"status,omitempty" yaml:"status,omitempty"`
}

// ApisixPluginConfigSpec defines the desired state of ApisixPluginConfigSpec.
type ApisixPluginConfigSpec struct {
	// +kubebuilder:validation:MinLength=1
	Desc string `json:"desc,omitempty" yaml:"desc,omitempty"`

	// Plugins contains a list of ApisixRouteHTTPPluginConfig
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Plugins []v2alpha1.ApisixRouteHTTPPluginConfig `json:"plugins" yaml:"plugins"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:generate=true

// ApisixPluginConfigList contains a list of ApisixPluginConfig.
type ApisixPluginConfigList struct {
	metav1.TypeMeta `json:",inline" yaml:",inline"`
	metav1.ListMeta `json:"metadata" yaml:"metadata"`
	Items           []ApisixPluginConfig `json:"items,omitempty" yaml:"items,omitempty"`
}
