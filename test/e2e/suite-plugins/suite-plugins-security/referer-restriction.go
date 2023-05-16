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

var _ = ginkgo.Describe("suite-plugins-security: referer-restriction plugin", func() {
	suites := func(scaffoldFunc func() *scaffold.Scaffold) {
		s := scaffoldFunc()
		ginkgo.It("configure a access list", func() {
			backendSvc, backendPorts := s.DefaultHTTPBackend()
			ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
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
   plugins:
   - name: referer-restriction
     enable: true
     config:
       whitelist:
         - test.com
         - "*.foo.com"
`, backendSvc, backendPorts[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

			err := s.EnsureNumApisixUpstreamsCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
			err = s.EnsureNumApisixRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

			// "Referer" match passed
			resp := s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("Referer", "http://test.com").
				Expect()
			resp.Status(http.StatusOK)
			resp.Body().Contains("origin")

			// "Referer" match failed
			resp = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("Referer", "http://www.test.com").
				Expect()
			resp.Status(http.StatusForbidden)
			resp.Body().Contains("Your referer host is not allowed")

			// "Referer" match passed
			resp = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("Referer", "http://www.foo.com").
				Expect()
			resp.Status(http.StatusOK)
			resp.Body().Contains("origin")

			// "Referer" is missing
			resp = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				Expect()
			resp.Status(http.StatusForbidden)
			resp.Body().Contains("Your referer host is not allowed")
		})

		ginkgo.It("configure a deny access list", func() {
			backendSvc, backendPorts := s.DefaultHTTPBackend()
			ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
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
   plugins:
   - name: referer-restriction
     enable: true
     config:
       blacklist:
         - test.com
         - "*.foo.com"
`, backendSvc, backendPorts[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

			err := s.EnsureNumApisixUpstreamsCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking the number of upstreams")
			err = s.EnsureNumApisixRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking the number of routes")

			// "Referer" match failed
			resp := s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("Referer", "http://test.com").
				Expect()
			resp.Status(http.StatusForbidden)
			resp.Body().Contains("Your referer host is not allowed")

			// "Referer" match passed
			resp = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("Referer", "http://www.test.com").
				Expect()
			resp.Status(http.StatusOK)
			resp.Body().Contains("origin")

			// "Referer" match failed
			resp = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("Referer", "http://www.foo.com").
				Expect()
			resp.Status(http.StatusForbidden)
			resp.Body().Contains("Your referer host is not allowed")

			// "Referer" is missing
			resp = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				Expect()
			resp.Status(http.StatusForbidden)
			resp.Body().Contains("Your referer host is not allowed")
		})

		ginkgo.It("customize return message", func() {
			backendSvc, backendPorts := s.DefaultHTTPBackend()
			ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
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
   plugins:
   - name: referer-restriction
     enable: true
     config:
       whitelist:
         - test.com
         - "*.foo.com"
       message: "You can customize the message any way you like"
`, backendSvc, backendPorts[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

			err := s.EnsureNumApisixUpstreamsCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking the number of upstreams")
			err = s.EnsureNumApisixRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking the number of routes")

			// "Referer" match failed
			resp := s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("Referer", "http://www.test.com").
				Expect()
			resp.Status(http.StatusForbidden)
			resp.Body().Contains("You can customize the message any way you like")

			// "Referer" is missing
			resp = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				Expect()
			resp.Status(http.StatusForbidden)
			resp.Body().Contains("You can customize the message any way you like")
		})

		ginkgo.It("configure bypass_missing field to true", func() {
			backendSvc, backendPorts := s.DefaultHTTPBackend()
			ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
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
   plugins:
   - name: referer-restriction
     enable: true
     config:
       whitelist:
         - test.com
       bypass_missing: true
`, backendSvc, backendPorts[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

			err := s.EnsureNumApisixUpstreamsCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking the number of upstreams")
			err = s.EnsureNumApisixRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking the number of routes")

			// "Referer" is missing
			resp := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect()
			resp.Status(http.StatusOK)
			resp.Body().Contains("origin")

			// "Referer" format is incorrect
			resp = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("Referer", "test.com").
				Expect()
			resp.Status(http.StatusOK)
			resp.Body().Contains("origin")
		})

		ginkgo.It("disable plugin", func() {
			backendSvc, backendPorts := s.DefaultHTTPBackend()
			ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
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
   plugins:
   - name: referer-restriction
     enable: false
     config:
       whitelist:
         - test.com
`, backendSvc, backendPorts[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

			err := s.EnsureNumApisixUpstreamsCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
			err = s.EnsureNumApisixRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

			// "Referer" is not in the whitelist.
			resp := s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("Referer", "http://foo.com").
				Expect()
			resp.Status(http.StatusOK)
			resp.Body().Contains("origin")

			// "Referer" is missing
			resp = s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect()
			resp.Status(http.StatusOK)
			resp.Body().Contains("origin")
		})
	}

	ginkgo.Describe("suite-plugins-security: scaffold v2", func() {
		suites(scaffold.NewDefaultV2Scaffold)
	})
})
