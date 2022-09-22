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

package validation

import (
	"context"
	"errors"
	"fmt"
	"strings"

	kwhmodel "github.com/slok/kubewebhook/v2/pkg/model"
	kwhvalidating "github.com/slok/kubewebhook/v2/pkg/webhook/validating"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/apisix-ingress-controller/pkg/apisix"
	v2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2"
	"github.com/apache/apisix-ingress-controller/pkg/log"
)

var (
	ApisixRouteV2GVR = metav1.GroupVersionResource{
		Group:    v2.GroupVersion.Group,
		Version:  v2.GroupVersion.Version,
		Resource: "apisixroutes",
	}

	ApisixPluginConfigV2GVR = metav1.GroupVersionResource{
		Group:    v2.GroupVersion.Group,
		Version:  v2.GroupVersion.Version,
		Resource: "apisixpluginconfigs",
	}
)

var (
	validateHandler = map[metav1.GroupVersionResource]kwhvalidating.ValidatorFunc{
		ApisixRouteV2GVR:        validateApisixRouteV2,
		ApisixPluginConfigV2GVR: validateApisixPluginConfigV2,
	}
)

var ApisixValidator = kwhvalidating.ValidatorFunc(
	func(ctx context.Context, review *kwhmodel.AdmissionReview, object metav1.Object) (result *kwhvalidating.ValidatorResult, err error) {
		GVR := review.RequestGVR
		if validatorFunc, ok := validateHandler[*GVR]; ok {
			return validatorFunc(ctx, review, object)
		}
		errStr := fmt.Sprintf("{group: %s, version: %s, Resource: %s} not supported", GVR.Group, GVR.Version, GVR.Resource)
		return &kwhvalidating.ValidatorResult{
			Valid:   false,
			Message: errStr,
		}, errors.New(errStr)
	},
)

// ApisixRouteValidator validates ApisixRoute and its plugins.
// When the validation of one plugin fails, it will continue to validate the rest of plugins.
func validateApisixRouteV2(ctx context.Context, review *kwhmodel.AdmissionReview, object metav1.Object) (result *kwhvalidating.ValidatorResult, err error) {
	log.Debugw("arrive ApisixRoute validator webhook", zap.Any("object", object))

	ar := object.(*v2.ApisixRoute)
	valid := true
	var msgs []string

	client, err := GetSchemaClient(&apisix.ClusterOptions{})
	if err != nil {
		msg := "failed to get the schema client"
		log.Errorf("%s: %s", msg, err)
		return &kwhvalidating.ValidatorResult{Valid: false, Message: msg}, err
	}

	for _, h := range ar.Spec.HTTP {
		for _, p := range h.Plugins {
			if p.Enable {
				if v, err := ValidatePlugin(client, p.Name, p.Config); !v {
					valid = false
					msgs = append(msgs, err.Error())
					log.Warnf("failed to validate plugin %s: %s", p.Name, err)
				}
			}
		}
	}

	return &kwhvalidating.ValidatorResult{Valid: valid, Message: strings.Join(msgs, "\n")}, nil
}

func validateApisixPluginConfigV2(ctx context.Context, review *kwhmodel.AdmissionReview, object metav1.Object) (result *kwhvalidating.ValidatorResult, err error) {
	log.Debugw("arrive ApisixPluginConfig validator webhook", zap.Any("object", object))

	apc := object.(*v2.ApisixPluginConfig)
	valid := true
	var msgs []string

	client, err := GetSchemaClient(&apisix.ClusterOptions{})
	if err != nil {
		msg := "failed to get the schema client"
		log.Errorf("%s: %s", msg, err)
		return &kwhvalidating.ValidatorResult{Valid: false, Message: msg}, err
	}

	for _, plugin := range apc.Spec.Plugins {
		if plugin.Enable {
			if v, err := ValidatePlugin(client, plugin.Name, plugin.Config); !v {
				valid = false
				msgs = append(msgs, err.Error())
				log.Warnf("failed to validate plugin %s: %s", plugin.Name, err)
			}
		}
	}

	return &kwhvalidating.ValidatorResult{Valid: valid, Message: strings.Join(msgs, "\n")}, nil
}
