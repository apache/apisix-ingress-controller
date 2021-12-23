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
package pluginutil

import (
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

/*
  This file setting default parameters.
*/

// GetNamespaces get namespace from k8s defaults flags. defaults: ingress-apisix
func GetNamespaces(flags *genericclioptions.ConfigFlags) string {
	ns, _, err := flags.ToRawKubeConfigLoader().Namespace()
	if err != nil || len(ns) == 0 {
		ns = apiv1.NamespaceDefault
		return ns
	}
	return ns
}

// GetKubeconfigFile get kubeconfig from k8s defaults flags. default: ~/.kube/conf
func GetKubeconfigFile(flags *genericclioptions.ConfigFlags) string {
	if *flags.KubeConfig == "" {
		return "~/.kube/conf"
	}
	return *flags.KubeConfig
}
