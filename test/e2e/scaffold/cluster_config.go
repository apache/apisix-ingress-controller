// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
package scaffold

import (
	"fmt"
	"time"

	"github.com/gruntwork-io/terratest/modules/k8s"
)

var (
	_apisixClusterConfigTemplate = `
apiVersion: %s
kind: ApisixClusterConfig
metadata:
  name: %s
spec:
  monitoring:
    prometheus:
      enable: %v
      prefer_name: %v
`
	_apisixClusterConfigV2beta3Template = `
apiVersion: %s
kind: ApisixClusterConfig
metadata:
  name: %s
spec:
  monitoring:
    prometheus:
      enable: %v
`
)

// NewApisixClusterConfig creates an ApisixClusterConfig CRD
func (s *Scaffold) NewApisixClusterConfig(name string, enable bool, enablePreferName bool) error {
	cc := fmt.Sprintf(_apisixClusterConfigTemplate, s.opts.ApisixResourceVersion, name, enable, enablePreferName)
	if err := s.CreateResourceFromString(cc); err != nil {
		return err
	}
	s.addFinalizers(func() {
		_ = s.DeleteResourceFromString(cc)
	})

	return nil
}

// DeleteApisixClusterConfig removes an ApisixClusterConfig CRD
func (s *Scaffold) DeleteApisixClusterConfig(name string, enable bool, enablePreferName bool) error {
	cc := fmt.Sprintf(_apisixClusterConfigTemplate, s.opts.ApisixResourceVersion, name, enable, enablePreferName)
	if err := k8s.KubectlDeleteFromStringE(s.t, s.kubectlOptions, cc); err != nil {
		return err
	}
	time.Sleep(5 * time.Second)
	return nil
}
