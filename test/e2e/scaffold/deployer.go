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

package scaffold

import (
	corev1 "k8s.io/api/core/v1"
)

// Deployer defines the interface for deploying data plane components
type Deployer interface {
	DeployDataplane(opts DeployDataplaneOptions)
	DeployIngress()
	ScaleIngress(replicas int)
	BeforeEach()
	AfterEach()
	CreateAdditionalGateway(namePrefix string) (string, *corev1.Service, error)
	CleanupAdditionalGateway(identifier string) error
	GetAdminEndpoint(...*corev1.Service) string
	DefaultDataplaneResource() DataplaneResource
}

var NewDeployer func(*Scaffold) Deployer

type DeployDataplaneOptions struct {
	Namespace         string
	ServiceType       string
	SkipCreateTunnels bool
	ServiceHTTPPort   int
	ServiceHTTPSPort  int
}
