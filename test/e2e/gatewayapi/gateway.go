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

package gatewayapi

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

const _secretName = "test-apisix-tls"

var Cert = strings.TrimSpace(framework.TestServerCert)

var Key = strings.TrimSpace(framework.TestServerKey)

func createSecret(s *scaffold.Scaffold, secretName string) {
	err := s.NewKubeTlsSecret(secretName, Cert, Key)
	assert.Nil(GinkgoT(), err, "create secret error")
}

var _ = Describe("Test Gateway", func() {
	s := scaffold.NewScaffold(&scaffold.Options{
		ControllerName: "apisix.apache.org/apisix-ingress-controller",
	})

	var gatewayProxyYaml = `
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
          value: "%s"
`

	Context("Gateway", func() {
		var defaultGatewayClass = `
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: apisix
spec:
  controllerName: "apisix.apache.org/apisix-ingress-controller"
`

		var defaultGateway = `
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: apisix
spec:
  gatewayClassName: apisix
  listeners:
    - name: http1
      protocol: HTTP
      port: 80
  infrastructure:
    parametersRef:
      group: apisix.apache.org
      kind: GatewayProxy
      name: apisix-proxy-config
`

		var noClassGateway = `
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: apisix-not-class
spec:
  gatewayClassName: apisix-not-exist
  listeners:
    - name: http1
      protocol: HTTP
      port: 80
  infrastructure:
    parametersRef:
      group: apisix.apache.org
      kind: GatewayProxy
      name: apisix-proxy-config
`

		It("Create Gateway", func() {
			By("create GatewayProxy")
			gatewayProxy := fmt.Sprintf(gatewayProxyYaml, s.Deployer.GetAdminEndpoint(), s.AdminKey())
			err := s.CreateResourceFromString(gatewayProxy)
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
			time.Sleep(5 * time.Second)

			By("create GatewayClass")
			err = s.CreateResourceFromStringWithNamespace(defaultGatewayClass, "")
			Expect(err).NotTo(HaveOccurred(), "creating GatewayClass")
			time.Sleep(5 * time.Second)

			By("check GatewayClass condition")
			gcyaml, err := s.GetResourceYaml("GatewayClass", "apisix")
			Expect(err).NotTo(HaveOccurred(), "getting GatewayClass yaml")
			Expect(gcyaml).To(ContainSubstring(`status: "True"`), "checking GatewayClass condition status")
			Expect(gcyaml).To(ContainSubstring("message: the gatewayclass has been accepted by the apisix-ingress-controller"), "checking GatewayClass condition message")

			By("create Gateway")
			err = s.CreateResourceFromStringWithNamespace(defaultGateway, s.
				Namespace())
			Expect(err).NotTo(HaveOccurred(), "creating Gateway")
			time.Sleep(5 * time.Second)

			By("check Gateway condition")
			gwyaml, err := s.GetResourceYaml("Gateway", "apisix")
			Expect(err).NotTo(HaveOccurred(), "getting Gateway yaml")
			Expect(gwyaml).To(ContainSubstring(`status: "True"`), "checking Gateway condition status")
			Expect(gwyaml).To(ContainSubstring("message: the gateway has been accepted by the apisix-ingress-controller"), "checking Gateway condition message")

			By("create Gateway with not accepted GatewayClass")
			err = s.CreateResourceFromStringWithNamespace(noClassGateway, s.Namespace())
			Expect(err).NotTo(HaveOccurred(), "creating Gateway")
			time.Sleep(5 * time.Second)

			By("check Gateway condition")
			gwyaml, err = s.GetResourceYaml("Gateway", "apisix-not-class")
			Expect(err).NotTo(HaveOccurred(), "getting Gateway yaml")
			Expect(gwyaml).To(ContainSubstring(`status: Unknown`), "checking Gateway condition status")
		})
	})

	Context("Gateway SSL", func() {
		It("Check if SSL resource was created", func() {
			By("create GatewayProxy")
			gatewayProxy := fmt.Sprintf(gatewayProxyYaml, s.Deployer.GetAdminEndpoint(), s.AdminKey())
			err := s.CreateResourceFromString(gatewayProxy)
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
			time.Sleep(5 * time.Second)

			By("create secret")
			secretName := _secretName
			host := "api6.com"
			createSecret(s, secretName)
			var defaultGatewayClass = `
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: apisix
spec:
  controllerName: "apisix.apache.org/apisix-ingress-controller"
`

			var defaultGateway = fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: apisix
spec:
  gatewayClassName: apisix
  listeners:
    - name: http1
      protocol: HTTPS
      port: 443
      hostname: %s
      tls:
        certificateRefs:
        - kind: Secret
          group: ""
          name: %s
  infrastructure:
    parametersRef:
      group: apisix.apache.org
      kind: GatewayProxy
      name: apisix-proxy-config
`, host, secretName)
			By("create GatewayClass")
			err = s.CreateResourceFromStringWithNamespace(defaultGatewayClass, "")
			Expect(err).NotTo(HaveOccurred(), "creating GatewayClass")
			time.Sleep(5 * time.Second)

			By("create Gateway")
			err = s.CreateResourceFromStringWithNamespace(defaultGateway, s.Namespace())
			Expect(err).NotTo(HaveOccurred(), "creating Gateway")
			time.Sleep(10 * time.Second)

			tls, err := s.DefaultDataplaneResource().SSL().List(context.Background())
			assert.Nil(GinkgoT(), err, "list tls error")
			assert.Len(GinkgoT(), tls, 1, "tls number not expect")
			assert.Len(GinkgoT(), tls[0].Certificates, 1, "length of certificates not expect")
			assert.Equal(GinkgoT(), Cert, tls[0].Certificates[0].Certificate, "tls cert not expect")
			assert.ElementsMatch(GinkgoT(), []string{host}, tls[0].Snis)
		})

		It("Gateway SSL with and without hostname", func() {
			By("create GatewayProxy")
			gatewayProxy := fmt.Sprintf(gatewayProxyYaml, s.Deployer.GetAdminEndpoint(), s.AdminKey())
			err := s.CreateResourceFromString(gatewayProxy)
			Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
			time.Sleep(5 * time.Second)

			secretName := _secretName
			createSecret(s, secretName)
			var defaultGatewayClass = `
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: apisix
spec:
  controllerName: "apisix.apache.org/apisix-ingress-controller"
`

			var defaultGateway = fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: same-namespace-with-https-listener
spec:
  gatewayClassName: apisix
  listeners:
  - name: https
    port: 443
    protocol: HTTPS
    allowedRoutes:
      namespaces:
        from: Same
    tls:
      certificateRefs:
      - group: ""
        kind: Secret
        name: %s
  - name: https-with-hostname
    port: 443
    hostname: api6.com
    protocol: HTTPS
    allowedRoutes:
      namespaces:
        from: Same
    tls:
      certificateRefs:
      - group: ""
        kind: Secret
        name: %s
  infrastructure:
    parametersRef:
      group: apisix.apache.org
      kind: GatewayProxy
      name: apisix-proxy-config
`, secretName, secretName)
			By("create GatewayClass")
			err = s.CreateResourceFromStringWithNamespace(defaultGatewayClass, "")
			Expect(err).NotTo(HaveOccurred(), "creating GatewayClass")
			time.Sleep(5 * time.Second)

			By("create Gateway")
			err = s.CreateResourceFromStringWithNamespace(defaultGateway, s.Namespace())
			Expect(err).NotTo(HaveOccurred(), "creating Gateway")
			time.Sleep(10 * time.Second)

			tls, err := s.DefaultDataplaneResource().SSL().List(context.Background())
			assert.Nil(GinkgoT(), err, "list tls error")
			assert.Len(GinkgoT(), tls, 1, "tls number not expect")
			assert.Len(GinkgoT(), tls[0].Certificates, 1, "length of certificates not expect")
			assert.Equal(GinkgoT(), Cert, tls[0].Certificates[0].Certificate, "tls cert not expect")
			assert.Equal(GinkgoT(), tls[0].Labels["k8s/controller-name"], "apisix.apache.org/apisix-ingress-controller")

			By("update secret")
			err = s.NewKubeTlsSecret(secretName, framework.TestCert, framework.TestKey)
			Expect(err).NotTo(HaveOccurred(), "update secret")
			Eventually(func() string {
				tls, err := s.DefaultDataplaneResource().SSL().List(context.Background())
				Expect(err).NotTo(HaveOccurred(), "list ssl")
				if len(tls) < 1 {
					return ""
				}
				if len(tls[0].Certificates) < 1 {
					return ""
				}
				return tls[0].Certificates[0].Certificate
			}).WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(framework.TestCert))
		})
	})
})
