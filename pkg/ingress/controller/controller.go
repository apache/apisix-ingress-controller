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
package controller

import (
	clientSet "github.com/gxthrj/apisix-ingress-types/pkg/client/clientset/versioned"
	"github.com/gxthrj/apisix-ingress-types/pkg/client/informers/externalversions"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	"github.com/api7/ingress-controller/pkg/log"
)

// recover any exception
func recoverException() {
	if err := recover(); err != nil {
		log.Error(err)
	}
}

type Api6Controller struct {
	KubeClientSet             kubernetes.Interface
	Api6ClientSet             clientSet.Interface
	SharedInformerFactory     externalversions.SharedInformerFactory
	CoreSharedInformerFactory informers.SharedInformerFactory
	Stop                      chan struct{}
}

func (api6 *Api6Controller) ApisixRoute() {
	arc := BuildApisixRouteController(
		api6.KubeClientSet,
		api6.Api6ClientSet,
		api6.SharedInformerFactory.Apisix().V1().ApisixRoutes())
	arc.Run(api6.Stop)
}

func (api6 *Api6Controller) ApisixUpstream() {
	auc := BuildApisixUpstreamController(
		api6.KubeClientSet,
		api6.Api6ClientSet,
		api6.SharedInformerFactory.Apisix().V1().ApisixUpstreams())
	auc.Run(api6.Stop)
}

func (api6 *Api6Controller) ApisixService() {
	auc := BuildApisixServiceController(
		api6.KubeClientSet,
		api6.Api6ClientSet,
		api6.SharedInformerFactory.Apisix().V1().ApisixServices())
	auc.Run(api6.Stop)
}

func (api6 *Api6Controller) ApisixTLS() {
	auc := BuildApisixTlsController(
		api6.KubeClientSet,
		api6.Api6ClientSet,
		api6.SharedInformerFactory.Apisix().V1().ApisixTlses())
	auc.Run(api6.Stop)
}

func (api6 *Api6Controller) Endpoint() {
	auc := BuildEndpointController(api6.KubeClientSet)
	//conf.EndpointsInformer)
	auc.Run(api6.Stop)
}
