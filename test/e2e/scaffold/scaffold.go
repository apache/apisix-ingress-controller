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
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gavv/httpexpect/v2"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/testing"
	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

type Options struct {
	Name                    string
	Kubeconfig              string
	APISIXConfigPath        string
	APISIXDefaultConfigPath string
	IngressAPISIXReplicas   int
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
	finializers       []func()

	// Used for template rendering.
	EtcdServiceFQDN string
}

// Getkubeconfig returns the kubeconfig file path.
// Order:
// env KUBECONFIG;
// ~/.kube/config;
// "" (in case in-cluster configuration will be used).
func GetKubeconfig() string {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		u, err := user.Current()
		if err != nil {
			panic(err)
		}
		kubeconfig = filepath.Join(u.HomeDir, ".kube", "config")
		if _, err := os.Stat(kubeconfig); err != nil && !os.IsNotExist(err) {
			kubeconfig = ""
		}
	}
	return kubeconfig
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

// NewDefaultScaffold creates a scaffold with some default options.
func NewDefaultScaffold() *Scaffold {
	opts := &Options{
		Name:                    "default",
		Kubeconfig:              GetKubeconfig(),
		APISIXConfigPath:        "testdata/apisix-gw-config.yaml",
		APISIXDefaultConfigPath: "testdata/apisix-gw-config-default.yaml",
		IngressAPISIXReplicas:   1,
	}
	return NewScaffold(opts)
}

// KillPod kill the pod which name is podName.
func (s *Scaffold) KillPod(podName string) error {
	cli, err := k8s.GetKubernetesClientE(s.t)
	if err != nil {
		return err
	}
	return cli.CoreV1().Pods(s.namespace).Delete(context.TODO(), podName, metav1.DeleteOptions{})
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

// NewAPISIXClient creates the default HTTP client.
func (s *Scaffold) NewAPISIXClient() *httpexpect.Expect {
	host, err := s.apisixServiceURL()
	assert.Nil(s.t, err, "getting apisix service url")
	u := url.URL{
		Scheme: "http",
		Host:   host,
	}
	return httpexpect.WithConfig(httpexpect.Config{
		BaseURL: u.String(),
		Client: &http.Client{
			Transport: &http.Transport{},
		},
		Reporter: httpexpect.NewAssertReporter(
			httpexpect.NewAssertReporter(ginkgo.GinkgoT()),
		),
	})
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

	err = s.waitAllEtcdPodsAvailable()
	assert.Nil(s.t, err, "waiting for etcd ready")

	s.apisixService, err = s.newAPISIX()
	assert.Nil(s.t, err, "initializing Apache APISIX")

	err = s.waitAllAPISIXPodsAvailable()
	assert.Nil(s.t, err, "waiting for apisix ready")

	s.httpbinService, err = s.newHTTPBIN()
	assert.Nil(s.t, err, "initializing httpbin")

	k8s.WaitUntilServiceAvailable(s.t, s.kubectlOptions, s.httpbinService.Name, 3, 2*time.Second)

	err = s.newIngressAPISIXController()
	assert.Nil(s.t, err, "initializing ingress apisix controller")

	err = s.waitAllIngressControllerPodsAvailable()
	assert.Nil(s.t, err, "waiting for ingress apisix controller ready")
}

func (s *Scaffold) afterEach() {
	defer ginkgo.GinkgoRecover()
	err := k8s.DeleteNamespaceE(s.t, s.kubectlOptions, s.namespace)
	assert.Nilf(ginkgo.GinkgoT(), err, "deleting namespace %s", s.namespace)

	for _, f := range s.finializers {
		f()
	}

	// Wait for a while to prevent the worker node being overwhelming
	// (new cases will be run).
	time.Sleep(3 * time.Second)
}

func (s *Scaffold) addFinializer(f func()) {
	s.finializers = append(s.finializers, f)
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

func waitExponentialBackoff(condFunc func() (bool, error)) error {
	backoff := wait.Backoff{
		Duration: 500 * time.Millisecond,
		Factor:   2,
		Steps:    8,
	}
	return wait.ExponentialBackoff(backoff, condFunc)
}
