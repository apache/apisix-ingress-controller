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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gavv/httpexpect/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/apache/apisix-ingress-controller/test/e2e/framework"
	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test ADC cache alignment", Label("networking.k8s.io", "ingress"), func() {
	// conf_version only exists in APISIX standalone mode. Bail out while the tree is
	// built rather than skipping in a BeforeEach: the scaffold registers its own
	// BeforeEach first, so a skip would already have paid for a data plane.
	if framework.ProviderType != framework.ProviderTypeAPISIXStandalone {
		return
	}

	s := scaffold.NewDefaultScaffold()

	const gatewayProxyYaml = `
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

	const ingressClassYaml = `
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: %s
spec:
  controller: "%s"
  parameters:
    apiGroup: "apisix.apache.org"
    kind: "GatewayProxy"
    name: "apisix-proxy-config"
    namespace: "%s"
    scope: "Namespace"
`

	const ingressYaml = `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: httpbin
spec:
  ingressClassName: %s
  rules:
  - host: conf-version.example
    http:
      paths:
      - path: %s
        pathType: Exact
        backend:
          service:
            name: httpbin-service-e2e-test
            port:
              number: 80
`

	// confVersions returns every *_conf_version the data plane currently holds, read
	// straight from its Admin API rather than through the controller.
	confVersions := func(admin *httpexpect.Expect) map[string]int64 {
		body := admin.GET("/apisix/admin/configs").
			WithHeader("X-API-KEY", s.AdminKey()).
			Expect().Status(http.StatusOK).Body().Raw()

		var config map[string]any
		Expect(json.Unmarshal([]byte(body), &config)).NotTo(HaveOccurred(), "decoding standalone config")

		versions := map[string]int64{}
		for key, value := range config {
			if number, ok := value.(float64); ok && strings.HasSuffix(key, "_conf_version") {
				versions[key] = int64(number)
			}
		}
		return versions
	}

	// bumpConfVersions rewrites every *_conf_version the data plane holds to a timestamp
	// an hour ahead and pushes the configuration back, without going through the
	// controller. This is what the leader in between would have left behind: it pushed
	// while this pod was on standby, so APISIX now holds versions newer than the ones the
	// ADC sidecar generated during this pod's previous term -- and the sidecar survives a
	// leadership change, so it still diffs against that older baseline.
	bumpConfVersions := func(admin *httpexpect.Expect) int64 {
		body := admin.GET("/apisix/admin/configs").
			WithHeader("X-API-KEY", s.AdminKey()).
			Expect().Status(http.StatusOK).Body().Raw()

		var config map[string]any
		Expect(json.Unmarshal([]byte(body), &config)).NotTo(HaveOccurred(), "decoding standalone config")

		future := time.Now().Add(time.Hour).UnixMilli()
		bumped := 0
		for key := range config {
			// APISIX echoes its own metadata back, but rejects it on write.
			if strings.HasPrefix(strings.ToUpper(key), "X-") {
				delete(config, key)
				continue
			}
			if strings.HasSuffix(key, "_conf_version") {
				config[key] = future
				bumped++
			}
		}
		Expect(bumped).NotTo(BeZero(), "the data plane should already hold conf_versions")

		admin.PUT("/apisix/admin/configs").
			WithHeader("X-API-KEY", s.AdminKey()).
			// APISIX requires a digest and compares it with the stored one to skip an
			// update that repeats it; it never recomputes it from the body.
			WithHeader("X-Digest", fmt.Sprintf("e2e-conf-version-bump-%d", future)).
			WithJSON(config).
			Expect().Status(http.StatusAccepted)

		return future
	}

	It("recovers when the data plane holds newer conf_versions than the ADC cache", func() {
		By("create GatewayProxy, IngressClass and Ingress")
		Expect(s.CreateResourceFromString(fmt.Sprintf(gatewayProxyYaml, s.Deployer.GetAdminEndpoint(), s.AdminKey()))).
			NotTo(HaveOccurred(), "creating GatewayProxy")
		Expect(s.CreateResourceFromStringWithNamespace(
			fmt.Sprintf(ingressClassYaml, s.Namespace(), s.GetControllerName(), s.Namespace()), "")).
			NotTo(HaveOccurred(), "creating IngressClass")
		Expect(s.CreateResourceFromString(fmt.Sprintf(ingressYaml, s.Namespace(), "/get"))).
			NotTo(HaveOccurred(), "creating Ingress")

		s.RequestAssert(&scaffold.RequestAssert{
			Method: "GET",
			Path:   "/get",
			Host:   "conf-version.example",
			Check:  scaffold.WithExpectedStatus(http.StatusOK),
		})

		admin := s.Deployer.AdminAPIClient()

		By("push newer conf_versions straight into the data plane")
		future := bumpConfVersions(admin)

		By("update the Ingress and expect the change to reach the data plane")
		// The ADC sidecar still diffs against its own baseline, whose conf_versions are
		// now older than the ones the data plane holds, and APISIX rejects the whole
		// configuration with "<resource>_conf_version must be greater than or equal to".
		// The controller has to rebuild that baseline from the data plane before anything
		// can land again.
		Expect(s.CreateResourceFromString(fmt.Sprintf(ingressYaml, s.Namespace(), "/headers"))).
			NotTo(HaveOccurred(), "updating Ingress")

		s.RequestAssert(&scaffold.RequestAssert{
			Method: "GET",
			Path:   "/headers",
			Host:   "conf-version.example",
			Check:  scaffold.WithExpectedStatus(http.StatusOK),
		})
		s.RequestAssert(&scaffold.RequestAssert{
			Method: "GET",
			Path:   "/get",
			Host:   "conf-version.example",
			Check:  scaffold.WithExpectedStatus(http.StatusNotFound),
		})

		By("the versions the data plane holds moved past the injected ones")
		s.RetryAssertion(func() error {
			versions := confVersions(admin)
			if len(versions) == 0 {
				return errors.New("the data plane reported no conf_version at all")
			}
			for key, version := range versions {
				if version < future {
					return fmt.Errorf("%s is %d, expected it to be at least %d", key, version, future)
				}
			}
			return nil
		}).ShouldNot(HaveOccurred(), "checking conf_versions")
	})
})
