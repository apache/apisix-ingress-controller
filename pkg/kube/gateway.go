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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
	gatewayv1listers "sigs.k8s.io/gateway-api/pkg/client/listers/apis/v1"
	gatewayv1beta1listers "sigs.k8s.io/gateway-api/pkg/client/listers/apis/v1beta1"
)

const (
	// IngressV1 represents the Ingress in networking/v1 group version.
	GatewayV1 = "gateway.networking.k8s.io/v1"
	// IngressV1beta1 represents the Ingress in networking/v1beta1 group version.
	GatewayV1beta1 = "gateway.networking.k8s.io/v1beta1"
)

// IngressLister is an encapsulation for the lister of Kubernetes
// Ingress, it aims at to be compatible with different Ingress versions.
type GatewayLister interface {
	// V1 gets the ingress in networking/v1.
	V1(string, string) (Gateway, error)
	// V1beta1 gets the ingress in networking/v1beta1.
	V1beta1(string, string) (Gateway, error)
}

// Ingress is an encapsulation for Kubernetes Ingress with different
// versions, for now, they are networking/v1 and networking/v1beta1.
type Gateway interface {
	// GroupVersion returns the api group version of the
	// real ingress.
	GroupVersion() string
	// V1 returns the ingress in networking/v1, the real
	// ingress must be in networking/v1, or V1() will panic.
	V1() *gatewayv1.Gateway
	// V1beta1 returns the ingress in networking/v1beta1, the real
	// ingress must be in networking/v1beta1, or V1beta1() will panic.
	V1beta1() *gatewayv1beta1.Gateway
	// ResourceVersion returns the the resource version field inside
	// the real Ingress.
	ResourceVersion() string

	metav1.Object
}
type gateway struct {
	groupVersion string
	v1           *gatewayv1.Gateway
	v1beta1      *gatewayv1beta1.Gateway
	metav1.Object
}

func (ing *gateway) V1() *gatewayv1.Gateway {
	if ing.groupVersion != GatewayV1 {
		panic("not a networking/v1 ingress")
	}
	return ing.v1
}

func (ing *gateway) V1beta1() *gatewayv1beta1.Gateway {
	if ing.groupVersion != GatewayV1beta1 {
		panic("not a networking/v1beta1 ingress")
	}
	return ing.v1beta1
}

func (ing *gateway) GroupVersion() string {
	return ing.groupVersion
}

func (ing *gateway) ResourceVersion() string {
	if ing.GroupVersion() == GatewayV1beta1 {
		return ing.V1beta1().ResourceVersion
	}
	return ing.V1().ResourceVersion
}

// MustNewIngress creates a kube.Ingress object according to the
// type of obj.
func MustNewGateway(obj interface{}) Gateway {
	switch ing := obj.(type) {
	case *gatewayv1.Gateway:
		return &gateway{
			groupVersion: GatewayV1,
			v1:           ing,
			Object:       ing,
		}
	case *gatewayv1beta1.Gateway:
		return &gateway{
			groupVersion: GatewayV1beta1,
			v1beta1:      ing,
			Object:       ing,
		}
	default:
		panic("invalid gateway type")
	}
}

// IngressEvents contains the ingress key (namespace/name)
// and the group version message.
type GatewayEvent struct {
	Key          string
	GroupVersion string
	OldObject    Gateway
}
type gatewayLister struct {
	v1Lister      gatewayv1listers.GatewayLister
	v1beta1Lister gatewayv1beta1listers.GatewayLister
}

func (l *gatewayLister) V1(namespace, name string) (Gateway, error) {
	gate, err := l.v1Lister.Gateways(namespace).Get(name)
	if err != nil {
		return nil, err
	}
	return &gateway{
		groupVersion: GatewayV1,
		v1:           gate,
		Object:       gate,
	}, nil
}

func (l *gatewayLister) V1beta1(namespace, name string) (Gateway, error) {
	gate, err := l.v1beta1Lister.Gateways(namespace).Get(name)
	if err != nil {
		return nil, err
	}
	return &gateway{
		groupVersion: GatewayV1beta1,
		v1beta1:      gate,
		Object:       gate,
	}, nil
}

// NewIngressLister creates an version-neutral Ingress lister.
func NewGatewayLister(v1 gatewayv1listers.GatewayLister, v1beta1 gatewayv1beta1listers.GatewayLister) GatewayLister {
	return &gatewayLister{
		v1Lister:      v1,
		v1beta1Lister: v1beta1,
	}
}
