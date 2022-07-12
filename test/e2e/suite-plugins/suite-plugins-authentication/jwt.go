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

var _ = ginkgo.Describe("suite-plugins-authentication: ApisixConsumer with jwtAuth", func() {
	suites := func(scaffoldFunc func() *scaffold.Scaffold) {
		s := scaffoldFunc()

		ginkgo.It("ApisixRoute with jwtAuth consumer", func() {
			ac := `
apiVersion: apisix.apache.org/v2beta3
kind: ApisixConsumer
metadata:
  name: foo
spec:
  authParameter:
    jwtAuth:
      value:
        key: foo-key
`
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ac), "creating jwtAuth ApisixConsumer")

			// Wait until the ApisixConsumer create event was delivered.
			time.Sleep(6 * time.Second)

			grs, err := s.ListApisixConsumers()
			assert.Nil(ginkgo.GinkgoT(), err, "listing consumer")
			assert.Len(ginkgo.GinkgoT(), grs, 1)
			assert.Len(ginkgo.GinkgoT(), grs[0].Plugins, 1)
			jwtAuth, _ := grs[0].Plugins["jwt-auth"].(map[string]interface{})
			assert.Equal(ginkgo.GinkgoT(), jwtAuth["key"], "foo-key")

			adminSvc, adminPort := s.ApisixAdminServiceAndPort()
			ar1 := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2beta3
kind: ApisixRoute
metadata:
  name: default
spec:
  http:
  - name: public-api
    match:
      paths:
      - /apisix/plugin/jwt/sign
    backends:
    - serviceName: %s
      servicePort: %d
    plugins:
    - name: public-api
      enable: true
`, adminSvc, adminPort)
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar1), "creating ApisixRoute")
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "Checking number of routes")
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "Checking number of upstreams")

			backendSvc, backendPorts := s.DefaultHTTPBackend()
			ar2 := fmt.Sprintf(`
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
   backends:
   - serviceName: %s
     servicePort: %d
   authentication:
     enable: true
     type: jwtAuth
`, backendSvc, backendPorts[0])
			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar2), "Creating ApisixRoute with jwtAuth")
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(2), "Checking number of routes")
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(2), "Checking number of upstreams")

			msg401 := s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				Expect().
				Status(http.StatusUnauthorized).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg401, "Missing JWT token in request")

			token := s.NewAPISIXClient().GET("/apisix/plugin/jwt/sign").
				WithQuery("key", "foo-key").
				Expect().
				Status(http.StatusOK).
				Body().
				NotEmpty().
				Raw()

			_ = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("Authorization", string(token)).
				Expect().
				Status(http.StatusOK)
		})

		ginkgo.It("ApisixRoute with jwtAuth consumer using secret", func() {
			secret := `
apiVersion: v1
kind: Secret
metadata:
  name: jwt
data:
  key: Zm9vLWtleQ==
`
			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(secret), "creating jwtAuth secret for ApisixConsumer")

			ac := `
apiVersion: apisix.apache.org/v2beta3
kind: ApisixConsumer
metadata:
  name: foo
spec:
  authParameter:
    jwtAuth:
      secretRef:
        name: jwt
`
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ac), "creating jwtAuth ApisixConsumer")

			// Wait until the ApisixConsumer create event was delivered.
			time.Sleep(6 * time.Second)

			grs, err := s.ListApisixConsumers()
			assert.Nil(ginkgo.GinkgoT(), err, "listing consumer")
			assert.Len(ginkgo.GinkgoT(), grs, 1)
			assert.Len(ginkgo.GinkgoT(), grs[0].Plugins, 1)
			jwtAuth, _ := grs[0].Plugins["jwt-auth"].(map[string]interface{})
			assert.Equal(ginkgo.GinkgoT(), jwtAuth["key"], "foo-key")

			adminSvc, adminPort := s.ApisixAdminServiceAndPort()
			ar1 := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2beta3
kind: ApisixRoute
metadata:
  name: default
spec:
  http:
  - name: public-api
    match:
      paths:
      - /apisix/plugin/jwt/sign
    backends:
    - serviceName: %s
      servicePort: %d
    plugins:
    - name: public-api
      enable: true
`, adminSvc, adminPort)
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar1), "creating ApisixRoute")
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "Checking number of routes")
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "Checking number of upstreams")

			backendSvc, backendPorts := s.DefaultHTTPBackend()
			ar2 := fmt.Sprintf(`
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
   backends:
   - serviceName: %s
     servicePort: %d
   authentication:
     enable: true
     type: jwtAuth
`, backendSvc, backendPorts[0])
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar2), "Creating ApisixRoute with jwtAuth")
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(2), "Checking number of routes")
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(2), "Checking number of upstreams")

			msg401 := s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				Expect().
				Status(http.StatusUnauthorized).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg401, "Missing JWT token in request")

			token := s.NewAPISIXClient().GET("/apisix/plugin/jwt/sign").
				WithQuery("key", "foo-key").
				Expect().
				Status(http.StatusOK).
				Body().
				NotEmpty().
				Raw()

			_ = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("Authorization", string(token)).
				Expect().
				Status(http.StatusOK)
		})
	}

	ginkgo.Describe("suite-plugins-authentication: scaffold v2beta3", func() {
		suites(scaffold.NewDefaultScaffold)
	})
	ginkgo.Describe("suite-plugins-authentication: scaffold v2", func() {
		suites(scaffold.NewDefaultV2Scaffold)
	})
})
