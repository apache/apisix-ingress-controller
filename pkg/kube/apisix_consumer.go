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

	"github.com/apache/apisix-ingress-controller/pkg/config"
	configv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	configv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta3"
	listersv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/listers/config/v2"
	listersv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/listers/config/v2beta3"
)

// ApisixConsumerLister is an encapsulation for the lister of ApisixConsumer,
// it aims at to be compatible with different ApisixConsumer versions.
type ApisixConsumerLister interface {
	// V2beta3 gets the ApisixConsumer in apisix.apache.org/v2beta3.
	V2beta3(string, string) (ApisixConsumer, error)
	// V2 gets the ApisixConsumer in apisix.apache.org/v2.
	V2(string, string) (ApisixConsumer, error)
}

// ApisixConsumerInformer is an encapsulation for the informer of ApisixConsumer,
// it aims at to be compatible with different ApisixConsumer versions.
type ApisixConsumerInformer interface {
	Run(chan struct{})
}

// ApisixConsumer is an encapsulation for ApisixConsumer resource with different
// versions, for now, they are apisix.apache.org/v2beta3 and apisix.apache.org/v2
type ApisixConsumer interface {
	// GroupVersion returns the api group version of the
	// real ApisixConsumer.
	GroupVersion() string
	// V2beta3 returns the ApisixConsumer in apisix.apache.org/v2beta3, the real
	// ApisixConsumer must be in this group version, otherwise will panic.
	V2beta3() *configv2beta3.ApisixConsumer
	// V2 returns the ApisixConsumer in apisix.apache.org/v2, the real
	// ApisixConsumer must be in this group version, otherwise will panic.
	V2() *configv2.ApisixConsumer
	// ResourceVersion returns the the resource version field inside
	// the real ApisixConsumer.
	ResourceVersion() string
}

// ApisixConsumerEvent contains the ApisixConsumer key (namespace/name)
// and the group version message.
type ApisixConsumerEvent struct {
	Key          string
	OldObject    ApisixConsumer
	GroupVersion string
}

type apisixConsumer struct {
	groupVersion string
	v2beta3      *configv2beta3.ApisixConsumer
	v2           *configv2.ApisixConsumer
}

func (ac *apisixConsumer) V2beta3() *configv2beta3.ApisixConsumer {
	if ac.groupVersion != config.ApisixV2beta3 {
		panic("not a apisix.apache.org/v2beta3 Consumer")
	}
	return ac.v2beta3
}

func (ac *apisixConsumer) V2() *configv2.ApisixConsumer {
	if ac.groupVersion != config.ApisixV2 {
		panic("not a apisix.apache.org/v2 Consumer")
	}
	return ac.v2
}

func (ac *apisixConsumer) GroupVersion() string {
	return ac.groupVersion
}

func (ac *apisixConsumer) ResourceVersion() string {
	if ac.groupVersion == config.ApisixV2beta3 {
		return ac.V2beta3().ResourceVersion
	}
	return ac.V2().ResourceVersion
}

type apisixConsumerLister struct {
	v2beta3Lister listersv2beta3.ApisixConsumerLister
	v2Lister      listersv2.ApisixConsumerLister
}

func (l *apisixConsumerLister) V2beta3(namespace, name string) (ApisixConsumer, error) {
	ac, err := l.v2beta3Lister.ApisixConsumers(namespace).Get(name)
	if err != nil {
		return nil, err
	}
	return &apisixConsumer{
		groupVersion: config.ApisixV2beta3,
		v2beta3:      ac,
	}, nil
}

func (l *apisixConsumerLister) V2(namespace, name string) (ApisixConsumer, error) {
	ac, err := l.v2Lister.ApisixConsumers(namespace).Get(name)
	if err != nil {
		return nil, err
	}
	return &apisixConsumer{
		groupVersion: config.ApisixV2,
		v2:           ac,
	}, nil
}

// MustNewApisixConsumer creates a kube.ApisixConsumer object according to the
// type of obj.
func MustNewApisixConsumer(obj interface{}) ApisixConsumer {
	switch ac := obj.(type) {
	case *configv2beta3.ApisixConsumer:
		return &apisixConsumer{
			groupVersion: config.ApisixV2beta3,
			v2beta3:      ac,
		}
	case *configv2.ApisixConsumer:
		return &apisixConsumer{
			groupVersion: config.ApisixV2,
			v2:           ac,
		}
	default:
		panic("invalid ApisixConsumer type")
	}
}

// NewApisixConsumer creates a kube.ApisixConsumer object according to the
// type of obj. It returns nil and the error reason when the
// type assertion fails.
func NewApisixConsumer(obj interface{}) (ApisixConsumer, error) {
	switch ac := obj.(type) {
	case *configv2beta3.ApisixConsumer:
		return &apisixConsumer{
			groupVersion: config.ApisixV2beta3,
			v2beta3:      ac,
		}, nil
	case *configv2.ApisixConsumer:
		return &apisixConsumer{
			groupVersion: config.ApisixV2,
			v2:           ac,
		}, nil
	default:
		return nil, errors.New("invalid ApisixConsumer type")
	}
}

func NewApisixConsumerLister(v2beta3 listersv2beta3.ApisixConsumerLister, v2 listersv2.ApisixConsumerLister) ApisixConsumerLister {
	return &apisixConsumerLister{
		v2beta3Lister: v2beta3,
		v2Lister:      v2,
	}
}
