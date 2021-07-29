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
	"context"
	"fmt"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("ApisixRoute stream Testing with v2beta1", func() {
	opts := &scaffold.Options{
		Name:                  "default",
		Kubeconfig:            scaffold.GetKubeconfig(),
		APISIXConfigPath:      "testdata/apisix-gw-config.yaml",
		IngressAPISIXReplicas: 1,
		HTTPBinServicePort:    80,
		APISIXRouteVersion:    "apisix.apache.org/v2beta1",
	}
	s := scaffold.NewScaffold(opts)
	ginkgo.It("stream tcp proxy", func() {
		backendSvc, backendSvcPort := s.DefaultHTTPBackend()
		apisixRoute := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2beta1
kind: ApisixRoute
metadata:
  name: httpbin-tcp-route
spec:
  stream:
  - name: rule1
    protocol: TCP
    match:
      ingressPort: 9100
    backend:
      serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0])

		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(apisixRoute))
		time.Sleep(9 * time.Second)

		err := s.EnsureNumApisixStreamRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		sr, err := s.ListApisixStreamRoutes()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), sr, 1)
		assert.Equal(ginkgo.GinkgoT(), sr[0].ServerPort, int32(9100))

		resp := s.NewAPISIXClientWithTCPProxy().GET("/ip").Expect()
		resp.Body().Contains("origin")

		resp = s.NewAPISIXClientWithTCPProxy().GET("/get").WithHeader("x-my-header", "x-my-value").Expect()
		resp.Body().Contains("x-my-value")
	})
	ginkgo.It("stream udp proxy", func() {
		apisixRoute := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2beta1
kind: ApisixRoute
metadata:
  name: httpbin-udp-route
spec:
  stream:
  - name: rule1
    protocol: UDP
    match:
      ingressPort: 9200
    backend:
      serviceName: kube-dns
      servicePort: 53
`)
		// update namespace only for this case
		s.UpdateNamespace("kube-system")
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(apisixRoute))
		time.Sleep(9 * time.Second)

		err := s.EnsureNumApisixStreamRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of routes")

		sr, err := s.ListApisixStreamRoutes()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), sr, 1)
		assert.Equal(ginkgo.GinkgoT(), sr[0].ServerPort, int32(9200))
		// test dns query
		r := s.DNSResolver()
		host := "httpbin.org"
		_, err = r.LookupIPAddr(context.Background(), host)
		assert.Nil(ginkgo.GinkgoT(), err, "dns query error")
	})
})
