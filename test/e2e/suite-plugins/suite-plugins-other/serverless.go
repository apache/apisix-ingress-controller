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
package plugins

import (
	"fmt"
	"net/http"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-plugins-other: serverless plugin", func() {
	suites := func(scaffoldFunc func() *scaffold.Scaffold) {
		s := scaffoldFunc()
		ginkgo.It("enable serverless plugin", func() {
			backendSvc, backendPorts := s.DefaultHTTPBackend()
			ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2beta3
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
   plugins:
   - name: serverless-pre-function
     enable: true
     config:
       phase: rewrite
       functions: ["return function() local core = require(\"apisix.core\"); core.response.exit(403); end"]         
`, backendSvc, backendPorts[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

			err := s.EnsureNumApisixRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

			resp := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect()
			resp.Status(http.StatusForbidden)

		})
		ginkgo.It("disable serverless plugin", func() {
			backendSvc, backendPorts := s.DefaultHTTPBackend()
			ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2beta3
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
`, backendSvc, backendPorts[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

			err := s.EnsureNumApisixRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

			resp := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect()
			resp.Status(http.StatusOK)

		})
	}

	ginkgo.Describe("suite-plugins-other: scaffold v2beta3", func() {
		suites(scaffold.NewDefaultV2beta3Scaffold)
	})
	ginkgo.Describe("suite-plugins-other: scaffold v2", func() {
		suites(scaffold.NewDefaultV2Scaffold)
	})
})
