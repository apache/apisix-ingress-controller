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
	"os/exec"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-plugins-authentication: ApisixConsumer with ldap", func() {
	suites := func(scaffoldFunc func() *scaffold.Scaffold) {
		s := scaffoldFunc()
		getLDAPServerURL := func() (string, error) {
			cmd := exec.Command("sh", "testdata/ldap/cmd.sh", "ip")
			ip, err := cmd.Output()
			if err != nil {
				return "", err
			}
			if len(ip) == 0 {
				return "", fmt.Errorf("ldap-server start failed")
			}
			return fmt.Sprintf("%s:1389", string(ip)), nil
		}

		ginkgo.It("ApisixRoute with ldapAuth consumer", func() {
			ac := `
apiVersion: apisix.apache.org/v2
kind: ApisixConsumer
metadata:
  name: jack
spec:
  authParameter:
    ldapAuth:
      value:
        user_dn: "cn=jack,ou=users,dc=ldap,dc=example,dc=org"
`
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ac), "creating ldapAuth ApisixConsumer")

			// Wait until the ApisixConsumer create event was delivered.
			time.Sleep(6 * time.Second)

			grs, err := s.ListApisixConsumers()
			assert.Nil(ginkgo.GinkgoT(), err, "listing consumer")
			assert.Len(ginkgo.GinkgoT(), grs, 1)
			assert.Len(ginkgo.GinkgoT(), grs[0].Plugins, 1)
			ldapAuth, _ := grs[0].Plugins["ldap-auth"].(map[string]interface{})
			assert.Equal(ginkgo.GinkgoT(), ldapAuth["user_dn"], "cn=jack,ou=users,dc=ldap,dc=example,dc=org")

			ldapSvr, err := getLDAPServerURL()
			assert.Nil(ginkgo.GinkgoT(), err, "check ldap server")
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
    backends:
    - serviceName: %s
      servicePort: %d
    authentication:
      enable: true
      type: ldapAuth
      ldapAuth: 
        ldap_uri: %s
        base_dn: "ou=users,dc=ldap,dc=example,dc=org"
        use_tls: false
        uid: "cn"
`, backendSvc, backendPorts[0], ldapSvr)
			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ar), "Creating ApisixRoute with ldapAuth")
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "Checking number of routes")
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "Checking number of upstreams")

			msg401CourseMissing := s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				Expect().
				Status(http.StatusUnauthorized).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg401CourseMissing, "Missing authorization in request")

			msg401CouseInvalid := s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithBasicAuth("jack", "invalid").
				Expect().
				Status(http.StatusUnauthorized).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg401CouseInvalid, "Invalid user authorization")

			_ = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithBasicAuth("jack", "jackPassword").
				Expect().
				Status(http.StatusOK)
		})

		ginkgo.It("ApisixRoute with ldapAuth consumer using secret", func() {
			secret := `
apiVersion: v1
kind: Secret
metadata:
  name: ldap
data:
  user_dn: Y249amFjayxvdT11c2VycyxkYz1sZGFwLGRjPWV4YW1wbGUsZGM9b3Jn
`
			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(secret), "creating ldapAuth secret for ApisixConsumer")

			ac := `
apiVersion: apisix.apache.org/v2
kind: ApisixConsumer
metadata:
  name: jack
spec:
  authParameter:
    ldapAuth:
      secretRef:
        name: ldap
`
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ac), "creating ldapAuth ApisixConsumer")

			// Wait until the ApisixConsumer create event was delivered.
			time.Sleep(6 * time.Second)

			grs, err := s.ListApisixConsumers()
			assert.Nil(ginkgo.GinkgoT(), err, "listing consumer")
			assert.Len(ginkgo.GinkgoT(), grs, 1)
			assert.Len(ginkgo.GinkgoT(), grs[0].Plugins, 1)
			ldapAuth, _ := grs[0].Plugins["ldap-auth"].(map[string]interface{})
			assert.Equal(ginkgo.GinkgoT(), ldapAuth["user_dn"], "cn=jack,ou=users,dc=ldap,dc=example,dc=org")

			ldapSvr, err := getLDAPServerURL()
			assert.Nil(ginkgo.GinkgoT(), err, "check ldap server")
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
    backends:
    - serviceName: %s
      servicePort: %d
    authentication:
      enable: true
      type: ldapAuth
      ldapAuth: 
        ldap_uri: %s
        base_dn: "ou=users,dc=ldap,dc=example,dc=org"
        use_tls: false
        uid: "cn"
`, backendSvc, backendPorts[0], ldapSvr)
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar), "Creating ApisixRoute with ldapAuth")
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "Checking number of routes")
			assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "Checking number of upstreams")

			msg401CouseMissing := s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				Expect().
				Status(http.StatusUnauthorized).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg401CouseMissing, "Missing authorization in request")

			msg401CourseInvalid := s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithBasicAuth("jack", "invalid").
				Expect().
				Status(http.StatusUnauthorized).
				Body().
				Raw()
			assert.Contains(ginkgo.GinkgoT(), msg401CourseInvalid, "Invalid user authorization")

			_ = s.NewAPISIXClient().GET("/ip").
				WithHeader("Host", "httpbin.org").
				WithBasicAuth("jack", "jackPassword").
				Expect().
				Status(http.StatusOK)
		})
	}

	ginkgo.Describe("suite-plugins-authentication: scaffold v2", func() {
		suites(scaffold.NewDefaultV2Scaffold)
	})
})
