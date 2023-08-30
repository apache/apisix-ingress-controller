// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

type ip struct {
	IP string `json:"origin"`
}

var _ = ginkgo.Describe("suite-ingress-features: single-route", func() {
	suites := func(s *scaffold.Scaffold) {
		ginkgo.It("/ip should return your ip", func() {
			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
			ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: httpbin-route
spec:
  http:
  - name: rule1
    match:
      paths:
      - /ip
    backends:
    - serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0])
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))
			err := s.EnsureNumApisixRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "checking number of routes")
			err = s.EnsureNumApisixUpstreamsCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "checking number of upstreams")

			// TODO When ingress controller can feedback the lifecycle of CRDs to the
			// status field, we can poll it rather than sleeping.
			time.Sleep(3 * time.Second)

			body := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusOK).Body().Raw()
			var placeholder ip
			err = json.Unmarshal([]byte(body), &placeholder)
			assert.Nil(ginkgo.GinkgoT(), err, "unmarshalling IP")
			assert.NotEqual(ginkgo.GinkgoT(), ip{}, placeholder)
			// It's not our focus point to check the IP address returned by httpbin,
			// so here skip the IP address validation.
		})
	}

	ginkgo.Describe("suite-ingress-features: scaffold v2", func() {
		suites(scaffold.NewDefaultV2Scaffold())
	})
})

var _ = ginkgo.Describe("suite-ingress-features: double-routes", func() {
	suites := func(s *scaffold.Scaffold) {
		ginkgo.It("double routes work independently", func() {
			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
			ar := `
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: httpbin-route
spec:
  http:
  - name: rule1
    match:
      paths:
      - /ip
    backends:
    - serviceName: %s
      servicePort: %d
  - name: rule2
    match:
      paths:
      - /json
    backends:
    - serviceName: %s
      servicePort: %d
`
			ar = fmt.Sprintf(ar, backendSvc, backendSvcPort[0], backendSvc, backendSvcPort[0])
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))

			err := s.EnsureNumApisixRoutesCreated(2)
			assert.Nil(ginkgo.GinkgoT(), err, "checking number of routes")
			err = s.EnsureNumApisixUpstreamsCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "checking number of upstreams")
			// TODO When ingress controller can feedback the lifecycle of CRDs to the
			// status field, we can poll it rather than sleeping.
			time.Sleep(3 * time.Second)
			body := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusOK).Body().Raw()
			var placeholder ip
			err = json.Unmarshal([]byte(body), &placeholder)
			assert.Nil(ginkgo.GinkgoT(), err, "unmarshalling IP")
			assert.NotEqual(ginkgo.GinkgoT(), ip{}, placeholder)

			body = s.NewAPISIXClient().GET("/json").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusOK).Body().Raw()
			var dummy map[string]interface{}
			err = json.Unmarshal([]byte(body), &dummy)
			assert.Nil(ginkgo.GinkgoT(), err, "unmarshalling json")
			// We don't care the json data, only make sure it's a normal json string.
		})
	}

	ginkgo.Describe("suite-ingress-features: scaffold v2", func() {
		suites(scaffold.NewDefaultV2Scaffold())
	})
})

var _ = ginkgo.Describe("suite-ingress-features: leader election", func() {
	suites := func(s *scaffold.Scaffold) {
		ginkgo.It("lease check", func() {
			pods, err := s.GetIngressPodDetails()
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.Len(ginkgo.GinkgoT(), pods, 2)
			lease, err := s.WaitGetLeaderLease()
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.Equal(ginkgo.GinkgoT(), *lease.Spec.LeaseDurationSeconds, int32(15))
			if *lease.Spec.HolderIdentity != pods[0].Name && *lease.Spec.HolderIdentity != pods[1].Name {
				assert.Fail(ginkgo.GinkgoT(), "bad leader lease holder identity")
			}
		})

		ginkgo.It("leader failover", func() {
			pods, err := s.GetIngressPodDetails()
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.Len(ginkgo.GinkgoT(), pods, 2)

			lease, err := s.WaitGetLeaderLease()
			assert.Nil(ginkgo.GinkgoT(), err)

			leaderIdx := 0
			if *lease.Spec.HolderIdentity == pods[1].Name {
				leaderIdx = 1
			}
			ginkgo.GinkgoT().Logf("lease is %s", *lease.Spec.HolderIdentity)
			assert.Nil(ginkgo.GinkgoT(), s.KillPod(pods[leaderIdx].Name))

			// Wait the old leader given up lease and new leader was elected.
			time.Sleep(5 * time.Second)

			newLease, err := s.WaitGetLeaderLease()
			assert.Nil(ginkgo.GinkgoT(), err)

			newPods, err := s.GetIngressPodDetails()
			assert.Nil(ginkgo.GinkgoT(), err)
			assert.Len(ginkgo.GinkgoT(), pods, 2)

			assert.NotEqual(ginkgo.GinkgoT(), *newLease.Spec.HolderIdentity, *lease.Spec.HolderIdentity)
			assert.Greater(ginkgo.GinkgoT(), *newLease.Spec.LeaseTransitions, *lease.Spec.LeaseTransitions)

			if *newLease.Spec.HolderIdentity != newPods[0].Name && *newLease.Spec.HolderIdentity != newPods[1].Name {
				assert.Failf(ginkgo.GinkgoT(), "bad leader lease holder identity: %s, should be %s or %s",
					*newLease.Spec.HolderIdentity, newPods[0].Name, newPods[1].Name)
			}
		})
	}

	ginkgo.Describe("suite-ingress-features: scaffold v2", func() {
		suites(scaffold.NewScaffold(&scaffold.Options{
			Name:                  "leaderelection",
			IngressAPISIXReplicas: 2,
			ApisixResourceVersion: scaffold.ApisixResourceVersion().V2,
		}))
	})
})

var _ = ginkgo.Describe("suite-ingress-features: stream_routes disabled", func() {
	suites := func(s *scaffold.Scaffold) {
		ginkgo.It("/ip should return your ip", func() {
			backendSvc, backendSvcPort := s.DefaultHTTPBackend()
			ar := fmt.Sprintf(`
apiVersion: apisix.apache.org/v2
kind: ApisixRoute
metadata:
  name: httpbin-route
spec:
  http:
  - name: rule1
    match:
      paths:
      - /ip
    backends:
    - serviceName: %s
      servicePort: %d
`, backendSvc, backendSvcPort[0])
			assert.Nil(ginkgo.GinkgoT(), s.CreateVersionedApisixResource(ar))
			err := s.EnsureNumApisixRoutesCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "checking number of routes")
			err = s.EnsureNumApisixUpstreamsCreated(1)
			assert.Nil(ginkgo.GinkgoT(), err, "checking number of upstreams")

			// TODO When ingress controller can feedback the lifecycle of CRDs to the
			// status field, we can poll it rather than sleeping.
			time.Sleep(3 * time.Second)

			body := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusOK).Body().Raw()
			var placeholder ip
			err = json.Unmarshal([]byte(body), &placeholder)
			assert.Nil(ginkgo.GinkgoT(), err, "unmarshalling IP")
			assert.NotEqual(ginkgo.GinkgoT(), ip{}, placeholder)
			// It's not our focus point to check the IP address returned by httpbin,
			// so here skip the IP address validation.
		})
	}

	ginkgo.Describe("suite-ingress-features: scaffold v2", func() {
		suites(scaffold.NewDefaultV2Scaffold())
	})
})
