// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package framework

import (
	"bytes"
	_ "embed"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/gruntwork-io/terratest/modules/k8s"
	. "github.com/onsi/gomega" //nolint:staticcheck
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	//go:embed manifests/ingress.yaml
	_ingressSpec   string
	IngressSpecTpl *template.Template
)

func init() {
	tpl, err := template.New("ingress").Funcs(sprig.TxtFuncMap()).Parse(_ingressSpec)
	if err != nil {
		panic(err)
	}
	IngressSpecTpl = tpl
}

type IngressDeployOpts struct {
	ControllerName     string
	ProviderType       string
	ProviderSyncPeriod time.Duration
	Namespace          string
	StatusAddress      string
	Replicas           int
	InitSyncDelay      time.Duration
}

func (f *Framework) DeployIngress(opts IngressDeployOpts) {
	buf := bytes.NewBuffer(nil)

	err := IngressSpecTpl.Execute(buf, opts)
	f.GomegaT.Expect(err).ToNot(HaveOccurred(), "rendering ingress spec")

	kubectlOpts := k8s.NewKubectlOptions("", "", opts.Namespace)

	k8s.KubectlApplyFromString(f.GinkgoT, kubectlOpts, buf.String())

	err = WaitPodsAvailable(f.GinkgoT, kubectlOpts, metav1.ListOptions{
		LabelSelector: "control-plane=controller-manager",
	})
	f.GomegaT.Expect(err).ToNot(HaveOccurred(), "waiting for controller-manager pod ready")
	f.WaitControllerManagerLog(opts.Namespace, "All cache synced successfully", 60, time.Minute)
}
