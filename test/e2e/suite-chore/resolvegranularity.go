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
	"fmt"
	"net/http"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-chore: ApisixRoute resolvegranularity Testing", func() {
	s := scaffold.NewDefaultScaffold()
	ginkgo.It("service and upstream [1:m]", func() {
		assert.Nil(ginkgo.GinkgoT(), s.ScaleHTTPBIN(2))
		time.Sleep(5 * time.Second)

		backendSvc, backendSvcPort := s.DefaultHTTPBackend()
		route1 := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: httpbin-route1
spec:
  http:
  - name: route1
    match:
      hosts:
      - httpbin.org
      paths:
      - /ip
    backends:
    - serviceName: %s
      servicePort: %d
      resolveGranularity: service
`, backendSvc, backendSvcPort[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(route1))
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "checking number of routes")
		ups, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), ups, 1)
		assert.Len(ginkgo.GinkgoT(), ups[0].Nodes, 1)
		_ = s.NewAPISIXClient().GET("/ip").
			WithHeader("Host", "httpbin.org").
			Expect().
			Status(http.StatusOK)

		route2 := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: httpbin-route2
spec:
  http:
  - name: route2
    match:
      hosts:
      - httpbin.com
      paths:
      - /get
    backends:
    - serviceName: %s
      servicePort: %d
      resolveGranularity: endpoint
`, backendSvc, backendSvcPort[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(route2))
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(2), "checking number of routes")
		ups, err = s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), ups, 2)
		if len(ups[0].Nodes) == 1 {
			assert.Len(ginkgo.GinkgoT(), ups[1].Nodes, 2)
		} else {
			assert.Len(ginkgo.GinkgoT(), ups[0].Nodes, 2)
			assert.Len(ginkgo.GinkgoT(), ups[1].Nodes, 1)
		}
		_ = s.NewAPISIXClient().GET("/get").
			WithHeader("Host", "httpbin.com").
			Expect().
			Status(http.StatusOK)

		s.RestartIngressControllerDeploy()
		time.Sleep(15 * time.Second)

		ups, err = s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), ups, 2)
		if len(ups[0].Nodes) == 1 {
			assert.Len(ginkgo.GinkgoT(), ups[1].Nodes, 2)
		} else {
			assert.Len(ginkgo.GinkgoT(), ups[0].Nodes, 2)
			assert.Len(ginkgo.GinkgoT(), ups[1].Nodes, 1)
		}
		_ = s.NewAPISIXClient().GET("/get").
			WithHeader("Host", "httpbin.com").
			Expect().
			Status(http.StatusOK)

		s.RestartIngressControllerDeploy()
		time.Sleep(15 * time.Second)

		ups, err = s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), ups, 2)
		if len(ups[0].Nodes) == 1 {
			assert.Len(ginkgo.GinkgoT(), ups[1].Nodes, 2)
		} else {
			assert.Len(ginkgo.GinkgoT(), ups[0].Nodes, 2)
			assert.Len(ginkgo.GinkgoT(), ups[1].Nodes, 1)
		}
		_ = s.NewAPISIXClient().GET("/get").
			WithHeader("Host", "httpbin.com").
			Expect().
			Status(http.StatusOK)

		s.RestartIngressControllerDeploy()
		time.Sleep(15 * time.Second)

		ups, err = s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), ups, 2)
		if len(ups[0].Nodes) == 1 {
			assert.Len(ginkgo.GinkgoT(), ups[1].Nodes, 2)
		} else {
			assert.Len(ginkgo.GinkgoT(), ups[0].Nodes, 2)
			assert.Len(ginkgo.GinkgoT(), ups[1].Nodes, 1)
		}
		_ = s.NewAPISIXClient().GET("/get").
			WithHeader("Host", "httpbin.com").
			Expect().
			Status(http.StatusOK)
	})
})
