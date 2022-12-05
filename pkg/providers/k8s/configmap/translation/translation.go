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
	corev1 "k8s.io/api/core/v1"

	v1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
)

type ConfigMap struct {
	ConfigYaml ConfigYAML
}

type ConfigYAML struct {
	// ClusterName => []*v1.PluginMetadata
	Data map[string][]*v1.PluginMetadata
}

// Only cluster default is supported
func TranslateConfigMap(cm *corev1.ConfigMap) (*ConfigMap, error) {
	data, err := parseDataOfConfigMap(cm)
	if err != nil {
		return nil, err
	}
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
		}
		configmap.ConfigYaml.Data[cluster.Cluster] = pluginMetadatas
	}
	return configmap, nil
}
