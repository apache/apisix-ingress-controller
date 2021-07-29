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

	configv1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v1"
	configv2alpha1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2alpha1"
	"github.com/apache/apisix-ingress-controller/pkg/log"
)

const (
	_conditionType        = "ResourcesAvailable"
	_commonSuccessMessage = "Sync Successfully"
)

// recordStatus record resources status
func (c *Controller) recordStatus(at interface{}, reason string, err error, status v1.ConditionStatus) {
	// build condition
	message := _commonSuccessMessage
	if err != nil {
		message = err.Error()
	}
	condition := metav1.Condition{
		Type:    _conditionType,
		Reason:  reason,
		Status:  status,
		Message: message,
	}
	client := c.kubeClient.APISIXClient

	switch v := at.(type) {
	case *configv1.ApisixTls:
		// set to status
		if v.Status.Conditions == nil {
			conditions := make([]metav1.Condition, 0)
			v.Status.Conditions = &conditions
		}
		meta.SetStatusCondition(v.Status.Conditions, condition)
		if _, errRecord := client.ApisixV1().ApisixTlses(v.Namespace).
			UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
			log.Errorw("failed to record status change for ApisixTls",
				zap.Error(errRecord),
				zap.String("name", v.Name),
				zap.String("namespace", v.Namespace),
			)
		}
	case *configv1.ApisixUpstream:
		// set to status
		if v.Status.Conditions == nil {
			conditions := make([]metav1.Condition, 0)
			v.Status.Conditions = &conditions
		}
		meta.SetStatusCondition(v.Status.Conditions, condition)
		if _, errRecord := client.ApisixV1().ApisixUpstreams(v.Namespace).
			UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
			log.Errorw("failed to record status change for ApisixUpstream",
				zap.Error(errRecord),
				zap.String("name", v.Name),
				zap.String("namespace", v.Namespace),
			)
		}
	case *configv2alpha1.ApisixRoute:
		// set to status
		if v.Status.Conditions == nil {
			conditions := make([]metav1.Condition, 0)
			v.Status.Conditions = &conditions
		}
		meta.SetStatusCondition(v.Status.Conditions, condition)
		if _, errRecord := client.ApisixV2alpha1().ApisixRoutes(v.Namespace).
			UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
			log.Errorw("failed to record status change for ApisixRoute",
				zap.Error(errRecord),
				zap.String("name", v.Name),
				zap.String("namespace", v.Namespace),
			)
		}
	case *configv2alpha1.ApisixConsumer:
		// set to status
		if v.Status.Conditions == nil {
			conditions := make([]metav1.Condition, 0)
			v.Status.Conditions = &conditions
		}
		meta.SetStatusCondition(v.Status.Conditions, condition)
		if _, errRecord := client.ApisixV2alpha1().ApisixConsumers(v.Namespace).
			UpdateStatus(context.TODO(), v, metav1.UpdateOptions{}); errRecord != nil {
			log.Errorw("failed to record status change for ApisixConsumer",
				zap.Error(errRecord),
				zap.String("name", v.Name),
				zap.String("namespace", v.Namespace),
			)
		}
	default:
		// This should not be executed
		log.Errorf("unsupported resource record: %s", v)
	}
}
