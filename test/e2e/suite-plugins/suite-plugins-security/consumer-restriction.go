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

var _ = ginkgo.Describe("suite-plugins-security: consumer-restriction plugin", func() {
	suites := func(scaffoldFunc func() *scaffold.Scaffold) {
		s := scaffoldFunc()

		ginkgo.It("restrict consumer_name", func() {
			err := s.ApisixConsumerBasicAuthCreated("jack1", "jack1-username", "jack1-password")
			assert.Nil(ginkgo.GinkgoT(), err, "creating basicAuth ApisixConsumer")

			err = s.ApisixConsumerBasicAuthCreated("jack2", "jack2-username", "jack2-password")
			assert.Nil(ginkgo.GinkgoT(), err, "creating basicAuth ApisixConsumer")

			// Wait until the ApisixConsumer create event was delivered.
			time.Sleep(6 * time.Second)

			grs, err := s.ListApisixConsumers()
			assert.Nil(ginkgo.GinkgoT(), err, "listing consumer")
			assert.Len(ginkgo.GinkgoT(), grs, 2)

			assert.Len(ginkgo.GinkgoT(), grs[0].Plugins, 1)
			assert.Len(ginkgo.GinkgoT(), grs[1].Plugins, 1)

			username := grs[0].Username
			basicAuth := grs[0].Plugins["basic-auth"]
			assert.Equal(ginkgo.GinkgoT(), basicAuth, map[string]interface{}{
				"username": "jack1-username",
				"password": "jack1-password",
			})

			basicAuth2 := grs[1].Plugins["basic-auth"]
			assert.Equal(ginkgo.GinkgoT(), basicAuth2, map[string]interface{}{
				"username": "jack2-username",
				"password": "jack2-password",
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
       - /anything
   backends:
   - serviceName: %s
     servicePort: %d
   authentication:
     enable: true
     type: basicAuth
   plugins:
   - name: consumer-restriction
     enable: true
     config:
       whitelist:
       - "%s"
`, backendSvc, backendPorts[0], username)
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar), "creating ApisixRoute with basicAuth")
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "Checking number of routes")
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "Checking number of upstreams")

			_ = s.NewAPISIXClient().GET("/anything").
				WithHeader("Host", "httpbin.org").
				WithHeader("Authorization", "Basic amFjazEtdXNlcm5hbWU6amFjazEtcGFzc3dvcmQ=").
				Expect().
				Status(http.StatusOK)

			msg401 := s.NewAPISIXClient().GET("/anything").
				WithHeader("Host", "httpbin.org").
				Expect().
				Status(http.StatusUnauthorized).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg401, "Missing authorization in request")

			msg403 := s.NewAPISIXClient().GET("/anything").
				WithHeader("Host", "httpbin.org").
				WithHeader("Authorization", "Basic amFjazItdXNlcm5hbWU6amFjazItcGFzc3dvcmQ=").
				Expect().
				Status(http.StatusForbidden).
				Body().
				Raw()

			assert.Contains(ginkgo.GinkgoT(), msg403, "The consumer_name is forbidden")
		})

		ginkgo.It("restrict allowed_by_methods", func() {
			err := s.ApisixConsumerBasicAuthCreated("jack1", "jack1-username", "jack1-password")
			assert.Nil(ginkgo.GinkgoT(), err, "creating basicAuth ApisixConsumer")

			err = s.ApisixConsumerBasicAuthCreated("jack2", "jack2-username", "jack2-password")
			assert.Nil(ginkgo.GinkgoT(), err, "creating basicAuth ApisixConsumer")

			// Wait until the ApisixConsumer create event was delivered.
			time.Sleep(6 * time.Second)

			grs, err := s.ListApisixConsumers()
			assert.Nil(ginkgo.GinkgoT(), err, "listing consumer")
			assert.Len(ginkgo.GinkgoT(), grs, 2)

			assert.Len(ginkgo.GinkgoT(), grs[0].Plugins, 1)
			assert.Len(ginkgo.GinkgoT(), grs[1].Plugins, 1)

			username := grs[0].Username
			basicAuth := grs[0].Plugins["basic-auth"]
			assert.Equal(ginkgo.GinkgoT(), basicAuth, map[string]interface{}{
				"username": "jack1-username",
				"password": "jack1-password",
			})

			username2 := grs[1].Username
			basicAuth2 := grs[1].Plugins["basic-auth"]
			assert.Equal(ginkgo.GinkgoT(), basicAuth2, map[string]interface{}{
				"username": "jack2-username",
				"password": "jack2-password",
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
       - /anything
   backends:
   - serviceName: %s
     servicePort: %d
   authentication:
     enable: true
     type: basicAuth
   plugins:
   - name: consumer-restriction
     enable: true
     config:
       allowed_by_methods:
       - user: "%s"
         methods:
         - "POST"
         - "GET"
       - user: "%s"
         methods:
         - "GET"
`, backendSvc, backendPorts[0], username, username2)
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar), "creating ApisixRoute with basicAuth")
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "Checking number of routes")
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "Checking number of upstreams")

			_ = s.NewAPISIXClient().GET("/anything").
				WithHeader("Host", "httpbin.org").
				WithHeader("Authorization", "Basic amFjazEtdXNlcm5hbWU6amFjazEtcGFzc3dvcmQ=").
				Expect().
				Status(http.StatusOK)

			_ = s.NewAPISIXClient().POST("/anything").
				WithHeader("Host", "httpbin.org").
				WithHeader("Authorization", "Basic amFjazEtdXNlcm5hbWU6amFjazEtcGFzc3dvcmQ=").
				Expect().
				Status(http.StatusOK)

			_ = s.NewAPISIXClient().GET("/anything").
				WithHeader("Host", "httpbin.org").
				WithHeader("Authorization", "Basic amFjazItdXNlcm5hbWU6amFjazItcGFzc3dvcmQ=").
				Expect().
				Status(http.StatusOK)

			msg403 := s.NewAPISIXClient().POST("/anything").
				WithHeader("Host", "httpbin.org").
				WithHeader("Authorization", "Basic amFjazItdXNlcm5hbWU6amFjazItcGFzc3dvcmQ=").
				Expect().
				Status(http.StatusForbidden).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg403, "The consumer_name is forbidden")

			msg401 := s.NewAPISIXClient().GET("/anything").
				WithHeader("Host", "httpbin.org").
				Expect().
				Status(http.StatusUnauthorized).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg401, "Missing authorization in request")
		})
	}

	ginkgo.Describe("suite-plugins: scaffold v2beta3", func() {
		suites(scaffold.NewDefaultScaffold)
	})
	ginkgo.Describe("suite-plugins: scaffold v2", func() {
		suites(scaffold.NewDefaultV2Scaffold)
	})
})
