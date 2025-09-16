// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package v2

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"

	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

type Headers map[string]string

func generateHMACHeaders(keyID, secretKey, method, path string) map[string]string {
	gmtTime := time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")
	signingString := fmt.Sprintf("%s\n%s %s\ndate: %s\n", keyID, method, path, gmtTime)

	// Create HMAC signature
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(signingString))
	signature := mac.Sum(nil)
	signatureBase64 := base64.StdEncoding.EncodeToString(signature)

	// Construct Authorization header
	authHeader := fmt.Sprintf(
		`Signature keyId="%s",algorithm="hmac-sha256",headers="@request-target date",signature="%s"`,
		keyID, signatureBase64,
	)

	return map[string]string{
		"Date":          gmtTime,
		"Authorization": authHeader,
	}
}

var _ = Describe("Test ApisixConsumer", Label("apisix.apache.org", "v2", "apisixconsumer"), func() {
	var (
		s       = scaffold.NewDefaultScaffold()
		applier = framework.NewApplier(s.GinkgoT, s.K8sClient, s.CreateResourceFromString)
	)

	BeforeEach(func() {
		By("create GatewayProxy")
		err := s.CreateResourceFromString(s.GetGatewayProxySpec())
		Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
		time.Sleep(5 * time.Second)

		By("create IngressClass")
		err = s.CreateResourceFromStringWithNamespace(s.GetIngressClassYaml(), "")
		Expect(err).NotTo(HaveOccurred(), "creating IngressClass")
		time.Sleep(5 * time.Second)
	})

	Context("Test KeyAuth", func() {
		const (
			keyAuth = `
apiVersion: apisix.apache.org/v2
kind: ApisixConsumer
metadata:
  name: test-consumer
spec:
  ingressClassName: %s
  authParameter:
    keyAuth:
      value:
        key: test-key
`
			defaultApisixRoute = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: default
spec:
  ingressClassName: %s
  http:
  - name: rule0
    match:
      hosts:
      - httpbin
      paths:
      - /get
      - /headers
      - /anything
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
    authentication:
      enable: true
      type: keyAuth
`
			secret = `
apiVersion: v1
kind: Secret
metadata:
  name: keyauth
data:
  # foo-key
  key: Zm9vLWtleQ==
`
			secretUpdated = `
apiVersion: v1
kind: Secret
metadata:
  name: keyauth
data:
  # foo2-key
  key: Zm9vMi1rZXk=
`
			keyAuthWiwhSecret = `
apiVersion: apisix.apache.org/v2
kind: ApisixConsumer
metadata:
  name: test-consumer
spec:
  ingressClassName: %s
  authParameter:
    keyAuth:
      secretRef:
        name: keyauth
`
		)
		request := func(path string, headers Headers) int {
			return s.NewAPISIXClient().GET(path).WithHeaders(headers).WithHost("httpbin").Expect().Raw().StatusCode
		}

		It("Basic tests", func() {
			By("apply ApisixRoute")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"}, &apiv2.ApisixRoute{}, fmt.Sprintf(defaultApisixRoute, s.Namespace()))

			By("apply ApisixConsumer")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "test-consumer"}, &apiv2.ApisixConsumer{}, fmt.Sprintf(keyAuth, s.Namespace()))

			By("verify ApisixRoute with ApisixConsumer")
			Eventually(request).WithArguments("/get", Headers{
				"apikey": "invalid-key",
			}).WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusUnauthorized))

			Eventually(request).WithArguments("/get", Headers{
				"apikey": "test-key",
			}).WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))

			By("Delete ApisixConsumer")
			err := s.DeleteResource("ApisixConsumer", "test-consumer")
			Expect(err).ShouldNot(HaveOccurred(), "deleting ApisixConsumer")
			Eventually(request).WithArguments("/get", Headers{
				"apikey": "test-key",
			}).WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusUnauthorized))

			By("delete ApisixRoute")
			err = s.DeleteResource("ApisixRoute", "default")
			Expect(err).ShouldNot(HaveOccurred(), "deleting ApisixRoute")
			Eventually(request).WithArguments("/headers", Headers{}).WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusNotFound))
		})

		It("SecretRef tests", func() {
			By("apply ApisixRoute")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"}, &apiv2.ApisixRoute{},
				fmt.Sprintf(defaultApisixRoute, s.Namespace()))

			By("apply Secret")
			err := s.CreateResourceFromString(secret)
			Expect(err).ShouldNot(HaveOccurred(), "creating Secret for ApisixConsumer")

			By("apply ApisixConsumer")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "test-consumer"}, &apiv2.ApisixConsumer{},
				fmt.Sprintf(keyAuthWiwhSecret, s.Namespace()))

			By("verify ApisixRoute with ApisixConsumer")
			Eventually(request).WithArguments("/get", Headers{
				"apikey": "invalid-key",
			}).WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusUnauthorized))

			Eventually(request).WithArguments("/get", Headers{
				"apikey": "foo-key",
			}).WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))

			By("update Secret")
			err = s.CreateResourceFromString(secretUpdated)
			Expect(err).ShouldNot(HaveOccurred(), "updating Secret for ApisixConsumer")

			Eventually(request).WithArguments("/get", Headers{
				"apikey": "foo-key",
			}).WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusUnauthorized))

			Eventually(request).WithArguments("/get", Headers{
				"apikey": "foo2-key",
			}).WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))

			By("Delete ApisixConsumer")
			err = s.DeleteResource("ApisixConsumer", "test-consumer")
			Expect(err).ShouldNot(HaveOccurred(), "deleting ApisixConsumer")
			Eventually(request).WithArguments("/get", Headers{
				"apikey": "test-key",
			}).WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusUnauthorized))

			By("delete ApisixRoute")
			err = s.DeleteResource("ApisixRoute", "default")
			Expect(err).ShouldNot(HaveOccurred(), "deleting ApisixRoute")
			Eventually(request).WithArguments("/headers", Headers{}).WithTimeout(5 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusNotFound))
		})
	})

	Context("Test BasicAuth", func() {
		const (
			basicAuth = `
apiVersion: apisix.apache.org/v2
kind: ApisixConsumer
metadata:
  name: test-consumer
spec:
  ingressClassName: %s
  authParameter:
    basicAuth:
      value:
        username: test-user
        password: test-password
`
			defaultApisixRoute = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: default
spec:
  ingressClassName: %s
  http:
  - name: rule0
    match:
      hosts:
      - httpbin
      paths:
      - /get
      - /headers
      - /anything
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
    authentication:
      enable: true
      type: basicAuth
`

			secret = `
apiVersion: v1
kind: Secret
metadata:
  name: basic
data:
  # foo:bar
  username: Zm9v
  password: YmFy
`
			secretUpdated = `
apiVersion: v1
kind: Secret
metadata:
  name: basic
data:
  # foo-new-user:bar-new-password
  username: Zm9vLW5ldy11c2Vy
  password: YmFyLW5ldy1wYXNzd29yZA==
`

			basicAuthWithSecret = `
apiVersion: apisix.apache.org/v2
kind: ApisixConsumer
metadata:
  name: test-consumer
spec:
  ingressClassName: %s
  authParameter:
    basicAuth:
      secretRef:
        name: basic
`
		)

		It("Basic tests", func() {
			By("apply ApisixRoute")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"}, &apiv2.ApisixRoute{}, fmt.Sprintf(defaultApisixRoute, s.Namespace()))

			By("apply ApisixConsumer")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "test-consumer"}, &apiv2.ApisixConsumer{}, fmt.Sprintf(basicAuth, s.Namespace()))

			By("verify ApisixRoute with ApisixConsumer")
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin",
				BasicAuth: &scaffold.BasicAuth{
					Username: "invalid-username",
					Password: "invalid-password",
				},
				Check: scaffold.WithExpectedStatus(http.StatusUnauthorized),
			})
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin",
				BasicAuth: &scaffold.BasicAuth{
					Username: "test-user",
					Password: "test-password",
				},
				Check: scaffold.WithExpectedStatus(http.StatusOK),
			})

			By("Delete ApisixConsumer")
			err := s.DeleteResource("ApisixConsumer", "test-consumer")
			Expect(err).ShouldNot(HaveOccurred(), "deleting ApisixConsumer")
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin",
				BasicAuth: &scaffold.BasicAuth{
					Username: "test-user",
					Password: "test-password",
				},
				Check: scaffold.WithExpectedStatus(http.StatusUnauthorized),
			})

			By("delete ApisixRoute")
			err = s.DeleteResource("ApisixRoute", "default")
			Expect(err).ShouldNot(HaveOccurred(), "deleting ApisixRoute")
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin",
				Check:  scaffold.WithExpectedStatus(http.StatusNotFound),
			})
		})

		It("SecretRef tests", func() {
			By("apply ApisixRoute")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"}, &apiv2.ApisixRoute{},
				fmt.Sprintf(defaultApisixRoute, s.Namespace()))

			By("apply Secret")
			err := s.CreateResourceFromString(secret)
			Expect(err).ShouldNot(HaveOccurred(), "creating Secret for ApisixConsumer")

			By("apply ApisixConsumer")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "test-consumer"},
				&apiv2.ApisixConsumer{}, fmt.Sprintf(basicAuthWithSecret, s.Namespace()))

			By("verify ApisixRoute with ApisixConsumer")
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin",
				Check:  scaffold.WithExpectedStatus(http.StatusUnauthorized),
			})
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin",
				BasicAuth: &scaffold.BasicAuth{
					Username: "foo",
					Password: "bar",
				},
				Check: scaffold.WithExpectedStatus(http.StatusOK),
			})

			By("update Secret")
			err = s.CreateResourceFromString(secretUpdated)
			Expect(err).ShouldNot(HaveOccurred(), "updating Secret for ApisixConsumer")
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin",
				BasicAuth: &scaffold.BasicAuth{
					Username: "foo",
					Password: "bar",
				},
				Check: scaffold.WithExpectedStatus(http.StatusUnauthorized),
			})
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin",
				BasicAuth: &scaffold.BasicAuth{
					Username: "foo-new-user",
					Password: "bar-new-password",
				},
				Check: scaffold.WithExpectedStatus(http.StatusOK),
			})

			By("Delete ApisixConsumer")
			err = s.DeleteResource("ApisixConsumer", "test-consumer")
			Expect(err).ShouldNot(HaveOccurred(), "deleting ApisixConsumer")
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin",
				BasicAuth: &scaffold.BasicAuth{
					Username: "foo-new-user",
					Password: "bar-new-password",
				},
				Check: scaffold.WithExpectedStatus(http.StatusUnauthorized),
			})

			By("delete ApisixRoute")
			err = s.DeleteResource("ApisixRoute", "default")
			Expect(err).ShouldNot(HaveOccurred(), "deleting ApisixRoute")
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin",
				Check:  scaffold.WithExpectedStatus(http.StatusNotFound),
			})
		})
	})

	Context("Test HMACAuth", func() {
		const (
			hmacAuthConsumer = `
apiVersion: apisix.apache.org/v2
kind: ApisixConsumer
metadata:
  name: hmac-consumer
spec:
  ingressClassName: %s
  authParameter:
    hmacAuth:
      value:
        key_id: papa
        secret_key: fatpa
`

			hmacAuthConsumerInvalid = `
apiVersion: apisix.apache.org/v2
kind: ApisixConsumer
metadata:
  name: hmac-consumer
spec:
  ingressClassName: %s
  authParameter:
    hmacAuth:
      value:
        secret_key: fatpa
`
			hmacRoute = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: hmac-route
spec:
  ingressClassName: %s
  http:
  - name: rule1
    match:
      hosts:
      - httpbin.org
      paths:
        - /ip
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
    authentication:
      enable: true
      type: hmacAuth
`
			hmacSecret = `
apiVersion: v1
kind: Secret
metadata:
  name: hmac
data:
  key_id: cGFwYQ==
  secret_key: ZmF0cGE=
`
			hmacAuthWithSecret = `
apiVersion: apisix.apache.org/v2
kind: ApisixConsumer
metadata:
  name: hmac-consumer
spec:
  ingressClassName: %s
  authParameter:
    hmacAuth:
      secretRef:
        name: hmac
`
		)

		It("Basic tests", func() {
			By("apply ApisixRoute")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "hmac-route"},
				&apiv2.ApisixRoute{}, fmt.Sprintf(hmacRoute, s.Namespace()))

			By("apply Invalid ApisixConsumer with missing required field")
			err := s.CreateResourceFromString(hmacAuthConsumerInvalid)
			Expect(err).Should(HaveOccurred(), "creating invalid ApisixConsumer")

			By("apply ApisixConsumer")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "hmac-consumer"},
				&apiv2.ApisixConsumer{}, fmt.Sprintf(hmacAuthConsumer, s.Namespace()))

			By("verify ApisixRoute with ApisixConsumer")
			// Generate HMAC headers dynamically
			hmacHeaders := generateHMACHeaders("papa", "fatpa", "GET", "/ip")

			// Test valid HMAC authentication
			s.RequestAssert(&scaffold.RequestAssert{
				Method:  "GET",
				Path:    "/ip",
				Host:    "httpbin.org",
				Headers: hmacHeaders,
				Check:   scaffold.WithExpectedStatus(http.StatusOK),
			})

			// Test missing authorization
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/ip",
				Host:   "httpbin.org",
				Check:  scaffold.WithExpectedStatus(http.StatusUnauthorized),
			})

			By("Delete resources")
			err = s.DeleteResource("ApisixConsumer", "hmac-consumer")
			Expect(err).ShouldNot(HaveOccurred(), "deleting ApisixConsumer")

			err = s.DeleteResource("ApisixRoute", "hmac-route")
			Expect(err).ShouldNot(HaveOccurred(), "deleting ApisixRoute")
		})

		It("SecretRef tests", func() {
			By("apply ApisixRoute")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "hmac-route"},
				&apiv2.ApisixRoute{}, fmt.Sprintf(hmacRoute, s.Namespace()))

			By("apply Secret")
			err := s.CreateResourceFromString(hmacSecret)
			Expect(err).ShouldNot(HaveOccurred(), "creating Secret for ApisixConsumer")

			By("apply ApisixConsumer")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "hmac-consumer"},
				&apiv2.ApisixConsumer{}, fmt.Sprintf(hmacAuthWithSecret, s.Namespace()))

			By("verify ApisixRoute with ApisixConsumer")
			// Generate HMAC headers dynamically
			hmacHeaders := generateHMACHeaders("papa", "fatpa", "GET", "/ip")

			// Test valid HMAC authentication
			s.RequestAssert(&scaffold.RequestAssert{
				Method:  "GET",
				Path:    "/ip",
				Host:    "httpbin.org",
				Headers: hmacHeaders,
				Check:   scaffold.WithExpectedStatus(http.StatusOK),
			})

			By("Delete resources")
			err = s.DeleteResource("ApisixConsumer", "hmac-consumer")
			Expect(err).ShouldNot(HaveOccurred(), "deleting ApisixConsumer")

			err = s.DeleteResource("ApisixRoute", "hmac-route")
			Expect(err).ShouldNot(HaveOccurred(), "deleting ApisixRoute")

			err = s.DeleteResource("Secret", "hmac")
			Expect(err).ShouldNot(HaveOccurred(), "deleting Secret")
		})
	})
})
