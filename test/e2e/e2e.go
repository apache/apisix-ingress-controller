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
package e2e

import (
	"os"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/onsi/ginkgo"

	_ "github.com/api7/ingress-controller/test/e2e/endpoints"
	_ "github.com/api7/ingress-controller/test/e2e/ingress"
	"github.com/api7/ingress-controller/test/e2e/scaffold"
)

var (
	_apisixRouteDef    string
	_apisixUpstreamDef string
	_apisixServiceDef  string
	_apisixTLSDef      string
)

func runE2E() {
	if v := os.Getenv("APISIX_ROUTE_DEF"); v != "" {
		_apisixRouteDef = v
	} else {
		panic("no specified ApisixRoute definition file")
	}
	if v := os.Getenv("APISIX_UPSTREAM_DEF"); v != "" {
		_apisixUpstreamDef = v
	} else {
		panic("no specified ApisixUpstream resource definition file")
	}
	if v := os.Getenv("APISIX_UPSTREAM_DEF"); v != "" {
		_apisixUpstreamDef = v
	} else {
		panic("no specified ApisixUpstream resource definition file")
	}
	if v := os.Getenv("APISIX_SERVICE_DEF"); v != "" {
		_apisixServiceDef = v
	} else {
		panic("no specified ApisixService resource definition file")
	}
	if v := os.Getenv("APISIX_TLS_DEF"); v != "" {
		_apisixTLSDef = v
	} else {
		panic("no specified ApisixTls resource definition file")
	}

	kubeconfig := scaffold.GetKubeconfig()
	opts := &k8s.KubectlOptions{
		ConfigPath: kubeconfig,
	}
	k8s.KubectlApply(ginkgo.GinkgoT(), opts, _apisixRouteDef)
	k8s.KubectlApply(ginkgo.GinkgoT(), opts, _apisixUpstreamDef)
	k8s.KubectlApply(ginkgo.GinkgoT(), opts, _apisixServiceDef)
	k8s.KubectlApply(ginkgo.GinkgoT(), opts, _apisixTLSDef)
}
