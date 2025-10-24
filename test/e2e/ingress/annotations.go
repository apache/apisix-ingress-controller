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
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/utils/ptr"

	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test Ingress With Annotations", Label("networking.k8s.io", "ingress"), func() {
	s := scaffold.NewDefaultScaffold()

	Context("Upstream", func() {
		var (
			ingressRetries = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: retries
  annotations:
    k8s.apisix.apache.org/upstream-retries: "3"
spec:
  ingressClassName: %s
  rules:
  - host: nginx.example
    http:
      paths:
      - path: /get
        pathType: Exact
        backend:
          service:
            name: nginx
            port:
              number: 80
`
			ingressSchemeHTTPS = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: https-backend
  annotations:
    k8s.apisix.apache.org/upstream-scheme: https
spec:
  ingressClassName: %s
  rules:
  - host: nginx.example
    http:
      paths:
      - path: /get
        pathType: Exact
        backend:
          service:
            name: nginx
            port:
              number: 7443
`

			ingressTimeouts = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: timeouts
  annotations:
    k8s.apisix.apache.org/upstream-read-timeout: "2s"
    k8s.apisix.apache.org/upstream-send-timeout: "3s"
    k8s.apisix.apache.org/upstream-connect-timeout: "4s"
spec:
  ingressClassName: %s
  rules:
  - host: nginx.example
    http:
      paths:
      - path: /delay
        pathType: Exact
        backend:
          service:
            name: nginx
            port:
              number: 443
`
		)
		BeforeEach(func() {
			s.DeployNginx(framework.NginxOptions{
				Namespace: s.Namespace(),
				Replicas:  ptr.To(int32(1)),
			})
			By("create GatewayProxy")
			Expect(s.CreateResourceFromString(s.GetGatewayProxySpec())).NotTo(HaveOccurred(), "creating GatewayProxy")

			By("create IngressClass")
			err := s.CreateResourceFromStringWithNamespace(s.GetIngressClassYaml(), "")
			Expect(err).NotTo(HaveOccurred(), "creating IngressClass")
			time.Sleep(5 * time.Second)
		})
		It("retries", func() {
			Expect(s.CreateResourceFromString(fmt.Sprintf(ingressRetries, s.Namespace()))).ShouldNot(HaveOccurred(), "creating Ingress")

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "nginx.example",
				Check:  scaffold.WithExpectedStatus(http.StatusOK),
			})
			upstreams, err := s.DefaultDataplaneResource().Upstream().List(context.Background())
			Expect(err).NotTo(HaveOccurred(), "listing Upstream")
			Expect(upstreams).To(HaveLen(1), "checking Upstream length")
			Expect(upstreams[0].Retries).To(Equal(ptr.To(int64(3))), "checking Upstream retries")
		})
		It("scheme", func() {
			Expect(s.CreateResourceFromString(fmt.Sprintf(ingressSchemeHTTPS, s.Namespace()))).ShouldNot(HaveOccurred(), "creating Ingress")

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "nginx.example",
				Check:  scaffold.WithExpectedStatus(http.StatusOK),
			})
			upstreams, err := s.DefaultDataplaneResource().Upstream().List(context.Background())
			Expect(err).NotTo(HaveOccurred(), "listing Upstream")
			Expect(upstreams).To(HaveLen(1), "checking Upstream length")
			Expect(upstreams[0].Scheme).To(Equal("https"), "checking Upstream scheme")
		})
		It("timeouts", func() {
			Expect(s.CreateResourceFromString(fmt.Sprintf(ingressTimeouts, s.Namespace()))).ShouldNot(HaveOccurred(), "creating Ingress")

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/delay",
				Host:   "nginx.example",
				Check:  scaffold.WithExpectedStatus(http.StatusOK),
			})

			_ = s.NewAPISIXClient().GET("/delay").WithQuery("delay", "10").
				WithHost("nginx.example").Expect().Status(http.StatusGatewayTimeout)

			_ = s.NewAPISIXClient().GET("/delay").WithHost("nginx.example").Expect().Status(http.StatusOK)

			upstreams, err := s.DefaultDataplaneResource().Upstream().List(context.Background())
			Expect(err).NotTo(HaveOccurred(), "listing Upstream")
			Expect(upstreams).To(HaveLen(1), "checking Upstream length")
			Expect(upstreams[0].Timeout).ToNot(BeNil(), "checking Upstream timeout")
			Expect(upstreams[0].Timeout.Read).To(Equal(2), "checking Upstream read timeout")
			Expect(upstreams[0].Timeout.Send).To(Equal(3), "checking Upstream send timeout")
			Expect(upstreams[0].Timeout.Connect).To(Equal(4), "checking Upstream connect timeout")
		})
	})
})
