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
package features

import (
	"time"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
)

var _ = ginkgo.Describe("ApisixConsumer", func() {
	s := scaffold.NewDefaultScaffold()

	ginkgo.It("create basicAuth consumer using value", func() {
		ac := `
apiVersion: apisix.apache.org/v2alpha1
kind: ApisixConsumer
metadata:
  name: basicvalue
spec:
  authParameter:
    basicAuth:
      value:
        username: foo
        password: bar
`
		err := s.CreateResourceFromString(ac)
		assert.Nil(ginkgo.GinkgoT(), err, "creating ApisixConsumer")

		defer func() {
			err := s.RemoveResourceByString(ac)
			assert.Nil(ginkgo.GinkgoT(), err)
		}()

		// Wait until the ApisixConsumer create event was delivered.
		time.Sleep(3 * time.Second)

		grs, err := s.ListApisixConsumers()
		assert.Nil(ginkgo.GinkgoT(), err, "listing consumer")
		assert.Len(ginkgo.GinkgoT(), grs, 1)
		assert.Equal(ginkgo.GinkgoT(), grs[0].Username, "default_basicvalue")
		assert.Len(ginkgo.GinkgoT(), grs[0].Plugins, 1)
		_, basicAuth := grs[0].Plugins["basic-auth"]
		assert.Equal(ginkgo.GinkgoT(), basicAuth, map[string]string{
			"username": "foo",
			"password": "bar",
		})
	})
})
