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
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

func CreateBasicAuthConsumer(name, username, password string) string {
	ac := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2beta3
kind: ApisixConsumer
metadata:
  name: %s
spec:
  authParameter:
    basicAuth:
      value:
        username: %s
        password: %s
`, name, username, password)
	return ac
}

func Authentication(basicAuthInfo interface{}) string {
	basicAuth, ok := basicAuthInfo.(map[string]interface{})
	if !ok {
		return ""
	}
	username, _ := basicAuth["username"].(string)
	password, _ := basicAuth["password"].(string)
	str := username + ":" + password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(str))
}

//consumer-restriction plugin testing
var _ = ginkgo.Describe("suite-plugins: consumer-restriction plugin", func() {
	opts := &scaffold.Options{
		Name:                  "default",
		Kubeconfig:            scaffold.GetKubeconfig(),
		APISIXConfigPath:      "testdata/apisix-gw-config.yaml",
		IngressAPISIXReplicas: 1,
		HTTPBinServicePort:    80,
		APISIXRouteVersion:    "apisix.apache.org/v2beta3",
	}
	s := scaffold.NewScaffold(opts)

	ginkgo.It("restrict consumer_name", func() {
		ac1 := CreateBasicAuthConsumer("jack1", "jack1-username", "jack1-password")
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ac1), "creating basicAuth ApisixConsumer")

		ac2 := CreateBasicAuthConsumer("jack2", "jack2-username", "jack2-password")
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ac2), "creating basicAuth ApisixConsumer")

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
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar), "creating ApisixRoute with basicAuth")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "Checking number of routes")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "Checking number of upstreams")

		_ = s.NewAPISIXClient().GET("/anything").
			WithHeader("Host", "httpbin.org").
			WithHeader("Authorization", Authentication(basicAuth)).
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
			WithHeader("Authorization", Authentication(basicAuth2)).
			Expect().
			Status(http.StatusForbidden).
			Body().
			Raw()

		assert.Contains(ginkgo.GinkgoT(), msg403, "The consumer_name is forbidden")
	})

	ginkgo.It("restrict allowed_by_methods", func() {
		ac1 := CreateBasicAuthConsumer("jack1", "jack1-username", "jack1-password")
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ac1), "creating basicAuth ApisixConsumer")

		ac2 := CreateBasicAuthConsumer("jack2", "jack2-username", "jack2-password")
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ac2), "creating basicAuth ApisixConsumer")

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
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar), "creating ApisixRoute with basicAuth")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "Checking number of routes")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "Checking number of upstreams")

		_ = s.NewAPISIXClient().GET("/anything").
			WithHeader("Host", "httpbin.org").
			WithHeader("Authorization", Authentication(basicAuth)).
			WithHeader("Content-type", "application/x-www-form-urlencoded").
			Expect().
			Status(http.StatusOK)

		_ = s.NewAPISIXClient().POST("/anything").
			WithHeader("Host", "httpbin.org").
			WithHeader("Authorization", Authentication(basicAuth)).
			Expect().
			Status(http.StatusOK)

		_ = s.NewAPISIXClient().GET("/anything").
			WithHeader("Host", "httpbin.org").
			WithHeader("Authorization", Authentication(basicAuth2)).
			Expect().
			Status(http.StatusOK)

		msg403 := s.NewAPISIXClient().POST("/anything").
			WithHeader("Host", "httpbin.org").
			WithHeader("Authorization", Authentication(basicAuth2)).
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
})
