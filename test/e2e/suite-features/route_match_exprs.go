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
package features

import (
	"fmt"
	"net/http"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-features: route match exprs", func() {
	suites := func(scaffoldFunc func() *scaffold.Scaffold) {
		s := scaffoldFunc()
		ginkgo.It("operator is equal", func() {
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
     exprs:
     - subject:
         scope: Header
         name: X-Foo
       op: Equal
       value: bar
   backends:
   - serviceName: %s
     servicePort: %d
`, backendSvc, backendPorts[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

			time.Sleep(6 * time.Second)
			err := s.EnsureNumApisixRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
			err = s.EnsureNumApisixUpstreamsCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")

			_ = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("X-Foo", "bar").
				Expect().
				Status(http.StatusOK)

			msg := s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("X-Foo", "baz").
				Expect().
				Status(http.StatusNotFound).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg, "404 Route Not Found")
		})

		ginkgo.It("operator is not_equal", func() {
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
     exprs:
     - subject:
         scope: Header
         name: X-Foo
       op: NotEqual
       value: bar
   backends:
   - serviceName: %s
     servicePort: %d
`, backendSvc, backendPorts[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

			err := s.EnsureNumApisixRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
			err = s.EnsureNumApisixUpstreamsCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")

			_ = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				Expect().
				Status(http.StatusOK)

			msg := s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("X-Foo", "bar").
				Expect().
				Status(http.StatusNotFound).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg, "404 Route Not Found")
		})

		ginkgo.It("operator is greater_than", func() {
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
     exprs:
     - subject:
         scope: Query
         name: id
       op: GreaterThan
       value: "13"
   backends:
   - serviceName: %s
     servicePort: %d
`, backendSvc, backendPorts[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

			time.Sleep(6 * time.Second)
			err := s.EnsureNumApisixRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
			err = s.EnsureNumApisixUpstreamsCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")

			_ = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithQuery("id", 100).
				Expect().
				Status(http.StatusOK)

			msg := s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithQuery("id", 3).
				Expect().
				Status(http.StatusNotFound).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg, "404 Route Not Found")

			msg = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				Expect().
				Status(http.StatusNotFound).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg, "404 Route Not Found")
		})

		ginkgo.It("operator is GreaterThanEqual", func() {
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
     exprs:
     - subject:
         scope: Query
         name: id
       op: GreaterThanEqual
       value: "13"
   backends:
   - serviceName: %s
     servicePort: %d
`, backendSvc, backendPorts[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

			time.Sleep(6 * time.Second)
			err := s.EnsureNumApisixRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
			err = s.EnsureNumApisixUpstreamsCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")

			_ = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithQuery("id", 100).
				Expect().
				Status(http.StatusOK)

			_ = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithQuery("id", 13).
				Expect().
				Status(http.StatusOK)

			msg := s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithQuery("id", 10).
				Expect().
				Status(http.StatusNotFound).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg, "404 Route Not Found")

			msg = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				Expect().
				Status(http.StatusNotFound).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg, "404 Route Not Found")
		})

		ginkgo.It("operator is less_than", func() {
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
     exprs:
     - subject:
         scope: Query
         name: ID
       op: LessThan
       value: "13"
   backends:
   - serviceName: %s
     servicePort: %d
`, backendSvc, backendPorts[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

			time.Sleep(6 * time.Second)
			err := s.EnsureNumApisixRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
			err = s.EnsureNumApisixUpstreamsCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")

			_ = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithQuery("ID", 12).
				Expect().
				Status(http.StatusOK)

			msg := s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithQuery("ID", 13).
				Expect().
				Status(http.StatusNotFound).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg, "404 Route Not Found")

			msg = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				Expect().
				Status(http.StatusNotFound).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg, "404 Route Not Found")
		})

		ginkgo.It("operator is LessThanEqual", func() {
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
     exprs:
     - subject:
         scope: Query
         name: ID
       op: LessThanEqual
       value: "13"
   backends:
   - serviceName: %s
     servicePort: %d
`, backendSvc, backendPorts[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

			time.Sleep(6 * time.Second)
			err := s.EnsureNumApisixRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
			err = s.EnsureNumApisixUpstreamsCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")

			_ = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithQuery("ID", 12).
				Expect().
				Status(http.StatusOK)

			_ = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithQuery("ID", 13).
				Expect().
				Status(http.StatusOK)

			msg := s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithQuery("ID", 14).
				Expect().
				Status(http.StatusNotFound).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg, "404 Route Not Found")

			msg = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				Expect().
				Status(http.StatusNotFound).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg, "404 Route Not Found")
		})

		ginkgo.It("operator is in", func() {
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
     exprs:
     - subject:
         scope: Header
         name: Content-Type
       op: In
       set: ["text/plain", "text/html", "image/jpeg"]
   backends:
   - serviceName: %s
     servicePort: %d
`, backendSvc, backendPorts[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

			err := s.EnsureNumApisixRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
			err = s.EnsureNumApisixUpstreamsCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")

			_ = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("Content-Type", "text/html").
				Expect().
				Status(http.StatusOK)

			msg := s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("Content-Type", "image/png").
				Expect().
				Status(http.StatusNotFound).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg, "404 Route Not Found")

			msg = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				Expect().
				Status(http.StatusNotFound).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg, "404 Route Not Found")
		})

		ginkgo.It("operator is not_in", func() {
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
     exprs:
     - subject:
         scope: Header
         name: Content-Type
       op: NotIn
       set: ["text/plain", "text/html", "image/jpeg"]
   backends:
   - serviceName: %s
     servicePort: %d
`, backendSvc, backendPorts[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

			err := s.EnsureNumApisixRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
			err = s.EnsureNumApisixUpstreamsCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")

			_ = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("Content-Type", "text/png").
				Expect().
				Status(http.StatusOK)

			msg := s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("Content-Type", "image/jpeg").
				Expect().
				Status(http.StatusNotFound).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg, "404 Route Not Found")

			_ = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				Expect().
				Status(http.StatusOK).
				Body().
				Raw()
		})

		ginkgo.It("operator is regex match", func() {
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
     exprs:
     - subject:
         scope: Header
         name: x-Real-URI
       op: RegexMatch
       value: "^/ip/0\\d{2}/.*$"
   backends:
   - serviceName: %s
     servicePort: %d
`, backendSvc, backendPorts[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))
			time.Sleep(6 * time.Second)

			err := s.EnsureNumApisixRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
			err = s.EnsureNumApisixUpstreamsCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")

			_ = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("X-Real-Uri", "/ip/098/v4").
				Expect().
				Status(http.StatusOK)

			msg := s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("X-Real-Uri", "/ip/0983/v4").
				Expect().
				Status(http.StatusNotFound).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg, "404 Route Not Found")

			msg = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				Expect().
				Status(http.StatusNotFound).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg, "404 Route Not Found")
		})

		ginkgo.It("operator is regex not match", func() {
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
     exprs:
     - subject:
         scope: Header
         name: X-Real-URI
       op: RegexNotMatch
       value: "^/ip/0\\d{2}/.*$"
   backends:
   - serviceName: %s
     servicePort: %d
`, backendSvc, backendPorts[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

			time.Sleep(6 * time.Second)
			err := s.EnsureNumApisixRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
			err = s.EnsureNumApisixUpstreamsCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")

			_ = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("X-Real-Uri", "/ip/0983/v4").
				Expect().
				Status(http.StatusOK)

			msg := s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("X-Real-Uri", "/ip/098/v4").
				Expect().
				Status(http.StatusNotFound).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg, "404 Route Not Found")

			_ = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				Expect().
				Status(http.StatusOK).
				Body().
				Raw()
		})

		ginkgo.It("operator is regex match in case insensitive mode", func() {
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
     exprs:
     - subject:
         scope: Header
         name: X-Real-URI
       op: RegexMatchCaseInsensitive
       value: "^/ip/0\\d{2}/.*$"
   backends:
   - serviceName: %s
     servicePort: %d
`, backendSvc, backendPorts[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

			time.Sleep(6 * time.Second)
			err := s.EnsureNumApisixRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
			err = s.EnsureNumApisixUpstreamsCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")

			_ = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("X-Real-Uri", "/IP/098/v4").
				Expect().
				Status(http.StatusOK)

			msg := s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("X-Real-Uri", "/ip/0983/v4").
				Expect().
				Status(http.StatusNotFound).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg, "404 Route Not Found")

			msg = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				Expect().
				Status(http.StatusNotFound).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg, "404 Route Not Found")
		})

		ginkgo.It("operator is regex not match in case insensitive mode", func() {
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
     exprs:
     - subject:
         scope: Header
         name: X-Real-URI
       op: RegexNotMatchCaseInsensitive
       value: "^/ip/0\\d{2}/.*$"
   backends:
   - serviceName: %s
     servicePort: %d
`, backendSvc, backendPorts[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

			err := s.EnsureNumApisixRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
			err = s.EnsureNumApisixUpstreamsCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")

			_ = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("X-Real-Uri", "/IP/0983/v4").
				Expect().
				Status(http.StatusOK)

			msg := s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("X-Real-Uri", "/IP/098/v4").
				Expect().
				Status(http.StatusNotFound).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg, "404 Route Not Found")

			_ = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				Expect().
				Status(http.StatusOK).
				Body().
				Raw()
		})
	}

	ginkgo.Describe("suite-features: scaffold v2", func() {
		suites(scaffold.NewDefaultV2Scaffold)
	})
})

var _ = ginkgo.Describe("suite-features: route match exprs bugfixes", func() {
	suites := func(scaffoldFunc func() *scaffold.Scaffold) {
		s := scaffoldFunc()
		ginkgo.It("exprs scope", func() {
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
     exprs:
     - subject:
         scope: Header
         name: X-Real-URI
       op: RegexMatchCaseInsensitive
       value: "^/ip/0\\d{2}/.*$"
   backends:
   - serviceName: %s
     servicePort: %d
 - name: rule2
   match:
     hosts:
     - httpbin.org
     paths:
     - /headers
   backends:
   - serviceName: %s
     servicePort: %d
`, backendSvc, backendPorts[0], backendSvc, backendPorts[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

			err := s.EnsureNumApisixRoutesCreated(2)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
			time.Sleep(6 * time.Second)
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1))

			routes, err := s.ListApisixRoutes()
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.Len(ginkgo.GinkgoT(), routes, 2)

			_ = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("X-Real-Uri", "/IP/093/v4").
				Expect().
				Status(http.StatusOK)

			_ = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("X-Real-Uri", "/IP/0981/v4").
				Expect().
				Status(http.StatusNotFound).
				Body().Contains("404 Route Not Found")

			// If the exprs in rule1 escaped to rulel2, then the following request
			// will throw 404 Route Not Found.
			_ = s.NewAPISIXClient().GET("/headers").
				WithHeader("Host", "httpbin.org").
				Expect().
				Status(http.StatusOK).
				Body().Contains("httpbin.org")
		})
	}

	ginkgo.Describe("suite-features: scaffold v2", func() {
		suites(scaffold.NewDefaultV2Scaffold)
	})
})

var _ = ginkgo.Describe("suite-features: route match exprs with variable", func() {
	s := scaffold.NewDefaultScaffold()
	ginkgo.It("exprs with request_method variable", func() {
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
       - /get
       - /post
       - /put
     exprs:
     - subject:
         scope: Variable
         name: request_method
       op: In
       set:
       - GET
       - POST
   backends:
   - serviceName: %s
     servicePort: %d
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar), "creating route")
		time.Sleep(6 * time.Second)
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "Checking number of routes")

		_ = s.NewAPISIXClient().GET("/get").
			WithHeader("Host", "httpbin.org").
			Expect().
			Status(http.StatusOK)

		_ = s.NewAPISIXClient().POST("/post").
			WithHeader("Host", "httpbin.org").
			Expect().
			Status(http.StatusOK)

		_ = s.NewAPISIXClient().PUT("/put").
			WithHeader("Host", "httpbin.org").
			Expect().
			Status(http.StatusNotFound)
	})

	ginkgo.It("exprs with host variable", func() {
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
     paths:
       - /ip
     exprs:
     - subject:
         scope: Variable
         name: host
       op: In
       set:
       - httpbin.net
       - httpbin.org
       - httpbin.com
   backends:
   - serviceName: %s
     servicePort: %d
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar), "creating route")
		time.Sleep(6 * time.Second)
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "Checking number of routes")

		_ = s.NewAPISIXClient().GET("/ip").
			WithHeader("Host", "httpbin.net").
			Expect().
			Status(http.StatusOK)

		_ = s.NewAPISIXClient().GET("/ip").
			WithHeader("Host", "httpbin.org").
			Expect().
			Status(http.StatusOK)

		_ = s.NewAPISIXClient().GET("/ip").
			WithHeader("Host", "httpbin.com").
			Expect().
			Status(http.StatusOK)
	})

	ginkgo.It("exprs request_method and host variable", func() {
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
     paths:
       - /*
     exprs:
     - subject:
         scope: Variable
         name: request_method
       op: In
       set:
       - GET
       - PUT
     - subject:
         scope: Variable
         name: host
       op: In
       set:
       - httpbin.org
       - httpbin.com
   backends:
   - serviceName: %s
     servicePort: %d
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar), "creating route")
		time.Sleep(6 * time.Second)
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "Checking number of routes")

		_ = s.NewAPISIXClient().GET("/get").
			WithHeader("Host", "httpbin.net").
			Expect().
			Status(http.StatusNotFound)

		_ = s.NewAPISIXClient().GET("/get").
			WithHeader("Host", "httpbin.org").
			Expect().
			Status(http.StatusOK)

		_ = s.NewAPISIXClient().POST("/post").
			WithHeader("Host", "httpbin.org").
			Expect().
			Status(http.StatusNotFound)

		_ = s.NewAPISIXClient().PUT("/put").
			WithHeader("Host", "httpbin.com").
			Expect().
			Status(http.StatusOK)
	})
})
