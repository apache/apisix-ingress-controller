// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
package gateway

import (
	"fmt"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-gateway: UDP Route", func() {
	s := scaffold.NewDefaultScaffold()

	ginkgo.It("UDPRoute", func() {
		// setup udp test service
		dnsSvc := s.NewCoreDNSService()
		route := fmt.Sprintf(`
apiVersion: gateway.networking.k8s.io/v1alpha2
kind: UDPRoute
metadata:
    name: "udp-route-test"
spec:
    rules:
    - backendRefs:
      - name: %s
        port: %d
`, dnsSvc.Name, dnsSvc.Spec.Ports[0].Port)
		err := s.CreateResourceFromString(route)
		assert.Nil(ginkgo.GinkgoT(), err, "create UDPRoute failed")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixStreamRoutesCreated(1), "Checking number of streamroute")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "Checking number of upstream")
		// test dns query
		output, err := s.RunDigDNSClientFromK8s("@apisix-service-e2e-test", "-p", "9200", "github.com")
		assert.Nil(ginkgo.GinkgoT(), err, "run dig error")
		assert.Contains(ginkgo.GinkgoT(), output, "ADDITIONAL SECTION")

		time.Sleep(3 * time.Second)
		output = s.GetDeploymentLogs(scaffold.CoreDNSDeployment)
		assert.Contains(ginkgo.GinkgoT(), output, "github.com. udp")
	})
})
