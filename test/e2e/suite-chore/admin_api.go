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

package chore

import (
	"context"

	v1 "github.com/apache/apisix-ingress-controller/pkg/types/apisix/v1"
	"github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-chore: admin-api sdk", func() {
	s := scaffold.NewDefaultScaffold()
	ginkgo.It("plugin metadata test with admin-api sdk", func() {
		client, err := s.ClusterClient()
		assert.Nil(ginkgo.GinkgoT(), err)

		//update plugin metadata
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

		client.PluginMetadata().Update(context.Background(), datadog, false)

		pluginMetadatas, err := client.PluginMetadata().List(context.Background())
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), pluginMetadatas, 1)
		assert.Equal(ginkgo.GinkgoT(), datadog, pluginMetadatas[0])

		// update plugin metadata
		updated := &v1.PluginMetadata{
			Name: "datadog",
			Metadata: map[string]interface{}{
				"host": "127.0.0.1",
				"port": float64(8126),
				"constant_tags": []interface{}{
					"source:ingress",
					"service:custom",
				},
				"namespace": "ingress",
			},
		}
		client.PluginMetadata().Update(context.Background(), updated, false)
		pluginMetadatas, err = client.PluginMetadata().List(context.Background())
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), pluginMetadatas, 1)
		assert.Equal(ginkgo.GinkgoT(), updated, pluginMetadatas[0])

		// delete plguin metadata
		client.PluginMetadata().Delete(context.Background(), datadog)
		pluginMetadatas, err = client.PluginMetadata().List(context.Background())
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), pluginMetadatas, 0)
	})

})
