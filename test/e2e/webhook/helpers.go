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

type routeWebhookTestCase struct {
	routeKind       string
	routeName       string
	missingService  string
	mirrorService   string
	servicePortName string
	servicePort     int
}

func setupGatewayResources(s *scaffold.Scaffold) {
	By("creating GatewayProxy")
	err := s.CreateResourceFromString(s.GetGatewayProxySpec())
	Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
	time.Sleep(5 * time.Second)

	By("creating GatewayClass")
	err = s.CreateResourceFromString(s.GetGatewayClassYaml())
	Expect(err).NotTo(HaveOccurred(), "creating GatewayClass")
	time.Sleep(2 * time.Second)

	By("creating Gateway")
	err = s.CreateResourceFromString(s.GetGatewayYaml())
	Expect(err).NotTo(HaveOccurred(), "creating Gateway")
	time.Sleep(5 * time.Second)
}

func verifyMissingBackendWarnings(s *scaffold.Scaffold, tc routeWebhookTestCase) {
	gatewayName := s.Namespace()
	routeYAML := fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1
kind: %s
metadata:
  name: %s
spec:
  parentRefs:
  - name: %s
  rules:
  - backendRefs:
    - name: %s
      port: %d
    filters:
    - type: RequestMirror
      requestMirror:
        backendRef:
          name: %s
          port: %d
`, tc.routeKind, tc.routeName, gatewayName, tc.missingService, tc.servicePort, tc.mirrorService, tc.servicePort)

	missingBackendWarning := fmt.Sprintf("Warning: Referenced Service '%s/%s' not found", gatewayName, tc.missingService)
	mirrorBackendWarning := fmt.Sprintf("Warning: Referenced Service '%s/%s' not found", gatewayName, tc.mirrorService)

	output, err := s.CreateResourceFromStringAndGetOutput(routeYAML)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(output).To(ContainSubstring(missingBackendWarning))
	Expect(output).To(ContainSubstring(mirrorBackendWarning))

	By("delete the " + tc.routeKind)
	err = s.DeleteResource(tc.routeKind, tc.routeName)
	Expect(err).NotTo(HaveOccurred())
	time.Sleep(2 * time.Second)

	By(fmt.Sprintf("creating referenced backend services for %s", tc.routeKind))
	serviceYAML := `
apiVersion: v1
kind: Service
metadata:
  name: %s
spec:
  selector:
    app: placeholder
  ports:
  - name: %s
    port: %d
    targetPort: %d
  type: ClusterIP
`

	backendService := fmt.Sprintf(serviceYAML, tc.missingService, tc.servicePortName, tc.servicePort, tc.servicePort)
	err = s.CreateResourceFromString(backendService)
	Expect(err).NotTo(HaveOccurred(), "creating primary backend service")

	mirrorService := fmt.Sprintf(serviceYAML, tc.mirrorService, tc.servicePortName, tc.servicePort, tc.servicePort)
	err = s.CreateResourceFromString(mirrorService)
	Expect(err).NotTo(HaveOccurred(), "creating mirror backend service")

	time.Sleep(2 * time.Second)

	output, err = s.CreateResourceFromStringAndGetOutput(routeYAML)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(output).NotTo(ContainSubstring(missingBackendWarning))
	Expect(output).NotTo(ContainSubstring(mirrorBackendWarning))
}
