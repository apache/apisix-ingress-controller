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

// ApisixGlobalRuleLister is an encapsulation for the lister of ApisixGlobalRule,
// it aims at to be compatible with different ApisixGlobalRule versions.
type ApisixGlobalRuleLister interface {
	// V2 gets the ApisixGlobalRule in apisix.apache.org/v2.
	V2(string, string) (ApisixGlobalRule, error)

	ApisixGlobalRule(string, string) (ApisixGlobalRule, error)
}

// ApisixGlobalRuleInformer is an encapsulation for the informer of ApisixGlobalRule,
// it aims at to be compatible with different ApisixGlobalRule versions.
type ApisixGlobalRuleInformer interface {
	Run(chan struct{})
}

// ApisixGlobalRule is an encapsulation for ApisixGlobalRule resource with different
// versions, for now, they are apisix.apache.org/v1 and apisix.apache.org/v2alpha1
type ApisixGlobalRule interface {
	// GroupVersion returns the api group version of the
	// real ApisixGlobalRule.
	GroupVersion() string
	// V2 returns the ApisixGlobalRule in apisix.apache.org/v2, the real
	// ApisixGlobalRule must be in this group version, otherwise will panic.
	V2() *configv2.ApisixGlobalRule
	// ResourceVersion returns the the resource version field inside
	// the real ApisixGlobalRule.
	ResourceVersion() string

	metav1.Object
}

// ApisixGlobalRuleEvent contains the ApisixGlobalRule key (namespace/name)
// and the group version message.
type ApisixGlobalRuleEvent struct {
	Key          string
	OldObject    ApisixGlobalRule
	GroupVersion string
}

type apisixGlobalRule struct {
	groupVersion string
	v2           *configv2.ApisixGlobalRule
	metav1.Object
}

func (agr *apisixGlobalRule) V2() *configv2.ApisixGlobalRule {
	if agr.groupVersion != config.ApisixV2 {
		panic("not a apisix.apache.org/v2 ApisixGlobalRule")
	}
	return agr.v2
}

func (agr *apisixGlobalRule) GroupVersion() string {
	return agr.groupVersion
}

func (agr *apisixGlobalRule) ResourceVersion() string {
	return agr.V2().ResourceVersion
}

type apisixGlobalRuleLister struct {
	groupVersion string
	v2Lister     listersv2.ApisixGlobalRuleLister
}

func (l *apisixGlobalRuleLister) V2(namespace, name string) (ApisixGlobalRule, error) {
	agr, err := l.v2Lister.ApisixGlobalRules(namespace).Get(name)
	if err != nil {
		return nil, err
	}
	return &apisixGlobalRule{
		groupVersion: config.ApisixV2,
		v2:           agr,
		Object:       agr.GetObjectMeta(),
	}, nil
}

func (l *apisixGlobalRuleLister) ApisixGlobalRule(namespace, name string) (ApisixGlobalRule, error) {
	switch l.groupVersion {
	case config.ApisixV2:
		agr, err := l.v2Lister.ApisixGlobalRules(namespace).Get(name)
		if err != nil {
			return nil, err
		}
		return &apisixGlobalRule{
			groupVersion: config.ApisixV2,
			v2:           agr,
		}, nil
	default:
		panic("invalid ApisixGlobalRule group version")
	}
}

// MustNewApisixGlobalRule creates a kube.ApisixGlobalRule object according to the
// type of obj.
func MustNewApisixGlobalRule(obj interface{}) ApisixGlobalRule {
	switch agr := obj.(type) {
	case *configv2.ApisixGlobalRule:
		return &apisixGlobalRule{
			groupVersion: config.ApisixV2,
			v2:           agr,
			Object:       agr.GetObjectMeta(),
		}
	default:
		panic("invalid ApisixGlobalRule type")
	}
}

// NewApisixGlobalRule creates a kube.ApisixGlobalRule object according to the
// type of obj. It returns nil and the error reason when the
// type assertion fails.
func NewApisixGlobalRule(obj interface{}) (ApisixGlobalRule, error) {
	switch agr := obj.(type) {
	case *configv2.ApisixGlobalRule:
		return &apisixGlobalRule{
			groupVersion: config.ApisixV2,
			v2:           agr,
			Object:       agr.GetObjectMeta(),
		}, nil
	default:
		return nil, errors.New("invalid ApisixGlobalRule type")
	}
}

func NewApisixGlobalRuleLister(apiVersion string, v2 listersv2.ApisixGlobalRuleLister) ApisixGlobalRuleLister {
	return &apisixGlobalRuleLister{
		groupVersion: apiVersion,
		v2Lister:     v2,
	}
}
