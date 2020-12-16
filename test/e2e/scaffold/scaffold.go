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
	"github.com/gavv/httpexpect/v2"
	clientset "github.com/gxthrj/apisix-ingress-types/pkg/client/clientset/versioned"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"net/http"

	"github.com/api7/ingress-controller/pkg/config"
)

type Options struct {
	Name                    string
	Kubeconfig              string
	IngressAPISIXImage      string
	ETCDImage               string
	APISIXImage             string
	HTTPBINImage            string
	APISIXConfigPath        string
	APISIXDefaultConfigPath string
	IngressAPISIXConfig     *config.Config
}

type Scaffold struct {
	opts         *Options
	namespace    string
	kubeconfig   clientcmd.ClientConfig
	clientset    kubernetes.Interface
	apisixClient clientset.Interface

	ingressAPISIXDeployment *appsv1.Deployment
	etcdDeployment          *appsv1.Deployment
	etcdService             *corev1.Service
	apisixDeployment        *appsv1.Deployment
	apisixService           *corev1.Service
	httpbinDeployment       *appsv1.Deployment
	httpbinService          *corev1.Service

	// Used for template rendering.
	EtcdServiceFQDN string
}

// NewScaffold creates an e2e test scaffold.
func NewScaffold(o *Options) *Scaffold {
	defer ginkgo.GinkgoRecover()

	s := &Scaffold{
		opts: o,
	}

	ginkgo.BeforeEach(s.beforeEach)
	ginkgo.AfterEach(s.afterEach)

	return s
}

func NewDefaultScaffold() *Scaffold {
	opts := &Options{
		Name:               "sample",
		Kubeconfig:         "/Users/alex/.kube/config",
		ETCDImage:          "bitnami/etcd:3.4.14-debian-10-r0",
		IngressAPISIXImage: "viewking/apisix-ingress-controller:dev",
		APISIXImage:        "apache/apisix:latest",
		HTTPBINImage:       "kennethreitz/httpbin",
		IngressAPISIXConfig: &config.Config{
			LogLevel:   "info",
			LogOutput:  "stdout",
			HTTPListen: ":8080",
			APISIX: config.APISIXConfig{
				// We don't use FQDN since we don't know the namespace in advance.
				BaseURL: "https://apisix-service-e2e-test:9180/apisix",
			},
		},
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
	return httpexpect.WithConfig(httpexpect.Config{
		BaseURL: s.apisixServiceURL(),
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
	s.kubeconfig = loadConfig(s.opts.Kubeconfig, "")
	restConfig, err := s.kubeconfig.ClientConfig()
	assert.Nil(ginkgo.GinkgoT(), err, "loading Kubernetes configuration")

	s.clientset, err = kubernetes.NewForConfig(restConfig)
	assert.Nil(ginkgo.GinkgoT(), err, "creating Kubernetes clientset")

	s.apisixClient, err = clientset.NewForConfig(restConfig)
	assert.Nil(ginkgo.GinkgoT(), err, "creating APISIX clientset")

	s.namespace, err = createNamespace(s.clientset, s.opts.Name)
	assert.Nil(ginkgo.GinkgoT(), err, "creating namespace")

	s.etcdDeployment, s.etcdService, err = s.newETCD()
	assert.Nil(ginkgo.GinkgoT(), err, "initializing etcd")

	s.apisixDeployment, s.apisixService, err = s.newAPISIX()
	assert.Nil(ginkgo.GinkgoT(), err, "initializing Apache APISIX")

	s.httpbinDeployment, s.httpbinService, err = s.newHTTPBIN()
	assert.Nil(ginkgo.GinkgoT(), err, "initializing httpbin")

	s.ingressAPISIXDeployment, err = s.newIngressAPISIXController()
	assert.Nil(ginkgo.GinkgoT(), err, "initializing ingress apisix controller")
}

func (s *Scaffold) afterEach() {
	//go func() {
	defer ginkgo.GinkgoRecover()
	err := deleteNamespace(s.clientset, s.namespace)
	assert.Nilf(ginkgo.GinkgoT(), err, "deleting namespace %s", s.namespace)
	//}()
}
