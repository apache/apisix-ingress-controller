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
	"net/http"

	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"

	"github.com/api7/ingress-controller/test/e2e/scaffold"
)

type ip struct {
	IP string `json:"ip"`
}

var _ = ginkgo.Describe("single-route", func() {
	s := scaffold.NewDefaultScaffold()
	ginkgo.It("/ip should return your ip", func() {
		backendSvc, backendSvcPort := s.DefaultHTTPBackend()
		s.CreateApisixRoute("httpbin-route", []scaffold.ApisixRouteRule{
			{
				Host: "httpbin.com",
				HTTP: scaffold.ApisixRouteRuleHTTP{
					Paths: []scaffold.ApisixRouteRuleHTTPPath{
						{
							Path: "/ip",
							Backend: scaffold.ApisixRouteRuleHTTPBackend{
								ServiceName: backendSvc,
								ServicePort: backendSvcPort[0],
							},
						},
					},
				},
			},
		})
		err := s.EnsureNumApisixRoutesCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "checking number of routes")
		err = s.EnsureNumApisixUpstreamsCreated(1)
		assert.Nil(ginkgo.GinkgoT(), err, "checking number of upstreams")
		body := s.NewAPISIXClient().GET("/ip").WithHeader("Host", "httpbin.com").Expect().Status(http.StatusOK).Body().Raw()
		var placeholder ip
		err = json.Unmarshal([]byte(body), &placeholder)
		assert.Nil(ginkgo.GinkgoT(), err, "unmarshalling IP")
		// It's not our focus point to check the IP address returned by httpbin,
		// so here skip the IP address validation.
	})
})
