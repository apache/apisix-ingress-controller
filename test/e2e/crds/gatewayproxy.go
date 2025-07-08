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
	"net/http"
	"time"

	"github.com/gruntwork-io/terratest/modules/k8s"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/ptr"

	"github.com/apache/apisix-ingress-controller/internal/provider/adc"
	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test GatewayProxy", Label("apisix.apache.org", "v1alpha1", "gatewayproxy"), func() {
	var (
		s   = scaffold.NewDefaultScaffold()
		err error
	)

	const gatewayProxySpec = `
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
  plugins:
  - name: response-rewrite
    enabled: true
    config: 
      headers:
        "X-Pod-Hostname": "$hostname"
`
	const gatewayClassSpec = `
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: %s
spec:
  controllerName: %s
`
	const gatewaySpec = `
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: apisix
spec:
  gatewayClassName: %s
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
	const httpRouteSpec = `
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: apisix
  hostnames:
  - "httpbin.org"
  rules:
  - matches: 
    - path:
        type: Exact
        value: /get
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`
	BeforeEach(func() {
		By("create GatewayProxy")
		err = s.CreateResourceFromString(fmt.Sprintf(gatewayProxySpec, framework.ProviderType, s.AdminKey()))
		Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
		time.Sleep(time.Second)

		By("create GatewayClass")
		gatewayClassName := fmt.Sprintf("apisix-%d", time.Now().Unix())
		err = s.CreateResourceFromString(fmt.Sprintf(gatewayClassSpec, gatewayClassName, s.GetControllerName()))
		Expect(err).NotTo(HaveOccurred(), "creating GatewayClass")
		time.Sleep(time.Second)

		By("create Gateway")
		err = s.CreateResourceFromStringWithNamespace(fmt.Sprintf(gatewaySpec, gatewayClassName), s.Namespace())
		Expect(err).NotTo(HaveOccurred(), "creating Gateway")
		time.Sleep(time.Second)

		By("create HTTPRoute")
		s.ApplyHTTPRoute(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin"}, httpRouteSpec)

		Eventually(func() int {
			return s.NewAPISIXClient().GET("/get").WithHost("httpbin.org").Expect().Raw().StatusCode
		}).WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))
	})

	Context("Test GatewayProxy update configs", func() {
		It("scaling apisix pods to test that the controller watches endpoints", func() {
			By("scale apisix to replicas 2")
			s.Deployer.DeployDataplane(scaffold.DeployDataplaneOptions{
				Replicas: ptr.To(2),
			})

			By("check pod ready")
			err = wait.PollUntilContextTimeout(context.Background(), time.Second, 10*time.Second, true, func(ctx context.Context) (done bool, err error) {
				pods := s.GetPods(s.Namespace(), "app.kubernetes.io/name=apisix")
				if len(pods) != 2 {
					return false, nil
				}
				for _, pod := range pods {
					if pod.Status.PodIP == "" {
						return false, nil
					}
				}
				return true, nil
			})
			Expect(err).NotTo(HaveOccurred(), "check pods ready")

			By("request every pod to check configuration effect")
			pods := s.GetPods(s.Namespace(), "app.kubernetes.io/name=apisix")
			for i, pod := range pods {
				s.Logf("pod name: %s", pod.GetName())
				tunnel := k8s.NewTunnel(s.KubeOpts(), k8s.ResourceTypePod, pod.GetName(), 9080+i, 9080)
				err := tunnel.ForwardPortE(s.GinkgoT)
				Expect(err).NotTo(HaveOccurred(), "forward pod: %s", pod.Name)

				err = wait.PollUntilContextTimeout(context.Background(), time.Second, 30*time.Second, true, func(ctx context.Context) (done bool, err error) {
					resp := scaffold.NewClient("http", tunnel.Endpoint()).
						GET("/get").WithHost("httpbin.org").Expect().Raw()
					return resp.StatusCode == http.StatusOK && resp.Header.Get("X-Pod-Hostname") == pod.GetName(), nil
				})
				Expect(err).NotTo(HaveOccurred(), "request the pod: %s", pod.GetName())

				tunnel.Close()
			}
		})
	})

	Context("Backend server", func() {
		It("backend server on apisix/apisix-standalone mode", func() {
			var (
				keyword string
			)

			if framework.ProviderType == adc.BackendModeAPISIX {
				keyword = fmt.Sprintf(`{"config.ServerAddrs": ["%s"]}`, s.Deployer.GetAdminEndpoint())
			} else {
				keyword = fmt.Sprintf(`{"config.ServerAddrs": ["http://%s:9180"]}`, s.GetPodIP(s.Namespace(), "app.kubernetes.io/name=apisix"))
			}

			By(fmt.Sprintf("wait for keyword: %s", keyword))
			s.WaitControllerManagerLog(s.Namespace(), keyword, 60, time.Minute)
		})
	})
})
