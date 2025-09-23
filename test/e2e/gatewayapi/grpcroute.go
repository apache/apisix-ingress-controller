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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc/metadata"
	pb "sigs.k8s.io/gateway-api/conformance/echo-basic/grpcechoserver"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test GRPCRoute", Label("networking.k8s.io", "grpcroute"), func() {
	s := scaffold.NewDefaultScaffold()

	BeforeEach(func() {
		By("deploy grpc backend")
		s.DeployGRPCBackend()

		By("create GatewayProxy")
		Expect(s.CreateResourceFromString(s.GetGatewayProxySpec())).NotTo(HaveOccurred(), "creating GatewayProxy")

		By("create GatewayClass")
		Expect(s.CreateResourceFromString(s.GetGatewayClassYaml())).NotTo(HaveOccurred(), "creating GatewayClass")

		s.RetryAssertion(func() string {
			gcyaml, _ := s.GetResourceYaml("GatewayClass", s.Namespace())
			return gcyaml
		}).Should(
			And(
				ContainSubstring(`status: "True"`),
				ContainSubstring("message: the gatewayclass has been accepted by the apisix-ingress-controller"),
			),
			"check GatewayClass condition",
		)

		By("create Gateway")
		Expect(s.CreateResourceFromString(s.GetGatewayYaml())).NotTo(HaveOccurred(), "creating Gateway")

		s.RetryAssertion(func() string {
			gcyaml, _ := s.GetResourceYaml("Gateway", s.Namespace())
			return gcyaml
		}).Should(
			And(
				ContainSubstring(`status: "True"`),
				ContainSubstring("message: the gateway has been accepted by the apisix-ingress-controlle"),
			),
			"check Gateway condition status",
		)
	})

	Context("GRPCRoute Filters", func() {
		var reqHeaderModifyWithAdd = `
apiVersion: gateway.networking.k8s.io/v1
kind: GRPCRoute
metadata:
  name: req-header-modify
spec:
  parentRefs:
  - name: %s
  rules:
  - matches: 
    filters:
    - type: RequestHeaderModifier
      requestHeaderModifier:
        add:
        - name: X-Req-Add
          value: "plugin-req-add"
        set:
        - name: X-Req-Set
          value: "plugin-req-set"
        remove:
        - X-Req-Removed
    backendRefs:
    - name: grpc-infra-backend-v1
      port: 8080
`
		var respHeaderModifyWithAdd = `
apiVersion: gateway.networking.k8s.io/v1
kind: GRPCRoute
metadata:
  name: resp-header-modify
spec:
  parentRefs:
  - name: %s
  rules:
  - matches: 
    filters:
    - type: ResponseHeaderModifier
      responseHeaderModifier:
        add:
        - name: X-Resp-Add
          value: "plugin-resp-add"
    backendRefs:
    - name: grpc-infra-backend-v1
      port: 8080
`
		It("GRPCRoute RequestHeaderModifier", func() {
			By("create GRPCRoute")
			s.ResourceApplied("GRPCRoute", "req-header-modify", fmt.Sprintf(reqHeaderModifyWithAdd, s.Namespace()), 1)

			testCases := []scaffold.ExpectedResponse{
				{
					EchoRequest: &pb.EchoRequest{},
				},
				{
					EchoRequest: &pb.EchoRequest{},
					Headers: map[string]string{
						"X-Req-Add": "plugin-req-add",
					},
				},
				{
					EchoRequest: &pb.EchoRequest{},
					RequestMetadata: &scaffold.RequestMetadata{
						Metadata: map[string]string{
							"X-Req-Set": "test-set",
						},
					},
					Headers: map[string]string{
						"X-Req-Set": "plugin-req-set",
					},
				},
				{
					EchoRequest: &pb.EchoRequest{},
					RequestMetadata: &scaffold.RequestMetadata{
						Metadata: map[string]string{
							"X-Req-Removed": "to-be-removed",
						},
					},
					Headers: map[string]string{
						"X-Req-Removed": "",
					},
				},
			}

			for i := range testCases {
				tc := testCases[i]
				s.RetryAssertion(func() error {
					return s.RequestEchoBackend(tc)
				}).ShouldNot(HaveOccurred(), "request grpc backend")
			}
		})

		It("GRPCRoute ResponseHeaderModifier", func() {
			By("create GRPCRoute")
			s.ResourceApplied("GRPCRoute", "resp-header-modify", fmt.Sprintf(respHeaderModifyWithAdd, s.Namespace()), 1)

			testCases := []scaffold.ExpectedResponse{
				{
					EchoRequest: &pb.EchoRequest{},
				},
				{
					EchoRequest: &pb.EchoRequest{},
					EchoResponse: scaffold.EchoResponse{
						Headers: &metadata.MD{
							"X-Resp-Add": []string{"plugin-resp-add"},
						},
					},
				},
			}

			for i := range testCases {
				tc := testCases[i]
				s.RetryAssertion(func() error {
					return s.RequestEchoBackend(tc)
				}).ShouldNot(HaveOccurred(), "request grpc backend")
			}
		})

		It("GRPCRoute ExtensionRef", func() {
			var rewritePlugin = `
apiVersion: apisix.apache.org/v1alpha1
kind: PluginConfig
metadata:
  name: rewrite
spec:
  plugins:
  - name: proxy-rewrite
    config:
      headers:
        add:
          x-req-add: "plugin-req-add"
`
			var rewritePluginUpdate = `
apiVersion: apisix.apache.org/v1alpha1
kind: PluginConfig
metadata:
  name: rewrite
spec:
  plugins:
  - name: proxy-rewrite
    config:
      headers:
        add:
          x-req-add: "plugin-req-add-v2"
`
			var extensionRefRewritePlugin = `
apiVersion: gateway.networking.k8s.io/v1
kind: GRPCRoute
metadata:
  name: rewrite
spec:
  parentRefs:
  - name: %s
  rules:
  - matches: 
    filters:
    - type: ExtensionRef
      extensionRef:
        group: apisix.apache.org
        kind: PluginConfig
        name: rewrite
    backendRefs:
    - name: grpc-infra-backend-v1
      port: 8080
`
			Expect(s.CreateResourceFromString(rewritePlugin)).NotTo(HaveOccurred(), "creating PluginConfig")
			s.ResourceApplied("GRPCRoute", "rewrite", fmt.Sprintf(extensionRefRewritePlugin, s.Namespace()), 1)

			testCases := []struct {
				scaffold.ExpectedResponse
				Helper func()
			}{
				{
					ExpectedResponse: scaffold.ExpectedResponse{
						EchoRequest: &pb.EchoRequest{},
					},
				},
				{
					ExpectedResponse: scaffold.ExpectedResponse{
						EchoRequest: &pb.EchoRequest{},
						Headers: map[string]string{
							"x-req-add": "plugin-req-add",
						},
					},
				},
				{
					ExpectedResponse: scaffold.ExpectedResponse{
						EchoRequest: &pb.EchoRequest{},
						Headers: map[string]string{
							"x-req-add": "plugin-req-add-v2",
						},
					},
					Helper: func() {
						Expect(s.CreateResourceFromString(rewritePluginUpdate)).NotTo(HaveOccurred(), "updating PluginConfig")
					},
				},
			}

			for i := range testCases {
				if testCases[i].Helper != nil {
					testCases[i].Helper()
				}
				tc := testCases[i].ExpectedResponse
				s.RetryAssertion(func() error {
					return s.RequestEchoBackend(tc)
				}).ShouldNot(HaveOccurred(), "request grpc backend")
			}
		})

		// TODO: add GRPCRoute RequestMirror test
		/*
			It("GRPCRoute RequestMirror", func() {})
		*/
	})

	// TODO: add BackendTrafficPolicy test
	/*
		Context("GRPCRoute With BackendTrafficPolicy", func() {})
	*/
})
