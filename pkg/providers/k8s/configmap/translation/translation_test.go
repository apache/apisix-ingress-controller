// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
package translation

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	v1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

var _configyaml = `- cluster: default
  plugins:
  - name : http-logger
    metadata:
      log_format:
        host: "$host"
        client_ip: "$remote_addr"
  - name: kafka-logger
    metadata:
      log_format:
        host: "$host"
  - name: datadog
    metadata:
      host: "DogStatsD.server.domain"
      port: 8125
      namespace: "apisix" 
- cluster: apisix-cluster
  plugins:
  - name: datadog
    metadata:
      host: "DogStatsD.server.domain"
- cluster: apisix1
`

func TestConfigYAML(t *testing.T) {
	cm := &corev1.ConfigMap{
		Data: map[string]string{
			"config.yaml": _configyaml,
		},
	}
	config, err := parseDataOfConfigMap(cm)
	assert.Nil(t, err)

	assert.NotNil(t, config)
	assert.Len(t, config.PluginMetadata, 3)
	assert.Equal(t, "default", config.PluginMetadata[0].Cluster)
	assert.Equal(t, "apisix-cluster", config.PluginMetadata[1].Cluster)
	assert.Equal(t, "apisix1", config.PluginMetadata[2].Cluster)

	assert.Equal(t, struct {
		Cluster string           "json:\"cluster\" yaml:\"cluster\""
		Plugins []PluginMetadata "json:\"plugins\" yaml:\"plugins\""
	}{
		Cluster: "apisix-cluster",
		Plugins: []PluginMetadata{
			{
				PluginName: "datadog",
				Metadata: map[string]interface{}{
					"host": "DogStatsD.server.domain",
				},
			},
		},
	}, config.PluginMetadata[1])
	for _, clu := range config.PluginMetadata {
		for _, metadata := range clu.Plugins {
			body, err := json.Marshal(metadata.Metadata)
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
			fmt.Println(metadata.PluginName, string(body))
		}
	}
	assert.Nil(t, cm)
}

func TestConfigMap(t *testing.T) {
	cm := &corev1.ConfigMap{
		Data: map[string]string{
			"config.yaml": _configyaml,
		},
	}
	data, err := parseDataOfConfigMap(cm)
	assert.Nil(t, err)
	configmap := &ConfigMap{
		ConfigYaml: ConfigYAML{
			Data: map[string][]*v1.PluginMetadata{},
		},
	}

	for _, cluster := range data.PluginMetadata {
		var pluginMetadatas []*v1.PluginMetadata
		for _, plugin := range cluster.Plugins {
			pluginMetadatas = append(pluginMetadatas, &v1.PluginMetadata{
				Name:     plugin.PluginName,
				Metadata: plugin.Metadata,
			})
			fmt.Println(plugin.Metadata)
		}
		configmap.ConfigYaml.Data[cluster.Cluster] = pluginMetadatas
	}
	assert.Nil(t, cm)
}
