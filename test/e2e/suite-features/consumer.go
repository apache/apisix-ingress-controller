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
	"time"

	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-features: ApisixConsumer", func() {
	s := scaffold.NewDefaultV2Scaffold()

	ginkgo.It("ApisixRoute with basicAuth consumer", func() {
		ac := `
apiVersion: apisix.apache.org/v2beta3
kind: ApisixConsumer
metadata:
  name: basicvalue
spec:
  authParameter:
    basicAuth:
      value:
        username: foo
        password: bar
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ac), "creating basicAuth ApisixConsumer")

		// Wait until the ApisixConsumer create event was delivered.
		time.Sleep(6 * time.Second)

		grs, err := s.ListApisixConsumers()
		assert.Nil(ginkgo.GinkgoT(), err, "listing consumer")
		assert.Len(ginkgo.GinkgoT(), grs, 1)
		assert.Len(ginkgo.GinkgoT(), grs[0].Plugins, 1)
		basicAuth, _ := grs[0].Plugins["basic-auth"]
		assert.Equal(ginkgo.GinkgoT(), basicAuth, map[string]interface{}{
			"username": "foo",
			"password": "bar",
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
     type: basicAuth
`, backendSvc, backendPorts[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar), "creating ApisixRoute with basicAuth")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "Checking number of routes")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "Checking number of upstreams")

		_ = s.NewAPISIXClient().GET("/ip").
			WithHeader("Host", "httpbin.org").
			WithHeader("X-Foo", "bar").
			WithHeader("Authorization", "Basic Zm9vOmJhcg==").
			Expect().
			Status(http.StatusOK)

		msg := s.NewAPISIXClient().GET("/ip").
			WithHeader("Host", "httpbin.org").
			WithHeader("X-Foo", "bar").
			Expect().
			Status(http.StatusUnauthorized).
			Body().
			Raw()
		assert.Contains(ginkgo.GinkgoT(), msg, "Missing authorization in request")

		msg = s.NewAPISIXClient().GET("/ip").
			WithHeader("Host", "httpbin.org").
			WithHeader("X-Foo", "baz").
			WithHeader("Authorization", "Basic Zm9vOmJhcg==").
			Expect().
			Status(http.StatusNotFound).
			Body().
			Raw()
		assert.Contains(ginkgo.GinkgoT(), msg, "404 Route Not Found")
	})

	ginkgo.It("ApisixRoute with basicAuth consumer using secret", func() {
		secret := `
apiVersion: v1
kind: Secret
metadata:
  name: basic
data:
  password: YmFy
  username: Zm9v
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(secret), "creating basic secret for ApisixConsumer")

		ac := `
apiVersion: apisix.apache.org/v2beta3
kind: ApisixConsumer
metadata:
  name: basicvalue
spec:
  authParameter:
    basicAuth:
      secretRef:
        name: basic
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ac), "creating basicAuth ApisixConsumer")

		// Wait until the ApisixConsumer create event was delivered.
		time.Sleep(6 * time.Second)

		grs, err := s.ListApisixConsumers()
		assert.Nil(ginkgo.GinkgoT(), err, "listing consumer")
		assert.Len(ginkgo.GinkgoT(), grs, 1)
		assert.Len(ginkgo.GinkgoT(), grs[0].Plugins, 1)
		basicAuth, _ := grs[0].Plugins["basic-auth"]
		assert.Equal(ginkgo.GinkgoT(), basicAuth, map[string]interface{}{
			"username": "foo",
			"password": "bar",
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
     type: basicAuth
`, backendSvc, backendPorts[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar), "creating ApisixRoute with basicAuth")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "Checking number of routes")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "Checking number of upstreams")

		_ = s.NewAPISIXClient().GET("/ip").
			WithHeader("Host", "httpbin.org").
			WithHeader("X-Foo", "bar").
			WithHeader("Authorization", "Basic Zm9vOmJhcg==").
			Expect().
			Status(http.StatusOK)

		msg := s.NewAPISIXClient().GET("/ip").
			WithHeader("Host", "httpbin.org").
			WithHeader("X-Foo", "bar").
			Expect().
			Status(http.StatusUnauthorized).
			Body().
			Raw()
		assert.Contains(ginkgo.GinkgoT(), msg, "Missing authorization in request")

		msg = s.NewAPISIXClient().GET("/ip").
			WithHeader("Host", "httpbin.org").
			WithHeader("X-Foo", "baz").
			WithHeader("Authorization", "Basic Zm9vOmJhcg==").
			Expect().
			Status(http.StatusNotFound).
			Body().
			Raw()
		assert.Contains(ginkgo.GinkgoT(), msg, "404 Route Not Found")
	})

	ginkgo.It("ApisixRoute with keyAuth consumer", func() {
		ac := `
apiVersion: apisix.apache.org/v2beta3
kind: ApisixConsumer
metadata:
  name: keyvalue
spec:
  authParameter:
    keyAuth:
      value:
        key: foo
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ac), "creating keyAuth ApisixConsumer")

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
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar), "creating ApisixRoute with keyAuth")
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

		ac := `
apiVersion: apisix.apache.org/v2beta3
kind: ApisixConsumer
metadata:
  name: keyvalue
spec:
  authParameter:
    keyAuth:
      secretRef:
        name: keyauth
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ac), "creating keyAuth ApisixConsumer")

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
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar), "creating ApisixRoute with keyAuth")
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

	ginkgo.It("ApisixRoute with wolfRbac consumer", func() {
		wolfSvr, err := s.WolfRbacSvrStartedURL()
		assert.Nil(ginkgo.GinkgoT(), err, "checking wolf-server")
		defer s.StopWolfRbacSvr()

		ac := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2beta3
kind: ApisixConsumer
metadata:
  name: wolf-user
spec:
  authParameter:
    wolfRbac:
      value:
        server: "%s"
        appid: "test-app"
        header_prefix: "X-"
`, wolfSvr)
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ac), "creating wolfRbac ApisixConsumer")

		// Wait until the ApisixConsumer create event was delivered.
		time.Sleep(6 * time.Second)

		grs, err := s.ListApisixConsumers()
		assert.Nil(ginkgo.GinkgoT(), err, "listing consumer")
		assert.Len(ginkgo.GinkgoT(), grs, 1)
		assert.Len(ginkgo.GinkgoT(), grs[0].Plugins, 1)
		wolfRbac, _ := grs[0].Plugins["wolf-rbac"].(map[string]interface{})
		assert.Equal(ginkgo.GinkgoT(), wolfRbac, map[string]interface{}{
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
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar1), "creating ApisixRoute")
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
     type: wolfRbac
`, backendSvc, backendPorts[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar2), "creating ApisixRoute")
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

	ginkgo.It("ApisixRoute with wolfRbac consumer using secret", func() {
		wolfSvr, err := s.WolfRbacSvrStartedURL()
		assert.Nil(ginkgo.GinkgoT(), err, "checking wolf-server")
		defer s.StopWolfRbacSvr()

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
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(secret), "creating wolfRbac secret for ApisixConsumer")

		ac := `
apiVersion: apisix.apache.org/v2beta3
kind: ApisixConsumer
metadata:
  name: wolf-user
spec:
  authParameter:
    wolfRbac:
      secretRef:
        name: rbac
`
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ac), "creating wolfRbac ApisixConsumer")

		// Wait until the ApisixConsumer create event was delivered.
		time.Sleep(6 * time.Second)

		grs, err := s.ListApisixConsumers()
		assert.Nil(ginkgo.GinkgoT(), err, "listing consumer")
		assert.Len(ginkgo.GinkgoT(), grs, 1)
		assert.Len(ginkgo.GinkgoT(), grs[0].Plugins, 1)
		wolfRbac, _ := grs[0].Plugins["wolf-rbac"].(map[string]interface{})
		assert.Equal(ginkgo.GinkgoT(), wolfRbac, map[string]interface{}{
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
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar1), "creating ApisixRoute")
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
     type: wolfRbac
`, backendSvc, backendPorts[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar2), "creating ApisixRoute")
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

	ginkgo.It("ApisixRoute without authentication", func() {
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
     enable: false
`, backendSvc, backendPorts[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar), "creating ApisixRoute without authentication")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "Checking number of routes")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "Checking number of upstreams")

		_ = s.NewAPISIXClient().GET("/ip").
			WithHeader("Host", "httpbin.org").
			WithHeader("X-Foo", "bar").
			Expect().
			Status(http.StatusOK)
	})
})
