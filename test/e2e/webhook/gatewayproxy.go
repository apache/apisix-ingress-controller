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

	It("should warn on missing service or secret references", func() {
		missingService := "missing-control-plane"
		missingSecret := "missing-admin-secret"
		gpName := "webhook-gateway-proxy"
		gpWithSecrets := `
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

		output, err := s.CreateResourceFromStringAndGetOutput(fmt.Sprintf(gpWithSecrets, gpName, missingService, missingSecret))
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

		output, err = s.CreateResourceFromStringAndGetOutput(fmt.Sprintf(gpWithSecrets, gpName, missingService, missingSecret))
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

		output, err = s.CreateResourceFromStringAndGetOutput(fmt.Sprintf(gpWithSecrets, gpName, missingService, missingSecret))
		Expect(err).ShouldNot(HaveOccurred())
		Expect(output).NotTo(ContainSubstring(fmt.Sprintf("Warning: Referenced Service '%s/%s' not found", s.Namespace(), missingService)))
		Expect(output).NotTo(ContainSubstring(fmt.Sprintf("Warning: Secret key 'token' not found in Secret '%s/%s'", s.Namespace(), missingSecret)))
	})
})
