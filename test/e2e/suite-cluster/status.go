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
package cluster

import (
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-cluster: CRDs status subresource Testing", func() {
	s := scaffold.NewDefaultScaffold()

	ginkgo.It("check ApisixClusterConfig status is recorded", func() {
		// create ApisixClusterConfig resource
		assert.Nil(ginkgo.GinkgoT(), s.NewApisixClusterConfig("default", true, true), "create cluster config error")
		time.Sleep(6 * time.Second)

		// status should be recorded as successful
		output, err := s.GetOutputFromString("acc", "default", "-o", "yaml")
		assert.Nil(ginkgo.GinkgoT(), err, "Get output of ApisixClusterConfig resource")
		assert.Contains(ginkgo.GinkgoT(), output, "type: ResourcesAvailable", "status.conditions.type is recorded")
		assert.Contains(ginkgo.GinkgoT(), output, "reason: ResourcesSynced", "status.conditions.reason  is recorded")
		assert.Contains(ginkgo.GinkgoT(), output, `status: "True"`, "status.conditions.status  is recorded")
		assert.Contains(ginkgo.GinkgoT(), output, "message: Sync Successfully", "status.conditions.message  is recorded")
	})
})
