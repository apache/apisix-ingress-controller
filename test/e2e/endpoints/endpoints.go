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
package endpoints

import (
	"fmt"
	"time"

	"github.com/api7/ingress-controller/test/e2e/scaffold"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
)

var _ = ginkgo.Describe("endpoints", func() {
	s := scaffold.NewDefaultScaffold()
	ginkgo.It("ignore applied only if there is an ApisixUpstream referenced", func() {
		time.Sleep(5 * time.Second)
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(0), "checking number of upstreams")
		backendSvc, backendSvcPort := s.DefaultHTTPBackend()
		ups := fmt.Sprintf(`
apiVersion: apisix.apache.org/v1
kind: ApisixUpstream
metadata:
  name: %s
spec:
  ports:
    - port: %d
      loadbalancer:
        type: roundbin
`, backendSvc, backendSvcPort[0])
		assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(ups))
		assert.Nil(ginkgo.GinkgoT(), s.EnsureNumApisixUpstreamsCreated(1), "checking number of upstreams")
	})
})
