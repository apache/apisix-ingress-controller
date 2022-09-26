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
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/pkg/types"
)

func TestNewConfigFromFile(t *testing.T) {
	cfg := &Config{
		LogLevel:                   "warn",
		LogOutput:                  "stdout",
		LogRotateOutputPath:        "",
		LogRotationMaxSize:         100,
		LogRotationMaxAge:          0,
		LogRotationMaxBackups:      0,
		HTTPListen:                 ":9090",
		HTTPSListen:                ":9443",
		IngressPublishService:      "",
		IngressStatusAddress:       []string{},
		CertFilePath:               "/etc/webhook/certs/cert.pem",
		KeyFilePath:                "/etc/webhook/certs/key.pem",
		EnableProfiling:            true,
		ApisixResourceSyncInterval: types.TimeDuration{Duration: 200 * time.Second},
		Kubernetes: KubernetesConfig{
			ResyncInterval: types.TimeDuration{Duration: time.Hour},
			Kubeconfig:     "/path/to/foo/baz",
			AppNamespaces:  []string{""},
			ElectionID:     "my-election-id",
			IngressClass:   IngressClass,
			IngressVersion: IngressNetworkingV1,
			APIVersion:     DefaultAPIVersion,
		},
		APISIX: APISIXConfig{
			DefaultClusterName:     "apisix",
			DefaultClusterBaseURL:  "http://127.0.0.1:8080/apisix",
			DefaultClusterAdminKey: "123456",
		},
	}

	jsonData, err := json.Marshal(cfg)
	assert.Nil(t, err, "failed to marshal config to json: %s", err)

	tmpJSON, err := ioutil.TempFile("/tmp", "config-*.json")
	assert.Nil(t, err, "failed to create temporary json configuration file: ", err)
	defer os.Remove(tmpJSON.Name())

	_, err = tmpJSON.Write(jsonData)
	assert.Nil(t, err, "failed to write json data: ", err)
	tmpJSON.Close()

	newCfg, err := NewConfigFromFile(tmpJSON.Name())
	assert.Nil(t, err, "failed to new config from file: ", err)
	assert.Nil(t, newCfg.Validate(), "failed to validate config")

	assert.Equal(t, cfg, newCfg, "bad configuration")

	// We constructs yaml data manually instead of using yaml.Marshal since
	// types.TimeDuration doesn't have a `yaml:",inline"` tag, if we add it,
	// error ",inline needs a struct value field" will be reported.
	// I don't know why.
	yamlData := `
log_level: warn
log_output: stdout
http_listen: :9090
https_listen: :9443
ingress_publish_service: ""
ingress_status_address: []
enable_profiling: true
apisix-resource-sync-interval: 200s
kubernetes:
  kubeconfig: /path/to/foo/baz
  resync_interval: 1h0m0s
  election_id: my-election-id
  ingress_class: apisix
  ingress_version: networking/v1
  api_version: apisix.apache.org/v2
apisix:
  default_cluster_base_url: http://127.0.0.1:8080/apisix
  default_cluster_admin_key: "123456"
  default_cluster_name: "apisix"
`
	tmpYAML, err := ioutil.TempFile("/tmp", "config-*.yaml")
	assert.Nil(t, err, "failed to create temporary yaml configuration file: ", err)
	defer os.Remove(tmpYAML.Name())

	_, err = tmpYAML.Write([]byte(yamlData))
	assert.Nil(t, err, "failed to write yaml data: ", err)
	tmpYAML.Close()

	newCfg, err = NewConfigFromFile(tmpYAML.Name())
	assert.Nil(t, err, "failed to new config from file: ", err)
	assert.Nil(t, newCfg.Validate(), "failed to validate config")

	assert.Equal(t, cfg, newCfg, "bad configuration")
}

func TestConfigWithEnvVar(t *testing.T) {
	cfg := &Config{
		LogLevel:                   "warn",
		LogOutput:                  "stdout",
		LogRotateOutputPath:        "",
		LogRotationMaxSize:         100,
		LogRotationMaxAge:          0,
		LogRotationMaxBackups:      0,
		HTTPListen:                 ":9090",
		HTTPSListen:                ":9443",
		IngressPublishService:      "",
		IngressStatusAddress:       []string{},
		CertFilePath:               "/etc/webhook/certs/cert.pem",
		KeyFilePath:                "/etc/webhook/certs/key.pem",
		EnableProfiling:            true,
		ApisixResourceSyncInterval: types.TimeDuration{Duration: 200 * time.Second},
		Kubernetes: KubernetesConfig{
			ResyncInterval: types.TimeDuration{Duration: time.Hour},
			Kubeconfig:     "",
			AppNamespaces:  []string{""},
			ElectionID:     "my-election-id",
			IngressClass:   IngressClass,
			IngressVersion: IngressNetworkingV1,
			APIVersion:     DefaultAPIVersion,
		},
		APISIX: APISIXConfig{
			DefaultClusterName:     "apisix",
			DefaultClusterBaseURL:  "http://127.0.0.1:8080/apisix",
			DefaultClusterAdminKey: "123456",
		},
	}

	defaultClusterBaseURLEnvName := "DEFAULT_CLUSTER_BASE_URL"
	defaultClusterAdminKeyEnvName := "DEFAULT_CLUSTER_ADMIN_KEY"
	defaultClusterNameEnvName := "DEFAULT_CLUSTER_NAME"
	kubeconfigEnvName := "KUBECONFIG"

	err := os.Setenv(defaultClusterBaseURLEnvName, "http://127.0.0.1:8080/apisix")
	assert.Nil(t, err, "failed to set env variable: ", err)
	_ = os.Setenv(defaultClusterAdminKeyEnvName, "123456")
	_ = os.Setenv(kubeconfigEnvName, "")
	_ = os.Setenv(defaultClusterNameEnvName, "apisix")

	jsonData := `
{
    "log_level": "warn",
    "log_output": "stdout",
    "http_listen": ":9090",
    "https_listen": ":9443",
	"ingress_publish_service": "",
	"ingress_status_address": [],
    "enable_profiling": true,
	"apisix-resource-sync-interval": "200s",
    "kubernetes": {
        "kubeconfig": "{{.KUBECONFIG}}",
        "resync_interval": "1h0m0s",
        "election_id": "my-election-id",
        "ingress_class": "apisix",
        "ingress_version": "networking/v1"
    },
    "apisix": {
        "default_cluster_base_url": "{{.DEFAULT_CLUSTER_BASE_URL}}",
        "default_cluster_admin_key": "{{.DEFAULT_CLUSTER_ADMIN_KEY}}",
        "default_cluster_name": "{{.DEFAULT_CLUSTER_NAME}}"
    }
}
`
	tmpJSON, err := ioutil.TempFile("/tmp", "config-*.json")
	assert.Nil(t, err, "failed to create temporary json configuration file: ", err)
	defer os.Remove(tmpJSON.Name())

	_, err = tmpJSON.Write([]byte(jsonData))
	assert.Nil(t, err, "failed to write json data: ", err)
	tmpJSON.Close()

	newCfg, err := NewConfigFromFile(tmpJSON.Name())
	assert.Nil(t, err, "failed to new config from file: ", err)
	assert.Nil(t, newCfg.Validate(), "failed to validate config")

	assert.Equal(t, cfg, newCfg, "bad configuration")

	yamlData := `
log_level: warn
log_output: stdout
http_listen: :9090
https_listen: :9443
ingress_publish_service: ""
ingress_status_address: []
enable_profiling: true
apisix-resource-sync-interval: 200s
kubernetes:
  resync_interval: 1h0m0s
  kubeconfig: "{{.KUBECONFIG}}"
  election_id: my-election-id
  ingress_class: apisix
  ingress_version: networking/v1
apisix:
  default_cluster_base_url: {{.DEFAULT_CLUSTER_BASE_URL}}
  default_cluster_admin_key: "{{.DEFAULT_CLUSTER_ADMIN_KEY}}"
  default_cluster_name: "{{.DEFAULT_CLUSTER_NAME}}"
`
	tmpYAML, err := ioutil.TempFile("/tmp", "config-*.yaml")
	assert.Nil(t, err, "failed to create temporary yaml configuration file: ", err)
	defer os.Remove(tmpYAML.Name())

	_, err = tmpYAML.Write([]byte(yamlData))
	assert.Nil(t, err, "failed to write yaml data: ", err)
	tmpYAML.Close()

	newCfg, err = NewConfigFromFile(tmpYAML.Name())
	assert.Nil(t, err, "failed to new config from file: ", err)
	assert.Nil(t, newCfg.Validate(), "failed to validate config")

	assert.Equal(t, cfg, newCfg, "bad configuration")

	_ = os.Unsetenv(defaultClusterBaseURLEnvName)

	_, err = NewConfigFromFile(tmpJSON.Name())
	assert.NotNil(t, err, "should failed because env variable missing")

	_, err = NewConfigFromFile(tmpYAML.Name())
	assert.NotNil(t, err, "should failed because env variable missing")
}

func TestConfigDefaultValue(t *testing.T) {
	yamlData := `
apisix:
  default_cluster_base_url: http://127.0.0.1:8080/apisix
`
	tmpYAML, err := ioutil.TempFile("/tmp", "config-*.yaml")
	assert.Nil(t, err, "failed to create temporary yaml configuration file: ", err)
	defer os.Remove(tmpYAML.Name())

	_, err = tmpYAML.Write([]byte(yamlData))
	assert.Nil(t, err, "failed to write yaml data: ", err)
	tmpYAML.Close()

	newCfg, err := NewConfigFromFile(tmpYAML.Name())
	assert.Nil(t, err, "failed to new config from file: ", err)
	assert.Nil(t, newCfg.Validate(), "failed to validate config")

	defaultCfg := NewDefaultConfig()
	defaultCfg.APISIX.DefaultClusterBaseURL = "http://127.0.0.1:8080/apisix"
	defaultCfg.APISIX.DefaultClusterName = "default"

	assert.Equal(t, defaultCfg, newCfg, "bad configuration")
}

func TestConfigInvalidation(t *testing.T) {
	yamlData := ``
	tmpYAML, err := ioutil.TempFile("/tmp", "config-*.yaml")
	assert.Nil(t, err, "failed to create temporary yaml configuration file: ", err)
	defer os.Remove(tmpYAML.Name())

	_, err = tmpYAML.Write([]byte(yamlData))
	assert.Nil(t, err, "failed to write yaml data: ", err)
	tmpYAML.Close()

	newCfg, err := NewConfigFromFile(tmpYAML.Name())
	assert.Nil(t, err, "failed to new config from file: ", err)
	err = newCfg.Validate()
	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), "apisix base url is required", "bad error: ", err)

	yamlData = `
kubernetes:
  resync_interval: 15s
apisix:
  default_cluster_base_url: http://127.0.0.1:1234/apisix
`
	tmpYAML, err = ioutil.TempFile("/tmp", "config-*.yaml")
	assert.Nil(t, err, "failed to create temporary yaml configuration file: ", err)
	defer os.Remove(tmpYAML.Name())

	_, err = tmpYAML.Write([]byte(yamlData))
	assert.Nil(t, err, "failed to write yaml data: ", err)
	tmpYAML.Close()

	newCfg, err = NewConfigFromFile(tmpYAML.Name())
	assert.Nil(t, err, "failed to new config from file: ", err)
	err = newCfg.Validate()
	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), "controller resync interval too small", "bad error: ", err)
}
