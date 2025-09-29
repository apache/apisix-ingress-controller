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

	"github.com/Masterminds/sprig/v3"
	"github.com/gruntwork-io/terratest/modules/k8s"
	. "github.com/onsi/gomega" //nolint:staticcheck
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	//go:embed manifests/nginx.yaml
	_ngxSpec   string
	ngxSpecTpl *template.Template
)

type NginxOptions struct {
	Namespace string
	Replicas  *int32
}

func init() {
	tpl, err := template.New("ngx").Funcs(sprig.TxtFuncMap()).Parse(_ngxSpec)
	if err != nil {
		panic(err)
	}
	ngxSpecTpl = tpl
}

func (f *Framework) DeployNginx(opts NginxOptions) *corev1.Service {
	buf := bytes.NewBuffer(nil)

	err := ngxSpecTpl.Execute(buf, opts)
	f.GomegaT.Expect(err).ToNot(HaveOccurred(), "rendering nginx spec")

	f.applySSLSecret(opts.Namespace, "nginx-ssl", []byte(TESTCert1), []byte(TestKey1), []byte(TestCACert))

	kubectlOpts := k8s.NewKubectlOptions("", "", opts.Namespace)

	k8s.KubectlApplyFromString(f.GinkgoT, kubectlOpts, buf.String())

	err = WaitPodsAvailable(f.GinkgoT, kubectlOpts, metav1.ListOptions{
		LabelSelector: "app=nginx",
	})
	Expect(err).ToNot(HaveOccurred(), "waiting for nginx pod ready")

	return k8s.GetService(f.GinkgoT, kubectlOpts, "nginx")
}
