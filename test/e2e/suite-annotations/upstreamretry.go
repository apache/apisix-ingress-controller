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

var _ = ginkgo.Describe("suite-annotations: annotations.networking/v1 upstream retry", func() {
	s := scaffold.NewDefaultScaffold()
	ginkgo.It("enable upstream retry to 3", func() {
		ing := `
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/retry: "3"
  name: ingress-ext-v1beta1
spec:
  rules:
  - host: e2e.apisix.local
    http:
      paths:
      - path: /testupstream/retry
        pathType: Exact
        backend:
          serviceName: test-backend-service-e2e-test
          servicePort: 8080
`
		assert.NoError(ginkgo.GinkgoT(), s.CreateResourceFromString(ing))
		err := s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		time.Sleep(2 * time.Second)
		//first try
		respGet := s.NewAPISIXClient().GET("/testupstream/retry").WithHeader("Host", "e2e.apisix.local").Expect()
		respGet.Status(http.StatusInternalServerError)

		//second try
		respGet = s.NewAPISIXClient().GET("/testupstream/retry").WithHeader("Host", "e2e.apisix.local").Expect()
		respGet.Status(http.StatusInternalServerError)

		//Should pass on 3rd try
		respGet = s.NewAPISIXClient().GET("/testupstream/retry").WithHeader("Host", "e2e.apisix.local").Expect()
		respGet.Status(http.StatusOK)
		respGet.Body().Contains("successful response after 2 attempts")
	})
})
