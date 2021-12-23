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

// Code generated by informer-gen. DO NOT EDIT.

package externalversions

import (
	"fmt"

	v2beta1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta1"
	v2beta2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta2"
	v2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta3"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	cache "k8s.io/client-go/tools/cache"
)

// GenericInformer is type of SharedIndexInformer which will locate and delegate to other
// sharedInformers based on type
type GenericInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() cache.GenericLister
}

type genericInformer struct {
	informer cache.SharedIndexInformer
	resource schema.GroupResource
}

// Informer returns the SharedIndexInformer.
func (f *genericInformer) Informer() cache.SharedIndexInformer {
	return f.informer
}

// Lister returns the GenericLister.
func (f *genericInformer) Lister() cache.GenericLister {
	return cache.NewGenericLister(f.Informer().GetIndexer(), f.resource)
}

// ForResource gives generic access to a shared informer of the matching type
// TODO extend this to unknown resources with a client pool
func (f *sharedInformerFactory) ForResource(resource schema.GroupVersionResource) (GenericInformer, error) {
	switch resource {
	// Group=apisix.apache.org, Version=v2beta1
	case v2beta1.SchemeGroupVersion.WithResource("apisixroutes"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Apisix().V2beta1().ApisixRoutes().Informer()}, nil

		// Group=apisix.apache.org, Version=v2beta2
	case v2beta2.SchemeGroupVersion.WithResource("apisixroutes"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Apisix().V2beta2().ApisixRoutes().Informer()}, nil

		// Group=apisix.apache.org, Version=v2beta3
	case v2beta3.SchemeGroupVersion.WithResource("apisixclusterconfigs"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Apisix().V2beta3().ApisixClusterConfigs().Informer()}, nil
	case v2beta3.SchemeGroupVersion.WithResource("apisixconsumers"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Apisix().V2beta3().ApisixConsumers().Informer()}, nil
	case v2beta3.SchemeGroupVersion.WithResource("apisixpluginconfigs"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Apisix().V2beta3().ApisixPluginConfigs().Informer()}, nil
	case v2beta3.SchemeGroupVersion.WithResource("apisixroutes"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Apisix().V2beta3().ApisixRoutes().Informer()}, nil
	case v2beta3.SchemeGroupVersion.WithResource("apisixtlses"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Apisix().V2beta3().ApisixTlses().Informer()}, nil
	case v2beta3.SchemeGroupVersion.WithResource("apisixupstreams"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Apisix().V2beta3().ApisixUpstreams().Informer()}, nil

	}

	return nil, fmt.Errorf("no informer found for %v", resource)
}
