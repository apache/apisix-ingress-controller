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

	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("namespacing filtering", func() {
	opts := &scaffold.Options{
		Name:                  "default",
		Kubeconfig:            scaffold.GetKubeconfig(),
		APISIXConfigPath:      "testdata/apisix-gw-config.yaml",
		IngressAPISIXReplicas: 1,
		HTTPBinServicePort:    80,
		APISIXRouteVersion:    "apisix.apache.org/v2alpha1",
	}
	s := scaffold.NewScaffold(opts)
	ginkgo.It("resources in other namespaces should be ignored", func() {
		backendSvc, backendSvcPort := s.DefaultHTTPBackend()
		route := fmt.Sprintf(`
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

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(route), "creating ApisixRoute")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixRoutesCreated(1), "checking number of routes")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "checking number of upstreams")

		// TODO When ingress controller can feedback the lifecycle of CRDs to the
		// status field, we can poll it rather than sleeping.
		time.Sleep(3 * time.Second)

		body := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusOK).Body().Raw()
		var placeholder ip
		err := json.Unmarshal([]byte(body), &placeholder)
		assert.Nil(ginkgo.GinkgoT(), err, "unmarshalling IP")

		// Now create another ApisixRoute in default namespace.
		route = fmt.Sprintf(`
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
      - /headers
    backend:
      serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromStringWithNamespace(route, "default"), "creating ApisixRoute")
		_ = s.NewAPISIXClient().GET("/headers").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusNotFound)
	})
})
