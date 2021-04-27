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
package plugins

import (
	"io/ioutil"
	"strings"
	"errors"

	"github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
)

var serverInfoKey = [...]string{"etcd_version", "up_time", "last_report_time", "id", "hostname", "version", "boot_time"}

var _ = ginkgo.FDescribe("server-info plugin", func() {
	opts := &scaffold.Options{
		Name:                    "default",
		Kubeconfig:              scaffold.GetKubeconfig(),
		APISIXConfigPath:        "testdata/apisix-gw-config.yaml",
		APISIXDefaultConfigPath: "testdata/apisix-gw-config-default.yaml",
		IngressAPISIXReplicas:   1,
		HTTPBinServicePort:      80,
		APISIXRouteVersion:      "apisix.apache.org/v2alpha1",
	}
	s := scaffold.NewScaffold(opts)
	ginkgo.It("check server info", func() {
		err := setServerInfoPluginStatus(opts.APISIXConfigPath, true)
		assert.Nil(ginkgo.GinkgoT(), err, "Enabling server-info plugin")
		serverInfo, err := s.GetServerInfo()
		assert.Nil(ginkgo.GinkgoT(), err)
		if assert.NotNil(ginkgo.GinkgoT(), serverInfo) {
			for _, key := range serverInfoKey {
				_, ok := serverInfo[key]
				assert.True(ginkgo.GinkgoT(), ok)
			}
		}
	})

	ginkgo.It("disable plugin", func() {
		err := setServerInfoPluginStatus(opts.APISIXConfigPath, false)
		assert.Nil(ginkgo.GinkgoT(), err, "Disabling server-info plugin")
		serverInfo, err := s.GetServerInfo()
		assert.Nil(ginkgo.GinkgoT(), serverInfo)
		assert.NotNil(ginkgo.GinkgoT(), err)
	})

	ginkgo.It("enable plugin and then delete it", func() {
		err := setServerInfoPluginStatus(opts.APISIXConfigPath, true)
		assert.Nil(ginkgo.GinkgoT(), err, "Enabling server-info plugin")
		serverInfo, err := s.GetServerInfo()
		assert.Nil(ginkgo.GinkgoT(), err)
		if assert.NotNil(ginkgo.GinkgoT(), serverInfo) {
			for _, key := range serverInfoKey {
				_, ok := serverInfo[key]
				assert.True(ginkgo.GinkgoT(), ok)
			}
		}

		err = setServerInfoPluginStatus(opts.APISIXConfigPath, false)
		assert.Nil(ginkgo.GinkgoT(), err, "Disabling server-info plugin")
		serverInfo, err = s.GetServerInfo()
		assert.Nil(ginkgo.GinkgoT(), serverInfo)
		assert.NotNil(ginkgo.GinkgoT(), err)
	})
})

// enable/disable server-info plugin in config file by (un)comment, assmue "server-info" exists
func setServerInfoPluginStatus(apisixConfigPath string, setEnable bool) error {
	yamlConfig, err := ioutil.ReadFile(apisixConfigPath)
	if err != nil {
		return err
	}
	lines := strings.Split(string(yamlConfig), "\n")
	hasServerInfo := false
	for i, line := range lines {
		if strings.Contains(line, "server-info") {
			hasServerInfo = true
			if setEnable {
				lines[i] = "  - server-info"
			} else {
				lines[i] = "//  - server-info"
			}
			break
		}
	}
	if !hasServerInfo {
		return errors.New("no server-info plugin in config")
	}
	newYamlConfig := strings.Join(lines, "\n")
	err = ioutil.WriteFile(apisixConfigPath, []byte(newYamlConfig), 0644)
	if err != nil {
		return err
	}
	return nil
}
