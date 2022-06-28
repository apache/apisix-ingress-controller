// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ingress

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gruntwork-io/terratest/modules/k8s"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

type headers struct {
	Headers struct {
		Accept    string `json:"Accept"`
		Host      string `json:"Host"`
		UserAgent string `json:"User-Agent"`
	} `json:"headers"`
}

var _ = ginkgo.Describe("suite-ingress: namespacing filtering enable", func() {
	s := scaffold.NewDefaultScaffold()

	ginkgo.Context("with namespace_selector", func() {
		ginkgo.It("resources in other namespaces should be ignored", func() {
			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
			route := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: httpbin-route
spec:
  ingressClassName: apisix
  rules:
  - host: httpbin.com
    http:
      paths:
      - path: /ip
        pathType: Exact
        backend:
          service:
            name: %s
            port:
              number: %d
`, backendSvc, backendSvcPort[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(route), "creating ApisixRoute")
			time.Sleep(6 * time.Second)
			// assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "checking number of routes")
			// assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "checking number of upstreams")

			body := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusOK).Body().Raw()
			var placeholder ip
			err := json.Unmarshal([]byte(body), &placeholder)
			assert.Nil(ginkgo.GinkgoT(), err, "unmarshalling IP")

			// Now create another ApisixRoute in default namespace.
			route = fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: httpbin-route
spec:
  ingressClassName: apisix
  rules:
  - host: httpbin.com
    http:
      paths:
      - path: /headers
        pathType: Exact
        backend:
          service:
            name: %s
            port:
              number: %d
`, backendSvc, backendSvcPort[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromStringWithNamespace(route, "default"), "creating ApisixRoute")
			_ = s.NewAPISIXClient().GET("/headers").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusNotFound)
		})
	})
})

var _ = ginkgo.Describe("suite-ingress: namespacing filtering disable", func() {
	s := scaffold.NewDefaultScaffold()
	ginkgo.Context("without namespace_selector", func() {
		// make namespace_selector empty
		s.DisableNamespaceSelector()
		namespace := "second-httpbin-service-namespace"

		// create another http-bin service in a new namespace.
		ginkgo.BeforeEach(func() {
			k8s.CreateNamespace(ginkgo.GinkgoT(), &k8s.KubectlOptions{
				ConfigPath: scaffold.GetKubeconfig(),
			}, namespace)
			_, err := s.NewHTTPBINWithNamespace(namespace)
			assert.Nil(ginkgo.GinkgoT(), err, "create second httpbin service")
		})

		// clean this tmp namespace when test case is done.
		ginkgo.AfterEach(func() {
			err := k8s.DeleteNamespaceE(ginkgo.GinkgoT(), &k8s.KubectlOptions{
				ConfigPath: scaffold.GetKubeconfig()}, namespace)
			assert.Nilf(ginkgo.GinkgoT(), err, "deleting namespace %s", namespace)
		})

		ginkgo.It("all resources will be watched", func() {
			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
			route := fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ingress-httpbin.com
spec:
  ingressClassName: apisix
  rules:
  - host: local.httpbin.org.host.only.734212
    http:
      paths:
      - path: /ip
        pathType: Exact
        backend:
          service:
            name: %s
            port:
              number: %d	  
`, backendSvc, backendSvcPort[0])
			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(route), "creating first ApisixRoute")
			time.Sleep(3 * time.Second)

			// Now create another ApisixRoute in another namespace.
			backendSvc, backendSvcPort = s.DefaultHTTPBackend()
			route = fmt.Sprintf(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ingress-second-httpbin-service-namespace.httpbin.com
spec:
  ingressClassName: apisix
  rules:
  - host: second-httpbin-service-namespace.httpbin.com
    http:
      paths:
      - path: /headers
        pathType: Exact
        backend:
          service:
            name: %s
            port:
              number: %d
`, backendSvc, backendSvcPort[0])

			assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromStringWithNamespace(route, namespace), "creating second ApisixRoute")

			// restart ingress-controller
			pods, err := s.GetIngressPodDetails()
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.Len(ginkgo.GinkgoT(), pods, 1)
			ginkgo.GinkgoT().Logf("restart apisix-ingress-controller pod %s", pods[0].Name)
			assert.Nil(ginkgo.GinkgoT(), s.KillPod(pods[0].Name))
			time.Sleep(6 * time.Second)
			// Two ApisixRoutes have been created at this time.
			// assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(2), "checking number of routes")
			// assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(2), "checking number of upstreams")

			body := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "local.httpbin.org.host.only.734212").Expect().Status(http.StatusOK).Body().Raw()
			var placeholder ip
			err = json.Unmarshal([]byte(body), &placeholder)
			assert.Nil(ginkgo.GinkgoT(), err, "unmarshalling IP")
			assert.NotEqual(ginkgo.GinkgoT(), ip{}, placeholder)
			body = s.NewAPISIXClient().GET("/headers").WithHeader("Host", "second-httpbin-service-namespace.httpbin.com").Expect().Status(http.StatusOK).Body().Raw()
			var headerResponse headers
			err = json.Unmarshal([]byte(body), &headerResponse)
			assert.Nil(ginkgo.GinkgoT(), err, "unmarshalling header")
			assert.NotEqual(ginkgo.GinkgoT(), headers{}, headerResponse)
		})
	})
})
