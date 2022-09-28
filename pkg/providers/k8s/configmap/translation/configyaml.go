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
	"fmt"

	"github.com/apache/apisix-ingress-controller/pkg/log"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

type ClusterMetadata []struct {
	Cluster string           `json:"cluster" yaml:"cluster"`
	Plugins []PluginMetadata `json:"plugins" yaml:"plugins"`
}

type PluginMetadata struct {
	PluginName string         `json:"name" yaml:"name"`
	Metadata   map[string]any `json:"metadata" yaml:"metadata"`
}

type configYAMLParser struct {
	key string
}

func newConfigYAMLParser(key string) *configYAMLParser {
	return &configYAMLParser{
		key: key,
	}
}

func (c *configYAMLParser) Key() string {
	return c.key
}

func (c *configYAMLParser) Parse(data string) (any, error) {
	log.Error("configmap data", zap.String("data", data))
	var clusters ClusterMetadata
	if err := yaml.Unmarshal([]byte(data), &clusters); err != nil {
		return nil, fmt.Errorf("unmarshal config.yaml faild, erros: %s", err.Error())
	}
	return clusters, nil
}
