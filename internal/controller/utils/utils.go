// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/utils"
)

func SetApisixCRDConditionWithGeneration(status *apiv2.ApisixStatus, generation int64, condition metav1.Condition) {
	condition.ObservedGeneration = generation
	SetApisixCRDCondition(status, condition)
}

func SetApisixCRDCondition(status *apiv2.ApisixStatus, condition metav1.Condition) {
	for i, cond := range status.Conditions {
		if cond.Type == condition.Type {
			if cond.Status == condition.Status &&
				cond.ObservedGeneration > condition.ObservedGeneration {
				return
			}
			status.Conditions[i] = condition
			return
		}
	}

	status.Conditions = append(status.Conditions, condition)
}

func NewConditionTypeAccepted(reason apiv2.ApisixRouteConditionReason, status bool, generation int64, msg string) metav1.Condition {
	var condition = metav1.Condition{
		Type:               string(apiv2.ConditionTypeAccepted),
		Status:             utils.ConditionStatus(status),
		ObservedGeneration: generation,
		LastTransitionTime: metav1.Now(),
		Reason:             string(reason),
		Message:            msg,
	}
	return condition
}

func MergeCondition(conditions []metav1.Condition, newCondition metav1.Condition) []metav1.Condition {
	if newCondition.LastTransitionTime.IsZero() {
		newCondition.LastTransitionTime = metav1.Now()
	}
	newConditions := []metav1.Condition{}
	for _, condition := range conditions {
		if condition.Type != newCondition.Type {
			newConditions = append(newConditions, condition)
		}
	}
	newConditions = append(newConditions, newCondition)
	return newConditions
}
