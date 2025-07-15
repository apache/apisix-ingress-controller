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

package apisix

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test ApisixRoute", Label("apisix.apache.org", "v2", "apisixroute"), func() {
	var (
		s = scaffold.NewScaffold(&scaffold.Options{
			ControllerName: "apisix.apache.org/apisix-ingress-controller",
		})
		applier = framework.NewApplier(s.GinkgoT, s.K8sClient, s.CreateResourceFromString)
	)

	BeforeEach(func() {
		By("create GatewayProxy")
		gatewayProxy := fmt.Sprintf(gatewayProxyYaml, s.Deployer.GetAdminEndpoint(), s.AdminKey())
		err := s.CreateResourceFromStringWithNamespace(gatewayProxy, "default")
		Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
		time.Sleep(5 * time.Second)

		By("create IngressClass")
		err = s.CreateResourceFromStringWithNamespace(ingressClassYaml, "")
		Expect(err).NotTo(HaveOccurred(), "creating IngressClass")
		time.Sleep(5 * time.Second)
	})

	Context("Test ApisixRoute", func() {

		It("Basic tests", func() {
			const apisixRouteSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: default
spec:
  ingressClassName: apisix
  http:
  - name: rule0
    match:
      hosts:
      - httpbin
      paths:
      - %s
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
`
			request := func(path string) int {
				return s.NewAPISIXClient().GET(path).WithHost("httpbin").Expect().Raw().StatusCode
			}

			By("apply ApisixRoute")
			var apisixRoute apiv2.ApisixRoute
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"}, &apisixRoute, fmt.Sprintf(apisixRouteSpec, "/get"))

			By("verify ApisixRoute works")
			Eventually(request).WithArguments("/get").WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))

			By("update ApisixRoute")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"}, &apisixRoute, fmt.Sprintf(apisixRouteSpec, "/headers"))
			Eventually(request).WithArguments("/get").WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusNotFound))
			s.NewAPISIXClient().GET("/headers").WithHost("httpbin").Expect().Status(http.StatusOK)

			By("delete ApisixRoute")
			err := s.DeleteResource("ApisixRoute", "default")
			Expect(err).ShouldNot(HaveOccurred(), "deleting ApisixRoute")
			Eventually(request).WithArguments("/headers").WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusNotFound))
		})

		It("Test plugins in ApisixRoute", func() {
			const apisixRouteSpecPart0 = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: default
spec:
  ingressClassName: apisix
  http:
  - name: rule0
    match:
      paths:
      - /*
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
`
			const apisixRouteSpecPart1 = ` 
    plugins:
    - name: response-rewrite
      enable: true
      config:
        headers:
          X-Global-Rule: "test-response-rewrite"
          X-Global-Test: "enabled"
`
			By("apply ApisixRoute without plugins")
			var apisixRoute apiv2.ApisixRoute
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"}, &apisixRoute, apisixRouteSpecPart0)

			By("verify ApisixRoute works")
			request := func() int {
				return s.NewAPISIXClient().GET("/get").Expect().Raw().StatusCode
			}
			Eventually(request).WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))

			By("apply ApisixRoute with plugins")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"}, &apisixRoute, apisixRouteSpecPart0+apisixRouteSpecPart1)
			time.Sleep(5 * time.Second)

			By("verify plugin works")
			resp := s.NewAPISIXClient().GET("/get").Expect().Status(http.StatusOK)
			resp.Header("X-Global-Rule").IsEqual("test-response-rewrite")
			resp.Header("X-Global-Test").IsEqual("enabled")

			By("remove plugin")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"}, &apisixRoute, apisixRouteSpecPart0)
			time.Sleep(5 * time.Second)

			By("verify no plugin works")
			resp = s.NewAPISIXClient().GET("/get").Expect().Status(http.StatusOK)
			resp.Header("X-Global-Rule").IsEmpty()
			resp.Header("X-Global-Test").IsEmpty()
		})

		It("Test ApisixRoute match by vars", func() {
			const apisixRouteSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: default
spec:
  ingressClassName: apisix
  http:
  - name: rule0
    match:
      paths:
      - /*
      exprs:
      - subject:
          scope: Header
          name: X-Foo
        op: Equal
        value: bar
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
`
			By("apply ApisixRoute")
			var apisixRoute apiv2.ApisixRoute
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"}, &apisixRoute, apisixRouteSpec)

			By("verify ApisixRoute works")
			request := func() int {
				return s.NewAPISIXClient().GET("/get").
					WithHeader("X-Foo", "bar").
					Expect().Raw().StatusCode
			}
			Eventually(request).WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))
			s.NewAPISIXClient().GET("/get").Expect().Status(http.StatusNotFound)
		})

		It("Test ApisixRoute filterFunc", func() {
			const apisixRouteSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: default
spec:
  ingressClassName: apisix
  http:
  - name: rule0
    match:
      paths:
      - /*
      filter_func: "function(vars)\n  local core = require ('apisix.core')\n  local body, err = core.request.get_body()\n  if not body then\n      return false\n  end\n\n  local data, err = core.json.decode(body)\n  if not data then\n      return false\n  end\n\n  if data['foo'] == 'bar' then\n      return true\n  end\n\n  return false\nend"
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
`
			By("apply ApisixRoute")
			var apisixRoute apiv2.ApisixRoute
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"}, &apisixRoute, apisixRouteSpec)

			By("verify ApisixRoute works")
			request := func() int {
				return s.NewAPISIXClient().GET("/get").
					WithJSON(map[string]string{"foo": "bar"}).
					Expect().Raw().StatusCode
			}
			Eventually(request).WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))
			s.NewAPISIXClient().GET("/get").Expect().Status(http.StatusNotFound)
		})

		It("Test ApisixRoute service not found", func() {
			const apisixRouteSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: default
spec:
  ingressClassName: apisix
  http:
  - name: rule0
    match:
      hosts:
      - httpbin
      paths:
      - %s
    backends:
    - serviceName: service-not-found
      servicePort: 80
`
			request := func(path string) int {
				return s.NewAPISIXClient().GET(path).WithHost("httpbin").Expect().Raw().StatusCode
			}

			By("apply ApisixRoute")
			var apisixRoute apiv2.ApisixRoute
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"}, &apisixRoute, fmt.Sprintf(apisixRouteSpec, "/get"))

			Eventually(request).WithArguments("/get").WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusServiceUnavailable))
		})

		It("Test ApisixRoute resolveGranularity", func() {
			const apisixRouteSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: default
spec:
  ingressClassName: apisix
  http:
  - name: rule0
    match:
      paths:
      - /*
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
      resolveGranularity: service
    plugins:
    - name: response-rewrite
      enable: true
      config:
        headers:
          set:
            "X-Upstream-IP": "$upstream_addr"
`
			By("apply ApisixRoute")
			var apisixRoute apiv2.ApisixRoute
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"}, &apisixRoute, apisixRouteSpec)

			By("verify ApisixRoute works")
			request := func() int {
				return s.NewAPISIXClient().GET("/get").Expect().Raw().StatusCode
			}
			Eventually(request).WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))

			By("assert that the request is proxied to the Service ClusterIP")
			service, err := s.GetServiceByName("httpbin-service-e2e-test")
			Expect(err).ShouldNot(HaveOccurred(), "get service")
			clusterIP := net.JoinHostPort(service.Spec.ClusterIP, "80")
			s.NewAPISIXClient().GET("/get").Expect().Header("X-Upstream-IP").IsEqual(clusterIP)
		})

		It("Test ApisixRoute subset", func() {
			const apisixRouteSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: default
spec:
  ingressClassName: apisix
  http:
  - name: rule0
    match:
      hosts:
      - httpbin
      paths:
      - /*
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
      subset: test-subset
`
			const apisixUpstreamSpec0 = `
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: httpbin-service-e2e-test
spec:
  ingressClassName: apisix
  subsets:
  - name: test-subset
    labels:
      unknown-key: unknown-value
`
			const apisixUpstreamSpec1 = `
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: httpbin-service-e2e-test
spec:
  ingressClassName: apisix
  subsets:
  - name: test-subset
    labels:
      app: httpbin-deployment-e2e-test
`
			request := func() int {
				return s.NewAPISIXClient().GET("/get").WithHost("httpbin").Expect().Raw().StatusCode
			}
			By("apply ApisixRoute")
			var apisixRoute apiv2.ApisixRoute
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"}, &apisixRoute, apisixRouteSpec)
			Eventually(request).WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))

			// no pod matches the subset label "unknown-key: unknown-value" so there will be no node in the upstream,
			// to request the route will get http.StatusServiceUnavailable
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin-service-e2e-test"}, new(apiv2.ApisixUpstream), apisixUpstreamSpec0)
			Eventually(request).WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusServiceUnavailable))

			// the pod matches the subset label "app: httpbin-deployment-e2e-test",
			// to request the route will be OK
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin-service-e2e-test"}, new(apiv2.ApisixUpstream), apisixUpstreamSpec1)
			Eventually(request).WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))
		})

		It("Multiple ApisixRoute with same prefix name", func() {
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
      hosts:
      - %s
      paths:
      - /*
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
`
			By("apply ApisixRoute")
			var apisixRoute apiv2.ApisixRoute
			for _, id := range []string{"11111", "1111", "111", "11", "1"} {
				name := fmt.Sprintf("route-%s", id)
				host := fmt.Sprintf("httpbin-%s", id)
				applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: name}, &apisixRoute, fmt.Sprintf(apisixRouteSpec, name, host))
			}

			By("verify ApisixRoute works")
			for _, id := range []string{"1", "11", "111", "1111", "11111"} {
				host := fmt.Sprintf("httpbin-%s", id)
				Eventually(func() int {
					return s.NewAPISIXClient().GET("/get").WithHost(host).Expect().Raw().StatusCode
				}).WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))
			}
		})
	})

	Context("Test ApisixRoute reference ApisixUpstream", func() {
		It("Test reference ApisixUpstream", func() {
			const apisixRouteSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: default
spec:
  ingressClassName: apisix
  http:
  - name: rule0
    match:
      paths:
      - /*
    upstreams:
    - name: default-upstream
`
			const apisixUpstreamSpec0 = `
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: default-upstream
spec:
  ingressClassName: apisix
  externalNodes:
  - type: Service
    name: httpbin-service-e2e-test
`
			const apisixUpstreamSpec1 = `
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: default-upstream
spec:
  ingressClassName: apisix
  externalNodes:
  - type: Service
    name: alias-httpbin-service-e2e-test
`
			const serviceSpec = `
apiVersion: v1
kind: Service
metadata:
  name: alias-httpbin-service-e2e-test
spec:
  type: ExternalName
  externalName: httpbin-service-e2e-test
`
			By("create Service, ApisixUpstream and ApisixRoute")
			err := s.CreateResourceFromString(serviceSpec)
			Expect(err).ShouldNot(HaveOccurred(), "apply service")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default-upstream"}, new(apiv2.ApisixUpstream), apisixUpstreamSpec0)

			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"}, new(apiv2.ApisixRoute), apisixRouteSpec)

			By("verify that the ApisixUpstream reference a Service which is not ExternalName should not request OK")
			request := func(path string) int {
				return s.NewAPISIXClient().GET(path).WithHost("httpbin").Expect().Raw().StatusCode
			}
			Eventually(request).WithArguments("/get").WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusServiceUnavailable))

			By("verify that ApisixUpstream reference a Service which is ExternalName should request OK")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default-upstream"}, new(apiv2.ApisixUpstream), apisixUpstreamSpec1)
			Eventually(request).WithArguments("/get").WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))
		})

		It("Test a Mix of Backends and Upstreams", func() {
			// apisixUpstreamSpec is an ApisixUpstream reference to the Service httpbin-service-e2e-test
			const apisixUpstreamSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: default-upstream
spec:
  ingressClassName: apisix
  externalNodes:
  - type: Domain
    name: httpbin-service-e2e-test
  passHost: node
`
			// apisixRouteSpec is an ApisixUpstream uses a backend and reference an upstream.
			// It contains a plugin response-rewrite that lets us know what upstream the gateway forwards the request to.
			const apisixRouteSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: default
spec:
  ingressClassName: apisix
  http:
  - name: rule0
    match:
      paths:
      - /*
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
    upstreams:
    - name: default-upstream
    plugins:
    - name: response-rewrite
      enable: true
      config:
        headers:
          set:
            "X-Upstream-Host": "$upstream_addr"
`
			By("apply ApisixUpstream")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default-upstream"}, new(apiv2.ApisixUpstream), apisixUpstreamSpec)

			By("apply ApisixRoute")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"}, new(apiv2.ApisixRoute), apisixRouteSpec)

			By("verify ApisixRoute works")
			request := func(path string) int {
				return s.NewAPISIXClient().GET(path).Expect().Raw().StatusCode
			}
			Eventually(request).WithArguments("/get").WithTimeout(8 * time.Second).ProbeEvery(time.Second).Should(Equal(http.StatusOK))

			By("verify the backends and the upstreams work commonly")
			// .backends -> Service httpbin-service-e2e-test -> Endpoint httpbin-service-e2e-test, so the $upstream_addr value we get is the Endpoint IP.
			// .upstreams -> Service httpbin-service-e2e-test, so the $upstream_addr value we get is the Service ClusterIP.
			var upstreamAddrs = make(map[string]struct{})
			for range 10 {
				upstreamAddr := s.NewAPISIXClient().GET("/get").Expect().Raw().Header.Get("X-Upstream-Host")
				upstreamAddrs[upstreamAddr] = struct{}{}
			}

			endpoints, err := s.GetServiceEndpoints(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin-service-e2e-test"})
			Expect(err).ShouldNot(HaveOccurred(), "get endpoints")
			Expect(endpoints).Should(HaveLen(1))
			endpoint := net.JoinHostPort(endpoints[0], "80")

			service, err := s.GetServiceByName("httpbin-service-e2e-test")
			Expect(err).ShouldNot(HaveOccurred(), "get service")
			clusterIP := net.JoinHostPort(service.Spec.ClusterIP, "80")

			Expect(upstreamAddrs).Should(HaveLen(2))
			Eventually(upstreamAddrs).Should(HaveKey(endpoint))
			Eventually(upstreamAddrs).Should(HaveKey(clusterIP))
		})

		It("Test backend implicit reference to apisixupstream", func() {
			var err error

			const apisixRouteSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: default
spec:
  ingressClassName: apisix
  http:
  - name: rule0
    match:
      hosts:
      - httpbin
      paths:
      - /*
    backends:
    - serviceName: httpbin-service-e2e-test
      servicePort: 80
    plugins:
    - name: response-rewrite
      enable: true
      config:
        headers:
          set:
            "X-Upstream-Host": "$upstream_host"

`
			const apisixUpstreamSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixUpstream
metadata:
  name: httpbin-service-e2e-test
spec:
  ingressClassName: apisix
  passHost: rewrite
  upstreamHost: hello.httpbin.org
  loadbalancer:
    type: "chash"
    hashOn: "vars"
    key: "server_name"
`
			expectUpstreamHostIs := func(expectedUpstreamHost string) func(ctx context.Context) (bool, error) {
				return func(ctx context.Context) (done bool, err error) {
					resp := s.NewAPISIXClient().GET("/get").WithHost("httpbin").Expect().Raw()
					return resp.StatusCode == http.StatusOK && resp.Header.Get("X-Upstream-Host") == expectedUpstreamHost, nil
				}
			}

			By("apply apisixroute")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "default"}, new(apiv2.ApisixRoute), apisixRouteSpec)

			By("verify ApisixRoute works")
			// expect upstream host is "httpbin"
			err = wait.PollUntilContextTimeout(context.Background(), time.Second, 10*time.Second, true, expectUpstreamHostIs("httpbin"))
			Expect(err).ShouldNot(HaveOccurred(), "verify ApisixRoute works")

			By("apply apisixupstream")
			applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin-service-e2e-test"}, new(apiv2.ApisixUpstream), apisixUpstreamSpec)

			By("verify backend implicit reference to apisixupstream works")
			// expect upstream host is "hello.httpbin.org" which is rewritten by the apisixupstream
			err = wait.PollUntilContextTimeout(context.Background(), time.Second, 10*time.Second, true, expectUpstreamHostIs("hello.httpbin.org"))
			Expect(err).ShouldNot(HaveOccurred(), "check apisixupstream is referenced")
		})
	})
})
