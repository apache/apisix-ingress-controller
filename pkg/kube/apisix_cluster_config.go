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
	"github.com/apache/apisix-ingress-controller/pkg/config"
	configv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	listersv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/listers/config/v2"

	configv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta3"
	listersv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/listers/config/v2beta3"
)

// ApisixClusterConfigLister is an encapsulation for the lister of ApisixClusterConfig,
// it aims at to be compatible with different ApisixClusterConfig versions.
type ApisixClusterConfigLister interface {
	// V2beta3 gets the ApisixClusterConfig in apisix.apache.org/v2beta3.
	V2beta3(string) (ApisixClusterConfig, error)
	// V2 gets the ApisixClusterConfig in apisix.apache.org/v2.
	V2(string) (ApisixClusterConfig, error)
}

// ApisixClusterConfigInformer is an encapsulation for the informer of ApisixClusterConfig,
// it aims at to be compatible with different ApisixClusterConfig versions.
type ApisixClusterConfigInformer interface {
	Run(chan struct{})
}

// ApisixClusterConfig is an encapsulation for ApisixClusterConfig resource with different
// versions, for now, they are apisix.apache.org/v1 and apisix.apache.org/v2alpha1
type ApisixClusterConfig interface {
	// GroupVersion returns the api group version of the
	// real ApisixClusterConfig.
	GroupVersion() string
	// V2beta3 returns the ApisixClusterConfig in apisix.apache.org/v2beta3, the real
	// ApisixClusterConfig must be in this group version, otherwise will panic.
	V2beta3() *configv2beta3.ApisixClusterConfig
	// V2 returns the ApisixClusterConfig in apisix.apache.org/v2, the real
	// ApisixClusterConfig must be in this group version, otherwise will panic.
	V2() *configv2.ApisixClusterConfig
	// ResourceVersion returns the the resource version field inside
	// the real ApisixClusterConfig.
	ResourceVersion() string
}

// ApisixClusterConfigEvent contains the ApisixClusterConfig key (namespace/name)
// and the group version message.
type ApisixClusterConfigEvent struct {
	Key          string
	OldObject    ApisixClusterConfig
	GroupVersion string
}

type apisixClusterConfig struct {
	groupVersion string
	v2beta3      *configv2beta3.ApisixClusterConfig
	v2           *configv2.ApisixClusterConfig
}

func (acc *apisixClusterConfig) V2beta3() *configv2beta3.ApisixClusterConfig {
	if acc.groupVersion != config.ApisixV2beta3 {
		panic("not a apisix.apache.org/v2beta3 apisixClusterConfig")
	}
	return acc.v2beta3
}

func (acc *apisixClusterConfig) V2() *configv2.ApisixClusterConfig {
	if acc.groupVersion != config.ApisixV2 {
		panic("not a apisix.apache.org/v2 apisixClusterConfig")
	}
	return acc.v2
}

func (acc *apisixClusterConfig) GroupVersion() string {
	return acc.groupVersion
}

func (acc *apisixClusterConfig) ResourceVersion() string {
	if acc.groupVersion == config.ApisixV2beta3 {
		return acc.V2beta3().ResourceVersion
	}
	return acc.V2().ResourceVersion
}

type apisixClusterConfigLister struct {
	v2beta3Lister listersv2beta3.ApisixClusterConfigLister
	v2Lister      listersv2.ApisixClusterConfigLister
}

func (l *apisixClusterConfigLister) V2beta3(name string) (ApisixClusterConfig, error) {
	acc, err := l.v2beta3Lister.Get(name)
	if err != nil {
		return nil, err
	}
	return &apisixClusterConfig{
		groupVersion: config.ApisixV2beta3,
		v2beta3:      acc,
	}, nil
}

func (l *apisixClusterConfigLister) V2(name string) (ApisixClusterConfig, error) {
	acc, err := l.v2Lister.Get(name)
	if err != nil {
		return nil, err
	}
	return &apisixClusterConfig{
		groupVersion: config.ApisixV2,
		v2:           acc,
	}, nil
}

// MustNewApisixClusterConfig creates a kube.ApisixClusterConfig object according to the
// type of obj.
func MustNewApisixClusterConfig(obj interface{}) ApisixClusterConfig {
	switch acc := obj.(type) {
	case *configv2beta3.ApisixClusterConfig:
		return &apisixClusterConfig{
			groupVersion: config.ApisixV2beta3,
			v2beta3:      acc,
		}
	case *configv2.ApisixClusterConfig:
		return &apisixClusterConfig{
			groupVersion: config.ApisixV2,
			v2:           acc,
		}
	default:
		panic("invalid ApisixClusterConfig type")
	}
}

// NewApisixClusterConfig creates a kube.ApisixClusterConfig object according to the
// type of obj. It returns nil and the error reason when the
// type assertion fails.
func NewApisixClusterConfig(obj interface{}) (ApisixClusterConfig, error) {
	switch acc := obj.(type) {
	case *configv2beta3.ApisixClusterConfig:
		return &apisixClusterConfig{
			groupVersion: config.ApisixV2beta3,
			v2beta3:      acc,
		}, nil
	case *configv2.ApisixClusterConfig:
		return &apisixClusterConfig{
			groupVersion: config.ApisixV2,
			v2:           acc,
		}, nil
	default:
		return nil, fmt.Errorf("invalid ApisixClusterConfig type %T", acc)
	}
}

func NewApisixClusterConfigLister(v2beta3 listersv2beta3.ApisixClusterConfigLister, v2 listersv2.ApisixClusterConfigLister) ApisixClusterConfigLister {
	return &apisixClusterConfigLister{
		v2beta3Lister: v2beta3,
		v2Lister:      v2,
	}
}
