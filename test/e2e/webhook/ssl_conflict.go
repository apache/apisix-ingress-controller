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

package webhook

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test SSL/TLS Conflict Detection", Label("webhook"), func() {
	s := scaffold.NewScaffold(scaffold.Options{
		Name:          "ssl-conflict-test",
		EnableWebhook: true,
	})

	BeforeEach(func() {
		By("creating GatewayProxy")
		err := s.CreateResourceFromString(s.GetGatewayProxySpec())
		Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")

		By("creating GatewayClass")
		err = s.CreateResourceFromString(s.GetGatewayClassYaml())
		Expect(err).NotTo(HaveOccurred(), "creating GatewayClass")

		By("creating IngressClass")
		err = s.CreateResourceFromStringWithNamespace(s.GetIngressClassYaml(), "")
		Expect(err).NotTo(HaveOccurred(), "creating IngressClass")
	})

	Context("ApisixTls conflict detection", func() {
		It("should reject ApisixTls with conflicting certificate for same host", func() {
			host := "conflict.example.com"
			secretA := "tls-cert-a"
			secretB := "tls-cert-b"

			By("creating two different TLS secrets")
			createApisixTLSSecret(s, secretA, host, "creating secret A")
			createApisixTLSSecret(s, secretB, host, "creating secret B")

			By("creating first ApisixTls with certificate A")
			tlsAYAML := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixTls
metadata:
  name: tls-a
  namespace: %s
spec:
  ingressClassName: %s
  hosts:
  - %s
  secret:
    name: %s
    namespace: %s
`, s.Namespace(), s.Namespace(), host, secretA, s.Namespace())
			err := s.CreateResourceFromString(tlsAYAML)
			Expect(err).NotTo(HaveOccurred(), "creating ApisixTls A")

			By("attempting to create second ApisixTls with certificate B for same host")
			tlsBYAML := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixTls
metadata:
  name: tls-b
  namespace: %s
spec:
  ingressClassName: %s
  hosts:
  - %s
  secret:
    name: %s
    namespace: %s
`, s.Namespace(), s.Namespace(), host, secretB, s.Namespace())
			Eventually(func(g Gomega) {
				err = s.CreateResourceFromString(tlsBYAML)
				g.Expect(err).Should(HaveOccurred(), "expecting conflict when creating ApisixTls B")
				g.Expect(err.Error()).To(ContainSubstring("SSL configuration conflicts detected"))
				g.Expect(err.Error()).To(ContainSubstring(host))
				g.Expect(err.Error()).To(ContainSubstring("ApisixTls"))
			}).WithTimeout(scaffold.DefaultTimeout).ProbeEvery(scaffold.DefaultInterval).Should(Succeed())
		})

		It("should allow ApisixTls with same certificate for same host", func() {
			host := "shared.example.com"
			sharedSecret := "tls-shared-cert"

			By("creating a shared TLS secret")
			createKubeTLSSecret(s, sharedSecret, host, "creating shared secret")

			By("creating first ApisixTls with shared certificate")
			tls1YAML := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixTls
metadata:
  name: tls-shared-1
  namespace: %s
spec:
  ingressClassName: %s
  hosts:
  - %s
  secret:
    name: %s
    namespace: %s
`, s.Namespace(), s.Namespace(), host, sharedSecret, s.Namespace())
			err := s.CreateResourceFromString(tls1YAML)
			Expect(err).NotTo(HaveOccurred(), "creating first ApisixTls")

			By("creating second ApisixTls with same certificate for same host")
			tls2YAML := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixTls
metadata:
  name: tls-shared-2
  namespace: %s
spec:
  ingressClassName: %s
  hosts:
  - %s
  secret:
    name: %s
    namespace: %s
`, s.Namespace(), s.Namespace(), host, sharedSecret, s.Namespace())
			Eventually(func(g Gomega) {
				err = s.CreateResourceFromString(tls2YAML)
				g.Expect(err).NotTo(HaveOccurred(), "second ApisixTls should be allowed with same certificate")
			}).WithTimeout(scaffold.DefaultTimeout).ProbeEvery(scaffold.DefaultInterval).Should(Succeed())
		})
	})

	Context("Gateway and ApisixTls conflict detection", func() {
		It("should reject Gateway with conflicting certificate against existing ApisixTls", func() {
			host := "gateway-vs-tls.example.com"
			secretA := "gateway-cert-a"
			secretB := "gateway-cert-b"

			By("creating two different TLS secrets")
			createKubeTLSSecret(s, secretA, host, "creating secret A")
			createKubeTLSSecret(s, secretB, host, "creating secret B")

			By("creating ApisixTls with certificate A")
			tlsYAML := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixTls
metadata:
  name: apisixtls-first
  namespace: %s
spec:
  ingressClassName: %s
  hosts:
  - %s
  secret:
    name: %s
    namespace: %s
`, s.Namespace(), s.Namespace(), host, secretA, s.Namespace())
			err := s.CreateResourceFromString(tlsYAML)
			Expect(err).NotTo(HaveOccurred(), "creating ApisixTls")

			By("attempting to create Gateway with certificate B for same host")
			hostname := host
			gatewayYAML := fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: gateway-conflict
  namespace: %s
spec:
  gatewayClassName: %s
  listeners:
  - name: https
    protocol: HTTPS
    port: 443
    hostname: %s
    tls:
      mode: Terminate
      certificateRefs:
      - name: %s
  infrastructure:
    parametersRef:
      group: apisix.apache.org
      kind: GatewayProxy
      name: apisix-proxy-config
`, s.Namespace(), s.Namespace(), hostname, secretB)
			Eventually(func(g Gomega) {
				err = s.CreateResourceFromString(gatewayYAML)
				g.Expect(err).Should(HaveOccurred(), "expecting conflict when creating Gateway")
				g.Expect(err.Error()).To(ContainSubstring("SSL configuration conflicts detected"))
				g.Expect(err.Error()).To(ContainSubstring(host))
			}).WithTimeout(scaffold.DefaultTimeout).ProbeEvery(scaffold.DefaultInterval).Should(Succeed())
		})

		It("should allow Gateway with same certificate as existing ApisixTls", func() {
			host := "gateway-tls-allowed.example.com"
			sharedSecret := "gateway-shared-cert"

			By("creating a shared TLS secret")
			createKubeTLSSecret(s, sharedSecret, host, "creating shared secret")

			By("creating ApisixTls with shared certificate")
			tlsYAML := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixTls
metadata:
  name: apisixtls-allowed
  namespace: %s
spec:
  ingressClassName: %s
  hosts:
  - %s
  secret:
    name: %s
    namespace: %s
`, s.Namespace(), s.Namespace(), host, sharedSecret, s.Namespace())
			err := s.CreateResourceFromString(tlsYAML)
			Expect(err).NotTo(HaveOccurred(), "creating ApisixTls")

			By("creating Gateway with same certificate")
			hostname := host
			gatewayYAML := fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: gateway-allowed
  namespace: %s
spec:
  gatewayClassName: %s
  listeners:
  - name: https
    protocol: HTTPS
    port: 443
    hostname: %s
    tls:
      mode: Terminate
      certificateRefs:
      - name: %s
  infrastructure:
    parametersRef:
      group: apisix.apache.org
      kind: GatewayProxy
      name: apisix-proxy-config
`, s.Namespace(), s.Namespace(), hostname, sharedSecret)
			Eventually(func(g Gomega) {
				err = s.CreateResourceFromString(gatewayYAML)
				g.Expect(err).NotTo(HaveOccurred(), "Gateway should be allowed with same certificate")
			}).WithTimeout(scaffold.DefaultTimeout).ProbeEvery(scaffold.DefaultInterval).Should(Succeed())
		})

		It("should reject ApisixTls when Gateway without hostname uses different certificate", func() {
			host := "gateway-no-host-conflict.example.com"
			secretA := "gateway-no-host-cert-a"
			secretB := "gateway-no-host-cert-b"

			By("creating two different TLS secrets")
			createKubeTLSSecret(s, secretA, host, "creating secret A")
			createKubeTLSSecret(s, secretB, host, "creating secret B")

			By("creating Gateway without explicit hostname using certificate A")
			gatewayYAML := fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: gateway-no-host
  namespace: %s
spec:
  gatewayClassName: %s
  listeners:
  - name: https
    protocol: HTTPS
    port: 443
    tls:
      mode: Terminate
      certificateRefs:
      - name: %s
  infrastructure:
    parametersRef:
      group: apisix.apache.org
      kind: GatewayProxy
      name: apisix-proxy-config
`, s.Namespace(), s.Namespace(), secretA)
			err := s.CreateResourceFromString(gatewayYAML)
			Expect(err).NotTo(HaveOccurred(), "creating Gateway without hostname")

			By("attempting to create ApisixTls with certificate B for same host")
			tlsYAML := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixTls
metadata:
  name: apisixtls-no-host-conflict
  namespace: %s
spec:
  ingressClassName: %s
  hosts:
  - %s
  secret:
    name: %s
    namespace: %s
`, s.Namespace(), s.Namespace(), host, secretB, s.Namespace())
			Eventually(func(g Gomega) {
				err = s.CreateResourceFromString(tlsYAML)
				g.Expect(err).Should(HaveOccurred(), "expecting conflict when creating ApisixTls without hostname on existing Gateway")
				g.Expect(err.Error()).To(ContainSubstring("SSL configuration conflicts detected"))
				g.Expect(err.Error()).To(ContainSubstring(host))
			}).WithTimeout(scaffold.DefaultTimeout).ProbeEvery(scaffold.DefaultInterval).Should(Succeed())
		})
	})

	Context("Gateway self-conflict detection", func() {
		It("should reject Gateway with multiple listeners using different certificates for same host", func() {
			host := "self-conflict.example.com"
			secretA := "gateway-self-cert-a"
			secretB := "gateway-self-cert-b"

			By("creating two different TLS secrets")
			createKubeTLSSecret(s, secretA, host, "creating secret A")
			createKubeTLSSecret(s, secretB, host, "creating secret B")

			By("attempting to create Gateway with two listeners using different certificates for same host")
			hostname := host
			gatewayYAML := fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: gateway-self-conflict
  namespace: %s
spec:
  gatewayClassName: %s
  listeners:
  - name: https-1
    protocol: HTTPS
    port: 443
    hostname: %s
    tls:
      mode: Terminate
      certificateRefs:
      - name: %s
  - name: https-2
    protocol: HTTPS
    port: 8443
    hostname: %s
    tls:
      mode: Terminate
      certificateRefs:
      - name: %s
  infrastructure:
    parametersRef:
      group: apisix.apache.org
      kind: GatewayProxy
      name: apisix-proxy-config
`, s.Namespace(), s.Namespace(), hostname, secretA, hostname, secretB)
			Eventually(func(g Gomega) {
				err := s.CreateResourceFromString(gatewayYAML)
				g.Expect(err).Should(HaveOccurred(), "expecting self-conflict in Gateway")
				g.Expect(err.Error()).To(ContainSubstring("SSL configuration conflicts detected"))
				g.Expect(err.Error()).To(ContainSubstring(host))
			}).WithTimeout(scaffold.DefaultTimeout).ProbeEvery(scaffold.DefaultInterval).Should(Succeed())
		})
	})

	Context("Ingress conflict detection", func() {
		It("should reject Ingress with conflicting certificate in its own TLS config", func() {
			host := "ingress-self-conflict.example.com"
			secretA := "ingress-self-cert-a"
			secretB := "ingress-self-cert-b"

			By("creating two different TLS secrets")
			createKubeTLSSecret(s, secretA, host, "creating secret A")
			createKubeTLSSecret(s, secretB, host, "creating secret B")

			By("creating a backend service for Ingress")
			serviceYAML := fmt.Sprintf(`
apiVersion: v1
kind: Service
metadata:
  name: test-service-self
  namespace: %s
spec:
  selector:
    app: test
  ports:
  - port: 80
    targetPort: 80
`, s.Namespace())
			err := s.CreateResourceFromString(serviceYAML)
			Expect(err).NotTo(HaveOccurred(), "creating service")

			By("attempting to create Ingress with two TLS configs using different certificates for same host")
			ingressYAML := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ingress-self-conflict
  namespace: %s
spec:
  ingressClassName: %s
  tls:
  - hosts:
    - %s
    secretName: %s
  - hosts:
    - %s
    secretName: %s
  rules:
  - host: %s
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: test-service-self
            port:
              number: 80
`, s.Namespace(), s.Namespace(), host, secretA, host, secretB, host)
			Eventually(func(g Gomega) {
				err = s.CreateResourceFromString(ingressYAML)
				g.Expect(err).Should(HaveOccurred(), "expecting self-conflict in Ingress")
				g.Expect(err.Error()).To(ContainSubstring("SSL configuration conflicts detected"))
				g.Expect(err.Error()).To(ContainSubstring(host))
			}).WithTimeout(scaffold.DefaultTimeout).ProbeEvery(scaffold.DefaultInterval).Should(Succeed())
		})

		It("should reject Ingress with conflicting certificate against existing ApisixTls", func() {
			host := "ingress-vs-tls.example.com"
			secretA := "ingress-cert-a"
			secretB := "ingress-cert-b"

			By("creating two different TLS secrets")
			createKubeTLSSecret(s, secretA, host, "creating secret A")
			createKubeTLSSecret(s, secretB, host, "creating secret B")

			By("creating ApisixTls with certificate A")
			tlsYAML := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixTls
metadata:
  name: apisixtls-ingress-test
  namespace: %s
spec:
  ingressClassName: %s
  hosts:
  - %s
  secret:
    name: %s
    namespace: %s
`, s.Namespace(), s.Namespace(), host, secretA, s.Namespace())
			err := s.CreateResourceFromString(tlsYAML)
			Expect(err).NotTo(HaveOccurred(), "creating ApisixTls")

			By("creating a backend service for Ingress")
			serviceYAML := fmt.Sprintf(`
apiVersion: v1
kind: Service
metadata:
  name: test-service
  namespace: %s
spec:
  selector:
    app: test
  ports:
  - port: 80
    targetPort: 80
`, s.Namespace())
			err = s.CreateResourceFromString(serviceYAML)
			Expect(err).NotTo(HaveOccurred(), "creating service")

			By("attempting to create Ingress with certificate B for same host")
			ingressYAML := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ingress-conflict
  namespace: %s
spec:
  ingressClassName: %s
  tls:
  - hosts:
    - %s
    secretName: %s
  rules:
  - host: %s
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: test-service
            port:
              number: 80
`, s.Namespace(), s.Namespace(), host, secretB, host)
			Eventually(func(g Gomega) {
				err = s.CreateResourceFromString(ingressYAML)
				g.Expect(err).Should(HaveOccurred(), "expecting conflict when creating Ingress")
				g.Expect(err.Error()).To(ContainSubstring("SSL configuration conflicts detected"))
				g.Expect(err.Error()).To(ContainSubstring(host))
			}).WithTimeout(scaffold.DefaultTimeout).ProbeEvery(scaffold.DefaultInterval).Should(Succeed())
		})

		It("should allow Ingress with same certificate as existing Gateway", func() {
			host := "ingress-gateway-allowed.example.com"
			sharedSecret := "ingress-gateway-shared-cert"

			By("creating a shared TLS secret")
			createKubeTLSSecret(s, sharedSecret, host, "creating shared secret")

			By("creating Gateway with shared certificate")
			hostname := host
			gatewayYAML := fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: gateway-for-ingress
  namespace: %s
spec:
  gatewayClassName: %s
  listeners:
  - name: https
    protocol: HTTPS
    port: 443
    hostname: %s
    tls:
      mode: Terminate
      certificateRefs:
      - name: %s
  infrastructure:
    parametersRef:
      group: apisix.apache.org
      kind: GatewayProxy
      name: apisix-proxy-config
`, s.Namespace(), s.Namespace(), hostname, sharedSecret)
			err := s.CreateResourceFromString(gatewayYAML)
			Expect(err).NotTo(HaveOccurred(), "creating Gateway")

			By("creating a backend service for Ingress")
			serviceYAML := fmt.Sprintf(`
apiVersion: v1
kind: Service
metadata:
  name: test-service-2
  namespace: %s
spec:
  selector:
    app: test
  ports:
  - port: 80
    targetPort: 80
`, s.Namespace())
			err = s.CreateResourceFromString(serviceYAML)
			Expect(err).NotTo(HaveOccurred(), "creating service")

			By("creating Ingress with same certificate")
			ingressYAML := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ingress-allowed
  namespace: %s
spec:
  ingressClassName: %s
  tls:
  - hosts:
    - %s
    secretName: %s
  rules:
  - host: %s
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: test-service-2
            port:
              number: 80
`, s.Namespace(), s.Namespace(), host, sharedSecret, host)
			Eventually(func(g Gomega) {
				err = s.CreateResourceFromString(ingressYAML)
				g.Expect(err).NotTo(HaveOccurred(), "Ingress should be allowed with same certificate")
			}).WithTimeout(scaffold.DefaultTimeout).ProbeEvery(scaffold.DefaultInterval).Should(Succeed())
		})

		It("should reject Ingress when Gateway without hostname uses different certificate", func() {
			host := "gateway-ingress-no-host-conflict.example.com"
			secretA := "gateway-ingress-no-host-cert-a"
			secretB := "gateway-ingress-no-host-cert-b"

			By("creating two different TLS secrets")
			createKubeTLSSecret(s, secretA, host, "creating secret A")
			createKubeTLSSecret(s, secretB, host, "creating secret B")

			By("creating Gateway without explicit hostname using certificate A")
			gatewayYAML := fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: gateway-ingress-no-host
  namespace: %s
spec:
  gatewayClassName: %s
  listeners:
  - name: https
    protocol: HTTPS
    port: 443
    tls:
      mode: Terminate
      certificateRefs:
      - name: %s
  infrastructure:
    parametersRef:
      group: apisix.apache.org
      kind: GatewayProxy
      name: apisix-proxy-config
`, s.Namespace(), s.Namespace(), secretA)
			err := s.CreateResourceFromString(gatewayYAML)
			Expect(err).NotTo(HaveOccurred(), "creating Gateway without hostname")

			By("creating a backend service for Ingress")
			serviceYAML := fmt.Sprintf(`
apiVersion: v1
kind: Service
metadata:
  name: test-service-ingress-no-host
  namespace: %s
spec:
  selector:
    app: test
  ports:
  - port: 80
    targetPort: 80
`, s.Namespace())
			err = s.CreateResourceFromString(serviceYAML)
			Expect(err).NotTo(HaveOccurred(), "creating service")

			By("attempting to create Ingress without explicit host using certificate B")
			ingressYAML := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ingress-no-host-conflict
  namespace: %s
spec:
  ingressClassName: %s
  tls:
  - secretName: %s
  rules:
  - http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: test-service-ingress-no-host
            port:
              number: 80
`, s.Namespace(), s.Namespace(), secretB)
			Eventually(func(g Gomega) {
				err = s.CreateResourceFromString(ingressYAML)
				g.Expect(err).Should(HaveOccurred(), "expecting conflict when creating Ingress without hostname")
				g.Expect(err.Error()).To(ContainSubstring("SSL configuration conflicts detected"))
				g.Expect(err.Error()).To(ContainSubstring(host))
			}).WithTimeout(scaffold.DefaultTimeout).ProbeEvery(scaffold.DefaultInterval).Should(Succeed())
		})
	})

	Context("Default IngressClass conflict detection", func() {
		It("should reject Ingress without explicit class when default class uses a different certificate", func() {
			host := "default-ingress-conflict.example.com"
			secretA := "default-ingress-cert-a"
			secretB := "default-ingress-cert-b"
			defaultClassName := fmt.Sprintf("%s-default", s.Namespace())

			By("creating TLS secrets for default ingress test")
			createKubeTLSSecret(s, secretA, host, "creating secret A")
			createKubeTLSSecret(s, secretB, host, "creating secret B")

			By("creating default IngressClass with APISIX controller")
			defaultIngressClassYAML := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: %s
  annotations:
    ingressclass.kubernetes.io/is-default-class: "true"
spec:
  controller: %s
  parameters:
    apiGroup: "apisix.apache.org"
    kind: "GatewayProxy"
    name: "apisix-proxy-config"
    namespace: %s
    scope: Namespace
`, defaultClassName, s.GetControllerName(), s.Namespace())
			err := s.CreateResourceFromStringWithNamespace(defaultIngressClassYAML, "")
			Expect(err).NotTo(HaveOccurred(), "creating default IngressClass")

			By("creating backend service for default ingress test")
			serviceYAML := fmt.Sprintf(`
apiVersion: v1
kind: Service
metadata:
  name: test-service-default
  namespace: %s
spec:
  selector:
    app: test
  ports:
  - port: 80
    targetPort: 80
`, s.Namespace())
			err = s.CreateResourceFromString(serviceYAML)
			Expect(err).NotTo(HaveOccurred(), "creating service")

			By("creating baseline Ingress with certificate A")
			ingressAYAML := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ingress-default-a
  namespace: %s
spec:
  tls:
  - hosts:
    - %s
    secretName: %s
  rules:
  - host: %s
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: test-service-default
            port:
              number: 80
`, s.Namespace(), host, secretA, host)
			err = s.CreateResourceFromString(ingressAYAML)
			Expect(err).NotTo(HaveOccurred(), "creating baseline Ingress")

			By("attempting to create second Ingress with conflicting certificate via default class")
			ingressBYAML := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ingress-default-b
  namespace: %s
spec:
  tls:
  - hosts:
    - %s
    secretName: %s
  rules:
  - host: %s
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: test-service-default
            port:
              number: 80
`, s.Namespace(), host, secretB, host)
			Eventually(func(g Gomega) {
				err = s.CreateResourceFromString(ingressBYAML)
				g.Expect(err).Should(HaveOccurred(), "expecting conflict when creating second Ingress")
				g.Expect(err.Error()).To(ContainSubstring("SSL configuration conflicts detected"))
				g.Expect(err.Error()).To(ContainSubstring(host))
			}).WithTimeout(scaffold.DefaultTimeout).ProbeEvery(scaffold.DefaultInterval).Should(Succeed())
		})

		It("should reject ApisixTls without explicit class when default class uses a different certificate", func() {
			host := "default-tls-conflict.example.com"
			secretA := "default-tls-cert-a"
			secretB := "default-tls-cert-b"
			defaultClassName := fmt.Sprintf("%s-default-tls", s.Namespace())

			By("creating TLS secrets for default ApisixTls test")
			createKubeTLSSecret(s, secretA, host, "creating secret A")
			createKubeTLSSecret(s, secretB, host, "creating secret B")

			By("creating default IngressClass required for ApisixTls admission")
			defaultIngressClassYAML := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: %s
  annotations:
    ingressclass.kubernetes.io/is-default-class: "true"
spec:
  controller: %s
  parameters:
    apiGroup: "apisix.apache.org"
    kind: "GatewayProxy"
    name: "apisix-proxy-config"
    namespace: %s
    scope: Namespace
`, defaultClassName, s.GetControllerName(), s.Namespace())
			err := s.CreateResourceFromStringWithNamespace(defaultIngressClassYAML, "")
			Expect(err).NotTo(HaveOccurred(), "creating default IngressClass")

			By("creating baseline ApisixTls without explicit ingress class")
			tlsAYAML := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixTls
metadata:
  name: tls-default-a
  namespace: %s
spec:
  hosts:
  - %s
  secret:
    name: %s
    namespace: %s
`, s.Namespace(), host, secretA, s.Namespace())
			err = s.CreateResourceFromString(tlsAYAML)
			Expect(err).NotTo(HaveOccurred(), "creating baseline ApisixTls")

			By("attempting to create ApisixTls with conflicting certificate without class override")
			tlsBYAML := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixTls
metadata:
  name: tls-default-b
  namespace: %s
spec:
  hosts:
  - %s
  secret:
    name: %s
    namespace: %s
`, s.Namespace(), host, secretB, s.Namespace())
			Eventually(func(g Gomega) {
				err = s.CreateResourceFromString(tlsBYAML)
				g.Expect(err).Should(HaveOccurred(), "expecting conflict when creating second ApisixTls")
				g.Expect(err.Error()).To(ContainSubstring("SSL configuration conflicts detected"))
				g.Expect(err.Error()).To(ContainSubstring(host))
			}).WithTimeout(scaffold.DefaultTimeout).ProbeEvery(scaffold.DefaultInterval).Should(Succeed())
		})
	})

	Context("Update scenario conflict detection", func() {
		It("should reject Ingress update that switches to a conflicting certificate", func() {
			host := "ingress-update-conflict.example.com"
			secretA := "ingress-update-cert-a"
			secretB := "ingress-update-cert-b"

			By("creating TLS secrets for ingress update test")
			createKubeTLSSecret(s, secretA, host, "creating secret A")
			createKubeTLSSecret(s, secretB, host, "creating secret B")

			By("creating ApisixTls with certificate A to establish existing mapping")
			tlsYAML := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixTls
metadata:
  name: tls-update-baseline
  namespace: %s
spec:
  ingressClassName: %s
  hosts:
  - %s
  secret:
    name: %s
    namespace: %s
`, s.Namespace(), s.Namespace(), host, secretA, s.Namespace())
			err := s.CreateResourceFromString(tlsYAML)
			Expect(err).NotTo(HaveOccurred(), "creating baseline ApisixTls for ingress update")

			By("creating backend service for ingress update test")
			serviceYAML := fmt.Sprintf(`
apiVersion: v1
kind: Service
metadata:
  name: test-service-update
  namespace: %s
spec:
  selector:
    app: test
  ports:
  - port: 80
    targetPort: 80
`, s.Namespace())
			err = s.CreateResourceFromString(serviceYAML)
			Expect(err).NotTo(HaveOccurred(), "creating service")

			By("creating initial Ingress with matching certificate")
			ingressBaseYAML := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ingress-update
  namespace: %s
spec:
  ingressClassName: %s
  tls:
  - hosts:
    - %s
    secretName: %s
  rules:
  - host: %s
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: test-service-update
            port:
              number: 80
`, s.Namespace(), s.Namespace(), host, secretA, host)
			err = s.CreateResourceFromString(ingressBaseYAML)
			Expect(err).NotTo(HaveOccurred(), "creating initial Ingress")

			By("attempting to update Ingress to use conflicting certificate B")
			ingressUpdatedYAML := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ingress-update
  namespace: %s
spec:
  ingressClassName: %s
  tls:
  - hosts:
    - %s
    secretName: %s
  rules:
  - host: %s
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: test-service-update
            port:
              number: 80
`, s.Namespace(), s.Namespace(), host, secretB, host)
			Eventually(func(g Gomega) {
				err = s.CreateResourceFromString(ingressUpdatedYAML)
				g.Expect(err).Should(HaveOccurred(), "expecting conflict when updating Ingress certificate")
				g.Expect(err.Error()).To(ContainSubstring("SSL configuration conflicts detected"))
				g.Expect(err.Error()).To(ContainSubstring(host))
			}).WithTimeout(scaffold.DefaultTimeout).ProbeEvery(scaffold.DefaultInterval).Should(Succeed())
		})

		It("should reject Gateway update that switches to a conflicting certificate", func() {
			host := "gateway-update-conflict.example.com"
			secretA := "gateway-update-cert-a"
			secretB := "gateway-update-cert-b"

			By("creating TLS secrets for gateway update test")
			createKubeTLSSecret(s, secretA, host, "creating secret A")
			createKubeTLSSecret(s, secretB, host, "creating secret B")

			By("creating ApisixTls with certificate A to establish host ownership")
			tlsYAML := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixTls
metadata:
  name: tls-gateway-update
  namespace: %s
spec:
  ingressClassName: %s
  hosts:
  - %s
  secret:
    name: %s
    namespace: %s
`, s.Namespace(), s.Namespace(), host, secretA, s.Namespace())
			err := s.CreateResourceFromString(tlsYAML)
			Expect(err).NotTo(HaveOccurred(), "creating baseline ApisixTls for gateway update")

			By("creating initial Gateway using certificate A")
			gatewayBaseYAML := fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: gateway-update
  namespace: %s
spec:
  gatewayClassName: %s
  listeners:
  - name: https
    protocol: HTTPS
    port: 443
    hostname: %s
    tls:
      mode: Terminate
      certificateRefs:
      - name: %s
  infrastructure:
    parametersRef:
      group: apisix.apache.org
      kind: GatewayProxy
      name: apisix-proxy-config
`, s.Namespace(), s.Namespace(), host, secretA)
			err = s.CreateResourceFromString(gatewayBaseYAML)
			Expect(err).NotTo(HaveOccurred(), "creating initial Gateway")

			By("attempting to update Gateway to use conflicting certificate B")
			gatewayUpdatedYAML := fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: gateway-update
  namespace: %s
spec:
  gatewayClassName: %s
  listeners:
  - name: https
    protocol: HTTPS
    port: 443
    hostname: %s
    tls:
      mode: Terminate
      certificateRefs:
      - name: %s
  infrastructure:
    parametersRef:
      group: apisix.apache.org
      kind: GatewayProxy
      name: apisix-proxy-config
`, s.Namespace(), s.Namespace(), host, secretB)
			Eventually(func(g Gomega) {
				err = s.CreateResourceFromString(gatewayUpdatedYAML)
				g.Expect(err).Should(HaveOccurred(), "expecting conflict when updating Gateway certificate")
				g.Expect(err.Error()).To(ContainSubstring("SSL configuration conflicts detected"))
				g.Expect(err.Error()).To(ContainSubstring(host))
			}).WithTimeout(scaffold.DefaultTimeout).ProbeEvery(scaffold.DefaultInterval).Should(Succeed())
		})
	})

	Context("Mixed resource conflict detection", func() {
		It("should handle conflicts among Gateway, Ingress, and ApisixTls", func() {
			host := "mixed.example.com"
			secretA := "mixed-cert-a"
			secretB := "mixed-cert-b"
			secretC := "mixed-cert-c"

			By("creating three different TLS secrets")
			createKubeTLSSecret(s, secretA, host, "creating secret A")
			createKubeTLSSecret(s, secretB, host, "creating secret B")
			createKubeTLSSecret(s, secretC, host, "creating secret C")

			By("creating Gateway with certificate A")
			hostname := host
			gatewayYAML := fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: gateway-mixed
  namespace: %s
spec:
  gatewayClassName: %s
  listeners:
  - name: https
    protocol: HTTPS
    port: 443
    hostname: %s
    tls:
      mode: Terminate
      certificateRefs:
      - name: %s
  infrastructure:
    parametersRef:
      group: apisix.apache.org
      kind: GatewayProxy
      name: apisix-proxy-config
`, s.Namespace(), s.Namespace(), hostname, secretA)
			err := s.CreateResourceFromString(gatewayYAML)
			Expect(err).NotTo(HaveOccurred(), "creating Gateway with cert A")

			By("attempting to create ApisixTls with certificate B")
			tlsYAML := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixTls
metadata:
  name: apisixtls-mixed
  namespace: %s
spec:
  ingressClassName: %s
  hosts:
  - %s
  secret:
    name: %s
    namespace: %s
`, s.Namespace(), s.Namespace(), host, secretB, s.Namespace())
			Eventually(func(g Gomega) {
				err = s.CreateResourceFromString(tlsYAML)
				g.Expect(err).Should(HaveOccurred(), "expecting conflict when creating ApisixTls with different cert")
				g.Expect(err.Error()).To(ContainSubstring("SSL configuration conflicts detected"))
			}).WithTimeout(scaffold.DefaultTimeout).ProbeEvery(scaffold.DefaultInterval).Should(Succeed())

			By("creating a backend service")
			serviceYAML := fmt.Sprintf(`
apiVersion: v1
kind: Service
metadata:
  name: test-service-3
  namespace: %s
spec:
  selector:
    app: test
  ports:
  - port: 80
    targetPort: 80
`, s.Namespace())
			err = s.CreateResourceFromString(serviceYAML)
			Expect(err).NotTo(HaveOccurred(), "creating service")

			By("attempting to create Ingress with certificate C")
			ingressYAML := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ingress-mixed
  namespace: %s
spec:
  ingressClassName: %s
  tls:
  - hosts:
    - %s
    secretName: %s
  rules:
  - host: %s
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: test-service-3
            port:
              number: 80
`, s.Namespace(), s.Namespace(), host, secretC, host)
			Eventually(func(g Gomega) {
				err = s.CreateResourceFromString(ingressYAML)
				g.Expect(err).Should(HaveOccurred(), "expecting conflict when creating Ingress with different cert")
				g.Expect(err.Error()).To(ContainSubstring("SSL configuration conflicts detected"))
			}).WithTimeout(scaffold.DefaultTimeout).ProbeEvery(scaffold.DefaultInterval).Should(Succeed())
		})
	})
})

func createApisixTLSSecret(s *scaffold.Scaffold, secretName, host, failureMessage string) {
	cert, key := s.GenerateCert(GinkgoT(), []string{host})
	err := s.NewSecret(secretName, cert.String(), key.String())
	Expect(err).NotTo(HaveOccurred(), failureMessage)
}

func createKubeTLSSecret(s *scaffold.Scaffold, secretName, host, failureMessage string) {
	cert, key := s.GenerateCert(GinkgoT(), []string{host})
	err := s.NewKubeTlsSecret(secretName, cert.String(), key.String())
	Expect(err).NotTo(HaveOccurred(), failureMessage)
}
