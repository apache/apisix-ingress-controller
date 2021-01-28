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
	"k8s.io/client-go/informers"
	coreinformers "k8s.io/client-go/informers/core/v1"
	discoveryinformers "k8s.io/client-go/informers/discovery/v1beta1"
	"k8s.io/client-go/kubernetes"

	"github.com/api7/ingress-controller/pkg/config"
	clientset "github.com/api7/ingress-controller/pkg/kube/apisix/client/clientset/versioned"
)

var (
	EndpointsInformer         coreinformers.EndpointsInformer
	EndpointSliceInformer     discoveryinformers.EndpointSliceInformer
	kubeClient                kubernetes.Interface
	apisixKubeClient          *clientset.Clientset
	CoreSharedInformerFactory informers.SharedInformerFactory
)

func GetKubeClient() kubernetes.Interface {
	return kubeClient
}

func GetApisixClient() clientset.Interface {
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

	apisixKubeClient, err = clientset.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	CoreSharedInformerFactory = informers.NewSharedInformerFactory(kubeClient, cfg.Kubernetes.ResyncInterval.Duration)

	return nil
}
