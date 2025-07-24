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

var _ = Describe("Test ApisixConsumer", Label("apisix.apache.org", "v2", "apisixconsumer"), func() {
	var (
		s = scaffold.NewScaffold(&scaffold.Options{
			ControllerName: "apisix.apache.org/apisix-ingress-controller",
		})
		applier = framework.NewApplier(s.GinkgoT, s.K8sClient, s.CreateResourceFromString)
	)

	BeforeEach(func() {
		By("create GatewayProxy")
		gatewayProxy := fmt.Sprintf(gatewayProxyYaml, s.Deployer.GetAdminEndpoint(), s.AdminKey())
		err := s.CreateResourceFromStringWithNamespace(gatewayProxy, "default")
		Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
		time.Sleep(5 * time.Second)

		By("create IngressClass")
		err = s.CreateResourceFromStringWithNamespace(ingressClassYaml, "")
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
  ingressClassName: apisix
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
  ingressClassName: apisix
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
  ingressClassName: apisix
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
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"}, &apiv2.ApisixRoute{}, defaultApisixRoute)

			By("apply ApisixConsumer")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "test-consumer"}, &apiv2.ApisixConsumer{}, keyAuth)

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
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"}, &apiv2.ApisixRoute{}, defaultApisixRoute)

			By("apply Secret")
			err := s.CreateResourceFromString(secret)
			Expect(err).ShouldNot(HaveOccurred(), "creating Secret for ApisixConsumer")

			By("apply ApisixConsumer")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "test-consumer"}, &apiv2.ApisixConsumer{}, keyAuthWiwhSecret)

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
  ingressClassName: apisix
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
  ingressClassName: apisix
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
  ingressClassName: apisix
  authParameter:
    basicAuth:
      secretRef:
        name: basic
`
		)

		It("Basic tests", func() {
			By("apply ApisixRoute")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"}, &apiv2.ApisixRoute{}, defaultApisixRoute)

			By("apply ApisixConsumer")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "test-consumer"}, &apiv2.ApisixConsumer{}, basicAuth)

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
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"}, &apiv2.ApisixRoute{}, defaultApisixRoute)

			By("apply Secret")
			err := s.CreateResourceFromString(secret)
			Expect(err).ShouldNot(HaveOccurred(), "creating Secret for ApisixConsumer")

			By("apply ApisixConsumer")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "test-consumer"}, &apiv2.ApisixConsumer{}, basicAuthWithSecret)

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
})
