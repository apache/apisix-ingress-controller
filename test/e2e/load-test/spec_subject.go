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

package load

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/api7/gopkg/pkg/log"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

const gatewayProxyYaml = `
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: apisix-proxy-config
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
          value: "%s"
`

const ingressClassYaml = `
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
    namespace: %s
    scope: "Namespace"
`

var _ = Describe("Load Test", func() {
	var (
		s = scaffold.NewScaffold(&scaffold.Options{
			ControllerName: "apisix.apache.org/apisix-ingress-controller",
		})
		controlAPIClient scaffold.ControlAPIClient
		err              error
	)

	BeforeEach(func() {
		By("create GatewayProxy")
		gatewayProxy := fmt.Sprintf(gatewayProxyYaml, framework.ProviderType, s.AdminKey())
		err = s.CreateResourceFromStringWithNamespace(gatewayProxy, s.Namespace())
		Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
		time.Sleep(5 * time.Second)

		By("create IngressClass")
		err = s.CreateResourceFromStringWithNamespace(fmt.Sprintf(ingressClassYaml, s.Namespace()), "")
		Expect(err).NotTo(HaveOccurred(), "creating IngressClass")
		time.Sleep(5 * time.Second)

		By("port-forward to control api service")
		controlAPIClient, err = s.ControlAPIClient()
		Expect(err).NotTo(HaveOccurred(), "create control api client")
	})

	Context("Load Test 2000 ApisixRoute", func() {
		It("test 2000 ApisixRoute", func() {
			const total = 2000

			const apisixRouteSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: %s
spec:
  ingressClassName: apisix
  http:
  - name: rule0
    match:
      paths:
      - /get
      exprs:
      - subject:
          scope: Header
          name: X-Route-Name
        op: Equal
        value: %s
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
`

			By(fmt.Sprintf("prepare %d ApisixRoutes", total))
			var text = bytes.NewBuffer(nil)
			for i := range total {
				name := getRouteName(i)
				_, err := fmt.Fprintf(text, apisixRouteSpec, name, name)
				Expect(err).NotTo(HaveOccurred())
				text.WriteString("\n---\n")
			}
			err := s.CreateResourceFromString(text.String())
			Expect(err).NotTo(HaveOccurred(), "creating ApisixRoutes")

			var (
				results []TestResult
				now     = time.Now()
			)
			By("Test the time required for applying a large number of ApisixRoutes to take effect")
			var times int
			err = wait.PollUntilContextTimeout(context.Background(), time.Second, 10*time.Minute, true, func(ctx context.Context) (done bool, err error) {
				times++
				results, _, err := controlAPIClient.ListServices()
				if err != nil {
					log.Errorw("failed to ListServices", zap.Error(err))
					return false, nil
				}
				if len(results) != total {
					log.Debugw("number of effective services", zap.Int("number", len(results)), zap.Int("times", times))
					return false, nil
				}
				return len(results) == total, nil
			})
			Expect(err).ShouldNot(HaveOccurred())
			results = append(results, TestResult{
				CaseName: fmt.Sprintf("Apply %d ApisixRoutes", total),
				CostTime: time.Since(now),
			})

			By("Test the time required for an ApisixRoute update to take effect")
			var apisixRouteSpec0 = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: %s
spec:
  ingressClassName: apisix
  http:
  - name: rule0
    match:
      paths:
      - /headers
      exprs:
      - subject:
          scope: Header
          name: X-Route-Name
        op: Equal
        value: %s
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
`
			name := getRouteName(10)
			err = s.CreateResourceFromString(fmt.Sprintf(apisixRouteSpec0, name, name))
			Expect(err).NotTo(HaveOccurred())
			now = time.Now()
			Eventually(func() int {
				return s.NewAPISIXClient().GET("/headers").WithHeader("X-Route-Name", name).Expect().Raw().StatusCode
			}).WithTimeout(time.Minute).ProbeEvery(100 * time.Millisecond).Should(Equal(http.StatusOK))
			results = append(results, TestResult{
				CaseName: fmt.Sprintf("Update a single ApisixRoute base on %d ApisixRoutes", total),
				CostTime: time.Since(now),
			})

			PrintResults(results)
		})
	})
})

func getRouteName(i int) string {
	return fmt.Sprintf("test-route-%04d", i)
}

type TestResult struct {
	CaseName string
	CostTime time.Duration
}

func (tr TestResult) String() string {
	return fmt.Sprintf("%s takes effect for %s", tr.CaseName, tr.CostTime)
}

func PrintResults(results []TestResult) {
	fmt.Printf("\n======================TEST RESULT ProviderSyncPeriod %s===============================\n", framework.ProviderSyncPeriod)
	fmt.Printf("%-70s", "Test Case")
	fmt.Printf("%-70s\n", "Time Required")
	fmt.Printf("%-70s\n", "--------------------------------------------------------------------------------------")
	for _, result := range results {
		fmt.Printf("%-70s", result.CaseName)
		fmt.Printf("%-70s\n", result.CostTime)
	}
	fmt.Println("======================================================================================")
	fmt.Println()
}
