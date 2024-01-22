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
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-plugins-security: ip-restriction plugin", func() {
	suites := func(scaffoldFunc func() *scaffold.Scaffold) {
		s := scaffoldFunc()
		ginkgo.It("ip whitelist", func() {
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
       - /hello
   backends:
   - serviceName: %s
     servicePort: %d
     weight: 10
   plugins:
   - name: ip-restriction
     enable: true
     config:
       whitelist:
       - "192.168.3.3"
`, backendSvc, backendPorts[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

			err := s.EnsureNumApisixUpstreamsCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
			err = s.EnsureNumApisixRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

			// As we use port forwarding so the ip address is 127.0.0.1
			s.NewAPISIXClient().GET("/hello").WithHeader("Host", "httpbin.org").
				Expect().
				Status(403).
				Body().
				Contains("Your IP address is not allowed")

			ar = fmt.Sprintf(`
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
   - name: ip-restriction
     enable: true
     config:
       whitelist:
       - "127.0.0.1"
`, backendSvc, backendPorts[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

			err = s.EnsureNumApisixUpstreamsCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
			err = s.EnsureNumApisixRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

			// As we use port forwarding so the ip address is 127.0.0.1
			s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").
				Expect().
				Status(200).
				Body().
				Contains("origin")
		})
		ginkgo.It("ip blacklist", func() {
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
       - /hello
   backends:
   - serviceName: %s
     servicePort: %d
     weight: 10
   plugins:
   - name: ip-restriction
     enable: true
     config:
       blacklist:
       - "127.0.0.1"
`, backendSvc, backendPorts[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

			err := s.EnsureNumApisixUpstreamsCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
			err = s.EnsureNumApisixRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

			// As we use port forwarding so the ip address is 127.0.0.1
			s.NewAPISIXClient().GET("/hello").WithHeader("Host", "httpbin.org").
				Expect().
				Status(403).
				Body().
				Contains("Your IP address is not allowed")

			ar = fmt.Sprintf(`
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
   - name: ip-restriction
     enable: true
     config:
       blacklist:
       - "192.168.12.12"
`, backendSvc, backendPorts[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

			// EnsureNumApisixRoutesCreated cannot be used to ensure update Correctness.
			time.Sleep(6 * time.Second)

			// As we use port forwarding so the ip address is 127.0.0.1
			s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").
				Expect().
				Status(200).
				Body().
				Contains("origin")
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
   - name: ip-restriction
     enable: false
     config:
       blacklist:
       - "127.0.0.1"
`, backendSvc, backendPorts[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

			err := s.EnsureNumApisixUpstreamsCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
			err = s.EnsureNumApisixRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

			// As we use port forwarding so the ip address is 127.0.0.1
			s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").
				Expect().
				Status(200).
				Body().
				Contains("origin")
		})
	}

	ginkgo.Describe("suite-plugins-security: scaffold v2", func() {
		suites(scaffold.NewDefaultV2Scaffold)
	})
})
