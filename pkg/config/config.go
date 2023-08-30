// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"text/template"
	"time"

	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/apache/apisix-ingress-controller/pkg/types"
)

const (
	// IngressAPISIXLeader is the default election id for the controller
	// leader election.
	IngressAPISIXLeader = "ingress-apisix-leader"
	// IngressClass is the default ingress class name, used for Ingress
	// object's IngressClassName field in Kubernetes clusters version v1.18.0
	// or higher, or the annotation "kubernetes.io/ingress.class" (deprecated).
	IngressClass = "apisix"

	// IngressNetworkingV1 represents ingress.networking/v1
	IngressNetworkingV1 = "networking/v1"
	// IngressNetworkingV1beta1 represents ingress.networking/v1beta1
	IngressNetworkingV1beta1 = "networking/v1beta1"
	// IngressExtensionsV1beta1 represents ingress.extensions/v1beta1
	// WARNING: ingress.extensions/v1beta1 is deprecated in v1.14+, and will be unavilable
	// in v1.22.
	IngressExtensionsV1beta1 = "extensions/v1beta1"
	// ApisixV2 represents apisix.apache.org/v2
	ApisixV2 = "apisix.apache.org/v2"
	// DefaultAPIVersion refers to the default resource version
	DefaultAPIVersion = ApisixV2

	_minimalResyncInterval = 30 * time.Second

	// ControllerName is the name of the controller used to identify
	// the controller of the GatewayClass.
	ControllerName = "apisix.apache.org/gateway-controller"

	// Process Ingress resources with ingressClass=apisix and all CRDs
	IngressClassApisixAndAll = "apisix-and-all"

	// Deployment mode
	DeploymentMode_AdminAPI = "admin-api"
	DeploymentMode_gRPC     = "grpc"
)

var (
	// Description information of API version, including default values and supported API version.
	APIVersionDescribe = fmt.Sprintf(`the default value of API version is "%s", support "%s".`, DefaultAPIVersion, ApisixV2)
)

// Config contains all config items which are necessary for
// apisix-ingress-controller's running.
type Config struct {
	CertFilePath                 string             `json:"cert_file" yaml:"cert_file"`
	KeyFilePath                  string             `json:"key_file" yaml:"key_file"`
	LogLevel                     string             `json:"log_level" yaml:"log_level"`
	LogOutput                    string             `json:"log_output" yaml:"log_output"`
	LogRotateOutputPath          string             `json:"log_rotate_output_path" yaml:"log_rotate_output_path"`
	LogRotationMaxSize           int                `json:"log_rotation_max_size" yaml:"log_rotation_max_size"`
	LogRotationMaxAge            int                `json:"log_rotation_max_age" yaml:"log_rotation_max_age"`
	LogRotationMaxBackups        int                `json:"log_rotation_max_backups" yaml:"log_rotation_max_backups"`
	HTTPListen                   string             `json:"http_listen" yaml:"http_listen"`
	HTTPSListen                  string             `json:"https_listen" yaml:"https_listen"`
	IngressPublishService        string             `json:"ingress_publish_service" yaml:"ingress_publish_service"`
	IngressStatusAddress         []string           `json:"ingress_status_address" yaml:"ingress_status_address"`
	EnableProfiling              bool               `json:"enable_profiling" yaml:"enable_profiling"`
	Kubernetes                   KubernetesConfig   `json:"kubernetes" yaml:"kubernetes"`
	APISIX                       APISIXConfig       `json:"apisix" yaml:"apisix"`
	ApisixResourceSyncInterval   types.TimeDuration `json:"apisix_resource_sync_interval" yaml:"apisix_resource_sync_interval"`
	ApisixResourceSyncComparison bool               `json:"apisix_resource_sync_comparison" yaml:"apisix_resource_sync_comparison"`
	PluginMetadataConfigMap      string             `json:"plugin_metadata_cm" yaml:"plugin_metadata_cm"`
	EtcdServer                   EtcdServerConfig   `json:"etcdserver" yaml:"etcdserver"`
}

type EtcdServerConfig struct {
	Enabled           bool   `json:"enabled" yaml:"enabled"`
	Prefix            string `json:"prefix" yaml:"prefix"`
	ListenAddress     string `json:"listen_address" yaml:"listen_address"`
	SSLKeyEncryptSalt string `json:"ssl_key_encrypt_salt" yaml:"ssl_key_encrypt_salt"`
}

// KubernetesConfig contains all Kubernetes related config items.
type KubernetesConfig struct {
	Kubeconfig           string             `json:"kubeconfig" yaml:"kubeconfig"`
	ResyncInterval       types.TimeDuration `json:"resync_interval" yaml:"resync_interval"`
	NamespaceSelector    []string           `json:"namespace_selector" yaml:"namespace_selector"`
	ElectionID           string             `json:"election_id" yaml:"election_id"`
	IngressClass         string             `json:"ingress_class" yaml:"ingress_class"`
	IngressVersion       string             `json:"ingress_version" yaml:"ingress_version"`
	WatchEndpointSlices  bool               `json:"watch_endpoint_slices" yaml:"watch_endpoint_slices"`
	APIVersion           string             `json:"api_version" yaml:"api_version"`
	EnableGatewayAPI     bool               `json:"enable_gateway_api" yaml:"enable_gateway_api"`
	DisableStatusUpdates bool               `json:"disable_status_updates" yaml:"disable_status_updates"`
	EnableAdmission      bool               `json:"enable_admission" yaml:"enable_admission"`
}

// APISIXConfig contains all APISIX related config items.
type APISIXConfig struct {
	// AdminAPIVersion is the APISIX admin API version
	AdminAPIVersion string `json:"admin_api_version" yaml:"admin_api_version"`
	// DefaultClusterName is the name of default cluster.
	DefaultClusterName string `json:"default_cluster_name" yaml:"default_cluster_name"`
	// DefaultClusterBaseURL is the base url configuration for the default cluster.
	DefaultClusterBaseURL string `json:"default_cluster_base_url" yaml:"default_cluster_base_url"`
	// DefaultClusterAdminKey is the admin key for the default cluster.
	// TODO: Obsolete the plain way to specify admin_key, which is insecure.
	DefaultClusterAdminKey string `json:"default_cluster_admin_key" yaml:"default_cluster_admin_key"`
}

// NewDefaultConfig creates a Config object which fills all config items with
// default value.
func NewDefaultConfig() *Config {
	return &Config{
		LogLevel:                     "warn",
		LogOutput:                    "stderr",
		LogRotateOutputPath:          "",
		LogRotationMaxSize:           100,
		LogRotationMaxAge:            0,
		LogRotationMaxBackups:        0,
		HTTPListen:                   ":8080",
		HTTPSListen:                  ":8443",
		IngressPublishService:        "",
		IngressStatusAddress:         []string{},
		CertFilePath:                 "/etc/webhook/certs/cert.pem",
		KeyFilePath:                  "/etc/webhook/certs/key.pem",
		EnableProfiling:              true,
		ApisixResourceSyncInterval:   types.TimeDuration{Duration: 1 * time.Hour},
		ApisixResourceSyncComparison: true,
		Kubernetes: KubernetesConfig{
			Kubeconfig:           "", // Use in-cluster configurations.
			ResyncInterval:       types.TimeDuration{Duration: 6 * time.Hour},
			ElectionID:           IngressAPISIXLeader,
			IngressClass:         IngressClassApisixAndAll,
			IngressVersion:       IngressNetworkingV1,
			APIVersion:           DefaultAPIVersion,
			WatchEndpointSlices:  false,
			EnableGatewayAPI:     false,
			DisableStatusUpdates: false,
			EnableAdmission:      false,
		},
		APISIX: APISIXConfig{
			AdminAPIVersion:    "v2",
			DefaultClusterName: "default",
		},
		EtcdServer: EtcdServerConfig{
			Enabled:           false,
			Prefix:            "/apisix",
			ListenAddress:     ":12379",
			SSLKeyEncryptSalt: "edd1c9f0985e76a2",
		},
	}
}

// NewConfigFromFile creates a Config object and fills all config items according
// to the configuration file. The file can be in JSON/YAML format, which will be
// distinguished according to the file suffix.
func NewConfigFromFile(filename string) (*Config, error) {
	cfg := NewDefaultConfig()
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	envVarMap := map[string]string{}
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		envVarMap[pair[0]] = pair[1]
	}

	tpl := template.New("text").Option("missingkey=error")
	tpl, err = tpl.Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("error parsing configuration template %v", err)
	}
	buf := bytes.NewBufferString("")
	err = tpl.Execute(buf, envVarMap)
	if err != nil {
		return nil, fmt.Errorf("error execute configuration template %v", err)
	}

	if strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml") {
		err = yaml.Unmarshal(buf.Bytes(), cfg)
	} else {
		err = json.Unmarshal(buf.Bytes(), cfg)
	}

	if err != nil {
		return nil, err
	}
	return cfg, nil
}

// Validate validates whether the Config is right.
func (cfg *Config) Validate() error {
	if cfg.Kubernetes.ResyncInterval.Duration < _minimalResyncInterval {
		return errors.New("controller resync interval too small")
	}
	if cfg.APISIX.DefaultClusterName == "" {
		cfg.APISIX.DefaultClusterName = "default"
	}
	if cfg.APISIX.DefaultClusterBaseURL == "" {
		return errors.New("apisix base url is required")
	}
	switch cfg.Kubernetes.IngressVersion {
	case IngressNetworkingV1, IngressNetworkingV1beta1:
		break
	default:
		return errors.New("unsupported ingress version")
	}
	ok, err := cfg.verifyNamespaceSelector()
	if !ok {
		return err
	}
	return nil
}

func (cfg *Config) verifyNamespaceSelector() (bool, error) {
	labels := cfg.Kubernetes.NamespaceSelector
	// default is [""]
	if len(labels) == 1 && labels[0] == "" {
		cfg.Kubernetes.NamespaceSelector = []string{}
	}

	for _, s := range cfg.Kubernetes.NamespaceSelector {
		parts := strings.Split(s, "=")
		if len(parts) != 2 {
			return false, fmt.Errorf("Illegal namespaceSelector: %s, should be key-value pairs divided by = ", s)
		} else {
			if err := cfg.validateLabelKey(parts[0]); err != nil {
				return false, err
			}
			if err := cfg.validateLabelValue(parts[1]); err != nil {
				return false, err
			}
		}
	}
	return true, nil
}

// validateLabelKey validate the key part of label
// ref: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set
func (cfg *Config) validateLabelKey(key string) error {
	errorMsg := validation.IsQualifiedName(key)
	msg := strings.Join(errorMsg, "\n")
	if msg == "" {
		return nil
	}
	return fmt.Errorf("Illegal namespaceSelector: %s, "+msg, key)
}

// validateLabelValue validate the value part of label
// ref: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set
func (cfg *Config) validateLabelValue(value string) error {
	errorMsg := validation.IsValidLabelValue(value)
	msg := strings.Join(errorMsg, "\n")
	if msg == "" {
		return nil
	}
	return fmt.Errorf("Illegal namespaceSelector: %s, "+msg, value)
}
