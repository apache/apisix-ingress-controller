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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-plugins-authentication: ApisixConsumer with wolfRBAC", func() {
	suites := func(scaffoldFunc func() *scaffold.Scaffold) {
		s := scaffoldFunc()

		ginkgo.Context("wolfRBAC-server", func() {
			getWolfRBACServerURL := func() (string, error) {
				cmd := exec.Command("sh", "testdata/wolf-rbac/cmd.sh", "ip")
				ip, err := cmd.Output()
				if err != nil {
					return "", err
				}
				if len(ip) == 0 {
					return "", fmt.Errorf("wolf-server start failed")
				}
				return fmt.Sprintf("http://%s:12180", string(ip)), nil
			}
			ginkgo.It("ApisixRoute with wolfRBAC consumer", func() {
				wolfSvr, err := getWolfRBACServerURL()
				assert.Nil(ginkgo.GinkgoT(), err, "checking wolf-server")
				ac := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2beta3
kind: ApisixConsumer
metadata:
  name: wolf-user
spec:
  authParameter:
    wolfRBAC:
      value:
        server: "%s"
        appid: "test-app"
        header_prefix: "X-"
`, wolfSvr)
				assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ac), "creating wolfRBAC ApisixConsumer")

				// Wait until the ApisixConsumer create event was delivered.
				time.Sleep(6 * time.Second)

				grs, err := s.ListApisixConsumers()
				assert.Nil(ginkgo.GinkgoT(), err, "listing consumer")
				assert.Len(ginkgo.GinkgoT(), grs, 1)
				assert.Len(ginkgo.GinkgoT(), grs[0].Plugins, 1)
				wolfRBAC, _ := grs[0].Plugins["wolf-rbac"].(map[string]interface{})
				assert.Equal(ginkgo.GinkgoT(), wolfRBAC, map[string]interface{}{
					"server":        wolfSvr,
					"appid":         "test-app",
					"header_prefix": "X-",
				})
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
      - /apisix/plugin/wolf-rbac/login
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
       - /*
   backends:
   - serviceName: %s
     servicePort: %d
   authentication:
     enable: true
     type: wolfRBAC
`, backendSvc, backendPorts[0])
				assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar2), "creating ApisixRoute")
				assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(2), "Checking number of routes")
				assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(2), "Checking number of upstreams")
				payload := []byte(`
{
	"appid": "test-app",
	"username": "test",
	"password": "test-123456",
	"authType": 1
}
		`)
				body := s.NewAPISIXClient().POST("/apisix/plugin/wolf-rbac/login").
					WithHeader("Content-Type", "application/json").
					WithBytes(payload).
					Expect().
					Status(http.StatusOK).
					Body().
					Contains("rbac_token").
					Raw()

				data := struct {
					Token string `json:"rbac_token"`
				}{}
				_ = json.Unmarshal([]byte(body), &data)

				_ = s.NewAPISIXClient().GET("").
					WithHeader("Host", "httpbin.org").
					WithHeader("Authorization", data.Token).
					Expect().
					Status(http.StatusOK)

				msg401 := s.NewAPISIXClient().GET("").
					WithHeader("Host", "httpbin.org").
					Expect().
					Status(http.StatusUnauthorized).
					Body().
					Raw()
				assert.Contains(ginkgo.GinkgoT(), msg401, "Missing rbac token in request")
			})

			ginkgo.It("ApisixRoute with wolfRBAC consumer using secret", func() {
				wolfSvr, err := getWolfRBACServerURL()
				assert.Nil(ginkgo.GinkgoT(), err, "checking wolf-server")
				secret := fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: rbac
data:
  server: %s
  appid: dGVzdC1hcHA=
  header_prefix: WC0=
`, base64.StdEncoding.EncodeToString([]byte(wolfSvr)))
				assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(secret), "creating wolfRBAC secret for ApisixConsumer")

				ac := `
apiVersion: apisix.apache.org/v2beta3
kind: ApisixConsumer
metadata:
  name: wolf-user
spec:
  authParameter:
    wolfRBAC:
      secretRef:
        name: rbac
`
				assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ac), "creating wolfRBAC ApisixConsumer")

				// Wait until the ApisixConsumer create event was delivered.
				time.Sleep(6 * time.Second)

				grs, err := s.ListApisixConsumers()
				assert.Nil(ginkgo.GinkgoT(), err, "listing consumer")
				assert.Len(ginkgo.GinkgoT(), grs, 1)
				assert.Len(ginkgo.GinkgoT(), grs[0].Plugins, 1)
				wolfRBAC, _ := grs[0].Plugins["wolf-rbac"].(map[string]interface{})
				assert.Equal(ginkgo.GinkgoT(), wolfRBAC, map[string]interface{}{
					"server":        wolfSvr,
					"appid":         "test-app",
					"header_prefix": "X-",
				})
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
      - /apisix/plugin/wolf-rbac/login
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
       - /*
   backends:
   - serviceName: %s
     servicePort: %d
   authentication:
     enable: true
     type: wolfRBAC
`, backendSvc, backendPorts[0])
				assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar2), "creating ApisixRoute")
				assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(2), "Checking number of routes")
				assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(2), "Checking number of upstreams")
				payload := []byte(`
{
	"appid": "test-app",
	"username": "test",
	"password": "test-123456",
	"authType": 1
}
		`)
				body := s.NewAPISIXClient().POST("/apisix/plugin/wolf-rbac/login").
					WithHeader("Content-Type", "application/json").
					WithBytes(payload).
					Expect().
					Status(http.StatusOK).
					Body().
					Contains("rbac_token").
					Raw()

				data := struct {
					Token string `json:"rbac_token"`
				}{}
				_ = json.Unmarshal([]byte(body), &data)

				_ = s.NewAPISIXClient().GET("").
					WithHeader("Host", "httpbin.org").
					WithHeader("Authorization", data.Token).
					Expect().
					Status(http.StatusOK)

				msg401 := s.NewAPISIXClient().GET("").
					WithHeader("Host", "httpbin.org").
					Expect().
					Status(http.StatusUnauthorized).
					Body().
					Raw()
				assert.Contains(ginkgo.GinkgoT(), msg401, "Missing rbac token in request")
			})
		})
	}

	ginkgo.Describe("suite-plugins-authentication: scaffold v2beta3", func() {
		suites(scaffold.NewDefaultScaffold)
	})
	ginkgo.Describe("suite-plugins-authentication: scaffold v2", func() {
		suites(scaffold.NewDefaultV2Scaffold)
	})
})
