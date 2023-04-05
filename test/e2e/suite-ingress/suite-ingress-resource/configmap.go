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

package ingress

import (
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _configmapConfigYAML = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: plugin-metadata-config-map
data:
  config.yaml: |
    - cluster: default
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
          constant_tags:
          - "source:apisix"
`

var _configmapConfigYAMLUpdate = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: plugin-metadata-config-map
data:
  config.yaml: |
    - cluster: default
      plugins:
      - name: datadog
        metadata:
          host: "DogStatsD.server.domain"
          port: 8125
          namespace: "ingress" 
          constant_tags:
          - "source:ingress"
`

var _configmapConfigYAML2 = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: plugin-metadata-config-map
data:
  config.yaml: |
    - cluster: non-existent
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
`

var _configmapConfigYAML3 = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: plugin-metadata
data:
  config.yaml: |
    - cluster: non-existent
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
`

var _ = ginkgo.Describe("suite-ingress-resource: configmap Testing", func() {
	s := scaffold.NewDefaultScaffold()
	ginkgo.It("create configmap and configure config.yaml", func() {
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(_configmapConfigYAML))
		time.Sleep(6 * time.Second)

		pluginMetadatas, err := s.ListPluginMetadatas()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), pluginMetadatas)
		assert.Len(ginkgo.GinkgoT(), pluginMetadatas, 3)
		actual := map[string]map[string]any{}
		for _, pluginMetadata := range pluginMetadatas {
			actual[pluginMetadata.Name] = pluginMetadata.Metadata
		}

		assert.Equal(ginkgo.GinkgoT(), map[string]map[string]any{
			"datadog": {
				"host":      "DogStatsD.server.domain",
				"port":      float64(8125),
				"namespace": "apisix",
				"constant_tags": []interface{}{
					"source:apisix",
				},
			},
			"http-logger": {
				"log_format": map[string]any{
					"client_ip": "$remote_addr",
					"host":      "$host",
				},
			},
			"kafka-logger": {
				"log_format": map[string]any{
					"host": "$host",
				},
			},
		}, actual)
	})

	ginkgo.It("create configmap and configure invalid cluster name", func() {
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(_configmapConfigYAML2))
		time.Sleep(6 * time.Second)

		pluginMetadatas, err := s.ListPluginMetadatas()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), pluginMetadatas, 0)
	})

	ginkgo.It("create configmap and configure invalid configmap name", func() {
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(_configmapConfigYAML3))
		time.Sleep(6 * time.Second)

		pluginMetadatas, err := s.ListPluginMetadatas()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), pluginMetadatas, 0)
	})

	ginkgo.It("update configmap and configure config.yaml", func() {
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(_configmapConfigYAML))
		time.Sleep(6 * time.Second)

		pluginMetadatas, err := s.ListPluginMetadatas()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), pluginMetadatas)
		assert.Len(ginkgo.GinkgoT(), pluginMetadatas, 3)

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(_configmapConfigYAMLUpdate))
		time.Sleep(6 * time.Second)

		pluginMetadatas, err = s.ListPluginMetadatas()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), pluginMetadatas)
		assert.Len(ginkgo.GinkgoT(), pluginMetadatas, 1)

		actual := map[string]map[string]any{}
		for _, pluginMetadata := range pluginMetadatas {
			actual[pluginMetadata.Name] = pluginMetadata.Metadata
		}

		assert.Equal(ginkgo.GinkgoT(), map[string]map[string]any{
			"datadog": {
				"host":      "DogStatsD.server.domain",
				"port":      float64(8125),
				"namespace": "ingress",
				"constant_tags": []interface{}{
					"source:ingress",
				},
			},
		}, actual)
	})

	ginkgo.It("delete configmap and configure config.yaml", func() {
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(_configmapConfigYAML))
		time.Sleep(6 * time.Second)

		pluginMetadatas, err := s.ListPluginMetadatas()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), pluginMetadatas)
		assert.Len(ginkgo.GinkgoT(), pluginMetadatas, 3)

		assert.Nil(ginkgo.GinkgoT(), s.DeleteResourceFromString(_configmapConfigYAML))
		time.Sleep(6 * time.Second)

		pluginMetadatas, err = s.ListPluginMetadatas()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), pluginMetadatas, 0)
	})
})
