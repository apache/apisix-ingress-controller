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

	configv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta3"
	listersv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/listers/config/v2beta3"
)

const (
	// ApisixPluginConfigV2beta3 represents the ApisixPluginConfig in apisix.apache.org/V2beta3 group version
	ApisixPluginConfigV2beta3 = "apisix.apache.org/V2beta3"
)

// ApisixPluginConfigLister is an encapsulation for the lister of ApisixPluginConfig,
// it aims at to be compatible with different ApisixPluginConfig versions.
type ApisixPluginConfigLister interface {
	// V2beta3 gets the ApisixPluginConfig in apisix.apache.org/V2beta3.
	V2beta3(string, string) (ApisixPluginConfig, error)
}

// ApisixPluginConfigInformer is an encapsulation for the informer of ApisixPluginConfig,
// it aims at to be compatible with different ApisixPluginConfig versions.
type ApisixPluginConfigInformer interface {
	Run(chan struct{})
}

// ApisixPluginConfig is an encapsulation for ApisixPluginConfig resource with different
// versions, for now, they are apisix.apache.org/v1 and apisix.apache.org/v2alpha1
type ApisixPluginConfig interface {
	// GroupVersion returns the api group version of the
	// real ApisixPluginConfig.
	GroupVersion() string
	// V2beta3 returns the ApisixPluginConfig in apisix.apache.org/V2beta3, the real
	// ApisixPluginConfig must be in this group version, otherwise will panic.
	V2beta3() *configv2beta3.ApisixPluginConfig
	// ResourceVersion returns the the resource version field inside
	// the real ApisixPluginConfig.
	ResourceVersion() string
}

// ApisixPluginConfigEvent contains the ApisixPluginConfig key (namespace/name)
// and the group version message.
type ApisixPluginConfigEvent struct {
	Key          string
	OldObject    ApisixPluginConfig
	GroupVersion string
}

type apisixPluginConfig struct {
	groupVersion string
	v2beta3      *configv2beta3.ApisixPluginConfig
}

func (apc *apisixPluginConfig) V2beta3() *configv2beta3.ApisixPluginConfig {
	if apc.groupVersion != ApisixPluginConfigV2beta3 {
		panic("not a apisix.apache.org/V2beta3 pluginConfig")
	}
	return apc.v2beta3
}

func (apc *apisixPluginConfig) GroupVersion() string {
	return apc.groupVersion
}

func (apc *apisixPluginConfig) ResourceVersion() string {
	return apc.V2beta3().ResourceVersion
}

type apisixPluginConfigLister struct {
	v2beta3Lister listersv2beta3.ApisixPluginConfigLister
}

func (l *apisixPluginConfigLister) V2beta3(namespace, name string) (ApisixPluginConfig, error) {
	apc, err := l.v2beta3Lister.ApisixPluginConfigs(namespace).Get(name)
	if err != nil {
		return nil, err
	}
	return &apisixPluginConfig{
		groupVersion: ApisixPluginConfigV2beta3,
		v2beta3:      apc,
	}, nil
}

// MustNewApisixPluginConfig creates a kube.ApisixPluginConfig object according to the
// type of obj.
func MustNewApisixPluginConfig(obj interface{}) ApisixPluginConfig {
	switch apc := obj.(type) {
	case *configv2beta3.ApisixPluginConfig:
		return &apisixPluginConfig{
			groupVersion: ApisixPluginConfigV2beta3,
			v2beta3:      apc,
		}
	default:
		panic("invalid ApisixPluginConfig type")
	}
}

// NewApisixPluginConfig creates a kube.ApisixPluginConfig object according to the
// type of obj. It returns nil and the error reason when the
// type assertion fails.
func NewApisixPluginConfig(obj interface{}) (ApisixPluginConfig, error) {
	switch apc := obj.(type) {
	case *configv2beta3.ApisixPluginConfig:
		return &apisixPluginConfig{
			groupVersion: ApisixPluginConfigV2beta3,
			v2beta3:      apc,
		}, nil
	default:
		return nil, errors.New("invalid ApisixPluginConfig type")
	}
}

func NewApisixPluginConfigLister(v2beta3 listersv2beta3.ApisixPluginConfigLister) ApisixPluginConfigLister {
	return &apisixPluginConfigLister{
		v2beta3Lister: v2beta3,
	}
}
