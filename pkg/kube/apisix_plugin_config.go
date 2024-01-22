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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/apisix-ingress-controller/pkg/config"
	configv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	listersv2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/client/listers/config/v2"
)

// ApisixPluginConfigLister is an encapsulation for the lister of ApisixPluginConfig,
// it aims at to be compatible with different ApisixPluginConfig versions.
type ApisixPluginConfigLister interface {
	// V2 gets the ApisixPluginConfig in apisix.apache.org/v2.
	V2(string, string) (ApisixPluginConfig, error)
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
	// V2 returns the ApisixPluginConfig in apisix.apache.org/v2, the real
	// ApisixPluginConfig must be in this group version, otherwise will panic.
	V2() *configv2.ApisixPluginConfig
	// ResourceVersion returns the the resource version field inside
	// the real ApisixPluginConfig.
	ResourceVersion() string

	metav1.Object
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
	v2           *configv2.ApisixPluginConfig
	metav1.Object
}

func (apc *apisixPluginConfig) V2() *configv2.ApisixPluginConfig {
	if apc.groupVersion != config.ApisixV2 {
		panic("not a apisix.apache.org/v2 pluginConfig")
	}
	return apc.v2
}

func (apc *apisixPluginConfig) GroupVersion() string {
	return apc.groupVersion
}

func (apc *apisixPluginConfig) ResourceVersion() string {
	return apc.V2().ResourceVersion
}

type apisixPluginConfigLister struct {
	v2Lister listersv2.ApisixPluginConfigLister
}

func (l *apisixPluginConfigLister) V2(namespace, name string) (ApisixPluginConfig, error) {
	apc, err := l.v2Lister.ApisixPluginConfigs(namespace).Get(name)
	if err != nil {
		return nil, err
	}
	return &apisixPluginConfig{
		groupVersion: config.ApisixV2,
		v2:           apc,
		Object:       apc,
	}, nil
}

// MustNewApisixPluginConfig creates a kube.ApisixPluginConfig object according to the
// type of obj.
func MustNewApisixPluginConfig(obj interface{}) ApisixPluginConfig {
	switch apc := obj.(type) {
	case *configv2.ApisixPluginConfig:
		return &apisixPluginConfig{
			groupVersion: config.ApisixV2,
			v2:           apc,
			Object:       apc,
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
	case *configv2.ApisixPluginConfig:
		return &apisixPluginConfig{
			groupVersion: config.ApisixV2,
			v2:           apc,
			Object:       apc,
		}, nil
	default:
		return nil, errors.New("invalid ApisixPluginConfig type")
	}
}

func NewApisixPluginConfigLister(v2 listersv2.ApisixPluginConfigLister) ApisixPluginConfigLister {
	return &apisixPluginConfigLister{
		v2Lister: v2,
	}
}
