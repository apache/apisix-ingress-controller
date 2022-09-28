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
	"context"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	v1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
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
`

var _ = ginkgo.Describe("suite-ingress-resource: configmap Testing", func() {
	s := scaffold.NewDefaultScaffold()
	ginkgo.It("create configmap", func() {
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(_configmapConfigYAML))
		time.Sleep(15 * time.Second)

		pluginMetadatas, err := s.ListPluginMetadatas()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.NotNil(ginkgo.GinkgoT(), pluginMetadatas)
	})

	ginkgo.It("plugin metadata test with admin-api sdk", func() {
		client, err := s.ClusterClient()
		assert.Nil(ginkgo.GinkgoT(), err)

		datadog := &v1.PluginMetadata{
			Name: "datadog",
			Metadata: map[string]interface{}{
				"host": "172.168.45.29",
				"port": float64(8126),
				"constant_tags": []interface{}{
					"source:apisix",
					"service:custom",
				},
				"namespace": "apisix",
			},
		}

		client.PluginMetadata().Create(context.Background(), datadog)

		pluginMetadatas, err := client.PluginMetadata().List(context.Background())
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), pluginMetadatas, 1)
		assert.Equal(ginkgo.GinkgoT(), datadog, pluginMetadatas[0])
	})

})
