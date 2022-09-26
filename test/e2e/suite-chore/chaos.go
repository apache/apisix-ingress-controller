// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-chore: Chaos Testing", func() {
	s := scaffold.NewDefaultScaffold()
	ginkgo.Context("simulate apisix deployment restart", func() {
		ginkgo.Specify("ingress controller can synchronize rules normally after apisix recovery", func() {
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(0), "checking number of upstreams")
			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
			route1 := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2beta2
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
`, backendSvc, backendSvcPort[0])
			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(route1))
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "checking number of routes")
			s.RestartAPISIXDeploy()
			route2 := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2beta2
kind: ApisixRoute
metadata:
  name: httpbin-route2
spec:
  http:
  - name: route2
    match:
      hosts:
      - httpbin.org
      paths:
      - /get
    backends:
    - serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0])
			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(route2))
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(2), "checking number of routes")
			s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusOK)
			s.NewAPISIXClient().GET("/get").WithHeader("Host", "httpbin.org").Expect().Status(http.StatusOK)
		})
	})

})
