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

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
)

var _ = ginkgo.Describe("fault-injection plugin", func() {
	opts := &scaffold.Options{
		Name:                  "default",
		Kubeconfig:            scaffold.GetKubeconfig(),
		APISIXConfigPath:      "testdata/apisix-gw-config.yaml",
		IngressAPISIXReplicas: 1,
		HTTPBinServicePort:    80,
		APISIXRouteVersion:    "apisix.apache.org/v2alpha1",
	}
	s := scaffold.NewScaffold(opts)
	ginkgo.It("empty config", func() {
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
   plugins:
   - name: cors
     enable: true
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar))
		err := s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		err = s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		resp := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect()
		resp.Status(http.StatusOK)
		resp.Header("Access-Control-Allow-Origin").Equal("*")
		resp.Header("Access-Control-Allow-Methods").Equal("*")
		resp.Header("Access-Control-Allow-Headers").Equal("*")
		resp.Header("Access-Control-Expose-Headers").Equal("*")
		resp.Header("Access-Control-Max-Age").Equal("5")
		resp.Body().Contains("origin")
	})
	ginkgo.It("finer granularity config", func() {
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
   plugins:
   - name: cors
     enable: true
     config:
       allow_origins: http://foo.bar.org
       allow_methods: "GET,POST"
       max_age: 3600
       expose_headers: x-foo,x-baz
       allow_headers: x-from-ingress
       allow_credential: true
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar))
		err := s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		err = s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		resp := s.NewAPISIXClient().GET("/ip").
			WithHeader("Host", "httpbin.org").
			WithHeader("Origin", "http://foo.bar.org").
			Expect()
		resp.Status(http.StatusOK)
		resp.Header("Access-Control-Allow-Origin").Equal("http://foo.bar.org")
		resp.Header("Access-Control-Allow-Methods").Equal("GET,POST")
		resp.Header("Access-Control-Allow-Headers").Equal("x-from-ingress")
		resp.Header("Access-Control-Expose-Headers").Equal("x-foo,x-baz")
		resp.Header("Access-Control-Max-Age").Equal("3600")
		resp.Header("Access-Control-Allow-Credentials").Equal("true")
		resp.Body().Contains("origin")

		resp = s.NewAPISIXClient().GET("/ip").
			WithHeader("Host", "httpbin.org").
			WithHeader("Origin", "http://foo.bar2.org").
			Expect()
		resp.Header("Access-Control-Allow-Methods").Empty()
		resp.Header("Access-Control-Allow-Headers").Empty()
		resp.Header("Access-Control-Expose-Headers").Empty()
		resp.Header("Access-Control-Max-Age").Empty()
		// httpbin set it by itself.
		//resp.Header("Access-Control-Allow-Credentials").Empty()
		resp.Body().Contains("origin")
	})
	ginkgo.It("allow_origins_by_regex", func() {
		ginkgo.Skip("APISIX version priors to 2.5 doesn't contain allow_origins_by_regex in cors plugin")
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
   plugins:
   - name: cors
     enable: true
     config:
       allow_origins_by_regex:
       - foo\\.(bar|baz)\\.org
       allow_methods: "GET,POST"
       max_age: 3600
       expose_headers: x-foo,x-baz
       allow_headers: x-from-ingress
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar))
		err := s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		err = s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		resp := s.NewAPISIXClient().GET("/ip").
			WithHeader("Host", "httpbin.org").
			WithHeader("Origin", "http://foo.bar.org").
			Expect()
		resp.Status(http.StatusOK)
		resp.Header("Access-Control-Allow-Origin").Equal("http://foo.bar.org")
		resp.Header("Access-Control-Allow-Methods").Equal("GET,POST")
		resp.Header("Access-Control-Allow-Headers").Equal("x-from-ingress")
		resp.Header("Access-Control-Expose-Headers").Equal("x-foo,x-baz")
		resp.Header("Access-Control-Max-Age").Equal("3600")
		resp.Header("Access-Control-Allow-Credentials").Equal("true")
		resp.Body().Contains("origin")

		resp = s.NewAPISIXClient().GET("/ip").
			WithHeader("Host", "httpbin.org").
			WithHeader("Origin", "http://foo.baz.org").
			Expect()
		resp.Status(http.StatusOK)
		resp.Header("Access-Control-Allow-Origin").Equal("http://foo.baz.org")
		resp.Header("Access-Control-Allow-Methods").Equal("GET,POST")
		resp.Header("Access-Control-Allow-Headers").Equal("x-from-ingress")
		resp.Header("Access-Control-Expose-Headers").Equal("x-foo,x-baz")
		resp.Header("Access-Control-Max-Age").Equal("3600")
		resp.Header("Access-Control-Allow-Credentials").Equal("true")
		resp.Body().Contains("origin")

		resp = s.NewAPISIXClient().GET("/ip").
			WithHeader("Host", "httpbin.org").
			WithHeader("Origin", "http://foo.bar2.org").
			Expect()
		resp.Header("Access-Control-Allow-Methods").Empty()
		resp.Header("Access-Control-Allow-Headers").Empty()
		resp.Header("Access-Control-Expose-Headers").Empty()
		resp.Header("Access-Control-Max-Age").Empty()
		// httpbin set it by itself.
		//resp.Header("Access-Control-Allow-Credentials").Empty()
		resp.Body().Contains("origin")
	})
	ginkgo.It("disable plugin", func() {
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
   plugins:
   - name: cors
     enable: false
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar))
		err := s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		err = s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		resp := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect()
		resp.Status(http.StatusOK)
		// httpbin sets this header by itself.
		//resp.Header("Access-Control-Allow-Origin").Empty()
		resp.Header("Access-Control-Allow-Methods").Empty()
		resp.Header("Access-Control-Allow-Headers").Empty()
		resp.Header("Access-Control-Expose-Headers").Empty()
		resp.Header("Access-Control-Max-Age").Empty()
		resp.Body().Contains("origin")
	})
	ginkgo.It("enable plugin and then delete it", func() {
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
   plugins:
   - name: cors
     enable: true
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar))
		err := s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		err = s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		resp := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect()
		resp.Status(http.StatusOK)

		resp.Header("Access-Control-Allow-Origin").Equal("*")
		resp.Header("Access-Control-Allow-Methods").Equal("*")
		resp.Header("Access-Control-Allow-Headers").Equal("*")
		resp.Header("Access-Control-Expose-Headers").Equal("*")
		resp.Header("Access-Control-Max-Age").Equal("5")
		resp.Body().Contains("origin")

		ar = fmt.Sprintf(`
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
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar))
		err = s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		err = s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		resp = s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect()
		resp.Status(http.StatusOK)

		// httpbin sets this header by itself.
		//resp.Header("Access-Control-Allow-Origin").Empty()
		resp.Header("Access-Control-Allow-Methods").Empty()
		resp.Header("Access-Control-Allow-Headers").Empty()
		resp.Header("Access-Control-Expose-Headers").Empty()
		resp.Header("Access-Control-Max-Age").Empty()
		resp.Body().Contains("origin")
	})
})
