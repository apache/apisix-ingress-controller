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
	"encoding/base64"
	"fmt"
	"net/http"
	"os/exec"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-plugins-authentication: ApisixConsumer with openIDConnect", func() {
	suites := func(scaffoldFunc func() *scaffold.Scaffold) {
		s := scaffoldFunc()
		getOpenIDConnectServerURL := func() (string, error) {
			cmd := exec.Command("sh", "testdata/openid-connect/cmd.sh", "ip")
			ip, err := cmd.Output()
			if err != nil {
				return "", err
			}
			if len(ip) == 0 {
				return "", fmt.Errorf("keycloak start failed")
			}
			return fmt.Sprintf("http://%s:8222", string(ip)), nil
		}
		getSecret := func() (string, error) {
			cmd := exec.Command("sh", "testdata/openid-connect/cmd.sh", "secret")
			secret, err := cmd.Output()
			if err != nil {
				return "", err
			}
			if len(secret) == 0 {
				return "", fmt.Errorf("get secret failed")
			}
			return string(secret), nil
		}
		getAccessToken := func() (string, error) {
			cmd := exec.Command("sh", "testdata/openid-connect/cmd.sh", "access_token")
			secret, err := cmd.Output()
			if err != nil {
				return "", err
			}
			if len(secret) == 0 {
				return "", fmt.Errorf("get secret failed")
			}
			return string(secret), nil
		}

		ginkgo.It("ApisixRoute with openIDConnect consumer", func() {
			keycloakSvr, err := getOpenIDConnectServerURL()
			assert.Nil(ginkgo.GinkgoT(), err, "checking keycloak-server")
			clientsSecret, err := getSecret()
			assert.Nil(ginkgo.GinkgoT(), err, "checking openid connect client secret")
			accessToken, err := getAccessToken()
			assert.Nil(ginkgo.GinkgoT(), err, "checking openid connect access token")
			ac := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2beta3
kind: ApisixConsumer
metadata:
  name: openidconnectvalue
spec:
  authParameter:
    openidConnect:
	  value:
	    client_id: apisix
		client_secret: %s
		discovery: %s/realms/apisix-realm/.well-known/openid-configuration
		realm: apisix-realm
		bearer_only: true
		introspection_endpoint: %s/auth/realms/apisix-realm/protocol/openid-connect/token/introspect
`, clientsSecret, keycloakSvr, keycloakSvr)
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ac), "creating openIDConnect ApisixConsumer")

			// Wait until the ApisixConsumer create event was delivered.
			time.Sleep(6 * time.Second)

			grs, err := s.ListApisixConsumers()
			assert.Nil(ginkgo.GinkgoT(), err, "listing consumer")
			assert.Len(ginkgo.GinkgoT(), grs, 1)
			assert.Len(ginkgo.GinkgoT(), grs[0].Plugins, 1)
			openidConnect, _ := grs[0].Plugins["openid-connect"].(map[string]interface{})
			assert.Equal(ginkgo.GinkgoT(), "apisix", openidConnect["client_id"])
			assert.Equal(ginkgo.GinkgoT(), clientsSecret, openidConnect["client_secret"])
			//assert.Equal(ginkgo.GinkgoT(), "http://127.0.0.1:8222/realms/apisix-realm/.well-known/openid-configuration", openidConnect["discovery"])
			assert.Equal(ginkgo.GinkgoT(), int64(3), openidConnect["timeout"])

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
     type: openidConnect
`, backendSvc, backendPorts[0])
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar), "creating ApisixRoute with openidConnect")
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "Checking number of routes")
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "Checking number of upstreams")

			_ = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("X-Foo", "bar").
				WithHeader("Authorization", fmt.Sprintf("Bearer %s", accessToken)).
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
				WithHeader("Authorization", fmt.Sprintf("Bearer %s", accessToken)).
				WithHeader("User-Agent", "curl/7.29.0").
				Expect().
				Status(http.StatusNotFound).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg, "404 Route Not Found")
		})

		ginkgo.It("ApisixRoute with openid connect consumer using secret", func() {
			keycloakSvr, err := getOpenIDConnectServerURL()
			assert.Nil(ginkgo.GinkgoT(), err, "checking keycloak-server")
			clientsSecret, err := getSecret()
			assert.Nil(ginkgo.GinkgoT(), err, "checking openid connect client secret")
			accessToken, err := getAccessToken()
			assert.Nil(ginkgo.GinkgoT(), err, "checking openid connect access token")
			secret := fmt.Sprintf(`
		apiVersion: v1
		kind: Secret
		metadata:
		 name: opennid-connect
		data:
		 client_id: YXBpc2l4
		 client_secret: %s
		 discovery: %s
		 realm: YXBpc2l4LXJlYWxt
		 bearer_only: dHJ1ZQ==
		 introspection_endpoint: %s
		
		`, base64.StdEncoding.EncodeToString([]byte(clientsSecret)),
				base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s/realms/apisix-realm/.well-known/openid-configuration", keycloakSvr))),
				base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s/auth/realms/apisix-realm/protocol/openid-connect/token/introspect", keycloakSvr))))
			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(secret), "creating opennid-connect secret for ApisixConsumer")

			ac := `
		apiVersion: apisix.apache.org/v2beta3
		kind: ApisixConsumer
		metadata:
		 name: opennidconnectvalue
		spec:
		 authParameter:
		   opennidConnect:
		     secretRef:
		       name: opennid-connect
		`
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ac), "creating opennidConnect ApisixConsumer")

			// Wait until the ApisixConsumer create event was delivered.
			time.Sleep(6 * time.Second)

			grs, err := s.ListApisixConsumers()
			assert.Nil(ginkgo.GinkgoT(), err, "listing consumer")
			assert.Len(ginkgo.GinkgoT(), grs, 1)
			assert.Len(ginkgo.GinkgoT(), grs[0].Plugins, 1)
			openidConnect, _ := grs[0].Plugins["openid-connect"].(map[string]interface{})
			assert.Equal(ginkgo.GinkgoT(), "apisix", openidConnect["client_id"])
			assert.Equal(ginkgo.GinkgoT(), clientsSecret, openidConnect["client_secret"])
			assert.Equal(ginkgo.GinkgoT(), "apisix-realm", openidConnect["realm"])
			assert.Equal(ginkgo.GinkgoT(), int64(3), openidConnect["timeout"])

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
		    type: openidConnect
		`, backendSvc, backendPorts[0])
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar), "creating ApisixRoute with openidConnect")
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "Checking number of routes")
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "Checking number of upstreams")

			_ = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithHeader("X-Foo", "bar").
				WithHeader("Authorization", fmt.Sprintf("Bearer %s", accessToken)).
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
				WithHeader("Authorization", fmt.Sprintf("Bearer %s", accessToken)).
				WithHeader("User-Agent", "curl/7.29.0").
				Expect().
				Status(http.StatusNotFound).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg, "404 Route Not Found")
		})
	}

	ginkgo.Describe("suite-plugins-authentication: scaffold v2beta3", func() {
		suites(scaffold.NewDefaultV2beta3Scaffold)
	})
	ginkgo.Describe("suite-plugins-authentication: scaffold v2", func() {
		suites(scaffold.NewDefaultV2Scaffold)
	})
})
