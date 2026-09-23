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
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = Describe("Test Namespace Selector", Label("apisix.apache.org", "v2", "apisixroute"), func() {
	// Entries on different keys are ANDed, a namespace needs both labels.
	const (
		teamLabel = "apisix.apache.org/e2e-namespace-team"
		envLabel  = "apisix.apache.org/e2e-namespace-env"
	)

	var (
		s = scaffold.NewScaffold(scaffold.Options{
			NamespaceSelector: []string{teamLabel + "=a", envLabel + "=prod"},
		})
		otherNamespace string
	)

	const (
		externalServiceSpec = `
apiVersion: v1
kind: Service
metadata:
  name: httpbin-external
spec:
  type: ExternalName
  externalName: httpbin-service-e2e-test.%s.svc
`
		apisixRouteSpec = `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: default
  namespace: %s
spec:
  ingressClassName: %s
  http:
  - name: rule0
    match:
      hosts:
      - %s
      paths:
      - /get
    backends:
    - serviceName: httpbin-external
      servicePort: 80
`
	)

	labelNamespace := func(ns string, labels ...string) {
		args := append([]string{"label", "namespace", ns, "--overwrite"}, labels...)
		_, err := s.RunKubectlAndGetOutput(args...)
		Expect(err).NotTo(HaveOccurred(), "labeling namespace %s", ns)
	}
	selectNamespace := func(ns string) { labelNamespace(ns, teamLabel+"=a", envLabel+"=prod") }

	request := func(host string) int {
		return s.NewAPISIXClient().GET("/get").WithHost(host).Expect().Raw().StatusCode
	}

	BeforeEach(func() {
		By("create GatewayProxy")
		Expect(s.CreateResourceFromString(s.GetGatewayProxySpec())).NotTo(HaveOccurred(), "creating GatewayProxy")

		By("create IngressClass")
		err := s.CreateResourceFromStringWithNamespace(s.GetIngressClassYaml(), "")
		Expect(err).NotTo(HaveOccurred(), "creating IngressClass")

		otherNamespace = s.Namespace() + "-other"
		s.CreateNamespace(otherNamespace)
		selectNamespace(s.Namespace())

		for _, ns := range []string{s.Namespace(), otherNamespace} {
			err := s.CreateResourceFromStringWithNamespace(fmt.Sprintf(externalServiceSpec, s.Namespace()), ns)
			Expect(err).NotTo(HaveOccurred(), "creating ExternalName Service in %s", ns)
		}
	})

	AfterEach(func() {
		s.DeleteNamespace(otherNamespace)
	})

	It("syncs only the resources in the selected namespaces", func() {
		By("create an ApisixRoute in the selected and in the unselected namespace")
		for ns, host := range map[string]string{s.Namespace(): "watched", otherNamespace: "unwatched"} {
			err := s.CreateResourceFromStringWithNamespace(fmt.Sprintf(apisixRouteSpec, ns, s.Namespace(), host), ns)
			Expect(err).NotTo(HaveOccurred(), "creating ApisixRoute in %s", ns)
		}

		Eventually(request).WithArguments("watched").WithTimeout(30 * time.Second).ProbeEvery(time.Second).
			Should(Equal(http.StatusOK))
		Consistently(request).WithArguments("unwatched").WithTimeout(10 * time.Second).ProbeEvery(time.Second).
			Should(Equal(http.StatusNotFound))

		By("label the other namespace with only one of the selected labels")
		labelNamespace(otherNamespace, teamLabel+"=a")
		Consistently(request).WithArguments("unwatched").WithTimeout(10 * time.Second).ProbeEvery(time.Second).
			Should(Equal(http.StatusNotFound))

		By("select the other namespace")
		selectNamespace(otherNamespace)
		Eventually(request).WithArguments("unwatched").WithTimeout(30 * time.Second).ProbeEvery(time.Second).
			Should(Equal(http.StatusOK))

		By("unselect the namespace, its configuration is retracted")
		labelNamespace(s.Namespace(), envLabel+"-")
		Eventually(request).WithArguments("watched").WithTimeout(30 * time.Second).ProbeEvery(time.Second).
			Should(Equal(http.StatusNotFound))
		Consistently(request).WithArguments("unwatched").WithTimeout(5 * time.Second).ProbeEvery(time.Second).
			Should(Equal(http.StatusOK))

		By("select the namespace again")
		selectNamespace(s.Namespace())
		Eventually(request).WithArguments("watched").WithTimeout(30 * time.Second).ProbeEvery(time.Second).
			Should(Equal(http.StatusOK))
	})
})
