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
	"errors"

	configv2beta1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta1"
	configv2beta2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta2"
	configv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta3"
	listersv2beta1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/listers/config/v2beta1"
	listersv2beta2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/listers/config/v2beta2"
	listersv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/listers/config/v2beta3"
)

const (
	// ApisixRouteV2beta1 represents the ApisixRoute in apisix.apache.org/V2beta1 group version
	ApisixRouteV2beta1 = "apisix.apache.org/V2beta1"
	// ApisixRouteV2beta2 represents the ApisixRoute in apisix.apache.org/V2beta3 group version
	ApisixRouteV2beta2 = "apisix.apache.org/V2beta2"
	// ApisixRouteV2beta3 represents the ApisixRoute in apisix.apache.org/V2beta3 group version
	ApisixRouteV2beta3 = "apisix.apache.org/V2beta3"
)

// ApisixRouteLister is an encapsulation for the lister of ApisixRoute,
// it aims at to be compatible with different ApisixRoute versions.
type ApisixRouteLister interface {
	// V2beta1 gets the ApisixRoute in apisix.apache.org/V2beta1.
	V2beta1(string, string) (ApisixRoute, error)
	// V2beta2 gets the ApisixRoute in apisix.apache.org/V2beta3.
	V2beta2(string, string) (ApisixRoute, error)
	// V2beta3 gets the ApisixRoute in apisix.apache.org/V2beta3.
	V2beta3(string, string) (ApisixRoute, error)
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
	GetGroupVersion() string
	// V2beta1 returns the ApisixRoute in apisix.apache.org/V2beta1, the real
	// ApisixRoute must be in this group version, otherwise will panic.
	GetV2beta1() *configv2beta1.ApisixRoute
	// V2beta2 returns the ApisixRoute in apisix.apache.org/V2beta3, the real
	// ApisixRoute must be in this group version, otherwise will panic.
	GetV2beta2() *configv2beta2.ApisixRoute
	// V2beta3 returns the ApisixRoute in apisix.apache.org/V2beta3, the real
	// ApisixRoute must be in this group version, otherwise will panic.
	GetV2beta3() *configv2beta3.ApisixRoute
	// ResourceVersion returns the the resource version field inside
	// the real ApisixRoute.
	ResourceVersion() string
}

// ApisixRouteEvent contains the ApisixRoute key (namespace/name)
// and the group version message.
type ApisixRouteEvent struct {
	Key          string
	OldObject    ApisixRoute
	GroupVersion string
}

type apisixRoute struct {
	GroupVersion string
	V2beta1 *configv2beta1.ApisixRoute
	V2beta2 *configv2beta2.ApisixRoute
	V2beta3 *configv2beta3.ApisixRoute
}

func (ar *apisixRoute) GetV2beta1() *configv2beta1.ApisixRoute {
	if ar.GroupVersion != ApisixRouteV2beta1 {
		panic("not a apisix.apache.org/V2beta1 route")
	}
	return ar.V2beta1
}
func (ar *apisixRoute) GetV2beta2() *configv2beta2.ApisixRoute {
	if ar.GroupVersion != ApisixRouteV2beta2 {
		panic("not a apisix.apache.org/V2beta3 route")
	}
	return ar.V2beta2
}

func (ar *apisixRoute) GetV2beta3() *configv2beta3.ApisixRoute {
	if ar.GroupVersion != ApisixRouteV2beta3 {
		panic("not a apisix.apache.org/V2beta3 route")
	}
	return ar.V2beta3
}

func (ar *apisixRoute) GetGroupVersion() string {
	return ar.GroupVersion
}

func (ar *apisixRoute) ResourceVersion() string {
	if ar.GroupVersion == ApisixRouteV2beta1 {
		return ar.GetV2beta1().ResourceVersion
	} else if ar.GroupVersion == ApisixRouteV2beta2 {
		return ar.GetV2beta2().ResourceVersion
	}
	return ar.GetV2beta3().ResourceVersion
}

type apisixRouteLister struct {
	v2beta1Lister listersv2beta1.ApisixRouteLister
	v2beta2Lister listersv2beta2.ApisixRouteLister
	v2beta3Lister listersv2beta3.ApisixRouteLister
}

func (l *apisixRouteLister) V2beta1(namespace, name string) (ApisixRoute, error) {
	ar, err := l.v2beta1Lister.ApisixRoutes(namespace).Get(name)
	if err != nil {
		return nil, err
	}
	return &apisixRoute{
		GroupVersion: ApisixRouteV2beta1,
		V2beta1:      ar,
	}, nil
}

func (l *apisixRouteLister) V2beta2(namespace, name string) (ApisixRoute, error) {
	ar, err := l.v2beta2Lister.ApisixRoutes(namespace).Get(name)
	if err != nil {
		return nil, err
	}
	return &apisixRoute{
		GroupVersion: ApisixRouteV2beta2,
		V2beta2:      ar,
	}, nil
}

func (l *apisixRouteLister) V2beta3(namespace, name string) (ApisixRoute, error) {
	ar, err := l.v2beta3Lister.ApisixRoutes(namespace).Get(name)
	if err != nil {
		return nil, err
	}
	return &apisixRoute{
		GroupVersion: ApisixRouteV2beta3,
		V2beta3:      ar,
	}, nil
}

// MustNewApisixRoute creates a kube.ApisixRoute object according to the
// type of obj.
func MustNewApisixRoute(obj interface{}) ApisixRoute {
	switch ar := obj.(type) {
	case *configv2beta1.ApisixRoute:
		return &apisixRoute{
			GroupVersion: ApisixRouteV2beta1,
			V2beta1:      ar,
		}
	case *configv2beta2.ApisixRoute:
		return &apisixRoute{
			GroupVersion: ApisixRouteV2beta2,
			V2beta2:      ar,
		}
	case *configv2beta3.ApisixRoute:
		return &apisixRoute{
			GroupVersion: ApisixRouteV2beta3,
			V2beta3:      ar,
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
	case *configv2beta1.ApisixRoute:
		return &apisixRoute{
			GroupVersion: ApisixRouteV2beta1,
			V2beta1:      ar,
		}, nil
	case *configv2beta2.ApisixRoute:
		return &apisixRoute{
			GroupVersion: ApisixRouteV2beta2,
			V2beta2:      ar,
		}, nil
	case *configv2beta3.ApisixRoute:
		return &apisixRoute{
			GroupVersion: ApisixRouteV2beta3,
			V2beta3:      ar,
		}, nil
	default:
		return nil, errors.New("invalid ApisixRoute type")
	}
}

func NewApisixRouteLister(v2beta1 listersv2beta1.ApisixRouteLister, v2beta2 listersv2beta2.ApisixRouteLister, v2beta3 listersv2beta3.ApisixRouteLister) ApisixRouteLister {
	return &apisixRouteLister{
		v2beta1Lister: v2beta1,
		v2beta2Lister: v2beta2,
		v2beta3Lister: v2beta3,
	}
}
