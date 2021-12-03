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

package ingress

import (
	"context"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configv2beta1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta1"
	configv2beta2 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta2"
	configv2beta3 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2beta3"
	"github.com/apache/apisix-ingress-controller/pkg/log"
)

const (
	_conditionType        = "ResourcesAvailable"
	_commonSuccessMessage = "Sync Successfully"
)

// verifyGeneration verify generation to decide whether to update status
func (c *Controller) verifyGeneration(conditions *[]metav1.Condition, newCondition metav1.Condition) bool {
	existingCondition := meta.FindStatusCondition(*conditions, newCondition.Type)
	if existingCondition != nil && existingCondition.ObservedGeneration >= newCondition.ObservedGeneration {
		return false
	}
	return true
}

// recordStatus record resources status
func (c *Controller) recordStatus(at interface{}, reason string, err error, status v1.ConditionStatus, generation int64) {
	// build condition
	message := _commonSuccessMessage
	if err != nil {
		message = err.Error()
	}
	condition := metav1.Condition{
		Type:               _conditionType,
		Reason:             reason,
		Status:             status,
		Message:            message,
		ObservedGeneration: generation,
	}
	client := c.kubeClient.APISIXClient

	switch v := at.(type) {
	case *configv2beta3.ApisixTls:
		// set to status
		if v.Status.Conditions == nil {
			conditions := make([]metav1.Condition, 0)
			v.Status.Conditions = conditions
		}
		if c.verifyGeneration(&v.Status.Conditions, condition) {
			meta.SetStatusCondition(&v.Status.Conditions, condition)
			if _, errRecord := client.ApisixV2beta3().ApisixTlses(v.Namespace).
				UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
				log.Errorw("failed to record status change for ApisixTls",
					zap.Error(errRecord),
					zap.String("name", v.Name),
					zap.String("namespace", v.Namespace),
				)
			}
		}
	case *configv2beta3.ApisixUpstream:
		// set to status
		if v.Status.Conditions == nil {
			conditions := make([]metav1.Condition, 0)
			v.Status.Conditions = conditions
		}
		if c.verifyGeneration(&v.Status.Conditions, condition) {
			meta.SetStatusCondition(&v.Status.Conditions, condition)
			if _, errRecord := client.ApisixV2beta3().ApisixUpstreams(v.Namespace).
				UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
				log.Errorw("failed to record status change for ApisixUpstream",
					zap.Error(errRecord),
					zap.String("name", v.Name),
					zap.String("namespace", v.Namespace),
				)
			}
		}
	case *configv2beta1.ApisixRoute:
		// set to status
		if v.Status.Conditions == nil {
			conditions := make([]metav1.Condition, 0)
			v.Status.Conditions = conditions
		}
		if c.verifyGeneration(&v.Status.Conditions, condition) {
			meta.SetStatusCondition(&v.Status.Conditions, condition)
			if _, errRecord := client.ApisixV2beta1().ApisixRoutes(v.Namespace).
				UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
				log.Errorw("failed to record status change for ApisixRoute",
					zap.Error(errRecord),
					zap.String("name", v.Name),
					zap.String("namespace", v.Namespace),
				)
			}
		}
	case *configv2beta2.ApisixRoute:
		// set to status
		if v.Status.Conditions == nil {
			conditions := make([]metav1.Condition, 0)
			v.Status.Conditions = conditions
		}
		if c.verifyGeneration(&v.Status.Conditions, condition) {
			meta.SetStatusCondition(&v.Status.Conditions, condition)
			if _, errRecord := client.ApisixV2beta2().ApisixRoutes(v.Namespace).
				UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
				log.Errorw("failed to record status change for ApisixRoute",
					zap.Error(errRecord),
					zap.String("name", v.Name),
					zap.String("namespace", v.Namespace),
				)
			}
		}
	case *configv2beta3.ApisixRoute:
		// set to status
		if v.Status.Conditions == nil {
			conditions := make([]metav1.Condition, 0)
			v.Status.Conditions = conditions
		}
		if c.verifyGeneration(&v.Status.Conditions, condition) {
			meta.SetStatusCondition(&v.Status.Conditions, condition)
			if _, errRecord := client.ApisixV2beta3().ApisixRoutes(v.Namespace).
				UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
				log.Errorw("failed to record status change for ApisixRoute",
					zap.Error(errRecord),
					zap.String("name", v.Name),
					zap.String("namespace", v.Namespace),
				)
			}
		}
	case *configv2beta3.ApisixConsumer:
		// set to status
		if v.Status.Conditions == nil {
			conditions := make([]metav1.Condition, 0)
			v.Status.Conditions = conditions
		}
		if c.verifyGeneration(&v.Status.Conditions, condition) {
			meta.SetStatusCondition(&v.Status.Conditions, condition)
			if _, errRecord := client.ApisixV2beta3().ApisixConsumers(v.Namespace).
				UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
				log.Errorw("failed to record status change for ApisixConsumer",
					zap.Error(errRecord),
					zap.String("name", v.Name),
					zap.String("namespace", v.Namespace),
				)
			}
		}
	default:
		// This should not be executed
		log.Errorf("unsupported resource record: %s", v)
	}
}
