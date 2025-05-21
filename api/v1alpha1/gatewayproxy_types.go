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

package v1alpha1

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// GatewayProxySpec defines the desired state of GatewayProxy.
type GatewayProxySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// PublishService specifies the LoadBalancer-type Service whose external address the controller uses to
	// update the status of Ingress resources.
	PublishService string `json:"publishService,omitempty"`
	// StatusAddress specifies the external IP addresses that the controller uses to populate the status field
	// of GatewayProxy or Ingress resources for developers to access.
	StatusAddress []string `json:"statusAddress,omitempty"`
	// Provider configures the provider details.
	Provider *GatewayProxyProvider `json:"provider,omitempty"`
	// Plugins configure global plugins.
	Plugins []GatewayProxyPlugin `json:"plugins,omitempty"`
	// PluginMetadata configures common configurations shared by all plugin instances of the same name.
	PluginMetadata map[string]apiextensionsv1.JSON `json:"pluginMetadata,omitempty"`
}

// ProviderType defines the type of provider.
// +kubebuilder:validation:Enum=ControlPlane
type ProviderType string

const (
	// ProviderTypeControlPlane represents the control plane provider type.
	ProviderTypeControlPlane ProviderType = "ControlPlane"
)

// GatewayProxyProvider defines the provider configuration for GatewayProxy.
// +kubebuilder:validation:XValidation:rule="self.type == 'ControlPlane' ? has(self.controlPlane) : true",message="controlPlane must be specified when type is ControlPlane"
type GatewayProxyProvider struct {
	// Type specifies the type of provider. Can only be `ControlPlane`.
	// +kubebuilder:validation:Required
	Type ProviderType `json:"type"`

	// ControlPlane specifies the configuration for control plane provider.
	// +optional
	ControlPlane *ControlPlaneProvider `json:"controlPlane,omitempty"`
}

// AuthType defines the type of authentication.
// +kubebuilder:validation:Enum=AdminKey
type AuthType string

const (
	// AuthTypeAdminKey represents the admin key authentication type.
	AuthTypeAdminKey AuthType = "AdminKey"
)

// SecretKeySelector defines a reference to a specific key within a Secret.
type SecretKeySelector struct {
	// Name is the name of the secret.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Key is the key in the secret to retrieve the secret from.
	// +kubebuilder:validation:Required
	Key string `json:"key"`
}

// AdminKeyAuth defines the admin key authentication configuration.
type AdminKeyAuth struct {
	// Value sets the admin key value explicitly (not recommended for production).
	// +optional
	Value string `json:"value,omitempty"`

	// ValueFrom specifies the source of the admin key.
	// +optional
	ValueFrom *AdminKeyValueFrom `json:"valueFrom,omitempty"`
}

// AdminKeyValueFrom defines the source of the admin key.
type AdminKeyValueFrom struct {
	// SecretKeyRef references a key in a Secret.
	// +optional
	SecretKeyRef *SecretKeySelector `json:"secretKeyRef,omitempty"`
}

// ControlPlaneAuth defines the authentication configuration for control plane.
type ControlPlaneAuth struct {
	// Type specifies the type of authentication.
	// Can only be `AdminKey`.
	// +kubebuilder:validation:Required
	Type AuthType `json:"type"`

	// AdminKey specifies the admin key authentication configuration.
	// +optional
	AdminKey *AdminKeyAuth `json:"adminKey,omitempty"`
}

// ControlPlaneProvider defines the configuration for control plane provider.
type ControlPlaneProvider struct {
	// Endpoints specifies the list of control plane endpoints.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Endpoints []string `json:"endpoints"`

	// TlsVerify specifies whether to verify the TLS certificate of the control plane.
	// +optional
	TlsVerify *bool `json:"tlsVerify,omitempty"`

	// Auth specifies the authentication configurations.
	// +kubebuilder:validation:Required
	Auth ControlPlaneAuth `json:"auth"`
}

// +kubebuilder:object:root=true
// GatewayProxy is the Schema for the gatewayproxies API.
type GatewayProxy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// GatewayProxySpec defines the desired state and configuration of a GatewayProxy,
	// including networking settings, global plugins, and plugin metadata.
	Spec GatewayProxySpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
// GatewayProxyList contains a list of GatewayProxy.
type GatewayProxyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GatewayProxy `json:"items"`
}

// GatewayProxyPlugin contains plugin configurations.
type GatewayProxyPlugin struct {
	// Name is the name of the plugin.
	Name string `json:"name,omitempty"`
	// Enabled defines whether the plugin is enabled.
	Enabled bool `json:"enabled,omitempty"`
	// Config defines the plugin's configuration details.
	Config apiextensionsv1.JSON `json:"config,omitempty"`
}

func init() {
	SchemeBuilder.Register(&GatewayProxy{}, &GatewayProxyList{})
}
