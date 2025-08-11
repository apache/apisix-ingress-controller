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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test Consumer", Label("apisix.apache.org", "v1alpha1", "consumer"), func() {
	var (
		s = scaffold.NewScaffold(&scaffold.Options{
			ControllerName: fmt.Sprintf("apisix.apache.org/apisix-ingress-controller-%d", time.Now().Unix()),
		})
		err error
	)

	var defaultGatewayProxy = `
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
      auth:
        type: AdminKey
        adminKey:
          value: "%s"
`

	var defaultGatewayClass = `
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: %s
spec:
  controllerName: %s
`

	var defaultGateway = `
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: %s
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
      name: %s
`

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
		gatewayName := s.Namespace()
		By("create GatewayProxy")
		gatewayProxyName := s.Namespace()
		err = s.CreateResourceFromStringWithNamespace(fmt.Sprintf(defaultGatewayProxy, gatewayProxyName, s.Deployer.GetAdminEndpoint(), s.AdminKey()), s.Namespace())
		Expect(err).NotTo(HaveOccurred(), "creating GatewayProxy")
		time.Sleep(time.Second)

		By("create GatewayClass")
		gatewayClassName := fmt.Sprintf("apisix-%d", time.Now().Unix())
		err = s.CreateResourceFromString(fmt.Sprintf(defaultGatewayClass, gatewayClassName, s.GetControllerName()))
		Expect(err).NotTo(HaveOccurred(), "creating GatewayClass")
		time.Sleep(time.Second)

		By("create Gateway")
		err = s.CreateResourceFromStringWithNamespace(fmt.Sprintf(defaultGateway, gatewayName, gatewayClassName, gatewayProxyName), s.Namespace())
		Expect(err).NotTo(HaveOccurred(), "creating Gateway")
		time.Sleep(time.Second)

		By("create HTTPRoute")
		s.ApplyHTTPRoute(types.NamespacedName{Namespace: s.Namespace(), Name: "httpbin"}, fmt.Sprintf(defaultHTTPRoute, gatewayName))
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
  name: %s
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
			updatedProxy := fmt.Sprintf(updatedGatewayProxy, s.Namespace(), s.Deployer.GetAdminEndpoint(resources.DataplaneService), resources.AdminAPIKey)
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
})
