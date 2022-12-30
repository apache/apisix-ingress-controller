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
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-annotations: annotations.networking/v1 upstream scheme", func() {
	s := scaffold.NewDefaultScaffold()
	ginkgo.It("sanity", func() {
		ing := `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/upstream-scheme: grpcs
  name: ingress-v1
spec:
  rules:
  - host: e2e.apisix.local
    http:
      paths:
      - path: /helloworld.Greeter/SayHello
        pathType: ImplementationSpecific
        backend:
          service:
            name: test-backend-service-e2e-test
            port:
              number: 50053
`
		assert.NoError(ginkgo.GinkgoT(), s.CreateResourceFromString(ing))
		err := s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		time.Sleep(2 * time.Second)
		ups, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), ups, 1)
		assert.Equal(ginkgo.GinkgoT(), ups[0].Scheme, "grpcs")
	})
})

var _ = ginkgo.Describe("suite-annotations-error: annotations.networking/v1 upstream scheme error", func() {
	s := scaffold.NewDefaultScaffold()
	ginkgo.It("sanity", func() {
		ing := `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/upstream-scheme: nothing
  name: ingress-v1
spec:
  rules:
  - host: e2e.apisix.local
    http:
      paths:
      - path: /helloworld.Greeter/SayHello
        pathType: ImplementationSpecific
        backend:
          service:
            name: test-backend-service-e2e-test
            port:
              number: 50053
`
		assert.NoError(ginkgo.GinkgoT(), s.CreateResourceFromString(ing))
		err := s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		time.Sleep(2 * time.Second)
		ups, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), ups, 1)
		assert.Equal(ginkgo.GinkgoT(), ups[0].Scheme, "http")
	})
})

var _ = ginkgo.Describe("suite-annotations: annotations.networking/v1beta1 upstream scheme", func() {
	s := scaffold.NewDefaultScaffold()
	ginkgo.It("sanity", func() {
		ing := `
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  name: ingress-v1beta1
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/upstream-scheme: grpcs
spec:
  rules:
  - host: e2e.apisix.local
    http:
      paths:
      - path: /helloworld.Greeter/SayHello
        pathType: ImplementationSpecific
        backend:
          serviceName: test-backend-service-e2e-test
          servicePort: 50053
`
		assert.NoError(ginkgo.GinkgoT(), s.CreateResourceFromString(ing))
		err := s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		time.Sleep(2 * time.Second)
		ups, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), ups, 1)
		assert.Equal(ginkgo.GinkgoT(), ups[0].Scheme, "grpcs")
	})
})

var _ = ginkgo.Describe("suite-annotations: annotations.extensions/v1beta1 upstream scheme", func() {
	s := scaffold.NewDefaultScaffold()
	ginkgo.It("sanity", func() {
		ing := `
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: apisix
    k8s.apisix.apache.org/upstream-scheme: grpcs
  name: ingress-ext-v1beta1
spec:
  rules:
  - host: e2e.apisix.local
    http:
      paths:
      - path: /helloworld.Greeter/SayHello
        pathType: ImplementationSpecific
        backend:
          serviceName: test-backend-service-e2e-test
          servicePort: 50053
`
		assert.NoError(ginkgo.GinkgoT(), s.CreateResourceFromString(ing))
		err := s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "Checking number of upstreams")
		time.Sleep(2 * time.Second)
		ups, err := s.ListApisixUpstreams()
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Len(ginkgo.GinkgoT(), ups, 1)
		assert.Equal(ginkgo.GinkgoT(), ups[0].Scheme, "grpcs")
	})
})
