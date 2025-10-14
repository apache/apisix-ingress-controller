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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test GatewayProxy Webhook", Label("webhook"), func() {
	s := scaffold.NewScaffold(scaffold.Options{
		Name:          "gatewayproxy-webhook-test",
		EnableWebhook: true,
	})

	gatewayProxyTemplate := `
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: %s
spec:
  provider:
    type: ControlPlane
    controlPlane:
      service:
        name: %s
        port: 9180
      auth:
        type: AdminKey
        adminKey:
          valueFrom:
            secretKeyRef:
              name: %s
              key: token
`

	It("should warn on missing service or secret references", func() {
		missingService := "missing-control-plane"
		missingSecret := "missing-admin-secret"
		gpName := "webhook-gateway-proxy"

		output, err := s.CreateResourceFromStringAndGetOutput(fmt.Sprintf(gatewayProxyTemplate, gpName, missingService, missingSecret))
		Expect(err).ShouldNot(HaveOccurred())
		Expect(output).To(ContainSubstring(fmt.Sprintf("Warning: Referenced Service '%s/%s' not found", s.Namespace(), missingService)))
		Expect(output).To(ContainSubstring(fmt.Sprintf("Warning: Referenced Secret '%s/%s' not found", s.Namespace(), missingSecret)))

		By("creating the referenced Service and Secret without the required key")
		serviceYAML := fmt.Sprintf(`
apiVersion: v1
kind: Service
metadata:
  name: %s
spec:
  selector:
    app: placeholder
  ports:
  - name: admin
    port: 9180
    targetPort: 9180
  type: ClusterIP
`, missingService)
		err = s.CreateResourceFromString(serviceYAML)
		Expect(err).NotTo(HaveOccurred(), "creating placeholder service")

		secretWithoutKey := fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: %s
stringData:
  another: value
`, missingSecret)
		err = s.CreateResourceFromString(secretWithoutKey)
		Expect(err).NotTo(HaveOccurred(), "creating placeholder secret without token key")

		time.Sleep(2 * time.Second)

		By("delete and reapply the GatewayProxy, because gatewayproxy has no change")
		err = s.DeleteResource("GatewayProxy", gpName)
		Expect(err).ShouldNot(HaveOccurred())

		output, err = s.CreateResourceFromStringAndGetOutput(fmt.Sprintf(gatewayProxyTemplate, gpName, missingService, missingSecret))
		Expect(err).ShouldNot(HaveOccurred())
		Expect(output).NotTo(ContainSubstring(fmt.Sprintf("Warning: Referenced Service '%s/%s' not found", s.Namespace(), missingService)))
		Expect(output).To(ContainSubstring(fmt.Sprintf("Warning: Secret key 'token' not found in Secret '%s/%s'", s.Namespace(), missingSecret)))

		By("updating the Secret to include the expected key")
		secretWithKey := fmt.Sprintf(`
apiVersion: v1
kind: Secret
metadata:
  name: %s
stringData:
  token: %s
`, missingSecret, s.AdminKey())
		err = s.CreateResourceFromString(secretWithKey)
		Expect(err).NotTo(HaveOccurred(), "adding token key to secret")

		time.Sleep(2 * time.Second)

		By("delete and reapply the GatewayProxy, because gatewayproxy has no change")
		err = s.DeleteResource("GatewayProxy", gpName)
		Expect(err).ShouldNot(HaveOccurred())

		output, err = s.CreateResourceFromStringAndGetOutput(fmt.Sprintf(gatewayProxyTemplate, gpName, missingService, missingSecret))
		Expect(err).ShouldNot(HaveOccurred())
		Expect(output).NotTo(ContainSubstring(fmt.Sprintf("Warning: Referenced Service '%s/%s' not found", s.Namespace(), missingService)))
		Expect(output).NotTo(ContainSubstring(fmt.Sprintf("Warning: Secret key 'token' not found in Secret '%s/%s'", s.Namespace(), missingSecret)))
	})

	Context("GatewayProxy configuration conflicts", func() {
		It("should reject GatewayProxy that reuses the same Service and AdminKey Secret as an existing one on create and update", func() {
			serviceTemplate := `
apiVersion: v1
kind: Service
metadata:
  name: %s
spec:
  selector:
    app: dummy-control-plane
  ports:
  - name: admin
    port: 9180
    targetPort: 9180
`
			secretTemplate := `
apiVersion: v1
kind: Secret
metadata:
  name: %s
type: Opaque
stringData:
  %s: %s
`
			serviceName := "gatewayproxy-shared-service"
			secretName := "gatewayproxy-shared-secret"
			initialProxy := "gatewayproxy-shared-primary"
			conflictingProxy := "gatewayproxy-shared-conflict"

			Expect(s.CreateResourceFromString(fmt.Sprintf(serviceTemplate, serviceName))).ShouldNot(HaveOccurred(), "creating shared Service")
			Expect(s.CreateResourceFromString(fmt.Sprintf(secretTemplate, secretName, "token", "value"))).ShouldNot(HaveOccurred(), "creating shared Secret")

			err := s.CreateResourceFromString(fmt.Sprintf(gatewayProxyTemplate, initialProxy, serviceName, secretName))
			Expect(err).ShouldNot(HaveOccurred(), "creating initial GatewayProxy")

			time.Sleep(2 * time.Second)

			err = s.CreateResourceFromString(fmt.Sprintf(gatewayProxyTemplate, conflictingProxy, serviceName, secretName))
			Expect(err).Should(HaveOccurred(), "expecting conflict for duplicated GatewayProxy")
			Expect(err.Error()).To(ContainSubstring("gateway proxy configuration conflict"))
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("%s/%s", s.Namespace(), conflictingProxy)))
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("%s/%s", s.Namespace(), initialProxy)))
			Expect(err.Error()).To(ContainSubstring("Service"))
			Expect(err.Error()).To(ContainSubstring("AdminKey secret"))

			Expect(s.DeleteResource("GatewayProxy", initialProxy)).ShouldNot(HaveOccurred())
			Expect(s.DeleteResource("Service", serviceName)).ShouldNot(HaveOccurred())
			Expect(s.DeleteResource("Secret", secretName)).ShouldNot(HaveOccurred())
		})

		It("should reject GatewayProxy that overlaps endpoints when sharing inline AdminKey value", func() {
			gatewayProxyTemplate := `
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: %s
spec:
  provider:
    type: ControlPlane
    controlPlane:
      endpoints:
      - %s
      - %s
      auth:
        type: AdminKey
        adminKey:
          value: "%s"
`

			existingProxy := "gatewayproxy-inline-primary"
			conflictingProxy := "gatewayproxy-inline-conflict"
			endpointA := "https://127.0.0.1:9443"
			endpointB := "https://10.0.0.1:9443"
			endpointC := "https://192.168.0.1:9443"
			inlineKey := "inline-credential"

			err := s.CreateResourceFromString(fmt.Sprintf(gatewayProxyTemplate, existingProxy, endpointA, endpointB, inlineKey))
			Expect(err).ShouldNot(HaveOccurred(), "creating GatewayProxy with inline AdminKey")

			time.Sleep(2 * time.Second)

			err = s.CreateResourceFromString(fmt.Sprintf(gatewayProxyTemplate, conflictingProxy, endpointB, endpointC, inlineKey))
			Expect(err).Should(HaveOccurred(), "expecting conflict for overlapping endpoints with shared AdminKey")
			Expect(err.Error()).To(ContainSubstring("gateway proxy configuration conflict"))
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("%s/%s", s.Namespace(), conflictingProxy)))
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("%s/%s", s.Namespace(), existingProxy)))
			Expect(err.Error()).To(ContainSubstring("control plane endpoints"))
			Expect(err.Error()).To(ContainSubstring("inline AdminKey value"))
		})

		It("should reject GatewayProxy update that creates conflict with another GatewayProxy", func() {
			serviceTemplate := `
apiVersion: v1
kind: Service
metadata:
  name: %s
spec:
  selector:
    app: dummy-control-plane
  ports:
  - name: admin
    port: 9180
    targetPort: 9180
`
			secretTemplate := `
apiVersion: v1
kind: Secret
metadata:
  name: %s
type: Opaque
stringData:
  %s: %s
`
			sharedServiceName := "gatewayproxy-update-shared-service"
			sharedSecretName := "gatewayproxy-update-shared-secret"
			uniqueServiceName := "gatewayproxy-update-unique-service"
			proxyA := "gatewayproxy-update-a"
			proxyB := "gatewayproxy-update-b"

			Expect(s.CreateResourceFromString(fmt.Sprintf(serviceTemplate, sharedServiceName))).ShouldNot(HaveOccurred(), "creating shared Service")
			Expect(s.CreateResourceFromString(fmt.Sprintf(serviceTemplate, uniqueServiceName))).ShouldNot(HaveOccurred(), "creating unique Service")
			Expect(s.CreateResourceFromString(fmt.Sprintf(secretTemplate, sharedSecretName, "token", "value"))).ShouldNot(HaveOccurred(), "creating shared Secret")

			err := s.CreateResourceFromString(fmt.Sprintf(gatewayProxyTemplate, proxyA, sharedServiceName, sharedSecretName))
			Expect(err).ShouldNot(HaveOccurred(), "creating GatewayProxy A with shared Service and Secret")

			time.Sleep(2 * time.Second)

			err = s.CreateResourceFromString(fmt.Sprintf(gatewayProxyTemplate, proxyB, uniqueServiceName, sharedSecretName))
			Expect(err).ShouldNot(HaveOccurred(), "creating GatewayProxy B with unique Service but same Secret")

			time.Sleep(2 * time.Second)

			By("updating GatewayProxy B to use the same Service as GatewayProxy A, causing conflict")
			err = s.CreateResourceFromString(fmt.Sprintf(gatewayProxyTemplate, proxyB, sharedServiceName, sharedSecretName))
			Expect(err).Should(HaveOccurred(), "expecting conflict when updating to same Service")
			Expect(err.Error()).To(ContainSubstring("gateway proxy configuration conflict"))
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("%s/%s", s.Namespace(), proxyA)))
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("%s/%s", s.Namespace(), proxyB)))
		})
	})
})
