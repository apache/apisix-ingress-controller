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
package proxy

import (
	"encoding/json"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"

	"github.com/api7/ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("proxy-sanity", func() {
	s := scaffold.NewDefaultScaffold()

	ginkgo.It("/ip should return your ip", func() {
		// Wait all containers ready.
		// FIXME Remove this limitation.
		time.Sleep(15)

		svcName, svcPorts := s.DefaultHTTPBackend()
		s.CreateApisixRoute(&scaffold.ApisixRouteDesc{
			Name: "ingress-apisix-e2e-test-proxy-sanity",
			Host: "foo.com",
			Paths: []scaffold.ApisixRoutePath{
				{
					Path: "/ip",
					Backend: scaffold.ApisixRouteBackend{
						ServiceName: svcName,
						ServicePort: int64(svcPorts[0]),
					},
				},
			},
		})
		time.Sleep(time.Second)

		body := s.NewHTTPClient().
			GET("/ip").
			WithHeader("Host", "foo.com").
			Expect().
			Status(200).
			Body().
			Raw()

		var resp struct {
			IP string `json:"ip"`
		}
		err := json.Unmarshal([]byte(body), &resp)
		assert.Nil(ginkgo.GinkgoT(), err)
		assert.Equal(ginkgo.GinkgoT(), resp.IP, "127.0.0.1")
	})
})
