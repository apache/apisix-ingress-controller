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
	"github.com/imdario/mergo"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"

	"github.com/apache/apisix-ingress-controller/pkg/log"
)

type ConfigMapData struct {
	PluginMetadata ClusterMetadata
}

var (
	_keyConfigYAML = "config.yaml"

	_cmDataParser = map[string]DataParser{
		"PluginMetadata": newConfigYAMLParser(_keyConfigYAML),
	}
)

type DataParser interface {
	Key() string
	Parse(string) (any, error)
}

func parseDataOfConfigMap(cm *corev1.ConfigMap) (*ConfigMapData, error) {
	config := &ConfigMapData{}
	data := make(map[string]interface{})
	for name, parser := range _cmDataParser {
		if value, ok := cm.Data[parser.Key()]; ok {
			result, err := parser.Parse(value)
			if err != nil {
				log.Warnw("failed to parse configmap",
					zap.Error(err),
				)
				return nil, err
			}
			if result != nil {
				data[name] = result
			}
		}
	}
	err := mergo.MapWithOverwrite(config, data)
	if err != nil {
		log.Errorw("unexpected error merging extracted configmap", zap.Error(err))
		return nil, err
	}
	return config, nil
}
