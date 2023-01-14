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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-plugins-authentication: ApisixConsumer with openIDConnect", func() {
	suites := func(scaffoldFunc func() *scaffold.Scaffold) {
		s := scaffoldFunc()

		ginkgo.Context("openid-connect-server", func() {
			getAccessToken := func() (string, error) {
				url := "http://127.0.0.1:8222/realms/master/protocol/openid-connect/token"
				method := "POST"

				payload := strings.NewReader("username=user&password=bitnami&grant_type=password&client_id=admin-cli")

				client := &http.Client{}
				req, err := http.NewRequest(method, url, payload)

				if err != nil {
					fmt.Println(err)
					return "", err
				}
				req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

				res, err := client.Do(req)
				if err != nil {
					fmt.Println(err)
					return "", err
				}
				defer res.Body.Close()
				body, err := ioutil.ReadAll(res.Body)
				if err != nil {
					fmt.Println(err)
					return "", err
				}
				type Token struct {
					AccessToken string `json:"access_token"`
				}
				tokenStruct := &Token{}

				err = json.Unmarshal(body, tokenStruct)
				if err != nil {
					return "", err
				}

				return tokenStruct.AccessToken, nil
			}
			getSecret := func() (string, error) {
				accessToken, err := getAccessToken()
				if err != nil {
					return "", err
				}

				url := "http://127.0.0.1:8222/admin/realms/apisix-realm/clients?clientId=apisix"
				method := "GET"

				client := &http.Client{}
				req, err := http.NewRequest(method, url, nil)

				if err != nil {
					fmt.Println(err)
					return "", err
				}
				req.Header.Add("clientId", "apisix")
				req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))

				res, err := client.Do(req)
				if err != nil {
					fmt.Println(err)
					return "", err
				}
				defer res.Body.Close()

				body, err := ioutil.ReadAll(res.Body)
				if err != nil {
					fmt.Println(err)
					return "", err
				}

				type ClientInfos []struct {
					ID       string `json:"id"`
					ClientID string `json:"clientId"`
					Secret   string `json:"secret"`
					Protocol string `json:"protocol"`
				}
				var clientInfoStructs ClientInfos

				err = json.Unmarshal(body, &clientInfoStructs)
				if err != nil {
					return "", err
				}
				if len(clientInfoStructs) <= 0 {
					return "", fmt.Errorf("client info got nul array form keycloak")
				}

				return clientInfoStructs[0].Secret, nil
			}

			ginkgo.It("ApisixRoute with openidConnect consumer", func() {
				keycloakSvr := "http://127.0.0.1"
				accessToken, err := getAccessToken()
				assert.Nil(ginkgo.GinkgoT(), err, "checking openid connect access token")
				clientsSecret, err := getSecret()
				assert.Nil(ginkgo.GinkgoT(), err, "checking openid connect client secret")

				ac := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
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
        introspection_endpoint: %s/realms/apisix-realm/protocol/openid-connect/token/intros`, clientsSecret, keycloakSvr, keycloakSvr)
				assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ac), "creating openIDConnect ApisixConsumer")

				//Wait until the ApisixConsumer create event was delivered.
				time.Sleep(30 * time.Second)
				grs, err := s.ListApisixConsumers()
				assert.Nil(ginkgo.GinkgoT(), err, "listing consumer")
				assert.Len(ginkgo.GinkgoT(), grs, 1)
				assert.Len(ginkgo.GinkgoT(), grs[0].Plugins, 1)
				openidConnect, _ := grs[0].Plugins["openid-connect"].(map[string]interface{})
				assert.Equal(ginkgo.GinkgoT(), "apisix", openidConnect["client_id"])
				assert.Equal(ginkgo.GinkgoT(), clientsSecret, openidConnect["client_secret"])
				assert.Equal(ginkgo.GinkgoT(), int64(3), openidConnect["timeout"])

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
      type: openidConnect`, backendSvc, backendPorts[0])
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
		})
	}

	ginkgo.Describe("suite-plugins-authentication: scaffold v2", func() {
		suites(scaffold.NewDefaultV2Scaffold)
	})
})
