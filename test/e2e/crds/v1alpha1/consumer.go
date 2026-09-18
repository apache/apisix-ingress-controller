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

package v1alpha1

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"
	"k8s.io/apimachinery/pkg/types"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test Consumer", Label("apisix.apache.org", "v1alpha1", "consumer"), func() {
	var (
		s   = scaffold.NewDefaultScaffold()
		err error
	)

	var defaultHTTPRoute = `
apiVersion: apisix.apache.org/v1alpha1
kind: PluginConfig
metadata:
  name: auth-plugin-config
spec:
  plugins:
    - name: multi-auth
      config:
        auth_plugins:
          - basic-auth: {}
          - key-auth:
              header: apikey
---

apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: httpbin
spec:
  parentRefs:
  - name: %s
  hostnames:
  - "httpbin.org"
  rules:
  - matches: 
    - path:
        type: Exact
        value: /get
    filters:
    - type: ExtensionRef
      extensionRef:
        group: apisix.apache.org
        kind: PluginConfig
        name: auth-plugin-config
    backendRefs:
    - name: httpbin-service-e2e-test
      port: 80
`

	BeforeEach(func() {
		By("create GatewayProxy, control plane using endpoints")
		err = s.CreateResourceFromString(s.GetGatewayProxySpec())
		Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")

		By("create GatewayClass")
		err = s.CreateResourceFromString(s.GetGatewayClassYaml())
		Expect(err).NotTo(HaveOccurred(), "creating GatewayClass")

		By("create Gateway")
		err = s.CreateResourceFromString(s.GetGatewayYaml())
		Expect(err).NotTo(HaveOccurred(), "creating Gateway")

		By("create HTTPRoute")
		s.ApplyHTTPRoute(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin"}, fmt.Sprintf(defaultHTTPRoute, s.Namespace()))
	})

	Context("Consumer plugins", func() {
		var limitCountConsumer = `
apiVersion: apisix.apache.org/v1alpha1
kind: Consumer
metadata:
  name: consumer-sample
spec:
  gatewayRef:
    name: %s
  credentials:
    - type: key-auth
      name: key-auth-sample
      config:
        key: sample-key
  plugins:
    - name: limit-count
      config:
        count: 2
        time_window: 60
        rejected_code: 503
        key: remote_addr
`

		var unlimitConsumer = `
apiVersion: apisix.apache.org/v1alpha1
kind: Consumer
metadata:
  name: consumer-sample2
spec:
  gatewayRef:
    name: %s
  credentials:
    - type: key-auth
      name: key-auth-sample
      config:
        key: sample-key2
`

		It("limit-count plugin", func() {
			s.ResourceApplied("Consumer", "consumer-sample", fmt.Sprintf(limitCountConsumer, s.Namespace()), 1)
			s.ResourceApplied("Consumer", "consumer-sample2", fmt.Sprintf(unlimitConsumer, s.Namespace()), 1)

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.org",
				Headers: map[string]string{
					"apikey": "sample-key",
				},
				Check: scaffold.WithExpectedStatus(200),
			})

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.org",
				Headers: map[string]string{
					"apikey": "sample-key",
				},
				Check: scaffold.WithExpectedStatus(200),
			})

			By("trigger limit-count")
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.org",
				Headers: map[string]string{
					"apikey": "sample-key",
				},
				Check: scaffold.WithExpectedStatus(503),
			})

			for range 10 {
				s.RequestAssert(&scaffold.RequestAssert{
					Method: "GET",
					Path:   "/get",
					Host:   "httpbin.org",
					Headers: map[string]string{
						"apikey": "sample-key2",
					},
					Check: scaffold.WithExpectedStatus(200),
				})
			}
		})
	})

	Context("Credential", func() {
		var defaultCredential = `
apiVersion: apisix.apache.org/v1alpha1
kind: Consumer
metadata:
  name: consumer-sample
spec:
  gatewayRef:
    name: %s
  credentials:
    - type: basic-auth
      name: basic-auth-sample
      config:
        username: sample-user
        password: sample-password
    - type: key-auth
      name: key-auth-sample
      config:
        key: sample-key
    - type: key-auth
      name: key-auth-sample2
      config:
        key: sample-key2
`
		var updateCredential = `apiVersion: apisix.apache.org/v1alpha1
kind: Consumer
metadata:
  name: consumer-sample
spec:
  gatewayRef:
    name: %s
  credentials:
    - type: basic-auth
      name: basic-auth-sample
      config:
        username: sample-user
        password: sample-password
  plugins:
    - name: key-auth
      config:
        key: consumer-key
`

		It("Create/Update/Delete", func() {
			s.ResourceApplied("Consumer", "consumer-sample", fmt.Sprintf(defaultCredential, s.Namespace()), 1)

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.org",
				Headers: map[string]string{
					"apikey": "sample-key",
				},
				Check: scaffold.WithExpectedStatus(200),
			})

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.org",
				Headers: map[string]string{
					"apikey": "sample-key2",
				},
				Check: scaffold.WithExpectedStatus(200),
			})

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.org",
				BasicAuth: &scaffold.BasicAuth{
					Username: "sample-user",
					Password: "sample-password",
				},
				Check: scaffold.WithExpectedStatus(200),
			})

			By("update Consumer")
			s.ResourceApplied("Consumer", "consumer-sample", fmt.Sprintf(updateCredential, s.Namespace()), 2)

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.org",
				Headers: map[string]string{
					"apikey": "sample-key",
				},
				Check: scaffold.WithExpectedStatus(401),
			})

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.org",
				Headers: map[string]string{
					"apikey": "sample-key2",
				},
				Check: scaffold.WithExpectedStatus(401),
			})

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.org",
				Headers: map[string]string{
					"apikey": "consumer-key",
				},
				Check: scaffold.WithExpectedStatus(200),
			})

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.org",
				BasicAuth: &scaffold.BasicAuth{
					Username: "sample-user",
					Password: "sample-password",
				},
				Check: scaffold.WithExpectedStatus(200),
			})

			By("delete Consumer")
			err := s.DeleteResourceFromString(fmt.Sprintf(updateCredential, s.Namespace()))
			Expect(err).NotTo(HaveOccurred(), "deleting Consumer")

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.org",
				BasicAuth: &scaffold.BasicAuth{
					Username: "sample-user",
					Password: "sample-password",
				},
				Check: scaffold.WithExpectedStatus(401),
			})
		})

	})
	Context("SecretRef", func() {
		var keyAuthSecret = `
apiVersion: v1
kind: Secret
metadata:
  name: key-auth-secret
data:
  key: c2FtcGxlLWtleQ==
`
		var basicAuthSecret = `
apiVersion: v1
kind: Secret
metadata:
  name: basic-auth-secret
data:
  username: c2FtcGxlLXVzZXI=
  password: c2FtcGxlLXBhc3N3b3Jk
`
		const basicAuthSecret2 = `
apiVersion: v1
kind: Secret
metadata:
  name: basic-auth-secret
data:
  username: c2FtcGxlLXVzZXI=
  password: c2FtcGxlLXBhc3N3b3JkLW5ldw==
`
		var defaultConsumer = `
apiVersion: apisix.apache.org/v1alpha1
kind: Consumer
metadata:
  name: consumer-sample
spec:
  gatewayRef:
    name: %s
  credentials:
    - type: basic-auth
      name: basic-auth-sample
      secretRef:
        name: basic-auth-secret
    - type: key-auth
      name: key-auth-sample
      secretRef:
        name: key-auth-secret
    - type: key-auth
      name: key-auth-sample2
      config:
        key: sample-key2
`
		It("Create/Update/Delete", func() {
			err := s.CreateResourceFromString(keyAuthSecret)
			Expect(err).NotTo(HaveOccurred(), "creating key-auth secret")
			err = s.CreateResourceFromString(basicAuthSecret)
			Expect(err).NotTo(HaveOccurred(), "creating basic-auth secret")
			s.ResourceApplied("Consumer", "consumer-sample", fmt.Sprintf(defaultConsumer, s.Namespace()), 1)

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.org",
				Headers: map[string]string{
					"apikey": "sample-key",
				},
				Check: scaffold.WithExpectedStatus(200),
			})

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.org",
				BasicAuth: &scaffold.BasicAuth{
					Username: "sample-user",
					Password: "sample-password",
				},
				Check: scaffold.WithExpectedStatus(200),
			})

			// update basic-auth password
			err = s.CreateResourceFromString(basicAuthSecret2)
			Expect(err).NotTo(HaveOccurred(), "creating basic-auth secret")

			// use the old password will get 401
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.org",
				BasicAuth: &scaffold.BasicAuth{
					Username: "sample-user",
					Password: "sample-password",
				},
				Check: scaffold.WithExpectedStatus(401),
			})

			// use the new password will get 200
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.org",
				BasicAuth: &scaffold.BasicAuth{
					Username: "sample-user",
					Password: "sample-password-new",
				},
				Check: scaffold.WithExpectedStatus(200),
			})

			By("delete consumer")
			err = s.DeleteResourceFromString(fmt.Sprintf(defaultConsumer, s.Namespace()))
			Expect(err).NotTo(HaveOccurred(), "deleting consumer")

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.org",
				Headers: map[string]string{
					"apikey": "sample-key",
				},
				Check: scaffold.WithExpectedStatus(401),
			})

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.org",
				BasicAuth: &scaffold.BasicAuth{
					Username: "sample-user",
					Password: "sample-password",
				},
				Check: scaffold.WithExpectedStatus(401),
			})
		})
	})

	Context("Consumer with GatewayProxy Update", func() {
		var additionalGatewayGroupID string

		var defaultCredential = `
apiVersion: apisix.apache.org/v1alpha1
kind: Consumer
metadata:
  name: consumer-sample
spec:
  gatewayRef:
    name: %s
  credentials:
    - type: basic-auth
      name: basic-auth-sample
      config:
        username: sample-user
        password: sample-password
`
		var updatedGatewayProxy = `
apiVersion: apisix.apache.org/v1alpha1
kind: GatewayProxy
metadata:
  name: apisix-proxy-config
spec:
  provider:
    type: ControlPlane
    controlPlane:
      endpoints:
      - %s
      auth:
        type: AdminKey
        adminKey:
          value: "%s"
`

		It("Should sync consumer when GatewayProxy is updated", func() {
			s.ResourceApplied("Consumer", "consumer-sample", fmt.Sprintf(defaultCredential, s.Namespace()), 1)

			// verify basic-auth works
			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.org",
				BasicAuth: &scaffold.BasicAuth{
					Username: "sample-user",
					Password: "sample-password",
				},
				Check: scaffold.WithExpectedStatus(200),
			})

			By("create additional gateway group to get new admin key")
			var err error
			additionalGatewayGroupID, _, err = s.Deployer.CreateAdditionalGateway("gateway-proxy-update")
			Expect(err).NotTo(HaveOccurred(), "creating additional gateway group")

			resources, exists := s.GetAdditionalGateway(additionalGatewayGroupID)
			Expect(exists).To(BeTrue(), "additional gateway group should exist")

			client, err := s.NewAPISIXClientForGateway(additionalGatewayGroupID)
			Expect(err).NotTo(HaveOccurred(), "creating APISIX client for additional gateway group")

			By("Consumer not found for additional gateway group")
			s.RequestAssert(&scaffold.RequestAssert{
				Client: client,
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.org",
				BasicAuth: &scaffold.BasicAuth{
					Username: "sample-user",
					Password: "sample-password",
				},
				Check: scaffold.WithExpectedStatus(404),
			})

			By("update GatewayProxy with new admin key")
			updatedProxy := fmt.Sprintf(updatedGatewayProxy, s.Deployer.GetAdminEndpoint(resources.DataplaneService), resources.AdminAPIKey)
			err = s.CreateResourceFromStringWithNamespace(updatedProxy, s.Namespace())
			Expect(err).NotTo(HaveOccurred(), "updating GatewayProxy")

			By("verify Consumer works for additional gateway group")
			s.RequestAssert(&scaffold.RequestAssert{
				Client: client,
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.org",
				BasicAuth: &scaffold.BasicAuth{
					Username: "sample-user",
					Password: "sample-password",
				},
				Check: scaffold.WithExpectedStatus(200),
			})
		})
	})

	Context("Test Consumer sync during startup", func() {
		var consumer1 = `
apiVersion: apisix.apache.org/v1alpha1
kind: Consumer
metadata:
  name: consumer-sample
spec:
  gatewayRef:
    name: %s
  credentials:
    - type: key-auth
      name: key-auth-sample
      config:
        key: sample-key
`
		var consumer2 = `
apiVersion: apisix.apache.org/v1alpha1
kind: Consumer
metadata:
  name: consumer-unused
spec:
  gatewayRef:
    name: apisix-non-existent
  credentials:
    - type: key-auth
      name: key-auth-sample
      config:
        key: sample-key2
`
		It("Should sync Consumer during startup", func() {
			Expect(s.CreateResourceFromString(consumer2)).NotTo(HaveOccurred(), "creating unused consumer")
			s.ResourceApplied("Consumer", "consumer-sample", fmt.Sprintf(consumer1, s.Namespace()), 1)

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.org",
				Headers: map[string]string{
					"apikey": "sample-key",
				},
				Check: scaffold.WithExpectedStatus(200),
			})

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.org",
				Headers: map[string]string{
					"apikey": "sample-key2",
				},
				Check: scaffold.WithExpectedStatus(401),
			})

			By("restarting the controller and dataplane")
			s.Deployer.ScaleIngress(0)
			s.Deployer.ScaleDataplane(0)
			s.Deployer.ScaleDataplane(1)
			s.Deployer.ScaleIngress(1)

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.org",
				Headers: map[string]string{
					"apikey": "sample-key",
				},
				Check: scaffold.WithExpectedStatus(200),
			})

			s.RequestAssert(&scaffold.RequestAssert{
				Method: "GET",
				Path:   "/get",
				Host:   "httpbin.org",
				Headers: map[string]string{
					"apikey": "sample-key2",
				},
				Check: scaffold.WithExpectedStatus(401),
			})
		})
	})

	// A Consumer the data plane rejects must not stop the other Consumers under the same
	// GatewayProxy from being applied, and one rejected credential must not take the
	// Consumer's other credentials with it. What makes them rejected is checked by every
	// backend: a known plugin configured in a way its own check_schema refuses, and a
	// credential config of the wrong type.
	Context("Bad resource isolation", func() {
		var consumerWithPlugin = `
apiVersion: apisix.apache.org/v1alpha1
kind: Consumer
metadata:
  name: consumer-rejected
spec:
  gatewayRef:
    name: %s
  credentials:
    - type: key-auth
      name: key-auth-sample
      config:
        key: rejected-key
  plugins:
    - name: limit-count
      config:
        count: %d
        time_window: 60
        rejected_code: 503
        key: remote_addr
`
		var validConsumer = `
apiVersion: apisix.apache.org/v1alpha1
kind: Consumer
metadata:
  name: consumer-valid
spec:
  gatewayRef:
    name: %s
  credentials:
    - type: key-auth
      name: key-auth-sample
      config:
        key: valid-key
`
		var consumerWithCredentials = `
apiVersion: apisix.apache.org/v1alpha1
kind: Consumer
metadata:
  name: consumer-mixed
spec:
  gatewayRef:
    name: %s
  credentials:
    - type: key-auth
      name: valid-credential
      config:
        key: mixed-key
    - type: key-auth
      name: rejected-credential
      config:
        key: %s
`
		authenticates := func(key string, status int) {
			s.RequestAssert(&scaffold.RequestAssert{
				Method:  "GET",
				Path:    "/get",
				Host:    "httpbin.org",
				Headers: map[string]string{"apikey": key},
				Check:   scaffold.WithExpectedStatus(status),
			})
		}
		consumerStatus := func(name string, matchers ...gomegatypes.GomegaMatcher) {
			s.RetryAssertion(func() string {
				output, _ := s.GetOutputFromString("consumer", name, "-o", "yaml", "-n", s.Namespace())
				return output
			}).Should(And(matchers...))
		}

		It("isolates a rejected Consumer", func() {
			By("apply a valid and a rejected Consumer")
			err = s.CreateResourceFromString(fmt.Sprintf(consumerWithPlugin, s.Namespace(), 0))
			Expect(err).NotTo(HaveOccurred(), "creating the rejected Consumer")
			err = s.CreateResourceFromString(fmt.Sprintf(validConsumer, s.Namespace()))
			Expect(err).NotTo(HaveOccurred(), "creating the valid Consumer")

			By("the valid Consumer authenticates, the rejected one does not")
			authenticates("valid-key", 200)
			authenticates("rejected-key", 401)
			consumerStatus("consumer-rejected",
				ContainSubstring(`status: "False"`),
				ContainSubstring(`reason: SyncFailed`),
			)

			By("fix the rejected Consumer")
			err = s.CreateResourceFromString(fmt.Sprintf(consumerWithPlugin, s.Namespace(), 100))
			Expect(err).NotTo(HaveOccurred(), "updating the Consumer")

			By("both Consumers authenticate")
			authenticates("valid-key", 200)
			authenticates("rejected-key", 200)
		})

		It("isolates a rejected credential without taking the Consumer's other credentials", func() {
			By("apply a Consumer with one valid and one rejected credential")
			// key has to be a string, so the data plane refuses this credential.
			err = s.CreateResourceFromString(fmt.Sprintf(consumerWithCredentials, s.Namespace(), "123"))
			Expect(err).NotTo(HaveOccurred(), "creating the Consumer")

			By("the valid credential authenticates and the Consumer reports the dropped one")
			authenticates("mixed-key", 200)
			consumerStatus("consumer-mixed",
				ContainSubstring(`type: PartiallyInvalid`),
				ContainSubstring(`rejected-credential`),
			)

			By("fix the rejected credential")
			err = s.CreateResourceFromString(fmt.Sprintf(consumerWithCredentials, s.Namespace(), `"fixed-key"`))
			Expect(err).NotTo(HaveOccurred(), "updating the Consumer")

			By("both credentials authenticate")
			authenticates("mixed-key", 200)
			authenticates("fixed-key", 200)
			consumerStatus("consumer-mixed", Not(ContainSubstring(`type: PartiallyInvalid`)))
		})
	})
})
