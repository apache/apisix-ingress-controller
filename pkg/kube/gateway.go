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
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
	gatewayv1listers "sigs.k8s.io/gateway-api/pkg/client/listers/apis/v1"
	gatewayv1beta1listers "sigs.k8s.io/gateway-api/pkg/client/listers/apis/v1beta1"
)

const (
	// GatewayV1 represents the Gateway in networking/v1 group version.
	GatewayV1 = "gateway.networking.k8s.io/v1"
	// GatewayV1beta1 represents the Gateway in networking/v1beta1 group version.
	GatewayV1beta1 = "gateway.networking.k8s.io/v1beta1"
)

// GatewayLister is an encapsulation for the lister of Kubernetes
// Gateway, it aims at to be compatible with different Gateway versions.
type GatewayLister interface {
	// V1 gets the gateway in networking/v1.
	V1(string, string) (Gateway, error)
	// V1beta1 gets the gateway in networking/v1beta1.
	V1beta1(string, string) (Gateway, error)
}

// Gateway is an encapsulation for Kubernetes Gateway with different
// versions, for now, they are networking/v1 and networking/v1beta1.
type Gateway interface {
	// GroupVersion returns the api group version of the
	// real gateway.
	GroupVersion() string
	// V1 returns the gateway in gateway/v1, the real
	// gateway must be in gateway/v1, or V1() will panic.
	V1() *gatewayv1.Gateway
	// V1beta1 returns the gateway in gateway/v1beta1, the real
	// gateway must be in gateway/v1beta1, or V1beta1() will panic.
	V1beta1() *gatewayv1beta1.Gateway
	// ResourceVersion returns the the resource version field inside
	// the real Gateway.
	ResourceVersion() string

	metav1.Object
}
type gateway struct {
	groupVersion string
	v1           *gatewayv1.Gateway
	v1beta1      *gatewayv1beta1.Gateway
	metav1.Object
}

func (gate *gateway) V1() *gatewayv1.Gateway {
	if gate.groupVersion != GatewayV1 {
		panic("not a networking/v1 gateway")
	}
	return gate.v1
}

func (gate *gateway) V1beta1() *gatewayv1beta1.Gateway {
	if gate.groupVersion != GatewayV1beta1 {
		panic("not a networking/v1beta1 gateway")
	}
	fmt.Println("v1beta1 ", gate.v1beta1)
	return gate.v1beta1
}

func (gate *gateway) GroupVersion() string {
	return gate.groupVersion
}

func (gate *gateway) ResourceVersion() string {
	if gate.GroupVersion() == GatewayV1beta1 {
		return gate.V1beta1().ResourceVersion
	}
	return gate.V1().ResourceVersion
}

// MustNewGateway creates a kube.Gateway object according to the
// type of obj.
func MustNewGateway(obj interface{}) Gateway {
	switch gate := obj.(type) {
	case *gatewayv1.Gateway:
		return &gateway{
			groupVersion: GatewayV1,
			v1:           gate,
			Object:       gate,
		}
	case *gatewayv1beta1.Gateway:
		fmt.Println("gateway v1beta1 is ", gate)
		return &gateway{
			groupVersion: GatewayV1beta1,
			v1beta1:      gate,
			Object:       gate,
		}
	default:
		panic("invalid gateway type")
	}
}

// GatewayEvents contains the gateway key (namespace/name)
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
	fmt.Println("gate from here: ", gate)
	return &gateway{
		groupVersion: GatewayV1beta1,
		v1beta1:      gate,
		Object:       gate,
	}, nil
}

// NewGatewayLister creates an version-neutral Gateway lister.
func NewGatewayLister(v1 gatewayv1listers.GatewayLister, v1beta1 gatewayv1beta1listers.GatewayLister) GatewayLister {
	return &gatewayLister{
		v1Lister:      v1,
		v1beta1Lister: v1beta1,
	}
}
