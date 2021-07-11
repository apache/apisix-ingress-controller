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

	configv1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v1"
	configv2alpha1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2alpha1"
	configv2beta1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta1"
	listersv1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/listers/config/v1"
	listersv2alpha1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/listers/config/v2alpha1"
	listersv2beta1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/listers/config/v2beta1"
)

const (
	// ApisixRouteV1 represents the ApisixRoute in apisix.apache.org/v1 group version.
	ApisixRouteV1 = "apisix.apache.org/v1"
	// ApisixRouteV2alpha1 represents the ApisixRoute in apisix.apache.org/v2alpha1 group version
	ApisixRouteV2alpha1 = "apisix.apache.org/v2alpha1"
	// ApisixRouteV2beta1 represents the ApisixRoute in apisix.apache.org/v2beta1 group version
	ApisixRouteV2beta1 = "apisix.apache.org/v2beta1"
)

// ApisixRouteLister is an encapsulation for the lister of ApisixRoute,
// it aims at to be compatible with different ApisixRoute versions.
type ApisixRouteLister interface {
	// V1 gets the ApisixRoute in apisix.apache.org/v1.
	V1(string, string) (ApisixRoute, error)
	// V2alpha1 gets the ApisixRoute in apisix.apache.org/v2alpha1.
	V2alpha1(string, string) (ApisixRoute, error)
	// V2beta1 gets the ApisixRoute in apisix.apache.org/v2beta1.
	V2beta1(string, string) (ApisixRoute, error)
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
	// V1 returns the ApisixRoute in apisix.apache.org/v1, the real
	// ApisixRoute must be in this group version, otherwise will panic.
	V1() *configv1.ApisixRoute
	// V2alpha1 returns the ApisixRoute in apisix.apache.org/v2alpha1, the real
	// ApisixRoute must be in this group version, otherwise will panic.
	V2alpha1() *configv2alpha1.ApisixRoute
	// V2beta1 returns the ApisixRoute in apisix.apache.org/v2beta1, the real
	// ApisixRoute must be in this group version, otherwise will panic.
	V2beta1() *configv2beta1.ApisixRoute
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
	groupVersion string
	v1           *configv1.ApisixRoute
	v2alpha1     *configv2alpha1.ApisixRoute
	v2beta1      *configv2beta1.ApisixRoute
}

func (ar *apisixRoute) V1() *configv1.ApisixRoute {
	if ar.groupVersion != ApisixRouteV1 {
		panic("not a apisix.apache.org/v1 ingress")
	}
	return ar.v1
}

func (ar *apisixRoute) V2alpha1() *configv2alpha1.ApisixRoute {
	if ar.groupVersion != ApisixRouteV2alpha1 {
		panic("not a apisix.apache.org/v2alpha1 ingress")
	}
	return ar.v2alpha1
}

func (ar *apisixRoute) V2beta1() *configv2beta1.ApisixRoute {
	if ar.groupVersion != ApisixRouteV2beta1 {
		panic("not a apisix.apache.org/v2beta1 ingress")
	}
	return ar.v2beta1
}

func (ar *apisixRoute) GroupVersion() string {
	return ar.groupVersion
}

func (ar *apisixRoute) ResourceVersion() string {
	if ar.groupVersion == ApisixRouteV1 {
		return ar.V1().ResourceVersion
	}
	return ar.V2alpha1().ResourceVersion
}

type apisixRouteLister struct {
	v1Lister       listersv1.ApisixRouteLister
	v2alpha1Lister listersv2alpha1.ApisixRouteLister
	v2beta1Lister  listersv2beta1.ApisixRouteLister
}

func (l *apisixRouteLister) V1(namespace, name string) (ApisixRoute, error) {
	ar, err := l.v1Lister.ApisixRoutes(namespace).Get(name)
	if err != nil {
		return nil, err
	}
	return &apisixRoute{
		groupVersion: ApisixRouteV1,
		v1:           ar,
	}, nil
}

func (l *apisixRouteLister) V2alpha1(namespace, name string) (ApisixRoute, error) {
	ar, err := l.v2alpha1Lister.ApisixRoutes(namespace).Get(name)
	if err != nil {
		return nil, err
	}
	return &apisixRoute{
		groupVersion: ApisixRouteV2alpha1,
		v2alpha1:     ar,
	}, nil
}

func (l *apisixRouteLister) V2beta1(namespace, name string) (ApisixRoute, error) {
	ar, err := l.v2beta1Lister.ApisixRoutes(namespace).Get(name)
	if err != nil {
		return nil, err
	}
	return &apisixRoute{
		groupVersion: ApisixRouteV2beta1,
		v2beta1:      ar,
	}, nil
}

// MustNewApisixRoute creates a kube.ApisixRoute object according to the
// type of obj.
func MustNewApisixRoute(obj interface{}) ApisixRoute {
	switch ar := obj.(type) {
	case *configv1.ApisixRoute:
		return &apisixRoute{
			groupVersion: ApisixRouteV1,
			v1:           ar,
		}
	case *configv2alpha1.ApisixRoute:
		return &apisixRoute{
			groupVersion: ApisixRouteV2alpha1,
			v2alpha1:     ar,
		}
	case *configv2beta1.ApisixRoute:
		return &apisixRoute{
			groupVersion: ApisixRouteV2beta1,
			v2beta1:      ar,
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
	case *configv1.ApisixRoute:
		return &apisixRoute{
			groupVersion: ApisixRouteV1,
			v1:           ar,
		}, nil
	case *configv2alpha1.ApisixRoute:
		return &apisixRoute{
			groupVersion: ApisixRouteV2alpha1,
			v2alpha1:     ar,
		}, nil
	default:
		return nil, errors.New("invalid ApisixRoute type")
	}
}

func NewApisixRouteLister(v1 listersv1.ApisixRouteLister, v2alpha1 listersv2alpha1.ApisixRouteLister, v2beta1 listersv2beta1.ApisixRouteLister) ApisixRouteLister {
	return &apisixRouteLister{
		v1Lister:       v1,
		v2alpha1Lister: v2alpha1,
		v2beta1Lister:  v2beta1,
	}
}
