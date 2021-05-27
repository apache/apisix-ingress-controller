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

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
)

var _ = ginkgo.Describe("route match exprs", func() {
	opts := &scaffold.Options{
		Name:                  "default",
		Kubeconfig:            scaffold.GetKubeconfig(),
		APISIXConfigPath:      "testdata/apisix-gw-config.yaml",
		IngressAPISIXReplicas: 1,
		HTTPBinServicePort:    80,
		APISIXRouteVersion:    "apisix.apache.org/v2alpha1",
	}
	s := scaffold.NewScaffold(opts)
	ginkgo.It("operator is equal", func() {
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
     exprs:
     - subject:
         scope: Header
         name: X-Foo
       op: Equal
       value: bar
   backend:
     serviceName: %s
     servicePort: %d
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar))

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
     exprs:
     - subject:
         scope: Header
         name: X-Foo
       op: NotEqual
       value: bar
   backend:
     serviceName: %s
     servicePort: %d
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar))

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
     exprs:
     - subject:
         scope: Query
         name: id
       op: GreaterThan
       value: "13"
   backend:
     serviceName: %s
     servicePort: %d
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar))

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

	ginkgo.It("operator is less_than", func() {
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
     exprs:
     - subject:
         scope: Query
         name: ID
       op: LessThan
       value: "13"
   backend:
     serviceName: %s
     servicePort: %d
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar))

		err := s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
		err = s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")

		_ = s.NewAPISIXClient().GET("/ip").
			WithHeader("Host", "httpbin.org").
			WithQuery("id", 12).
			Expect().
			Status(http.StatusOK)

		msg := s.NewAPISIXClient().GET("/ip").
			WithHeader("Host", "httpbin.org").
			WithQuery("id", 13).
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
     exprs:
     - subject:
         scope: Header
         name: Content-Type
       op: In
       set: ["text/plain", "text/html", "image/jpeg"]
   backend:
     serviceName: %s
     servicePort: %d
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar))

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
     exprs:
     - subject:
         scope: Header
         name: Content-Type
       op: NotIn
       set: ["text/plain", "text/html", "image/jpeg"]
   backend:
     serviceName: %s
     servicePort: %d
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar))

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
     exprs:
     - subject:
         scope: Header
         name: x-Real-URI
       op: RegexMatch
       value: "^/ip/0\\d{2}/.*$"
   backend:
     serviceName: %s
     servicePort: %d
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar))

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
     exprs:
     - subject:
         scope: Header
         name: X-Real-URI
       op: RegexNotMatch
       value: "^/ip/0\\d{2}/.*$"
   backend:
     serviceName: %s
     servicePort: %d
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar))

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
     exprs:
     - subject:
         scope: Header
         name: X-Real-URI
       op: RegexMatchCaseInsensitive
       value: "^/ip/0\\d{2}/.*$"
   backend:
     serviceName: %s
     servicePort: %d
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar))

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
     exprs:
     - subject:
         scope: Header
         name: X-Real-URI
       op: RegexNotMatchCaseInsensitive
       value: "^/ip/0\\d{2}/.*$"
   backend:
     serviceName: %s
     servicePort: %d
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar))

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
})

var _ = ginkgo.Describe("route match exprs bugfixes", func() {
	opts := &scaffold.Options{
		Name:                  "default",
		Kubeconfig:            scaffold.GetKubeconfig(),
		APISIXConfigPath:      "testdata/apisix-gw-config.yaml",
		IngressAPISIXReplicas: 1,
		HTTPBinServicePort:    80,
		APISIXRouteVersion:    "apisix.apache.org/v2alpha1",
	}
	s := scaffold.NewScaffold(opts)
	ginkgo.It("exprs scope", func() {
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
     exprs:
     - subject:
         scope: Header
         name: X-Real-URI
       op: RegexMatchCaseInsensitive
       value: "^/ip/0\\d{2}/.*$"
   backend:
     serviceName: %s
     servicePort: %d
 - name: rule2
   match:
     hosts:
     - httpbin.org
     paths:
     - /headers
   backend:
     serviceName: %s
     servicePort: %d
`, backendSvc, backendPorts[0], backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar))

		err := s.EnsureNumApisixRoutesCreated(2)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
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
})
