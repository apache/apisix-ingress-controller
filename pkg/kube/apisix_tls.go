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
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/apisix-ingress-controller/pkg/config"
	configv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	configv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta3"
	listersv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/listers/config/v2"
	listersv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/listers/config/v2beta3"
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

	metav1.Object
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

	metav1.Object
}

func (atls *apisixTls) V2beta3() *configv2beta3.ApisixTls {
	if atls.groupVersion != config.ApisixV2beta3 {
		panic("not a apisix.apache.org/v2beta3 ApisixTls")
	}
	return atls.v2beta3
}
func (atls *apisixTls) V2() *configv2.ApisixTls {
	if atls.groupVersion != config.ApisixV2 {
		panic("not a apisix.apache.org/v2 ApisixTls")
	}
	return atls.v2
}

func (atls *apisixTls) GroupVersion() string {
	return atls.groupVersion
}

func (atls *apisixTls) ResourceVersion() string {
	if atls.groupVersion == config.ApisixV2beta3 {
		return atls.V2beta3().ResourceVersion
	}
	return atls.V2().ResourceVersion
}

type apisixTlsLister struct {
	v2beta3Lister listersv2beta3.ApisixTlsLister
	v2Lister      listersv2.ApisixTlsLister
}

func (l *apisixTlsLister) V2beta3(namespace, name string) (ApisixTls, error) {
	at, err := l.v2beta3Lister.ApisixTlses(namespace).Get(name)
	if err != nil {
		return nil, err
	}
	return &apisixTls{
		groupVersion: config.ApisixV2beta3,
		v2beta3:      at,
		Object:       at,
	}, nil
}
func (l *apisixTlsLister) V2(namespace, name string) (ApisixTls, error) {
	at, err := l.v2Lister.ApisixTlses(namespace).Get(name)
	if err != nil {
		return nil, err
	}
	return &apisixTls{
		groupVersion: config.ApisixV2,
		v2:           at,
		Object:       at,
	}, nil
}

// MustNewApisixTls creates a kube.ApisixTls object according to the
// type of obj.
func MustNewApisixTls(obj interface{}) ApisixTls {
	switch at := obj.(type) {
	case *configv2beta3.ApisixTls:
		return &apisixTls{
			groupVersion: config.ApisixV2beta3,
			v2beta3:      at,
			Object:       at,
		}
	case *configv2.ApisixTls:
		return &apisixTls{
			groupVersion: config.ApisixV2,
			v2:           at,
			Object:       at,
		}
	default:
		panic("invalid ApisixTls type")
	}
}

// NewApisixTls creates a kube.ApisixTls object according to the
// type of obj. It returns nil and the error reason when the
// type assertion fails.
func NewApisixTls(obj interface{}) (ApisixTls, error) {
	switch at := obj.(type) {
	case *configv2beta3.ApisixTls:
		return &apisixTls{
			groupVersion: config.ApisixV2beta3,
			v2beta3:      at,
			Object:       at,
		}, nil
	case *configv2.ApisixTls:
		return &apisixTls{
			groupVersion: config.ApisixV2,
			v2:           at,
			Object:       at,
		}, nil
	default:
		return nil, fmt.Errorf("invalid ApisixTls type %T", at)
	}
}

func NewApisixTlsLister(v2beta3 listersv2beta3.ApisixTlsLister, v2 listersv2.ApisixTlsLister) ApisixTlsLister {
	return &apisixTlsLister{
		v2beta3Lister: v2beta3,
		v2Lister:      v2,
	}
}
