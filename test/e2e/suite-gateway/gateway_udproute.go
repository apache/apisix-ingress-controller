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
	"context"
	"fmt"

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
    name: %s
spec:
    rules:
    - backendRefs:
    - name: %s
        port: %d
`, "udp-route-test", dnsSvc.Name, dnsSvc.Spec.Ports[0].Port)
		err := s.CreateResourceFromString(route)
		assert.Nil(ginkgo.GinkgoT(), err, "create UDPRoute failed")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixStreamRoutesCreated(1), "Checking number of streamroute")
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "Checking number of upstream")
		// test dns query
		r := s.DNSResolver()
		host := "httpbin.org"
		_, err = r.LookupIPAddr(context.Background(), host)
		assert.Nil(ginkgo.GinkgoT(), err, "dns query error")
	})
})
