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
	"fmt"
	"net/http"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("ApisixRoute Testing", func() {

	opts := &scaffold.Options{
		Name:                    "default",
		Kubeconfig:              scaffold.GetKubeconfig(),
		APISIXConfigPath:        "testdata/apisix-gw-config.yaml",
		APISIXDefaultConfigPath: "testdata/apisix-gw-config-default.yaml",
		IngressAPISIXReplicas:   1,
		HTTPBinServicePort:      80,
		APISIXRouteVersion:      "apisix.apache.org/v2alpha1",
	}
	s := scaffold.NewScaffold(opts)
	ginkgo.FIt("create and then scale upstream pods to 2 ", func() {
		backendSvc, backendSvcPort := s.DefaultHTTPBackend()
		apisixRoute := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2alpha1
kind: ApisixRoute
metadata:
  name: httpbin-route
spec:
  http:
  - name: rule1
    match:
      hosts:
      - httpbin.com
      paths:
      - /ip
    backend:
      serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(apisixRoute))

		err := s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
		err = s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		assert.Nil(ginkgo.GinkgoT(), s.ScaleHTTPBIN(2), "scaling number of httpbin instances")
		assert.Nil(ginkgo.GinkgoT(), s.WaitAllHTTPBINPoddsAvailable(), "waiting for all httpbin pods ready")
		// TODO When ingress controller can feedback the lifecycle of CRDs to the
		// status field, we can poll it rather than sleeping.
		time.Sleep(10 * time.Second)
		ups, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err, "list upstreams error")
		assert.Len(ginkgo.GinkgoT(), ups[0].Nodes, 2, "upstreams nodes not expect")
		s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusOK).Body().Raw()
	})

	ginkgo.It("create and then remove", func() {
		backendSvc, backendSvcPort := s.DefaultHTTPBackend()
		apisixRoute := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2alpha1
kind: ApisixRoute
metadata:
  name: httpbin-route
spec:
  http:
  - name: rule1
    match:
      hosts:
      - httpbin.com
      paths:
      - /ip
    backend:
      serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(apisixRoute), "creating ApisixRoute")
		err := s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")
		err = s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")

		// remove
		assert.Nil(ginkgo.GinkgoT(), s.RemoveResourceByString(apisixRoute))

		// TODO When ingress controller can feedback the lifecycle of CRDs to the
		// status field, we can poll it rather than sleeping.
		time.Sleep(10 * time.Second)
		ups, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err, "list upstreams error")
		assert.Len(ginkgo.GinkgoT(), ups, 0, "upstreams nodes not expect")
	})
})
