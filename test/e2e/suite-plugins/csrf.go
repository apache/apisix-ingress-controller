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
	"fmt"
	"net/http"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-plugins: csrf plugin", func() {
	suites := func(scaffoldFunc func() *scaffold.Scaffold) {
		s := scaffoldFunc()
		ginkgo.It("prevent csrf", func() {
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
       - /anything
   backends:
   - serviceName: %s
     servicePort: %d
   plugins:
   - name: csrf
     enable: true
     config:
       key: "edd1c9f034335f136f87ad84b625c8f1"
`, backendSvc, backendPorts[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

			err := s.EnsureNumApisixUpstreamsCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
			err = s.EnsureNumApisixRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

			msg401 := s.NewAPISIXClient().
				POST("/anything").
				WithHeader("Host", "httpbin.org").
				Expect().
				Status(http.StatusUnauthorized).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg401, "no csrf token in headers")

			resp := s.NewAPISIXClient().
				GET("/anything").
				WithHeader("Host", "httpbin.org").
				Expect().
				Status(http.StatusOK)
			resp.Header("Set-Cookie").NotEmpty()

			cookie := resp.Cookie("apisix-csrf-token")
			token := cookie.Value().Raw()

			_ = s.NewAPISIXClient().
				POST("/anything").
				WithHeader("Host", "httpbin.org").
				WithHeader("apisix-csrf-token", token).
				WithCookie("apisix-csrf-token", token).
				Expect().
				Status(http.StatusOK)
		})

		ginkgo.It("disable plugin", func() {
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
       - /anything
   backends:
   - serviceName: %s
     servicePort: %d
   plugins:
   - name: csrf
     enable: false
     config:
       key: "edd1c9f034335f136f87ad84b625c8f1"
`, backendSvc, backendPorts[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

			err := s.EnsureNumApisixUpstreamsCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
			err = s.EnsureNumApisixRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

			_ = s.NewAPISIXClient().
				GET("/anything").
				WithHeader("Host", "httpbin.org").
				Expect().
				Status(http.StatusOK)

			_ = s.NewAPISIXClient().
				POST("/anything").
				WithHeader("Host", "httpbin.org").
				Expect().
				Status(http.StatusOK)
		})
	}

	ginkgo.Describe("suite-plugins: scaffold v2beta3", func() {
		suites(scaffold.NewDefaultScaffold)
	})
	ginkgo.Describe("suite-plugins: scaffold v2", func() {
		suites(scaffold.NewDefaultV2Scaffold)
	})
})
