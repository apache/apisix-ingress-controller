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

package framework

import (
	"bytes"
	_ "embed"
	"text/template"
	"time"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/stretchr/testify/assert"
)

var (
	//go:embed manifests/grpc-backend.yaml
	_grpcBackendDeployment string
	grpcBackendTpl         *template.Template
)

type GRPCBackendOpts struct {
	KubectlOptions *k8s.KubectlOptions
}

func init() {
	tpl, err := template.New("grpc-backend").Parse(_grpcBackendDeployment)
	if err != nil {
		panic(err)
	}
	grpcBackendTpl = tpl
}

func (f *Framework) DeployGRPCBackend(opts GRPCBackendOpts) {
	if opts.KubectlOptions == nil {
		opts.KubectlOptions = f.kubectlOpts
	}
	buf := bytes.NewBuffer(nil)

	err := grpcBackendTpl.Execute(buf, opts)
	assert.Nil(f.GinkgoT, err, "rendering grpc backend spec")

	k8s.KubectlApplyFromString(f.GinkgoT, opts.KubectlOptions, buf.String())

	k8s.WaitUntilDeploymentAvailable(f.GinkgoT, opts.KubectlOptions, "grpc-infra-backend-v1", 10, 10*time.Second)
}
