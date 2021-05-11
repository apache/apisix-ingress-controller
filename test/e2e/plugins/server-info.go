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
package plugins

import (
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
)

var _ = ginkgo.FDescribe("server-info plugin", func() {
	opts := &scaffold.Options{
		Name:                    "default",
		Kubeconfig:              scaffold.GetKubeconfig(),
		APISIXConfigPath:        "testdata/apisix-gw-config.yaml",
		APISIXDefaultConfigPath: "testdata/apisix-gw-config-default.yaml",
		IngressAPISIXReplicas:   1,
		HTTPBinServicePort:      80,
		APISIXRouteVersion:      "apisix.apache.org/v2alpha1",
	}
	sEnabled := scaffold.NewScaffold(opts)

	ginkgo.It("enable server-info plugin", func() {
		serverInfoKey := [...]string{"etcd_version", "up_time", "last_report_time", "id", "hostname", "version", "boot_time"}
		serverInfo, err := sEnabled.GetServerInfo()
		assert.Nil(ginkgo.GinkgoT(), err)
		if assert.NotNil(ginkgo.GinkgoT(), serverInfo) {
			for _, key := range serverInfoKey {
				_, ok := serverInfo[key]
				assert.True(ginkgo.GinkgoT(), ok)
			}
		}

	})

	optsDisabled := &scaffold.Options{
		Name:                    "default",
		Kubeconfig:              scaffold.GetKubeconfig(),
		APISIXConfigPath:        "testdata/apisix-gw-config-srv-info-disabled.yaml",
		APISIXDefaultConfigPath: "testdata/apisix-gw-config-default.yaml",
		IngressAPISIXReplicas:   1,
		HTTPBinServicePort:      80,
		APISIXRouteVersion:      "apisix.apache.org/v2alpha1",
	}
	sDisabled := scaffold.NewScaffold(optsDisabled)

	ginkgo.It("disable plugin", func() {
		serverInfo, err := sDisabled.GetServerInfo()
		assert.Equal(ginkgo.GinkgoT(), len(serverInfo), 0)
		assert.Nil(ginkgo.GinkgoT(), err)
	})
})
