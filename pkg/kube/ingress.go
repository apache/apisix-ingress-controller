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

	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	listersextensionsv1beta1 "k8s.io/client-go/listers/extensions/v1beta1"
	listersnetworkingv1 "k8s.io/client-go/listers/networking/v1"
	listersnetworkingv1beta1 "k8s.io/client-go/listers/networking/v1beta1"
)

const (
	// IngressV1 represents the Ingress in networking/v1 group version.
	IngressV1 = "networking/v1"
	// IngressV1beta1 represents the Ingress in networking/v1beta1 group version.
	IngressV1beta1 = "networking/v1beta1"
	// IngressExtensionsV1beta1 represents the Ingress in extensions/v1beta1 group version.
	IngressExtensionsV1beta1 = "extensions/v1beta1"
)

// IngressLister is an encapsulation for the lister of Kubernetes
// Ingress, it aims at to be compatible with different Ingress versions.
type IngressLister interface {
	// V1 gets the ingress in networking/v1.
	V1(string, string) (Ingress, error)
	// V1beta1 gets the ingress in networking/v1beta1.
	V1beta1(string, string) (Ingress, error)
	// ExtensionsV1beta1 gets the ingress in extensions/v1beta1.
	ExtensionsV1beta1(string, string) (Ingress, error)
}

// IngressInformer is an encapsulation for the informer of Kubernetes
// Ingress, it aims at to be compatible with different Ingress versions.
type IngressInformer interface {
	Run(chan struct{})
}

// Ingress is an encapsulation for Kubernetes Ingress with different
// versions, for now, they are networking/v1 and networking/v1beta1.
type Ingress interface {
	// GroupVersion returns the api group version of the
	// real ingress.
	GroupVersion() string
	// V1 returns the ingress in networking/v1, the real
	// ingress must be in networking/v1, or V1() will panic.
	V1() *networkingv1.Ingress
	// V1beta1 returns the ingress in networking/v1beta1, the real
	// ingress must be in networking/v1beta1, or V1beta1() will panic.
	V1beta1() *networkingv1beta1.Ingress
	// ExtensionsV1beta1 returns the ingress in extensions/v1beta1, the real
	// ingress must be in extensions/v1beta1, or ExtensionsV1beta1() will panic.
	ExtensionsV1beta1() *extensionsv1beta1.Ingress
	// ResourceVersion returns the the resource version field inside
	// the real Ingress.
	ResourceVersion() string
}

// IngressEvents contains the ingress key (namespace/name)
// and the group version message.
type IngressEvent struct {
	Key          string
	GroupVersion string
	OldObject    Ingress
}

type ingress struct {
	groupVersion      string
	v1                *networkingv1.Ingress
	v1beta1           *networkingv1beta1.Ingress
	extensionsV1beta1 *extensionsv1beta1.Ingress
}

func (ing *ingress) V1() *networkingv1.Ingress {
	if ing.groupVersion != IngressV1 {
		panic("not a networking/v1 ingress")
	}
	return ing.v1
}

func (ing *ingress) V1beta1() *networkingv1beta1.Ingress {
	if ing.groupVersion != IngressV1beta1 {
		panic("not a networking/v1beta1 ingress")
	}
	return ing.v1beta1
}

func (ing *ingress) ExtensionsV1beta1() *extensionsv1beta1.Ingress {
	if ing.groupVersion != IngressExtensionsV1beta1 {
		panic("not a extensions/v1beta1 ingress")
	}
	return ing.extensionsV1beta1
}

func (ing *ingress) GroupVersion() string {
	return ing.groupVersion
}

func (ing *ingress) ResourceVersion() string {
	if ing.GroupVersion() == IngressV1 {
		return ing.V1().ResourceVersion
	}
	if ing.GroupVersion() == IngressV1beta1 {
		return ing.V1beta1().ResourceVersion
	}
	return ing.ExtensionsV1beta1().ResourceVersion
}

type ingressLister struct {
	v1Lister                listersnetworkingv1.IngressLister
	v1beta1Lister           listersnetworkingv1beta1.IngressLister
	extensionsV1beta1Lister listersextensionsv1beta1.IngressLister
}

func (l *ingressLister) V1(namespace, name string) (Ingress, error) {
	ing, err := l.v1Lister.Ingresses(namespace).Get(name)
	if err != nil {
		return nil, err
	}
	return &ingress{
		groupVersion: IngressV1,
		v1:           ing,
	}, nil
}

func (l *ingressLister) V1beta1(namespace, name string) (Ingress, error) {
	ing, err := l.v1beta1Lister.Ingresses(namespace).Get(name)
	if err != nil {
		return nil, err
	}
	return &ingress{
		groupVersion: IngressV1beta1,
		v1beta1:      ing,
	}, nil
}

func (l *ingressLister) ExtensionsV1beta1(namespace, name string) (Ingress, error) {
	ing, err := l.extensionsV1beta1Lister.Ingresses(namespace).Get(name)
	if err != nil {
		return nil, err
	}
	return &ingress{
		groupVersion:      IngressExtensionsV1beta1,
		extensionsV1beta1: ing,
	}, nil
}

// MustNewIngress creates a kube.Ingress object according to the
// type of obj.
func MustNewIngress(obj interface{}) Ingress {
	switch ing := obj.(type) {
	case *networkingv1.Ingress:
		return &ingress{
			groupVersion: IngressV1,
			v1:           ing,
		}
	case *networkingv1beta1.Ingress:
		return &ingress{
			groupVersion: IngressV1beta1,
			v1beta1:      ing,
		}
	case *extensionsv1beta1.Ingress:
		return &ingress{
			groupVersion:      IngressExtensionsV1beta1,
			extensionsV1beta1: ing,
		}
	default:
		panic("invalid ingress type")
	}
}

// NewIngress creates a kube.Ingress object according to the
// type of obj. It returns nil and the error reason when the
// type assertion fails.
func NewIngress(obj interface{}) (Ingress, error) {
	switch ing := obj.(type) {
	case *networkingv1.Ingress:
		return &ingress{
			groupVersion: IngressV1,
			v1:           ing,
		}, nil
	case *networkingv1beta1.Ingress:
		return &ingress{
			groupVersion: IngressV1beta1,
			v1beta1:      ing,
		}, nil
	case *extensionsv1beta1.Ingress:
		return &ingress{
			groupVersion:      IngressExtensionsV1beta1,
			extensionsV1beta1: ing,
		}, nil
	default:
		return nil, errors.New("invalid ingress type")
	}
}

// NewIngressLister creates an version-neutral Ingress lister.
func NewIngressLister(v1 listersnetworkingv1.IngressLister, v1beta1 listersnetworkingv1beta1.IngressLister,
	extensionsv1beta1 listersextensionsv1beta1.IngressLister) IngressLister {
	return &ingressLister{
		v1Lister:                v1,
		v1beta1Lister:           v1beta1,
		extensionsV1beta1Lister: extensionsv1beta1,
	}
}
