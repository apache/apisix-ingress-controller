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
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-plugins-authentication: ApisixConsumer with hmacAuth", func() {
	suites := func(scaffoldFunc func() *scaffold.Scaffold) {
		s := scaffoldFunc()

		ginkgo.It("ApisixRoute with hmacAuth consumer", func() {
			ac := `
apiVersion: apisix.apache.org/v2beta3
kind: ApisixConsumer
metadata:
  name: hmacvalue
spec:
  authParameter:
    hmacAuth:
      value:
        access_key: papa
        secret_key: fatpa
        algorithm: "hmac-sha256"
        clock_skew: 0
`
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ac), "creating hmacAuth ApisixConsumer")

			// Wait until the ApisixConsumer create event was delivered.
			time.Sleep(6 * time.Second)

			grs, err := s.ListApisixConsumers()
			assert.Nil(ginkgo.GinkgoT(), err, "listing consumer")
			assert.Len(ginkgo.GinkgoT(), grs, 1)
			assert.Len(ginkgo.GinkgoT(), grs[0].Plugins, 1)
			hmacAuth, _ := grs[0].Plugins["hmac-auth"].(map[string]interface{})
			assert.Equal(ginkgo.GinkgoT(), "papa", hmacAuth["access_key"])
			assert.Equal(ginkgo.GinkgoT(), "fatpa", hmacAuth["secret_key"])
			assert.Equal(ginkgo.GinkgoT(), "hmac-sha256", hmacAuth["algorithm"])
			assert.Equal(ginkgo.GinkgoT(), float64(0), hmacAuth["clock_skew"])

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
     type: hmacAuth
`, backendSvc, backendPorts[0])
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar), "creating ApisixRoute with hmacAuth")
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "Checking number of routes")
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "Checking number of upstreams")

			_ = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("X-Foo", "bar").
				WithHeader("X-HMAC-SIGNATURE", "l3Uka7E1kxPA/owQ2+OqJUmflRppjD5q8xPcWbyKKrg=").
				WithHeader("X-HMAC-ACCESS-KEY", "papa").
				WithHeader("X-HMAC-ALGORITHM", "hmac-sha256").
				WithHeader("X-HMAC-SIGNED-HEADERS", "User-Agent;X-Foo").
				WithHeader("User-Agent", "curl/7.29.0").
				Expect().
				Status(http.StatusOK)

			msg := s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("X-Foo", "bar").
				Expect().
				Status(http.StatusUnauthorized).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg, "client request can't be validated")

			msg = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("X-Foo", "baz").
				WithHeader("X-HMAC-SIGNATURE", "MhGJMkEYFD+98qtvoDPlvCGIUSmmUaw0In/D0vt2Z4E=").
				WithHeader("X-HMAC-ACCESS-KEY", "papa").
				WithHeader("X-HMAC-ALGORITHM", "hmac-sha256").
				WithHeader("X-HMAC-SIGNED-HEADERS", "User-Agent;X-Foo").
				WithHeader("User-Agent", "curl/7.29.0").
				Expect().
				Status(http.StatusNotFound).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg, "404 Route Not Found")
		})

		ginkgo.It("ApisixRoute with hmacAuth consumer using secret", func() {
			secret := `
apiVersion: v1
kind: Secret
metadata:
  name: hmac
data:
  access_key: cGFwYQ==
  secret_key: ZmF0cGE=
  algorithm: aG1hYy1zaGEyNTY=
  clock_skew: MA==
`
			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(secret), "creating hmac secret for ApisixConsumer")

			ac := `
apiVersion: apisix.apache.org/v2beta3
kind: ApisixConsumer
metadata:
  name: hmacvalue
spec:
  authParameter:
    hmacAuth:
      secretRef:
        name: hmac
`
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ac), "creating hmacAuth ApisixConsumer")

			// Wait until the ApisixConsumer create event was delivered.
			time.Sleep(6 * time.Second)

			grs, err := s.ListApisixConsumers()
			assert.Nil(ginkgo.GinkgoT(), err, "listing consumer")
			assert.Len(ginkgo.GinkgoT(), grs, 1)
			assert.Len(ginkgo.GinkgoT(), grs[0].Plugins, 1)
			hmacAuth, _ := grs[0].Plugins["hmac-auth"].(map[string]interface{})
			assert.Equal(ginkgo.GinkgoT(), "papa", hmacAuth["access_key"])
			assert.Equal(ginkgo.GinkgoT(), "fatpa", hmacAuth["secret_key"])
			assert.Equal(ginkgo.GinkgoT(), "hmac-sha256", hmacAuth["algorithm"])
			assert.Equal(ginkgo.GinkgoT(), float64(0), hmacAuth["clock_skew"])

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
     type: hmacAuth
`, backendSvc, backendPorts[0])
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar), "creating ApisixRoute with hmacAuth")
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "Checking number of routes")
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "Checking number of upstreams")

			_ = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("X-Foo", "bar").
				WithHeader("X-HMAC-SIGNATURE", "l3Uka7E1kxPA/owQ2+OqJUmflRppjD5q8xPcWbyKKrg=").
				WithHeader("X-HMAC-ACCESS-KEY", "papa").
				WithHeader("X-HMAC-ALGORITHM", "hmac-sha256").
				WithHeader("X-HMAC-SIGNED-HEADERS", "User-Agent;X-Foo").
				WithHeader("User-Agent", "curl/7.29.0").
				Expect().
				Status(http.StatusOK)

			msg := s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("X-Foo", "bar").
				Expect().
				Status(http.StatusUnauthorized).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg, "client request can't be validated")

			msg = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("X-Foo", "baz").
				WithHeader("X-HMAC-SIGNATURE", "MhGJMkEYFD+98qtvoDPlvCGIUSmmUaw0In/D0vt2Z4E=").
				WithHeader("X-HMAC-ACCESS-KEY", "papa").
				WithHeader("X-HMAC-ALGORITHM", "hmac-sha256").
				WithHeader("X-HMAC-SIGNED-HEADERS", "User-Agent;X-Foo").
				WithHeader("User-Agent", "curl/7.29.0").
				Expect().
				Status(http.StatusNotFound).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg, "404 Route Not Found")
		})
	}

	ginkgo.Describe("suite-plugins-authentication: scaffold v2", func() {
		suites(scaffold.NewDefaultV2Scaffold)
	})
})
