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
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test GatewayClass", Label("networking.k8s.io", "gatewayclass"), func() {
	s := scaffold.NewScaffold(&scaffold.Options{
		ControllerName: fmt.Sprintf("apisix.apache.org/apisix-ingress-controller-%d", time.Now().Unix()),
	})

	Context("Create GatewayClass", func() {
		var defautlGatewayClass = `
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: %s
spec:
  controllerName: "%s"
`

		var noGatewayClass = `
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: apisix-not-accepeted
spec:
  controllerName: "apisix.apache.org/not-exist"
`
		const defaultGateway = `
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: %s
spec:
  gatewayClassName: %s
  listeners:
    - name: http1
      protocol: HTTP
      port: 80
`
		It("Create GatewayClass", func() {
			gatewayClassName := fmt.Sprintf("apisix-%d", time.Now().Unix())
			By("create default GatewayClass")
			err := s.CreateResourceFromString(fmt.Sprintf(defautlGatewayClass, gatewayClassName, s.GetControllerName()))
			Expect(err).NotTo(HaveOccurred(), "creating GatewayClass")
			time.Sleep(5 * time.Second)

			gcyaml, err := s.GetResourceYaml("GatewayClass", gatewayClassName)
			Expect(err).NotTo(HaveOccurred(), "getting GatewayClass yaml")
			Expect(gcyaml).To(ContainSubstring(`status: "True"`), "checking GatewayClass condition status")
			Expect(gcyaml).To(ContainSubstring("message: the gatewayclass has been accepted by the apisix-ingress-controller"), "checking GatewayClass condition message")

			By("create GatewayClass with not accepted")
			err = s.CreateResourceFromString(noGatewayClass)
			Expect(err).NotTo(HaveOccurred(), "creating GatewayClass")
			time.Sleep(5 * time.Second)

			gcyaml, err = s.GetResourceYaml("GatewayClass", "apisix-not-accepeted")
			Expect(err).NotTo(HaveOccurred(), "getting GatewayClass yaml")
			Expect(gcyaml).To(ContainSubstring(`status: Unknown`), "checking GatewayClass condition status")
			Expect(gcyaml).To(ContainSubstring("message: Waiting for controller"), "checking GatewayClass condition message")
		})

		It("Delete GatewayClass", func() {
			gatewayClassName := fmt.Sprintf("apisix-%d", time.Now().Unix())
			By("create default GatewayClass")
			err := s.CreateResourceFromString(fmt.Sprintf(defautlGatewayClass, gatewayClassName, s.GetControllerName()))
			Expect(err).NotTo(HaveOccurred(), "creating GatewayClass")
			Eventually(func() string {
				spec, err := s.GetResourceYaml("GatewayClass", gatewayClassName)
				Expect(err).NotTo(HaveOccurred(), "get resource yaml")
				return spec
			}).WithTimeout(20 * time.Second).ProbeEvery(time.Second).Should(ContainSubstring(`status: "True"`))

			By("create a Gateway")
			gatewayName := s.Namespace()
			err = s.CreateResourceFromString(fmt.Sprintf(defaultGateway, gatewayName, gatewayClassName))
			Expect(err).NotTo(HaveOccurred(), "creating Gateway")
			time.Sleep(time.Second)

			By("try to delete the GatewayClass")
			_, err = s.RunKubectlAndGetOutput("delete", "GatewayClass", gatewayClassName, "--wait=false")
			Expect(err).NotTo(HaveOccurred())

			_, err = s.GetResourceYaml("GatewayClass", gatewayClassName)
			Expect(err).NotTo(HaveOccurred(), "get resource yaml")

			output, err := s.RunKubectlAndGetOutput("describe", "GatewayClass", gatewayClassName)
			Expect(err).NotTo(HaveOccurred(), "describe GatewayClass apisix")
			Expect(output).To(And(
				ContainSubstring("Warning"),
				ContainSubstring("DeletionBlocked"),
				ContainSubstring("gatewayclass-controller"),
				ContainSubstring("the GatewayClass is still used by Gateways"),
			))

			By("delete the Gateway")
			err = s.DeleteResource("Gateway", gatewayName)
			Expect(err).NotTo(HaveOccurred(), "deleting Gateway")
			time.Sleep(time.Second)

			By("try to delete the GatewayClass again")
			err = s.DeleteResource("GatewayClass", gatewayClassName)
			Expect(err).NotTo(HaveOccurred())

			_, err = s.GetResourceYaml("GatewayClass", gatewayClassName)
			Expect(err).To(HaveOccurred(), "get resource yaml")
			Expect(err.Error()).To(ContainSubstring("not found"))
		})
	})
})
