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
package kube

import (
	"errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/apisix-ingress-controller/pkg/config"
	configv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	listersv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/listers/config/v2"
)

// ApisixRouteLister is an encapsulation for the lister of ApisixRoute,
// it aims at to be compatible with different ApisixRoute versions.
type ApisixRouteLister interface {
	// V2 gets the ApisixRoute in apisix.apache.org/v2.
	V2(string, string) (ApisixRoute, error)
	// V2Lister gets the v2 lister
	V2Lister() listersv2.ApisixRouteLister
}

// ApisixRouteInformer is an encapsulation for the informer of ApisixRoute,
// it aims at to be compatible with different ApisixRoute versions.
type ApisixRouteInformer interface {
	Run(chan struct{})
}

// ApisixRoute is an encapsulation for ApisixRoute resource with different
// versions, for now, they are apisix.apache.org/v1 and apisix.apache.org/v2alpha1
type ApisixRoute interface {
	// GroupVersion returns the api group version of the
	// real ApisixRoute.
	GroupVersion() string
	// V2 returns the ApisixRoute in apisix.apache.org/v2, the real
	// ApisixRoute must be in this group version, otherwise will panic.
	V2() *configv2.ApisixRoute
	// ResourceVersion returns the the resource version field inside
	// the real ApisixRoute.
	ResourceVersion() string

	metav1.Object
}

// ApisixRouteEvent contains the ApisixRoute key (namespace/name)
// and the group version message.
type ApisixRouteEvent struct {
	Key          string
	OldObject    ApisixRoute
	GroupVersion string
}

type apisixRoute struct {
	groupVersion string
	v2           *configv2.ApisixRoute
	metav1.Object
}

func (l *apisixRouteLister) V2Lister() listersv2.ApisixRouteLister {
	return l.v2Lister
}

func (ar *apisixRoute) V2() *configv2.ApisixRoute {
	if ar.groupVersion != config.ApisixV2 {
		panic("not a apisix.apache.org/v2 route")
	}
	return ar.v2
}

func (ar *apisixRoute) GroupVersion() string {
	return ar.groupVersion
}

func (ar *apisixRoute) ResourceVersion() string {
	return ar.V2().ResourceVersion
}

type apisixRouteLister struct {
	v2Lister listersv2.ApisixRouteLister
}

func (l *apisixRouteLister) V2(namespace, name string) (ApisixRoute, error) {
	ar, err := l.v2Lister.ApisixRoutes(namespace).Get(name)
	if err != nil {
		return nil, err
	}
	return &apisixRoute{
		groupVersion: config.ApisixV2,
		v2:           ar,
		Object:       ar,
	}, nil
}

// MustNewApisixRoute creates a kube.ApisixRoute object according to the
// type of obj.
func MustNewApisixRoute(obj interface{}) ApisixRoute {
	switch ar := obj.(type) {
	case *configv2.ApisixRoute:
		return &apisixRoute{
			groupVersion: config.ApisixV2,
			v2:           ar,
			Object:       ar,
		}
	default:
		panic("invalid ApisixRoute type")
	}
}

// NewApisixRoute creates a kube.ApisixRoute object according to the
// type of obj. It returns nil and the error reason when the
// type assertion fails.
func NewApisixRoute(obj interface{}) (ApisixRoute, error) {
	switch ar := obj.(type) {
	case *configv2.ApisixRoute:
		return &apisixRoute{
			groupVersion: config.ApisixV2,
			v2:           ar,
			Object:       ar,
		}, nil
	default:
		return nil, errors.New("invalid ApisixRoute type")
	}
}

func NewApisixRouteLister(v2 listersv2.ApisixRouteLister) ApisixRouteLister {
	return &apisixRouteLister{
		v2Lister: v2,
	}
}
