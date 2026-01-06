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

package v2

import (
	"context"
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Issue 2689: Service Inline Upstream Not Updated on Endpoint Changes", Label("apisix.apache.org", "v2", "apisixroute", "issue-2689"), func() {
	var (
		s       = scaffold.NewDefaultScaffold()
		applier = framework.NewApplier(s.GinkgoT, s.K8sClient, s.CreateResourceFromString)
	)

	BeforeEach(func() {
		By("create GatewayProxy")
		err := s.CreateResourceFromString(s.GetGatewayProxySpec())
		Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
		time.Sleep(5 * time.Second)

		By("create IngressClass")
		err = s.CreateResourceFromStringWithNamespace(s.GetIngressClassYaml(), "")
		Expect(err).NotTo(HaveOccurred(), "creating IngressClass")
		time.Sleep(5 * time.Second)
	})

	It("Service inline upstream nodes should be updated when Pod IP changes", func() {
		const apisixRouteSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: issue-2689-test
  namespace: %s
spec:
  ingressClassName: %s
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
`

		By("apply ApisixRoute")
		var apisixRoute apiv2.ApisixRoute
		applier.MustApplyAPIv2(types.NamespacedName{Namespace: s.Namespace(), Name: "issue-2689-test"},
			&apisixRoute, fmt.Sprintf(apisixRouteSpec, s.Namespace(), s.Namespace()))

		By("verify ApisixRoute works")
		s.RequestAssert(&scaffold.RequestAssert{
			Method: "GET",
			Path:   "/get",
			Host:   "httpbin",
			Check:  scaffold.WithExpectedStatus(http.StatusOK),
		})

		By("get initial Kubernetes Service endpoints")
		initialEndpoints, err := s.GetServiceEndpoints(types.NamespacedName{
			Namespace: s.Namespace(),
			Name:      "httpbin-service-e2e-test",
		})
		Expect(err).NotTo(HaveOccurred(), "getting initial service endpoints")
		Expect(initialEndpoints).NotTo(BeEmpty(), "initial endpoints should not be empty")
		initialPodIP := initialEndpoints[0]
		GinkgoWriter.Printf("Initial Pod IP: %s\n", initialPodIP)

		By("get initial APISIX Service configuration")
		var initialService *adctypes.Service
		var serviceName string
		err = wait.PollUntilContextTimeout(context.Background(), 2*time.Second, 30*time.Second, true, func(ctx context.Context) (done bool, err error) {
			services, err := s.DefaultDataplaneResource().Service().List(ctx)
			if err != nil {
				return false, err
			}
			if len(services) == 0 {
				return false, nil
			}
			// Find the service that matches our route
			// Service name should contain the namespace and route name
			for _, svc := range services {
				if svc.Upstream != nil && len(svc.Upstream.Nodes) > 0 {
					// Check if this service's upstream nodes match the initial endpoint
					for _, node := range svc.Upstream.Nodes {
						if node.Host == initialPodIP {
							serviceName = svc.Name
							initialService = svc
							GinkgoWriter.Printf("Found matching Service: Name=%s\n", serviceName)
							GinkgoWriter.Printf("Initial Service inline upstream nodes: %v\n", svc.Upstream.Nodes)
							return true, nil
						}
					}
				}
			}
			return false, nil
		})
		Expect(err).NotTo(HaveOccurred(), "finding initial APISIX service")
		Expect(initialService).NotTo(BeNil(), "initial service should be found")

		// Record initial upstream nodes
		initialUpstreamNodes := make(map[string]int) // host -> port
		if initialService.Upstream != nil {
			for _, node := range initialService.Upstream.Nodes {
				initialUpstreamNodes[node.Host] = node.Port
			}
		}
		GinkgoWriter.Printf("Initial upstream nodes: %v\n", initialUpstreamNodes)

		By("scale httpbin deployment to 0 to trigger pod deletion")
		err = s.ScaleHTTPBIN(0)
		Expect(err).NotTo(HaveOccurred(), "scaling httpbin deployment to 0")

		By("wait for endpoints to be empty")
		err = wait.PollUntilContextTimeout(context.Background(), 1*time.Second, 30*time.Second, true, func(ctx context.Context) (done bool, err error) {
			endpoints, err := s.GetServiceEndpoints(types.NamespacedName{
				Namespace: s.Namespace(),
				Name:      "httpbin-service-e2e-test",
			})
			if err != nil {
				return false, err
			}
			return len(endpoints) == 0, nil
		})
		Expect(err).NotTo(HaveOccurred(), "waiting for endpoints to be empty")

		By("scale httpbin deployment to 1 to trigger new pod creation")
		err = s.ScaleHTTPBIN(1)
		Expect(err).NotTo(HaveOccurred(), "scaling httpbin deployment to 1")

		By("wait for new pod to be ready and get new endpoint IP")
		var newPodIP string
		err = wait.PollUntilContextTimeout(context.Background(), 2*time.Second, 60*time.Second, true, func(ctx context.Context) (done bool, err error) {
			endpoints, err := s.GetServiceEndpoints(types.NamespacedName{
				Namespace: s.Namespace(),
				Name:      "httpbin-service-e2e-test",
			})
			if err != nil {
				return false, err
			}
			if len(endpoints) == 0 {
				return false, nil
			}
			newPodIP = endpoints[0]
			// Verify that the new IP is different from the old one
			if newPodIP != initialPodIP {
				GinkgoWriter.Printf("New Pod IP: %s (different from initial: %s)\n", newPodIP, initialPodIP)
				return true, nil
			}
			return false, nil
		})
		Expect(err).NotTo(HaveOccurred(), "waiting for new pod IP")
		Expect(newPodIP).NotTo(Equal(initialPodIP), "new pod IP should be different from initial IP")

		By("wait for controller sync period (default 1m) plus some buffer")
		// Wait for sync period to ensure controller has time to sync
		time.Sleep(70 * time.Second)

		By("verify APISIX Service inline upstream nodes are updated")
		err = wait.PollUntilContextTimeout(context.Background(), 5*time.Second, 2*time.Minute, true, func(ctx context.Context) (done bool, err error) {
			services, err := s.DefaultDataplaneResource().Service().List(ctx)
			if err != nil {
				GinkgoWriter.Printf("Error listing services: %v\n", err)
				return false, err
			}

			// Find the service by ID or name
			for _, svc := range services {
				if svc.Name == serviceName {
					if svc.Upstream == nil {
						GinkgoWriter.Printf("Service %s has nil upstream\n", svc.ID)
						return false, nil
					}
					if len(svc.Upstream.Nodes) == 0 {
						GinkgoWriter.Printf("Service %s has empty upstream nodes\n", svc.ID)
						return false, nil
					}

					// Check if any node matches the new pod IP
					foundNewIP := false
					stillHasOldIP := false
					currentNodes := make(map[string]int)
					for _, node := range svc.Upstream.Nodes {
						currentNodes[node.Host] = node.Port
						if node.Host == newPodIP {
							foundNewIP = true
						}
						if node.Host == initialPodIP {
							stillHasOldIP = true
						}
					}

					GinkgoWriter.Printf("Service %s current upstream nodes: %v\n", svc.ID, currentNodes)
					GinkgoWriter.Printf("Expected new Pod IP: %s, Found: %v\n", newPodIP, foundNewIP)
					GinkgoWriter.Printf("Old Pod IP still present: %v\n", stillHasOldIP)

					// The service should have the new IP and not have the old IP
					if foundNewIP && !stillHasOldIP {
						return true, nil
					}
					return false, nil
				}
			}

			GinkgoWriter.Printf("Service %s not found in APISIX\n", serviceName)
			return false, nil
		})
		Expect(err).NotTo(HaveOccurred(), "waiting for service upstream nodes to update")

		By("verify the route still works with new pod IP")
		s.RequestAssert(&scaffold.RequestAssert{
			Method: "GET",
			Path:   "/get",
			Host:   "httpbin",
			Check:  scaffold.WithExpectedStatus(http.StatusOK),
		})

		By("final verification: get APISIX Service configuration and verify nodes")
		services, err := s.DefaultDataplaneResource().Service().List(context.Background())
		Expect(err).NotTo(HaveOccurred(), "getting final service configuration")
		foundService := false
		for _, svc := range services {
			if svc.Name == serviceName {
				foundService = true
				Expect(svc.Upstream).NotTo(BeNil(), "service upstream should not be nil")
				Expect(len(svc.Upstream.Nodes)).To(BeNumerically(">", 0), "service upstream should have nodes")

				// Verify nodes contain new IP
				hasNewIP := false
				hasOldIP := false
				for _, node := range svc.Upstream.Nodes {
					if node.Host == newPodIP {
						hasNewIP = true
					}
					if node.Host == initialPodIP {
						hasOldIP = true
					}
				}

				GinkgoWriter.Printf("Final Service %s upstream nodes: %v\n", svc.ID, svc.Upstream.Nodes)
				Expect(hasNewIP).To(BeTrue(), fmt.Sprintf("service upstream should contain new pod IP %s", newPodIP))
				Expect(hasOldIP).To(BeFalse(), fmt.Sprintf("service upstream should not contain old pod IP %s", initialPodIP))
				break
			}
		}
		Expect(foundService).To(BeTrue(), "service should be found in final verification")
	})
})
