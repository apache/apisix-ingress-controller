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

package annotations

import (
	"net/http"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-annotations: annotations.networking/v1 upstream", func() {
	s := scaffold.NewDefaultScaffold()
	ginkgo.It("Test timeout: 1 retry and long timeout", func() {
		ing := `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    k8s.apisix.apache.org/upstream-retries: "1"
    k8s.apisix.apache.org/upstream-read-timeout: "20s"
  name: ingress-ext-v1beta1
spec:
  ingressClassName: apisix
  rules:
    - host: e2e.apisix.local
      http:
        paths:
         - path: /retry
           pathType: Exact
           backend:
             service:
               name: gobackend-service
               port:
                 number: 9280
`
		err := s.CreateResourceFromString(ing)
		assert.Nil(ginkgo.GinkgoT(), err, "creating ingress")
		time.Sleep(5 * time.Second)
		err = s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "checking upstreams")
		time.Sleep(2 * time.Second)

		respGet := s.NewAPISIXClient().GET("/retry").WithHeader("Host", "e2e.apisix.local").Expect()
		respGet.Status(http.StatusOK)
	})

	ginkgo.It("Test retry: 2 retry and short timeout", func() {
		ing := `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    k8s.apisix.apache.org/upstream-retries: "2"
    k8s.apisix.apache.org/upstream-read-timeout: "2s"
  name: ingress-ext-v1beta1
spec:
  ingressClassName: apisix
  rules:
    - host: e2e.apisix.local
      http:
        paths:
         - path: /retry
           pathType: Exact
           backend:
             service:
               name: gobackend-service
               port:
                 number: 9280
`
		err := s.CreateResourceFromString(ing)
		assert.Nil(ginkgo.GinkgoT(), err, "creating ingress")
		time.Sleep(5 * time.Second)
		err = s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "checking upstreams")
		time.Sleep(2 * time.Second)

		respGet := s.NewAPISIXClient().GET("/retry").WithHeader("Host", "e2e.apisix.local").Expect()
		respGet.Status(http.StatusOK)
	})
})
