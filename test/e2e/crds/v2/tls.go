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
  name: %s
spec:
  controller: %s
  parameters:
    apiGroup: "apisix.apache.org"
    kind: "GatewayProxy"
    name: "apisix-proxy-tls"
    namespace: "%s"
    scope: "Namespace"
`

const apisixRouteYamlTls = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: test-route-tls
spec:
  ingressClassName: %s
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
		s       = scaffold.NewDefaultScaffold()
		applier = framework.NewApplier(s.GinkgoT, s.K8sClient, s.CreateResourceFromString)
	)

	Context("Test ApisixTls", func() {
		BeforeEach(func() {
			By("create GatewayProxy")
			gatewayProxy := fmt.Sprintf(gatewayProxyYamlTls, s.Deployer.GetAdminEndpoint(), s.AdminKey())
			err := s.CreateResourceFromString(gatewayProxy)
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
			time.Sleep(5 * time.Second)

			By("create IngressClass")
			err = s.CreateResourceFromStringWithNamespace(fmt.Sprintf(ingressClassYamlTls, s.Namespace(), s.GetControllerName(), s.Namespace()), "")
			Expect(err).NotTo(HaveOccurred(), "creating IngressClass")
			time.Sleep(5 * time.Second)

			By("create ApisixRoute for TLS testing")
			var apisixRoute apiv2.ApisixRoute
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "test-route-tls"}, &apisixRoute, fmt.Sprintf(apisixRouteYamlTls, s.Namespace()))
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
  ingressClassName: %s
  hosts:
  - api6.com
  secret:
    name: test-tls-secret
    namespace: %s
`

			By("apply ApisixTls")
			var apisixTls apiv2.ApisixTls
			tlsSpec := fmt.Sprintf(apisixTlsSpec, s.Namespace(), s.Namespace())
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
			}).WithTimeout(30 * time.Second).ProbeEvery(1 * time.Second).Should(BeTrue())

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
			}).WithTimeout(30 * time.Second).ProbeEvery(1 * time.Second).Should(Equal(http.StatusOK))

			s.NewAPISIXHttpsClient("api6.com").
				GET("/get").
				WithHost("api6.com").
				Expect().
				Status(200)

			err = s.NewKubeTlsSecret("test-tls-secret", framework.TestCert, framework.TestKey)
			Expect(err).NotTo(HaveOccurred(), "updating TLS secret")

			Eventually(func() error {
				tlss, err := s.DefaultDataplaneResource().SSL().List(context.Background())
				if err != nil {
					return err
				}
				if len(tlss) != 1 {
					return fmt.Errorf("expected 1 tls, got %d", len(tls))
				}
				certs := tlss[0].Certificates
				if len(certs) != 1 {
					return fmt.Errorf("expected 1 certificate, got %d", len(certs))
				}
				if !strings.Contains(certs[0].Certificate, framework.TestCert) {
					return fmt.Errorf("certificate not updated yet")
				}
				return nil
			}).WithTimeout(30*time.Second).ProbeEvery(1*time.Second).ShouldNot(HaveOccurred(), "tls secret updated in dataplane")
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
  ingressClassName: %s
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
			tlsSpec := fmt.Sprintf(apisixTlsSpec, s.Namespace(), s.Namespace(), s.Namespace())
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
			}).WithTimeout(30 * time.Second).ProbeEvery(1 * time.Second).Should(BeTrue())

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
		It("ApisixTls with skip_mtls_uri_regex test", func() {
			const host = "api6.com"
			const skipMtlsUriRegex = "/ip.*"

			By("generate mTLS certificates")
			caCertBytes, serverCertBytes, serverKeyBytes, _, _ := s.GenerateMACert(GinkgoT(), []string{host})
			caCert := caCertBytes.String()
			serverCert := serverCertBytes.String()
			serverKey := serverKeyBytes.String()

			By("create server TLS secret")
			err := s.NewKubeTlsSecret("test-mtls-server-secret", serverCert, serverKey)
			Expect(err).NotTo(HaveOccurred(), "creating server TLS secret")

			By("create client CA secret")
			err = s.NewClientCASecret("test-client-ca-secret", caCert, "")
			Expect(err).NotTo(HaveOccurred(), "creating client CA secret")

			const apisixTlsSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixTls
metadata:
  name: test-mtls-skip-regex
spec:
  ingressClassName: %s
  hosts:
  - %s
  secret:
    name: test-mtls-server-secret
    namespace: %s
  client:
    caSecret:
      name: test-client-ca-secret
      namespace: %s
    depth: 10
    skip_mtls_uri_regex:
    - %s
`

			By("apply ApisixTls with mTLS and skip_mtls_uri_regex")
			var apisixTls apiv2.ApisixTls
			tlsSpec := fmt.Sprintf(apisixTlsSpec, s.Namespace(), host, s.Namespace(), s.Namespace(), skipMtlsUriRegex)
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "test-mtls-skip-regex"}, &apisixTls, tlsSpec)

			By("verify mTLS configuration with skip_mtls_uri_regex")
			Eventually(func() bool {
				tls, err := s.DefaultDataplaneResource().SSL().List(context.Background())
				if err != nil {
					return false
				}
				if len(tls) != 1 {
					return false
				}
				return tls[0].Client != nil &&
					tls[0].Client.CA != "" &&
					len(tls[0].Client.SkipMtlsURIRegex) > 0 &&
					tls[0].Client.SkipMtlsURIRegex[0] == skipMtlsUriRegex
			}).WithTimeout(30 * time.Second).ProbeEvery(1 * time.Second).Should(BeTrue())

			By("test HTTPS request to path matching skip_mtls_uri_regex without client cert")
			Eventually(func() int {
				return s.NewAPISIXHttpsClient(host).
					GET("/ip").
					WithHost(host).
					Expect().
					Raw().StatusCode
			}).WithTimeout(30 * time.Second).ProbeEvery(1 * time.Second).Should(Equal(http.StatusOK))

			By("test HTTPS request to non-matching path without client cert should fail")
			Eventually(func() bool {
				resp := s.NewAPISIXHttpsClient(host).
					GET("/get").
					WithHost(host).
					Expect().
					Raw()
				return resp.StatusCode == http.StatusBadRequest ||
					resp.StatusCode == http.StatusForbidden ||
					resp.StatusCode >= 500
			}).WithTimeout(30 * time.Second).ProbeEvery(1 * time.Second).Should(BeTrue())

			// Verify the configuration details
			tls, err := s.DefaultDataplaneResource().SSL().List(context.Background())
			assert.Nil(GinkgoT(), err, "list tls error")
			assert.Len(GinkgoT(), tls, 1, "tls number not expect")
			assert.NotNil(GinkgoT(), tls[0].Client, "client configuration should not be nil")
			assert.NotEmpty(GinkgoT(), tls[0].Client.CA, "client CA should not be empty")
			assert.Equal(GinkgoT(), caCert, tls[0].Client.CA, "client CA should match")
			assert.Equal(GinkgoT(), int64(10), *tls[0].Client.Depth, "client depth should be 10")
			assert.Contains(GinkgoT(), tls[0].Client.SkipMtlsURIRegex, skipMtlsUriRegex, "skip_mtls_uri_regex should be set")
		})

		It("ApisixTls and Ingress with same certificate but different hosts", func() {
			By("create shared TLS secret")
			err := s.NewKubeTlsSecret("shared-tls-secret", Cert, Key)
			Expect(err).NotTo(HaveOccurred(), "creating shared TLS secret")

			const apisixTlsSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixTls
metadata:
  name: test-apisixtls-shared
spec:
  ingressClassName: %s
  hosts:
  - api6.com
  secret:
    name: shared-tls-secret
    namespace: %s
`

			By("apply ApisixTls with api6.com")
			var apisixTls apiv2.ApisixTls
			tlsSpec := fmt.Sprintf(apisixTlsSpec, s.Namespace(), s.Namespace())
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "test-apisixtls-shared"}, &apisixTls, tlsSpec)

			const ingressYamlWithTLS = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: test-ingress-tls-shared
spec:
  ingressClassName: %s
  tls:
  - hosts:
    - api7.com
    secretName: shared-tls-secret
  rules:
  - host: api7.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: httpbin-service-e2e-test
            port:
              number: 80
`

			By("apply Ingress with api7.com using same certificate")
			err = s.CreateResourceFromString(fmt.Sprintf(ingressYamlWithTLS, s.Namespace()))
			Expect(err).NotTo(HaveOccurred(), "creating Ingress")

			By("verify two SSL objects exist in control plane")
			Eventually(func() bool {
				tls, err := s.DefaultDataplaneResource().SSL().List(context.Background())
				if err != nil {
					return false
				}
				return len(tls) == 2
			}).WithTimeout(30 * time.Second).ProbeEvery(1 * time.Second).Should(BeTrue())

			tls, err := s.DefaultDataplaneResource().SSL().List(context.Background())
			assert.Nil(GinkgoT(), err, "list tls error")
			assert.Len(GinkgoT(), tls, 2, "should have exactly 2 SSL objects")

			By("verify SSL objects have different IDs and SNIs")
			sniFound := make(map[string]bool)

			for i := range tls {
				// Check certificate content is the same
				assert.Len(GinkgoT(), tls[i].Certificates, 1, "each SSL should have 1 certificate")
				assert.Equal(GinkgoT(), Cert, tls[i].Certificates[0].Certificate, "certificate should match")

				// Track SNIs
				for _, sni := range tls[i].Snis {
					sniFound[sni] = true
				}
			}

			By("verify both hosts are covered")
			assert.True(GinkgoT(), sniFound["api6.com"], "api6.com should be in SNIs")
			assert.True(GinkgoT(), sniFound["api7.com"], "api7.com should be in SNIs")

			By("test HTTPS request to api6.com")
			Eventually(func() int {
				return s.NewAPISIXHttpsClient("api6.com").
					GET("/get").
					WithHost("api6.com").
					Expect().
					Raw().StatusCode
			}).WithTimeout(30 * time.Second).ProbeEvery(1 * time.Second).Should(Equal(http.StatusOK))

			By("test HTTPS request to api7.com")
			Eventually(func() int {
				return s.NewAPISIXHttpsClient("api7.com").
					GET("/get").
					WithHost("api7.com").
					Expect().
					Raw().StatusCode
			}).WithTimeout(30 * time.Second).ProbeEvery(1 * time.Second).Should(Equal(http.StatusOK))
		})

	})
})
