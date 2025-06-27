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

package apisix

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"

	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

const gatewayProxyYamlTls = `
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: apisix-proxy-tls
  namespace: default
spec:
  provider:
    type: ControlPlane
    controlPlane:
      endpoints:
      - %s
      auth:
        type: AdminKey
        adminKey:
          value: "%s"
`

const ingressClassYamlTls = `
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: apisix-tls
spec:
  controller: "apisix.apache.org/apisix-ingress-controller"
  parameters:
    apiGroup: "apisix.apache.org"
    kind: "GatewayProxy"
    name: "apisix-proxy-tls"
    namespace: "default"
    scope: "Namespace"
`

const apisixRouteYamlTls = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: test-route-tls
spec:
  ingressClassName: apisix-tls
  http:
  - name: rule0
    match:
      paths:
      - /*
      hosts:
      - api6.com
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
`

var Cert = strings.TrimSpace(framework.TestServerCert)

var Key = strings.TrimSpace(framework.TestServerKey)

var _ = Describe("Test ApisixTls", Label("apisix.apache.org", "v2", "apisixtls"), func() {
	var (
		s = scaffold.NewScaffold(&scaffold.Options{
			ControllerName: "apisix.apache.org/apisix-ingress-controller",
		})
		applier = framework.NewApplier(s.GinkgoT, s.K8sClient, s.CreateResourceFromString)
	)

	Context("Test ApisixTls", func() {
		BeforeEach(func() {
			By("create GatewayProxy")
			gatewayProxy := fmt.Sprintf(gatewayProxyYamlTls, s.Deployer.GetAdminEndpoint(), s.AdminKey())
			err := s.CreateResourceFromStringWithNamespace(gatewayProxy, "default")
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
			time.Sleep(5 * time.Second)

			By("create IngressClass")
			err = s.CreateResourceFromStringWithNamespace(ingressClassYamlTls, "")
			Expect(err).NotTo(HaveOccurred(), "creating IngressClass")
			time.Sleep(5 * time.Second)

			By("create ApisixRoute for TLS testing")
			var apisixRoute apiv2.ApisixRoute
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "test-route-tls"}, &apisixRoute, apisixRouteYamlTls)
		})

		AfterEach(func() {
			By("delete GatewayProxy")
			gatewayProxy := fmt.Sprintf(gatewayProxyYamlTls, s.Deployer.GetAdminEndpoint(), s.AdminKey())
			err := s.DeleteResourceFromStringWithNamespace(gatewayProxy, "default")
			Expect(err).ShouldNot(HaveOccurred(), "deleting GatewayProxy")

			By("delete IngressClass")
			err = s.DeleteResourceFromStringWithNamespace(ingressClassYamlTls, "")
			Expect(err).ShouldNot(HaveOccurred(), "deleting IngressClass")
		})

		It("Basic ApisixTls test", func() {
			const host = "api6.com"

			By("create TLS secret")
			err := s.NewKubeTlsSecret("test-tls-secret", Cert, Key)
			Expect(err).NotTo(HaveOccurred(), "creating TLS secret")

			const apisixTlsSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixTls
metadata:
  name: test-tls
spec:
  ingressClassName: apisix-tls
  hosts:
  - api6.com
  secret:
    name: test-tls-secret
    namespace: %s
`

			By("apply ApisixTls")
			var apisixTls apiv2.ApisixTls
			tlsSpec := fmt.Sprintf(apisixTlsSpec, s.Namespace())
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "test-tls"}, &apisixTls, tlsSpec)

			By("verify TLS configuration in control plane")
			Eventually(func() bool {
				tls, err := s.DefaultDataplaneResource().SSL().List(context.Background())
				if err != nil {
					return false
				}
				if len(tls) != 1 {
					return false
				}
				if len(tls[0].Certificates) != 1 {
					return false
				}
				return true
			}).WithTimeout(30 * time.Second).ProbeEvery(2 * time.Second).Should(BeTrue())

			tls, err := s.DefaultDataplaneResource().SSL().List(context.Background())
			assert.Nil(GinkgoT(), err, "list tls error")
			assert.Len(GinkgoT(), tls, 1, "tls number not expect")
			assert.Len(GinkgoT(), tls[0].Certificates, 1, "length of certificates not expect")
			assert.Equal(GinkgoT(), Cert, tls[0].Certificates[0].Certificate, "tls cert not expect")
			assert.ElementsMatch(GinkgoT(), []string{host}, tls[0].Snis)

			By("test HTTPS request to dataplane")
			Eventually(func() int {
				return s.NewAPISIXHttpsClient("api6.com").
					GET("/get").
					WithHost("api6.com").
					Expect().
					Raw().StatusCode
			}).WithTimeout(30 * time.Second).ProbeEvery(2 * time.Second).Should(Equal(http.StatusOK))

			s.NewAPISIXHttpsClient("api6.com").
				GET("/get").
				WithHost("api6.com").
				Expect().
				Status(200)
		})

		It("ApisixTls with mTLS test", func() {
			const host = "api6.com"

			By("generate mTLS certificates")
			caCertBytes, serverCertBytes, serverKeyBytes, _, _ := s.GenerateMACert(GinkgoT(), []string{host})
			caCert := caCertBytes.String()
			serverCert := serverCertBytes.String()
			serverKey := serverKeyBytes.String()

			By("create TLS secret")
			err := s.NewKubeTlsSecret("test-mtls-secret", serverCert, serverKey)
			Expect(err).NotTo(HaveOccurred(), "creating TLS secret")

			By("create CA secret")
			err = s.NewClientCASecret("test-ca-secret", caCert, "")
			Expect(err).NotTo(HaveOccurred(), "creating CA secret")

			const apisixTlsSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixTls
metadata:
  name: test-mtls
spec:
  ingressClassName: apisix-tls
  hosts:
  - api6.com
  secret:
    name: test-mtls-secret
    namespace: %s
  client:
    caSecret:
      name: test-ca-secret
      namespace: %s
    depth: 1
`

			By("apply ApisixTls with mTLS")
			var apisixTls apiv2.ApisixTls
			tlsSpec := fmt.Sprintf(apisixTlsSpec, s.Namespace(), s.Namespace())
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "test-mtls"}, &apisixTls, tlsSpec)

			By("verify mTLS configuration in control plane")
			Eventually(func() bool {
				tls, err := s.DefaultDataplaneResource().SSL().List(context.Background())
				if err != nil {
					return false
				}
				if len(tls) != 1 {
					return false
				}
				if len(tls[0].Certificates) != 1 {
					return false
				}
				// Check if client CA is configured
				return tls[0].Client != nil && tls[0].Client.CA != ""
			}).WithTimeout(30 * time.Second).ProbeEvery(2 * time.Second).Should(BeTrue())

			tls, err := s.DefaultDataplaneResource().SSL().List(context.Background())
			assert.Nil(GinkgoT(), err, "list tls error")
			assert.Len(GinkgoT(), tls, 1, "tls number not expect")
			assert.Len(GinkgoT(), tls[0].Certificates, 1, "length of certificates not expect")
			assert.Equal(GinkgoT(), serverCert, tls[0].Certificates[0].Certificate, "tls cert not expect")
			assert.ElementsMatch(GinkgoT(), []string{host}, tls[0].Snis)
			assert.NotNil(GinkgoT(), tls[0].Client, "client configuration should not be nil")
			assert.NotEmpty(GinkgoT(), tls[0].Client.CA, "client CA should not be empty")
			assert.Equal(GinkgoT(), caCert, tls[0].Client.CA, "client CA should be test-ca-secret")
			assert.Equal(GinkgoT(), int64(1), *tls[0].Client.Depth, "client depth should be 1")
		})

	})
})
