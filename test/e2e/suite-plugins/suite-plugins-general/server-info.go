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
package plugins

import (
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-plugins-general: server-info plugin", func() {
	s := scaffold.NewDefaultScaffold()

	ginkgo.It("enable server-info plugin", func() {
		//TODO: Need to support etcdserver mode
		if s.IsEtcdServer() {
			ginkgo.Skip("Does not support etcdserver mode, temporarily skipping test cases, waiting for fix")
		}
		serverInfoKey := [...]string{"etcd_version", "id", "hostname", "version", "boot_time"}
		serverInfo, err := s.GetServerInfo()
		assert.Nil(ginkgo.GinkgoT(), err)
		for _, key := range serverInfoKey {
			_, ok := serverInfo[key]
			assert.True(ginkgo.GinkgoT(), ok)
		}
	})
})
