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

	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
	"github.com/onsi/ginkgo"
)

var _ = ginkgo.Describe("request-id plugin", func() {
	opts := &scaffold.Options{
		Name:                  "default",
		Kubeconfig:            scaffold.GetKubeconfig(),
		APISIXConfigPath:      "testdata/apisix-gw-config.yaml",
		IngressAPISIXReplicas: 1,
		HTTPBinServicePort:    80,
		APISIXRouteVersion:    "apisix.apache.org/v2alpha1",
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
   plugins:
   - name: request-id
     enable: true
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar))

		err := s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		err = s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		resp := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect()
		resp.Status(200)
		resp.Header("X-Request-Id").NotEmpty()
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
   - name: request-id
     enable: false
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar))

		err := s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		err = s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		resp := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect()
		resp.Status(200)
		resp.Header("X-Request-Id").Empty()
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
   - name: request-id
     enable: true
`, backendSvc, backendPorts[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar))

		err := s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		err = s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		resp := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect()
		resp.Status(200)
		resp.Header("X-Request-Id").NotEmpty()
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
		resp.Status(200)
		resp.Header("X-Request-Id").Empty()
		resp.Body().Contains("origin")
	})
})
