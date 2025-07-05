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

package ingress

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"

	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

const _secretName = "test-ingress-tls"

var Cert = strings.TrimSpace(framework.TestServerCert)

var Key = strings.TrimSpace(framework.TestServerKey)

func createSecret(s *scaffold.Scaffold, secretName string) {
	err := s.NewKubeTlsSecret(secretName, Cert, Key)
	assert.Nil(GinkgoT(), err, "create secret error")
}

var _ = Describe("Test Ingress", Label("networking.k8s.io", "ingress"), func() {
	s := scaffold.NewScaffold(&scaffold.Options{
		ControllerName: "apisix.apache.org/apisix-ingress-controller",
	})

	var gatewayProxyYaml = `
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: apisix-proxy-config
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

	Context("Ingress TLS", func() {
		It("Check if SSL resource was created", func() {
			By("create GatewayProxy")
			gatewayProxy := fmt.Sprintf(gatewayProxyYaml, s.Deployer.GetAdminEndpoint(), s.AdminKey())

			By("create GatewayProxy")
			err := s.CreateResourceFromStringWithNamespace(gatewayProxy, "default")
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
			time.Sleep(5 * time.Second)

			secretName := _secretName
			host := "api6.com"
			createSecret(s, secretName)

			var defaultIngressClass = `
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: apisix
spec:
  controller: "apisix.apache.org/apisix-ingress-controller"
  parameters:
    apiGroup: "apisix.apache.org"
    kind: "GatewayProxy"
    name: "apisix-proxy-config"
    namespace: "default"
    scope: "Namespace"
`

			var tlsIngress = fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: apisix-ingress-tls
spec:
  ingressClassName: apisix
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
            name: httpbin-service-e2e-test
            port:
              number: 80
`, host, secretName, host)

			By("create IngressClass")
			err = s.CreateResourceFromStringWithNamespace(defaultIngressClass, "")
			Expect(err).NotTo(HaveOccurred(), "creating IngressClass")
			time.Sleep(5 * time.Second)

			By("create Ingress with TLS")
			err = s.CreateResourceFromString(tlsIngress)
			Expect(err).NotTo(HaveOccurred(), "creating Ingress with TLS")
			time.Sleep(5 * time.Second)

			By("check TLS configuration")
			tls, err := s.DefaultDataplaneResource().SSL().List(context.Background())
			assert.Nil(GinkgoT(), err, "list tls error")
			assert.Len(GinkgoT(), tls, 1, "tls number not expect")
			assert.Len(GinkgoT(), tls[0].Certificates, 1, "length of certificates not expect")
			assert.Equal(GinkgoT(), Cert, tls[0].Certificates[0].Certificate, "tls cert not expect")
			assert.ElementsMatch(GinkgoT(), []string{host}, tls[0].Snis)
		})
	})

	Context("IngressClass Selection", func() {
		var defaultIngressClass = `
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: apisix-default
  annotations:
    ingressclass.kubernetes.io/is-default-class: "true"
spec:
  controller: "apisix.apache.org/apisix-ingress-controller"
  parameters:
    apiGroup: "apisix.apache.org"
    kind: "GatewayProxy"
    name: "apisix-proxy-config"
    namespace: "default"
    scope: "Namespace"
`

		var defaultIngress = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: apisix-ingress-default
spec:
  rules:
  - host: default.example.com
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

		var ingressWithExternalName = `
apiVersion: v1
kind: Service
metadata:
  name: httpbin-external-domain
spec:
  type: ExternalName
  externalName: postman-echo.com
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: apisix-ingress-external
spec:
  rules:
  - host: httpbin.external
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: httpbin-external-domain
            port:
              number: 80
`

		It("Test IngressClass Selection", func() {
			By("create GatewayProxy")
			gatewayProxy := fmt.Sprintf(gatewayProxyYaml, s.Deployer.GetAdminEndpoint(), s.AdminKey())
			err := s.CreateResourceFromStringWithNamespace(gatewayProxy, "default")
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
			time.Sleep(5 * time.Second)

			By("create Default IngressClass")
			err = s.CreateResourceFromStringWithNamespace(defaultIngressClass, "")
			Expect(err).NotTo(HaveOccurred(), "creating Default IngressClass")
			time.Sleep(5 * time.Second)

			By("create Ingress without IngressClass")
			err = s.CreateResourceFromString(defaultIngress)
			Expect(err).NotTo(HaveOccurred(), "creating Ingress without IngressClass")
			time.Sleep(5 * time.Second)

			By("verify default ingress")
			s.NewAPISIXClient().
				GET("/get").
				WithHost("default.example.com").
				Expect().
				Status(200)
		})

		It("Proxy External Service", func() {
			By("create GatewayProxy")
			gatewayProxy := fmt.Sprintf(gatewayProxyYaml, s.Deployer.GetAdminEndpoint(), s.AdminKey())
			err := s.CreateResourceFromStringWithNamespace(gatewayProxy, "default")
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
			time.Sleep(5 * time.Second)

			By("create Default IngressClass")
			err = s.CreateResourceFromStringWithNamespace(defaultIngressClass, "")
			Expect(err).NotTo(HaveOccurred(), "creating Default IngressClass")
			time.Sleep(5 * time.Second)

			By("create Ingress")
			err = s.CreateResourceFromString(ingressWithExternalName)
			Expect(err).NotTo(HaveOccurred(), "creating Ingress without IngressClass")
			time.Sleep(5 * time.Second)

			By("checking the external service response")
			s.NewAPISIXClient().
				GET("/get").
				WithHost("httpbin.external").
				Expect().
				Status(http.StatusMovedPermanently)
		})

		It("Delete Ingress during restart", func() {
			By("create GatewayProxy")
			gatewayProxy := fmt.Sprintf(gatewayProxyYaml, s.Deployer.GetAdminEndpoint(), s.AdminKey())
			err := s.CreateResourceFromStringWithNamespace(gatewayProxy, "default")
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
			time.Sleep(5 * time.Second)

			By("create Default IngressClass")
			err = s.CreateResourceFromStringWithNamespace(defaultIngressClass, "")
			Expect(err).NotTo(HaveOccurred(), "creating Default IngressClass")
			time.Sleep(5 * time.Second)

			By("create Ingress with ExternalName")
			err = s.CreateResourceFromString(ingressWithExternalName)
			Expect(err).NotTo(HaveOccurred(), "creating Ingress without IngressClass")
			time.Sleep(5 * time.Second)

			By("create Ingress")
			err = s.CreateResourceFromString(defaultIngress)
			Expect(err).NotTo(HaveOccurred(), "creating Ingress without IngressClass")
			time.Sleep(5 * time.Second)

			By("checking the external service response")
			s.NewAPISIXClient().
				GET("/get").
				WithHost("httpbin.external").
				Expect().
				Status(http.StatusMovedPermanently)

			s.NewAPISIXClient().
				GET("/get").
				WithHost("default.example.com").
				Expect().
				Status(200)

			s.Deployer.ScaleIngress(0)

			By("delete Ingress")
			err = s.DeleteResourceFromString(defaultIngress)
			Expect(err).NotTo(HaveOccurred(), "deleting Ingress without IngressClass")

			s.Deployer.ScaleIngress(1)
			time.Sleep(1 * time.Minute)

			s.NewAPISIXClient().
				GET("/get").
				WithHost("httpbin.external").
				Expect().
				Status(http.StatusMovedPermanently)

			s.NewAPISIXClient().
				GET("/get").
				WithHost("default.example.com").
				Expect().
				Status(404)
		})
	})

	Context("IngressClass with GatewayProxy", func() {
		gatewayProxyYaml := `
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: apisix-proxy-config
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
  plugins:
  - name: response-rewrite
    enabled: true
    config: 
      headers:
        X-Proxy-Test: "enabled"
`

		gatewayProxyWithSecretYaml := `
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: apisix-proxy-config-with-secret
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
          valueFrom:
            secretKeyRef:
              name: admin-secret
              key: admin-key
  plugins:
  - name: response-rewrite
    enabled: true
    config: 
      headers:
        X-Proxy-Test: "enabled"
`

		var ingressClassWithProxy = `
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: apisix-with-proxy
  annotations:
    ingressclass.kubernetes.io/is-default-class: "true"
spec:
  controller: "apisix.apache.org/apisix-ingress-controller"
  parameters:
    apiGroup: "apisix.apache.org"
    kind: "GatewayProxy"
    name: "apisix-proxy-config"
    namespace: "default"
    scope: "Namespace"
`

		var ingressClassWithProxySecret = `
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: apisix-with-proxy-secret
spec:
  controller: "apisix.apache.org/apisix-ingress-controller"
  parameters:
    apiGroup: "apisix.apache.org"
    kind: "GatewayProxy"
    name: "apisix-proxy-config-with-secret"
    namespace: "default"
    scope: "Namespace"
`

		var testIngress = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: apisix-ingress-with-proxy
spec:
  ingressClassName: apisix-with-proxy
  rules:
  - host: proxy.example.com
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

		var testIngressWithSecret = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: apisix-ingress-with-proxy-secret
spec:
  ingressClassName: apisix-with-proxy-secret
  rules:
  - host: proxy-secret.example.com
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

		It("Test IngressClass with GatewayProxy", func() {
			By("create GatewayProxy")
			gatewayProxy := fmt.Sprintf(gatewayProxyYaml, s.Deployer.GetAdminEndpoint(), s.AdminKey())

			By("create GatewayProxy")
			err := s.CreateResourceFromStringWithNamespace(gatewayProxy, "default")
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
			time.Sleep(5 * time.Second)

			By("create IngressClass with GatewayProxy reference")
			err = s.CreateResourceFromStringWithNamespace(ingressClassWithProxy, "")
			Expect(err).NotTo(HaveOccurred(), "creating IngressClass with GatewayProxy")
			time.Sleep(5 * time.Second)

			By("create Ingress with GatewayProxy IngressClass")
			err = s.CreateResourceFromString(testIngress)
			Expect(err).NotTo(HaveOccurred(), "creating Ingress with GatewayProxy IngressClass")
			time.Sleep(5 * time.Second)

			By("verify HTTP request")
			resp := s.NewAPISIXClient().
				GET("/get").
				WithHost("proxy.example.com").
				Expect().
				Status(200)
			resp.Header("X-Proxy-Test").IsEqual("enabled")
		})

		It("Test IngressClass with GatewayProxy using Secret", func() {
			By("create admin key secret")
			adminSecret := fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: admin-secret
  namespace: default
type: Opaque
stringData:
  admin-key: %s
`, s.AdminKey())
			err := s.CreateResourceFromStringWithNamespace(adminSecret, "default")
			Expect(err).NotTo(HaveOccurred(), "creating admin secret")
			time.Sleep(5 * time.Second)

			By("create GatewayProxy with Secret reference")
			gatewayProxy := fmt.Sprintf(gatewayProxyWithSecretYaml, s.Deployer.GetAdminEndpoint())
			err = s.CreateResourceFromStringWithNamespace(gatewayProxy, "default")
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy with Secret")
			time.Sleep(5 * time.Second)

			By("create IngressClass with GatewayProxy reference")
			err = s.CreateResourceFromStringWithNamespace(ingressClassWithProxySecret, "")
			Expect(err).NotTo(HaveOccurred(), "creating IngressClass with GatewayProxy")
			time.Sleep(5 * time.Second)

			By("create Ingress with GatewayProxy IngressClass")
			err = s.CreateResourceFromString(testIngressWithSecret)
			Expect(err).NotTo(HaveOccurred(), "creating Ingress with GatewayProxy IngressClass")
			time.Sleep(5 * time.Second)

			By("verify HTTP request")
			resp := s.NewAPISIXClient().
				GET("/get").
				WithHost("proxy-secret.example.com").
				Expect().
				Status(200)
			resp.Header("X-Proxy-Test").IsEqual("enabled")
		})
	})

	Context("HTTPRoutePolicy for Ingress", func() {
		getGatewayProxySpec := func() string {
			return fmt.Sprintf(`
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: apisix-proxy-config
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
`, s.Deployer.GetAdminEndpoint(), s.AdminKey())
		}

		const ingressClassSpec = `
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: apisix
spec:
  controller: "apisix.apache.org/apisix-ingress-controller"
  parameters:
    apiGroup: "apisix.apache.org"
    kind: "GatewayProxy"
    name: "apisix-proxy-config"
    namespace: "default"
    scope: "Namespace"
`
		const ingressSpec = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: default
spec:
  ingressClassName: apisix
  rules:
  - host: example.com
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
		const httpRoutePolicySpec0 = `
apiVersion: apisix.apache.org/v1alpha1
kind: HTTPRoutePolicy
metadata:
  name: http-route-policy-0
spec:
  targetRefs:
  - group: networking.k8s.io
    kind: Ingress
    name: default
  priority: 10
  vars:
  - - http_x_hrp_name
    - ==
    - http-route-policy-0
`
		const httpRoutePolicySpec1 = `
apiVersion: apisix.apache.org/v1alpha1
kind: HTTPRoutePolicy
metadata:
  name: http-route-policy-0
spec:
  targetRefs:
  - group: networking.k8s.io
    kind: Ingress
    name: default
  priority: 10
  vars:
  - - arg_hrp_name
    - ==
    - http-route-policy-0
`
		const httpRoutePolicySpec2 = `
apiVersion: apisix.apache.org/v1alpha1
kind: HTTPRoutePolicy
metadata:
  name: http-route-policy-0
spec:
  targetRefs:
  - group: networking.k8s.io
    kind: Ingress
    name: other
  priority: 10
  vars:
  - - arg_hrp_name
    - ==
    - http-route-policy-0
`
		const httpRoutePolicySpec3 = `
apiVersion: apisix.apache.org/v1alpha1
kind: HTTPRoutePolicy
metadata:
  name: http-route-policy-1
spec:
  targetRefs:
  - group: networking.k8s.io
    kind: Ingress
    name: default
  priority: 20
  vars:
  - - arg_hrp_name
    - ==
    - http-route-policy-0
`
		BeforeEach(func() {
			By("create GatewayProxy")
			err := s.CreateResourceFromStringWithNamespace(getGatewayProxySpec(), "default")
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
			time.Sleep(5 * time.Second)

			By("create IngressClass")
			err = s.CreateResourceFromStringWithNamespace(ingressClassSpec, "")
			Expect(err).NotTo(HaveOccurred(), "creating GatewayClass")
			time.Sleep(5 * time.Second)
		})

		It("HTTPRoutePolicy targetRef an Ingress", func() {
			By("create Ingress")
			err := s.CreateResourceFromString(ingressSpec)
			Expect(err).NotTo(HaveOccurred(), "creating Ingress")

			By("request the route should be OK")
			Eventually(func() int {
				return s.NewAPISIXClient().GET("/get").WithHost("example.com").Expect().Raw().StatusCode
			}).
				WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))

			By("create HTTPRoutePolicy")
			err = s.CreateResourceFromString(httpRoutePolicySpec0)
			Expect(err).NotTo(HaveOccurred(), "creating HTTPRoutePolicy")
			Eventually(func() string {
				spec, err := s.GetResourceYaml("HTTPRoutePolicy", "http-route-policy-0")
				Expect(err).NotTo(HaveOccurred(), "HTTPRoutePolicy status should be True")
				return spec
			}).
				WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(ContainSubstring(`status: "True"`))

			By("request the route without vars should be Not Found")
			Eventually(func() int {
				return s.NewAPISIXClient().GET("/get").WithHost("example.com").Expect().Raw().StatusCode
			}).
				WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusNotFound))

			By("request the route with the correct vars should be OK")
			s.NewAPISIXClient().GET("/get").WithHost("example.com").
				WithHeader("X-HRP-Name", "http-route-policy-0").Expect().Status(http.StatusOK)

			By("update the HTTPRoutePolicy")
			err = s.CreateResourceFromString(httpRoutePolicySpec1)
			Expect(err).NotTo(HaveOccurred(), "updating HTTPRoutePolicy")

			By("request with the old vars should be Not Found")
			Eventually(func() int {
				return s.NewAPISIXClient().GET("/get").WithHost("example.com").
					WithHeader("X-HRP-Name", "http-route-policy-0").Expect().Raw().StatusCode
			}).
				WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusNotFound))

			By("request with the new vars should be OK")
			s.NewAPISIXClient().GET("/get").WithHost("example.com").
				WithQuery("hrp_name", "http-route-policy-0").Expect().Status(http.StatusOK)

			By("update the HTTPRoutePolicy's targetRef")
			err = s.CreateResourceFromString(httpRoutePolicySpec2)
			Expect(err).NotTo(HaveOccurred(), "updating HTTPRoutePolicy")

			By("request the route without vars should be OK")
			Eventually(func() int {
				return s.NewAPISIXClient().GET("/get").WithHost("example.com").Expect().Raw().StatusCode
			}).
				WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))

			By("revert the HTTPRoutePolicy")
			err = s.CreateResourceFromString(httpRoutePolicySpec0)
			Expect(err).NotTo(HaveOccurred(), "creating HTTPRoutePolicy")

			By("request the route without vars should be Not Found")
			Eventually(func() int {
				return s.NewAPISIXClient().GET("/get").WithHost("example.com").Expect().Raw().StatusCode
			}).
				WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusNotFound))

			By("request the route with the correct vars should be OK")
			s.NewAPISIXClient().GET("/get").WithHost("example.com").
				WithHeader("X-HRP-Name", "http-route-policy-0").Expect().Status(http.StatusOK)

			By("apply conflict HTTPRoutePolicy")
			err = s.CreateResourceFromString(httpRoutePolicySpec3)
			Expect(err).NotTo(HaveOccurred(), "creating HTTPRoutePolicy")
			Eventually(func() string {
				spec, err := s.GetResourceYaml("HTTPRoutePolicy", "http-route-policy-1")
				Expect(err).NotTo(HaveOccurred(), "get HTTPRoutePolicy")
				return spec
			}).WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(ContainSubstring("reason: Conflicted"))

			By("delete the HTTPRoutePolicy")
			for _, name := range []string{"http-route-policy-0", "http-route-policy-1"} {
				err = s.DeleteResource("HTTPRoutePolicy", name)
				Expect(err).NotTo(HaveOccurred(), "deleting HTTPRoutePolicy")
			}

			By("request the route without vars should be OK")
			Eventually(func() int {
				return s.NewAPISIXClient().GET("/get").WithHost("example.com").Expect().Raw().StatusCode
			}).
				WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))
		})

		It("HTTPRoutePolicy status changes on Ingress deleting", func() {
			By("create Ingress")
			err := s.CreateResourceFromString(ingressSpec)
			Expect(err).NotTo(HaveOccurred(), "creating Ingress")

			By("create HTTPRoutePolicy")
			s.ApplyHTTPRoutePolicy(
				types.NamespacedName{Name: "apisix"},
				types.NamespacedName{Namespace: s.Namespace(), Name: "http-route-policy-0"},
				httpRoutePolicySpec0,
			)

			By("delete ingress")
			err = s.DeleteResource("Ingress", "default")
			Expect(err).NotTo(HaveOccurred(), "delete Ingress")

			err = framework.PollUntilHTTPRoutePolicyHaveStatus(s.K8sClient, 20*time.Second,
				types.NamespacedName{Namespace: s.Namespace(), Name: "http-route-policy-0"},
				func(hrp *v1alpha1.HTTPRoutePolicy) bool {
					return len(hrp.Status.Ancestors) == 0
				},
			)
			Expect(err).NotTo(HaveOccurred(), "expected HTTPRoutePolicy.Status has no Ancestor")
		})
	})

	Context("Ingress with GatewayProxy Update", func() {
		var additionalGatewayGroupID string

		var ingressClass = `
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: apisix-ingress-class
spec:
  controller: "apisix.apache.org/apisix-ingress-controller"
  parameters:
    apiGroup: "apisix.apache.org"
    kind: "GatewayProxy"
    name: "apisix-proxy-config"
    namespace: "default"
    scope: "Namespace"
`
		var ingress = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: apisix-ingress
spec:
  ingressClassName: apisix-ingress-class
  rules:
  - host: ingress.example.com
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
		var updatedGatewayProxy = `
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: apisix-proxy-config
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

		BeforeEach(func() {
			By("create GatewayProxy")
			gatewayProxy := fmt.Sprintf(gatewayProxyYaml, s.Deployer.GetAdminEndpoint(), s.AdminKey())
			err := s.CreateResourceFromStringWithNamespace(gatewayProxy, "default")
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
			time.Sleep(5 * time.Second)

			By("create IngressClass")
			err = s.CreateResourceFromStringWithNamespace(ingressClass, "")
			Expect(err).NotTo(HaveOccurred(), "creating IngressClass")
			time.Sleep(5 * time.Second)
		})

		It("Should sync Ingress when GatewayProxy is updated", func() {
			By("create Ingress")
			err := s.CreateResourceFromString(ingress)
			Expect(err).NotTo(HaveOccurred(), "creating Ingress")
			time.Sleep(5 * time.Second)

			By("verify Ingress works")
			s.NewAPISIXClient().
				GET("/get").
				WithHost("ingress.example.com").
				Expect().
				Status(200)

			By("create additional gateway group to get new admin key")
			additionalGatewayGroupID, _, err = s.Deployer.CreateAdditionalGateway("gateway-proxy-update")
			Expect(err).NotTo(HaveOccurred(), "creating additional gateway group")

			client, err := s.NewAPISIXClientForGateway(additionalGatewayGroupID)
			Expect(err).NotTo(HaveOccurred(), "creating APISIX client for additional gateway group")

			By("Ingress not found for additional gateway group")
			client.
				GET("/get").
				WithHost("ingress.example.com").
				Expect().
				Status(404)

			resources, exists := s.GetAdditionalGateway(additionalGatewayGroupID)
			Expect(exists).To(BeTrue(), "additional gateway group should exist")

			By("update GatewayProxy with new admin key")
			updatedProxy := fmt.Sprintf(updatedGatewayProxy, s.Deployer.GetAdminEndpoint(resources.DataplaneService), resources.AdminAPIKey)
			err = s.CreateResourceFromStringWithNamespace(updatedProxy, "default")
			Expect(err).NotTo(HaveOccurred(), "updating GatewayProxy")
			time.Sleep(5 * time.Second)

			By("verify Ingress works for additional gateway group")
			client.
				GET("/get").
				WithHost("ingress.example.com").
				Expect().
				Status(200)
		})
	})

	PContext("GatewayProxy reference Secret", func() {
		const secretSpec = `
apiVersion: v1
kind: Secret
metadata:
  name: control-plane-secret
data:
  admin-key: %s
`
		const gatewayProxySpec = `
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: apisix-proxy-config
spec:
  provider:
    type: ControlPlane
    controlPlane:
      endpoints:
      - %s
      auth:
        type: AdminKey
        adminKey:
          valueFrom:
            secretKeyRef:
              name: control-plane-secret
              key: admin-key
  plugins:
  - name: response-rewrite
    enabled: true
    config: 
      headers:
        X-Proxy-Test: "enabled"
`
		const ingressClassSpec = `
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: apisix-ingress-class
spec:
  controller: "apisix.apache.org/apisix-ingress-controller"
  parameters:
    apiGroup: "apisix.apache.org"
    kind: "GatewayProxy"
    name: "apisix-proxy-config"
    namespace: %s
    scope: "Namespace"
`
		const ingressSpec = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: apisix-ingress
spec:
  ingressClassName: apisix-ingress-class
  rules:
  - host: ingress.example.com
    http:
      paths:
      - path: /get
        pathType: Prefix
        backend:
          service:
            name: httpbin-service-e2e-test
            port:
              number: 80
`
		var (
			additionalGatewayGroupID string
			err                      error
		)

		It("GatewayProxy reference Secret", func() {
			By("create Secret")
			err = s.CreateResourceFromStringWithNamespace(fmt.Sprintf(secretSpec, base64.StdEncoding.EncodeToString([]byte(s.AdminKey()))), s.Namespace())
			Expect(err).NotTo(HaveOccurred(), "creating secret")

			By("create GatewayProxy")
			err = s.CreateResourceFromStringWithNamespace(fmt.Sprintf(gatewayProxySpec, s.Deployer.GetAdminEndpoint()), s.Namespace())
			Expect(err).NotTo(HaveOccurred(), "creating gateway proxy")

			By("create IngressClass")
			err = s.CreateResourceFromStringWithNamespace(fmt.Sprintf(ingressClassSpec, s.Namespace()), s.Namespace())
			Expect(err).NotTo(HaveOccurred(), "creating IngressClass")

			By("creat Ingress")
			err = s.CreateResourceFromStringWithNamespace(ingressSpec, s.Namespace())
			Expect(err).NotTo(HaveOccurred(), "creating Ingress")

			By("verify Ingress works")
			Eventually(func() int {
				return s.NewAPISIXClient().
					GET("/get").
					WithHost("ingress.example.com").
					Expect().Raw().StatusCode
			}).WithTimeout(8 * time.Second).ProbeEvery(time.Second).
				Should(Equal(http.StatusOK))
			s.NewAPISIXClient().
				GET("/get").
				WithHost("ingress.example.com").
				Expect().Header("X-Proxy-Test").IsEqual("enabled")

			By("create additional gateway group to get new admin key")
			additionalGatewayGroupID, _, err = s.Deployer.CreateAdditionalGateway("gateway-proxy-update")
			Expect(err).NotTo(HaveOccurred(), "creating additional gateway group")

			client, err := s.NewAPISIXClientForGateway(additionalGatewayGroupID)
			Expect(err).NotTo(HaveOccurred(), "creating APISIX client for additional gateway group")

			By("Ingress not found for additional gateway group")
			client.
				GET("/get").
				WithHost("ingress.example.com").
				Expect().
				Status(http.StatusNotFound)

			resources, exists := s.GetAdditionalGateway(additionalGatewayGroupID)
			Expect(exists).To(BeTrue(), "additional gateway group should exist")

			By("update secret")
			err = s.CreateResourceFromStringWithNamespace(fmt.Sprintf(secretSpec, base64.StdEncoding.EncodeToString([]byte(resources.AdminAPIKey))), s.Namespace())
			Expect(err).NotTo(HaveOccurred(), "creating secret")

			By("verify Ingress works for additional gateway group")
			Eventually(func() int {
				return client.
					GET("/get").
					WithHost("ingress.example.com").
					Expect().Raw().StatusCode
			}).WithTimeout(8 * time.Second).ProbeEvery(time.Second).
				Should(Equal(http.StatusOK))
			Eventually(func() string {
				return client.
					GET("/get").
					WithHost("ingress.example.com").
					Expect().Raw().Header.Get("X-Proxy-Test")
			}).WithTimeout(8 * time.Second).ProbeEvery(time.Second).
				Should(Equal("enabled"))
		})
	})
})
