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
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-plugins-authentication: ApisixConsumer with keyAuth", func() {
	suites := func(scaffoldFunc func() *scaffold.Scaffold) {
		s := scaffoldFunc()

		ginkgo.It("ApisixRoute with keyAuth consumer", func() {
			assert.Nil(ginkgo.GinkgoT(), s.ApisixConsumerKeyAuthCreated("keyvalue", "foo"), "creating keyAuth ApisixConsumer")

			// Wait until the ApisixConsumer create event was delivered.
			time.Sleep(6 * time.Second)

			grs, err := s.ListApisixConsumers()
			assert.Nil(ginkgo.GinkgoT(), err, "listing consumer")
			assert.Len(ginkgo.GinkgoT(), grs, 1)
			assert.Len(ginkgo.GinkgoT(), grs[0].Plugins, 1)
			basicAuth, _ := grs[0].Plugins["key-auth"]
			assert.Equal(ginkgo.GinkgoT(), basicAuth, map[string]interface{}{
				"key": "foo",
			})

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
   authentication:
     enable: true
     type: keyAuth
`, backendSvc, backendPorts[0])
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar), "creating ApisixRoute with keyAuth")
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "Checking number of routes")
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "Checking number of upstreams")

			_ = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("X-Foo", "bar").
				WithHeader("apikey", "foo").
				Expect().
				Status(http.StatusOK)

			msg := s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("X-Foo", "bar").
				Expect().
				Status(http.StatusUnauthorized).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg, "Missing API key found in request")

			msg = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("X-Foo", "baz").
				WithHeader("apikey", "baz").
				Expect().
				Status(http.StatusNotFound).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg, "404 Route Not Found")
		})

		ginkgo.It("ApisixRoute with keyAuth consumer using secret", func() {
			secret := `
apiVersion: v1
kind: Secret
metadata:
  name: keyauth
data:
  key: Zm9v
`
			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(secret), "creating keyauth secret for ApisixConsumer")
			assert.Nil(ginkgo.GinkgoT(), s.ApisixConsumerKeyAuthSecretCreated("keyvalue", "keyauth"), "creating keyAuth ApisixConsumer")

			// Wait until the ApisixConsumer create event was delivered.
			time.Sleep(6 * time.Second)

			grs, err := s.ListApisixConsumers()
			assert.Nil(ginkgo.GinkgoT(), err, "listing consumer")
			assert.Len(ginkgo.GinkgoT(), grs, 1)
			assert.Len(ginkgo.GinkgoT(), grs[0].Plugins, 1)
			basicAuth, _ := grs[0].Plugins["key-auth"]
			assert.Equal(ginkgo.GinkgoT(), basicAuth, map[string]interface{}{
				"key": "foo",
			})

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
   authentication:
     enable: true
     type: keyAuth
`, backendSvc, backendPorts[0])
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar), "creating ApisixRoute with keyAuth")
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "Checking number of routes")
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "Checking number of upstreams")

			_ = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("X-Foo", "bar").
				WithHeader("apikey", "foo").
				Expect().
				Status(http.StatusOK)

			msg := s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("X-Foo", "bar").
				Expect().
				Status(http.StatusUnauthorized).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg, "Missing API key found in request")

			msg = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("X-Foo", "baz").
				WithHeader("apikey", "baz").
				Expect().
				Status(http.StatusNotFound).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg, "404 Route Not Found")
		})
	}

	ginkgo.Describe("suite-plugins-authentication: scaffold v2beta3", func() {
		suites(scaffold.NewDefaultScaffold)
	})
	ginkgo.Describe("suite-plugins-authentication: scaffold v2", func() {
		suites(scaffold.NewDefaultV2Scaffold)
	})
})
