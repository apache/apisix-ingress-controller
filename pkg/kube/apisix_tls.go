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
	"fmt"

	configv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	configv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta3"
	listersv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/listers/config/v2"
	listersv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/listers/config/v2beta3"
)

const (
	// ApisixTlsV2beta3 represents the ApisixTls in apisix.apache.org/v2beta3 group version
	ApisixTlsV2beta3 = "apisix.apache.org/v2beta3"
	// ApisixTlsV2 represents the ApisixTls in apisix.apache.org/v2 group version
	ApisixTlsV2 = "apisix.apache.org/v2"
)

// ApisixTlsLister is an encapsulation for the lister of ApisixTls,
// it aims at to be compatible with different ApisixTls versions.
type ApisixTlsLister interface {
	// V2beta3 gets the ApisixTls in apisix.apache.org/v2beta3.
	V2beta3(string, string) (ApisixTls, error)
	// V2 gets the ApisixTls in apisix.apache.org/v2.
	V2(string, string) (ApisixTls, error)
}

// ApisixTlsInformer is an encapsulation for the informer of ApisixTls,
// it aims at to be compatible with different ApisixTls versions.
type ApisixTlsInformer interface {
	Run(chan struct{})
}

// ApisixTls is an encapsulation for ApisixTls resource with different
// versions, for now, they are apisix.apache.org/v1 and apisix.apache.org/v2alpha1
type ApisixTls interface {
	// GroupVersion returns the api group version of the
	// real ApisixTls.
	GroupVersion() string
	// V2beta3 returns the ApisixTls in apisix.apache.org/v2beta3, the real
	// ApisixTls must be in this group version, otherwise will panic.
	V2beta3() *configv2beta3.ApisixTls
	// V2 returns the ApisixTls in apisix.apache.org/v2, the real
	// ApisixTls must be in this group version, otherwise will panic.
	V2() *configv2.ApisixTls
	// ResourceVersion returns the the resource version field inside
	// the real ApisixTls.
	ResourceVersion() string
}

// ApisixTlsEvent contains the ApisixTls key (namespace/name)
// and the group version message.
type ApisixTlsEvent struct {
	Key          string
	OldObject    ApisixTls
	GroupVersion string
}

type apisixTls struct {
	groupVersion string
	v2beta3      *configv2beta3.ApisixTls
	v2           *configv2.ApisixTls
}

func (ar *apisixTls) V2beta3() *configv2beta3.ApisixTls {
	if ar.groupVersion != ApisixTlsV2beta3 {
		panic("not a apisix.apache.org/v2beta3 route")
	}
	return ar.v2beta3
}
func (ar *apisixTls) V2() *configv2.ApisixTls {
	if ar.groupVersion != ApisixTlsV2 {
		panic("not a apisix.apache.org/v2 route")
	}
	return ar.v2
}

func (ar *apisixTls) GroupVersion() string {
	return ar.groupVersion
}

func (ar *apisixTls) ResourceVersion() string {
	if ar.groupVersion == ApisixTlsV2beta3 {
		return ar.V2beta3().ResourceVersion
	}
	return ar.V2().ResourceVersion
}

type apisixTlsLister struct {
	v2beta3Lister listersv2beta3.ApisixTlsLister
	v2Lister      listersv2.ApisixTlsLister
}

func (l *apisixTlsLister) V2beta3(namespace, name string) (ApisixTls, error) {
	ar, err := l.v2beta3Lister.ApisixTlses(namespace).Get(name)
	if err != nil {
		return nil, err
	}
	return &apisixTls{
		groupVersion: ApisixTlsV2beta3,
		v2beta3:      ar,
	}, nil
}
func (l *apisixTlsLister) V2(namespace, name string) (ApisixTls, error) {
	ar, err := l.v2Lister.ApisixTlses(namespace).Get(name)
	if err != nil {
		return nil, err
	}
	return &apisixTls{
		groupVersion: ApisixTlsV2,
		v2:           ar,
	}, nil
}

// MustNewApisixTls creates a kube.ApisixTls object according to the
// type of obj.
func MustNewApisixTls(obj interface{}) ApisixTls {
	switch ar := obj.(type) {
	case *configv2beta3.ApisixTls:
		return &apisixTls{
			groupVersion: ApisixTlsV2beta3,
			v2beta3:      ar,
		}
	case *configv2.ApisixTls:
		return &apisixTls{
			groupVersion: ApisixTlsV2,
			v2:           ar,
		}
	default:
		panic("invalid ApisixTls type")
	}
}

// NewApisixTls creates a kube.ApisixTls object according to the
// type of obj. It returns nil and the error reason when the
// type assertion fails.
func NewApisixTls(obj interface{}) (ApisixTls, error) {
	switch ar := obj.(type) {
	case *configv2beta3.ApisixTls:
		return &apisixTls{
			groupVersion: ApisixTlsV2beta3,
			v2beta3:      ar,
		}, nil
	case *configv2.ApisixTls:
		return &apisixTls{
			groupVersion: ApisixTlsV2,
			v2:           ar,
		}, nil
	default:
		return nil, fmt.Errorf("invalid ApisixTls type %T", ar)
	}
}

func NewApisixTlsLister(v2beta3 listersv2beta3.ApisixTlsLister, v2 listersv2.ApisixTlsLister) ApisixTlsLister {
	return &apisixTlsLister{
		v2beta3Lister: v2beta3,
		v2Lister:      v2,
	}
}
