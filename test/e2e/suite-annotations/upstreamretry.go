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
	"fmt"
	"net/http"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-annotations: annotations.networking/v1 upstream", func() {
	s := scaffold.NewDefaultScaffold()
	ginkgo.It("1 retry and short timeout", func() {
		ing := `
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/retry: "1"
    k8s.apisix.apache.org/timeout.read: "2s"
  name: ingress-ext-v1beta1
spec:
  rules:
  - host: e2e.apisix.local
    http:
      paths:
      - path: /retry
        pathType: Exact
        backend:
          serviceName: gobackend-service
          servicePort: 9280
`
		assert.NoError(ginkgo.GinkgoT(), s.CreateResourceFromString(ing))
		err := s.EnsureNumApisixUpstreamsCreated(1)
		if err != nil {
			fmt.Println("should not err", err.Error())
		}
		time.Sleep(2 * time.Second)

		respGet := s.NewAPISIXClient().GET("/retry").WithHeader("Host", "e2e.apisix.local").Expect()
		respGet.Status(http.StatusGatewayTimeout)
	})

	ginkgo.It("1 retry and long timeout", func() {
		ing := `
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/retry: "1"
    k8s.apisix.apache.org/timeout.read: "20s"
  name: ingress-ext-v1beta1
spec:
  rules:
  - host: e2e.apisix.local
    http:
      paths:
      - path: /retry
        pathType: Exact
        backend:
          serviceName: gobackend-service
          servicePort: 9280
`
		assert.NoError(ginkgo.GinkgoT(), s.CreateResourceFromString(ing))
		err := s.EnsureNumApisixUpstreamsCreated(1)
		if err != nil {
			fmt.Println("should not err", err.Error())
		}
		time.Sleep(2 * time.Second)
		time.Sleep(2 * time.Second)

		respGet := s.NewAPISIXClient().GET("/retry").WithHeader("Host", "e2e.apisix.local").Expect()
		respGet.Status(http.StatusOK)
	})

	ginkgo.It("2 retry and short timeout", func() {
		ing := `
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/retry: "1"
    k8s.apisix.apache.org/timeout.read: "20s"
  name: ingress-ext-v1beta1
spec:
  rules:
  - host: e2e.apisix.local
    http:
      paths:
      - path: /retry
        pathType: Exact
        backend:
          serviceName: gobackend-service
          servicePort: 9280
`
		assert.NoError(ginkgo.GinkgoT(), s.CreateResourceFromString(ing))
		err := s.EnsureNumApisixUpstreamsCreated(1)
		if err != nil {
			fmt.Println("should not err", err.Error())
		}
		time.Sleep(2 * time.Second)
		time.Sleep(2 * time.Second)

		respGet := s.NewAPISIXClient().GET("/retry").WithHeader("Host", "e2e.apisix.local").Expect()
		respGet.Status(http.StatusOK)
	})
})
