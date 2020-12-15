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

	"github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/api7/ingress-controller/pkg/config"
)

type Options struct {
	Name                string
	Kubeconfig          string
	IngressAPISIXImage  string
	ETCDImage           string
	APISIXImage         string
	APISIXConfig        string
	IngressAPISIXConfig *config.Config
}

type Scaffold struct {
	context    context.Context
	ctxCancel  context.CancelFunc
	opts       *Options
	namespace  string
	kubeconfig clientcmd.ClientConfig
	clientset  kubernetes.Interface

	ingressAPISIXDeployment *appsv1.Deployment
	etcdDeployment          *appsv1.Deployment
	etcdService             *corev1.Service
	apisixDeployment        *appsv1.Deployment
	apisixService           *corev1.Service
}

// NewScaffold creates an e2e test scaffold.
func NewScaffold(o *Options) *Scaffold {
	defer ginkgo.GinkgoRecover()

	ctx, cancel := context.WithCancel(context.Background())

	s := &Scaffold{
		opts:      o,
		context:   ctx,
		ctxCancel: cancel,
	}

	ginkgo.BeforeEach(s.beforeEach)
	ginkgo.AfterEach(s.afterEach)

	return s
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

	s.namespace, err = createNamespace(s.context, s.clientset, s.opts.Name)
	assert.Nil(ginkgo.GinkgoT(), err, "creating namespace")

	s.etcdDeployment, s.etcdService, err = s.newETCD()
	assert.Nil(ginkgo.GinkgoT(), err, "initializing etcd")

	s.apisixDeployment, s.apisixService, err = s.newAPISIX()
	assert.Nil(ginkgo.GinkgoT(), err, "initializing Apache APISIX")

	s.ingressAPISIXDeployment, err = s.newIngressAPISIXController()
	assert.Nil(ginkgo.GinkgoT(), err, "initializing ingress apisix controller")
}

func (s *Scaffold) afterEach() {
	go func() {
		defer ginkgo.GinkgoRecover()
		err := deleteNamespace(s.clientset, s.namespace)
		assert.Nilf(ginkgo.GinkgoT(), err, "deleting namespace %s", s.namespace)
	}()
}
