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

	configv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	configv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta3"
	listersv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/listers/config/v2"
	listersv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/listers/config/v2beta3"
)

const (
	// ApisixUpstreamV2beta3 represents the ApisixUpstream in apisix.apache.org/v2beta3 group version
	ApisixUpstreamV2beta3 = "apisix.apache.org/v2beta3"
	// ApisixUpstreamV2 represents the ApisixUpstream in apisix.apache.org/v2 group version
	ApisixUpstreamV2 = "apisix.apache.org/v2"
)

// ApisixUpstreamLister is an encapsulation for the lister of ApisixUpstream,
// it aims at to be compatible with different ApisixUpstream versions.
type ApisixUpstreamLister interface {
	// V2beta3 gets the ApisixUpstream in apisix.apache.org/v2beta3.
	V2beta3(string, string) (ApisixUpstream, error)
	// V2 gets the ApisixUpstream in apisix.apache.org/v2.
	V2(string, string) (ApisixUpstream, error)
}

// ApisixUpstreamInformer is an encapsulation for the informer of ApisixUpstream,
// it aims at to be compatible with different ApisixUpstream versions.
type ApisixUpstreamInformer interface {
	Run(chan struct{})
}

// ApisixUpstream is an encapsulation for ApisixUpstream resource with different
// versions, for now, they are apisix.apache.org/v1 and apisix.apache.org/v2alpha1
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

func (ar *apisixUpstream) V2beta3() *configv2beta3.ApisixUpstream {
	if ar.groupVersion != ApisixUpstreamV2beta3 {
		panic("not a apisix.apache.org/v2beta3 Upstream")
	}
	return ar.v2beta3
}
func (ar *apisixUpstream) V2() *configv2.ApisixUpstream {
	if ar.groupVersion != ApisixUpstreamV2 {
		panic("not a apisix.apache.org/v2 Upstream")
	}
	return ar.v2
}

func (ar *apisixUpstream) GroupVersion() string {
	return ar.groupVersion
}

func (ar *apisixUpstream) ResourceVersion() string {
	if ar.groupVersion == ApisixUpstreamV2beta3 {
		return ar.V2beta3().ResourceVersion
	}
	return ar.V2().ResourceVersion
}

type apisixUpstreamLister struct {
	v2beta3Lister listersv2beta3.ApisixUpstreamLister
	v2Lister      listersv2.ApisixUpstreamLister
}

func (l *apisixUpstreamLister) V2beta3(namespace, name string) (ApisixUpstream, error) {
	ar, err := l.v2beta3Lister.ApisixUpstreams(namespace).Get(name)
	if err != nil {
		return nil, err
	}
	return &apisixUpstream{
		groupVersion: ApisixUpstreamV2beta3,
		v2beta3:      ar,
	}, nil
}
func (l *apisixUpstreamLister) V2(namespace, name string) (ApisixUpstream, error) {
	ar, err := l.v2Lister.ApisixUpstreams(namespace).Get(name)
	if err != nil {
		return nil, err
	}
	return &apisixUpstream{
		groupVersion: ApisixUpstreamV2,
		v2:           ar,
	}, nil
}

// MustNewApisixUpstream creates a kube.ApisixUpstream object according to the
// type of obj.
func MustNewApisixUpstream(obj interface{}) ApisixUpstream {
	switch ar := obj.(type) {
	case *configv2beta3.ApisixUpstream:
		return &apisixUpstream{
			groupVersion: ApisixUpstreamV2beta3,
			v2beta3:      ar,
		}
	case *configv2.ApisixUpstream:
		return &apisixUpstream{
			groupVersion: ApisixUpstreamV2,
			v2:           ar,
		}
	default:
		panic("invalid ApisixUpstream type")
	}
}

// NewApisixUpstream creates a kube.ApisixUpstream object according to the
// type of obj. It returns nil and the error reason when the
// type assertion fails.
func NewApisixUpstream(obj interface{}) (ApisixUpstream, error) {
	switch ar := obj.(type) {
	case *configv2beta3.ApisixUpstream:
		return &apisixUpstream{
			groupVersion: ApisixUpstreamV2beta3,
			v2beta3:      ar,
		}, nil
	case *configv2.ApisixUpstream:
		return &apisixUpstream{
			groupVersion: ApisixUpstreamV2,
			v2:           ar,
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
