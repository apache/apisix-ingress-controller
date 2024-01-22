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

var _ = ginkgo.Describe("suite-plugins-general: config plugin with secretRef", func() {
	suites := func(scaffoldFunc func() *scaffold.Scaffold) {
		s := scaffoldFunc()
		ginkgo.It("suite-plugins-general: echo plugin config with secretRef", func() {
			backendSvc, backendPorts := s.DefaultHTTPBackend()
			secret := `
apiVersion: v1
kind: Secret
metadata:
  name: echo
data:
  # content is "This is the replaced preface"
  before_body: IlRoaXMgaXMgdGhlIHJlcGxhY2VkIHByZWZhY2Ui
  # content is "my custom body"
  body: Im15IGN1c3RvbSBib2R5Ig==
  
`
			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(secret), "creating echo secret for ApisixRoute")
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
   - name: echo
     enable: true
     config:
       before_body: "This is the preface"
       after_body: "This is the epilogue"
       headers:
         X-Foo: v1
         X-Foo2: v2
     secretRef: echo
       
`, backendSvc, backendPorts[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

			err := s.EnsureNumApisixUpstreamsCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
			err = s.EnsureNumApisixRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

			resp := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect()
			resp.Status(http.StatusOK)
			resp.Header("X-Foo").Equal("v1")
			resp.Header("X-Foo2").Equal("v2")
			resp.Body().Contains("This is the replaced preface")
			resp.Body().Contains("This is the epilogue")
			resp.Body().Contains("my custom body")
		})

		ginkgo.It("suite-plugins-general: nested plugin config with secretRef", func() {
			backendSvc, backendPorts := s.DefaultHTTPBackend()
			secret := `
apiVersion: v1
kind: Secret
metadata:
 name: echo
data:
 headers.X-Foo: djI=
 # content is "my custom body"
 body: Im15IGN1c3RvbSBib2R5Ig==
`
			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(secret), "creating echo secret for ApisixRoute")
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
   - name: echo
     enable: true
     config:
       before_body: "This is the preface"
       after_body: "This is the epilogue"
       headers:
         X-Foo: v1
     secretRef: echo
       
`, backendSvc, backendPorts[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

			err := s.EnsureNumApisixUpstreamsCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
			err = s.EnsureNumApisixRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

			resp := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.org").Expect()
			resp.Status(http.StatusOK)
			resp.Header("X-Foo").Equal("v2")
		})
	}

	ginkgo.Describe("suite-plugins-general: scaffold v2", func() {
		suites(scaffold.NewDefaultV2Scaffold)
	})
})
