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
package scaffold

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/gavv/httpexpect/v2"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/testing"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

type Options struct {
	Name                    string
	Kubeconfig              string
	APISIXConfigPath        string
	APISIXDefaultConfigPath string
}

type Scaffold struct {
	opts              *Options
	kubectlOptions    *k8s.KubectlOptions
	namespace         string
	t                 testing.TestingT
	nodes             []corev1.Node
	etcdService       *corev1.Service
	apisixService     *corev1.Service
	httpbinDeployment *appsv1.Deployment
	httpbinService    *corev1.Service

	// Used for template rendering.
	EtcdServiceFQDN string
}

// NewScaffold creates an e2e test scaffold.
func NewScaffold(o *Options) *Scaffold {
	defer ginkgo.GinkgoRecover()

	s := &Scaffold{
		opts: o,
		t:    ginkgo.GinkgoT(),
	}

	ginkgo.BeforeEach(s.beforeEach)
	ginkgo.AfterEach(s.afterEach)

	return s
}

func NewDefaultScaffold() *Scaffold {
	opts := &Options{
		Name:                    "sample",
		Kubeconfig:              "/Users/alex/.kube/config",
		APISIXConfigPath:        "/Users/alex/Workstation/tokers/apisix-ingress-controller/test/e2e/testdata/apisix-gw-config.yaml",
		APISIXDefaultConfigPath: "/Users/alex/Workstation/tokers/apisix-ingress-controller/test/e2e/testdata/apisix-gw-config-default.yaml",
	}
	return NewScaffold(opts)
}

// DefaultHTTPBackend returns the service name and service ports
// of the default http backend.
func (s *Scaffold) DefaultHTTPBackend() (string, []int32) {
	var ports []int32
	for _, p := range s.httpbinService.Spec.Ports {
		ports = append(ports, p.Port)
	}
	return s.httpbinService.Name, ports
}

// NewHTTPClient creates the default HTTP client.
func (s *Scaffold) NewHTTPClient() *httpexpect.Expect {
	url, err := s.apisixServiceURL()
	assert.Nil(s.t, err, "getting apisix service url")
	return httpexpect.WithConfig(httpexpect.Config{
		BaseURL: url,
		Client: &http.Client{
			Transport: &http.Transport{},
		},
		Reporter: httpexpect.NewAssertReporter(
			httpexpect.NewAssertReporter(ginkgo.GinkgoT()),
		),
	})
}

func (s *Scaffold) BeforeEach() {
	s.beforeEach()
}

func (s *Scaffold) AfterEach() {
	s.afterEach()
}

func (s *Scaffold) beforeEach() {
	var err error
	s.namespace = fmt.Sprintf("ingress-apisix-e2e-tests-%s-%d", s.opts.Name, time.Now().Nanosecond())
	s.kubectlOptions = &k8s.KubectlOptions{
		ConfigPath: s.opts.Kubeconfig,
		Namespace:  s.namespace,
	}
	k8s.CreateNamespace(s.t, s.kubectlOptions, s.namespace)

	s.nodes, err = k8s.GetReadyNodesE(s.t, s.kubectlOptions)
	assert.Nil(s.t, err, "querying ready nodes")

	s.etcdService, err = s.newEtcd()
	assert.Nil(s.t, err, "initializing etcd")

	k8s.WaitUntilServiceAvailable(s.t, s.kubectlOptions, s.etcdService.Name, 3, 2*time.Second)

	s.apisixService, err = s.newAPISIX()
	assert.Nil(s.t, err, "initializing Apache APISIX")

	k8s.WaitUntilServiceAvailable(s.t, s.kubectlOptions, s.apisixService.Name, 3, 2*time.Second)

	s.httpbinService, err = s.newHTTPBIN()
	assert.Nil(s.t, err, "initializing httpbin")

	err = s.newIngressAPISIXController()
	assert.Nil(s.t, err, "initializing ingress apisix controller")
}

func (s *Scaffold) afterEach() {
	defer ginkgo.GinkgoRecover()
	err := k8s.DeleteNamespaceE(s.t, s.kubectlOptions, s.namespace)
	assert.Nilf(ginkgo.GinkgoT(), err, "deleting namespace %s", s.namespace)
}

func (s *Scaffold) renderConfig(path string) (string, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	t := template.Must(template.New(path).Parse(string(data)))
	if err := t.Execute(&buf, s); err != nil {
		return "", err
	}
	return buf.String(), nil
}
