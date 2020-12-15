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
package kube

import (
	clientSet "github.com/gxthrj/apisix-ingress-types/pkg/client/clientset/versioned"
	"k8s.io/client-go/informers"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/api7/ingress-controller/pkg/config"
)

var (
	EndpointsInformer         coreinformers.EndpointsInformer
	kubeClient                kubernetes.Interface
	apisixKubeClient          *clientSet.Clientset
	CoreSharedInformerFactory informers.SharedInformerFactory
)

func GetKubeClient() kubernetes.Interface {
	return kubeClient
}

func GetApisixClient() clientSet.Interface {
	return apisixKubeClient
}

// initInformer initializes all related shared informers.
// Deprecate: will be refactored in the future without notification.
func InitInformer(cfg *config.Config) error {
	var err error
	restConfig, err := BuildRestConfig(cfg.Kubernetes.Kubeconfig, "")
	if err != nil {
		return err
	}
	kubeClient, err = kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	apisixKubeClient, err = clientSet.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	CoreSharedInformerFactory = informers.NewSharedInformerFactory(kubeClient, 0)

	return nil
}
