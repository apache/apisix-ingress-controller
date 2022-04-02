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
package features

import (
	"fmt"
	"net/http"
	"time"

	"github.com/apache/apisix-ingress-controller/pkg/id"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("ApisixClusterConfig", func() {
	opts := &scaffold.Options{
		Name:                  "default",
		Kubeconfig:            scaffold.GetKubeconfig(),
		APISIXConfigPath:      "testdata/apisix-gw-config.yaml",
		IngressAPISIXReplicas: 1,
		HTTPBinServicePort:    80,
		APISIXRouteVersion:    "apisix.apache.org/v2beta3",
	}
	s := scaffold.NewScaffold(opts)
	ginkgo.It("enable prometheus", func() {
		adminSvc, adminPort := s.ApisixAdminServiceAndPort()
		acc := `
apiVersion: apisix.apache.org/v2beta3
kind: ApisixClusterConfig
metadata:
  name: default
spec:
  monitoring:
    prometheus:
      enable: true
`
		err := s.CreateResourceFromString(acc)
		assert.Nil(ginkgo.GinkgoT(), err, "creating ApisixClusterConfig")

		defer func() {
			err := s.RemoveResourceByString(acc)
			assert.Nil(ginkgo.GinkgoT(), err)
		}()

		// Wait until the ApisixClusterConfig create event was delivered.
		time.Sleep(3 * time.Second)

		arr := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2beta3
kind: ApisixRoute
metadata:
  name: httpbin-route
spec:
  http:
  - name: public-api
    match:
      hosts:
      - httpbin.org
      paths:
      - /apisix/prometheus/metrics
    backends:
    - serviceName: %s
      servicePort: %d
    plugins:
    - name: public-api
      enable: true
`, adminSvc, adminPort)

		err = s.CreateResourceFromString(arr)
		assert.Nil(ginkgo.GinkgoT(), err, "creating ApisixRouteConfig")

		defer func() {
			err := s.RemoveResourceByString(arr)
			assert.Nil(ginkgo.GinkgoT(), err)
		}()

		time.Sleep(3 * time.Second)

		grs, err := s.ListApisixGlobalRules()
		assert.Nil(ginkgo.GinkgoT(), err, "listing global_rules")
		assert.Len(ginkgo.GinkgoT(), grs, 1)
		assert.Equal(ginkgo.GinkgoT(), grs[0].ID, id.GenID("default"))
		assert.Len(ginkgo.GinkgoT(), grs[0].Plugins, 1)
		_, ok := grs[0].Plugins["prometheus"]
		assert.Equal(ginkgo.GinkgoT(), ok, true)

		resp := s.NewAPISIXClient().GET("/apisix/prometheus/metrics").WithHeader("Host", "httpbin.org").Expect()
		resp.Status(http.StatusOK)
		resp.Body().Contains("# HELP apisix_etcd_modify_indexes Etcd modify index for APISIX keys")
		resp.Body().Contains("# HELP apisix_etcd_reachable Config server etcd reachable from APISIX, 0 is unreachable")
		resp.Body().Contains("# HELP apisix_node_info Info of APISIX node")
	})
})
