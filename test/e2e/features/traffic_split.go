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

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
)

var _ = ginkgo.Describe("traffic split", func() {
	opts := &scaffold.Options{
		Name:                    "default",
		Kubeconfig:              scaffold.GetKubeconfig(),
		APISIXConfigPath:        "testdata/apisix-gw-config.yaml",
		APISIXDefaultConfigPath: "testdata/apisix-gw-config-default.yaml",
		IngressAPISIXReplicas:   1,
		HTTPBinServicePort:      80,
		APISIXRouteVersion:      "apisix.apache.org/v2alpha1",
	}
	s := scaffold.NewScaffold(opts)
	ginkgo.It("sanity", func() {
		backendSvc, backendPorts := s.DefaultHTTPBackend()
		ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2alpha1
kind: ApisixRoute
metadata:
 name: httpbin-route
spec:
 http:
 - name: rule1
   match:
     hosts:
     - httpbin.org
     paths:
       - /ip
   backends:
   - serviceName: %s
     servicePort: %d
     weight: 10
   - serviceName: %s
     servicePort: %d
     weight: 5
`, backendSvc, backendPorts[0], backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar))

		err := s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		err = s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		ups, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), ups, 1)

		// TODO Send requests to APISIX. Currently, traffic-split plugin in APISIX has
		// a bug when using upstream_id.
		routes, err := s.ListApisixRoutes()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), routes, 1)
		ts, ok := routes[0].Plugins["traffic-split"]
		assert.Equal(ginkgo.GinkgoT(), ok, true)

		rawRules, ok := ts.(map[string]interface{})["rules"]
		assert.Equal(ginkgo.GinkgoT(), ok, true)
		assert.Len(ginkgo.GinkgoT(), rawRules.([]interface{}), 1)
		rawWeightedUpstreams, ok := rawRules.([]interface{})[0].(map[string]interface{})["weighted_upstreams"]
		assert.Equal(ginkgo.GinkgoT(), ok, true)
		weightedUpstreams := rawWeightedUpstreams.([]interface{})
		assert.Len(ginkgo.GinkgoT(), weightedUpstreams, 2)
		assert.Equal(ginkgo.GinkgoT(), ups[0].ID, weightedUpstreams[0].(map[string]interface{})["upstream_id"].(string))
		assert.Equal(ginkgo.GinkgoT(), float64(5), weightedUpstreams[0].(map[string]interface{})["weight"].(float64))

		assert.Equal(ginkgo.GinkgoT(), float64(10), weightedUpstreams[1].(map[string]interface{})["weight"].(float64))
		_, ok = weightedUpstreams[1].(map[string]interface{})["upstream_id"]
		assert.Equal(ginkgo.GinkgoT(), ok, false)
	})
})
