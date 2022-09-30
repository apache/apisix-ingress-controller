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

	"k8s.io/apimachinery/pkg/labels"

	"github.com/apache/apisix-ingress-controller/pkg/config"
	configv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	configv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta3"
	listersv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/listers/config/v2"
	listersv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/listers/config/v2beta3"
)

// ApisixUpstreamLister is an encapsulation for the lister of ApisixUpstream,
// it aims at to be compatible with different ApisixUpstream versions.
type ApisixUpstreamLister interface {
	// V2beta3 gets the ApisixUpstream in apisix.apache.org/v2beta3.
	V2beta3(namespace, name string) (ApisixUpstream, error)
	// V2 gets the ApisixUpstream in apisix.apache.org/v2.
	V2(namespace, name string) (ApisixUpstream, error)
	// ListV2 gets v2.ApisixUpstreams
	ListV2(namespace string) ([]*configv2.ApisixUpstream, error)
}

// ApisixUpstreamInformer is an encapsulation for the informer of ApisixUpstream,
// it aims at to be compatible with different ApisixUpstream versions.
type ApisixUpstreamInformer interface {
	Run(chan struct{})
}

// ApisixUpstream is an encapsulation for ApisixUpstream resource with different
// versions, for now, they are apisix.apache.org/v2beta3 and apisix.apache.org/v2
type ApisixUpstream interface {
	// GroupVersion returns the api group version of the
	// real ApisixUpstream.
	GroupVersion() string
	// V2beta3 returns the ApisixUpstream in apisix.apache.org/v2beta3, the real
	// ApisixUpstream must be in this group version, otherwise will panic.
	V2beta3() *configv2beta3.ApisixUpstream
	// V2 returns the ApisixUpstream in apisix.apache.org/v2, the real
	// ApisixUpstream must be in this group version, otherwise will panic.
	V2() *configv2.ApisixUpstream
	// ResourceVersion returns the the resource version field inside
	// the real ApisixUpstream.
	ResourceVersion() string
}

// ApisixUpstreamEvent contains the ApisixUpstream key (namespace/name)
// and the group version message.
type ApisixUpstreamEvent struct {
	Key          string
	OldObject    ApisixUpstream
	GroupVersion string
}

type apisixUpstream struct {
	groupVersion string
	v2beta3      *configv2beta3.ApisixUpstream
	v2           *configv2.ApisixUpstream
}

func (au *apisixUpstream) V2beta3() *configv2beta3.ApisixUpstream {
	if au.groupVersion != config.ApisixV2beta3 {
		panic("not a apisix.apache.org/v2beta3 Upstream")
	}
	return au.v2beta3
}
func (au *apisixUpstream) V2() *configv2.ApisixUpstream {
	if au.groupVersion != config.ApisixV2 {
		panic("not a apisix.apache.org/v2 Upstream")
	}
	return au.v2
}

func (au *apisixUpstream) GroupVersion() string {
	return au.groupVersion
}

func (au *apisixUpstream) ResourceVersion() string {
	if au.groupVersion == config.ApisixV2beta3 {
		return au.V2beta3().ResourceVersion
	}
	return au.V2().ResourceVersion
}

type apisixUpstreamLister struct {
	v2beta3Lister listersv2beta3.ApisixUpstreamLister
	v2Lister      listersv2.ApisixUpstreamLister
}

func (l *apisixUpstreamLister) V2beta3(namespace, name string) (ApisixUpstream, error) {
	au, err := l.v2beta3Lister.ApisixUpstreams(namespace).Get(name)
	if err != nil {
		return nil, err
	}
	return &apisixUpstream{
		groupVersion: config.ApisixV2beta3,
		v2beta3:      au,
	}, nil
}
func (l *apisixUpstreamLister) V2(namespace, name string) (ApisixUpstream, error) {
	au, err := l.v2Lister.ApisixUpstreams(namespace).Get(name)
	if err != nil {
		return nil, err
	}
	return &apisixUpstream{
		groupVersion: config.ApisixV2,
		v2:           au,
	}, nil
}

func (l *apisixUpstreamLister) ListV2(namespace string) ([]*configv2.ApisixUpstream, error) {
	return l.v2Lister.ApisixUpstreams(namespace).List(labels.Everything())
}

// MustNewApisixUpstream creates a kube.ApisixUpstream object according to the
// type of obj.
func MustNewApisixUpstream(obj interface{}) ApisixUpstream {
	switch au := obj.(type) {
	case *configv2beta3.ApisixUpstream:
		return &apisixUpstream{
			groupVersion: config.ApisixV2beta3,
			v2beta3:      au,
		}
	case *configv2.ApisixUpstream:
		return &apisixUpstream{
			groupVersion: config.ApisixV2,
			v2:           au,
		}
	default:
		panic("invalid ApisixUpstream type")
	}
}

// NewApisixUpstream creates a kube.ApisixUpstream object according to the
// type of obj. It returns nil and the error reason when the
// type assertion fails.
func NewApisixUpstream(obj interface{}) (ApisixUpstream, error) {
	switch au := obj.(type) {
	case *configv2beta3.ApisixUpstream:
		return &apisixUpstream{
			groupVersion: config.ApisixV2beta3,
			v2beta3:      au,
		}, nil
	case *configv2.ApisixUpstream:
		return &apisixUpstream{
			groupVersion: config.ApisixV2,
			v2:           au,
		}, nil
	default:
		return nil, errors.New("invalid ApisixUpstream type")
	}
}

func NewApisixUpstreamLister(v2beta3 listersv2beta3.ApisixUpstreamLister, v2 listersv2.ApisixUpstreamLister) ApisixUpstreamLister {
	return &apisixUpstreamLister{
		v2beta3Lister: v2beta3,
		v2Lister:      v2,
	}
}
