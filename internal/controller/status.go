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

package controller

import (
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	"github.com/apache/apisix-ingress-controller/internal/provider"
)

const (
	ConditionTypeAvailable   string = "Available"
	ConditionTypeProgressing string = "Progressing"
	ConditionTypeDegraded    string = "Degraded"

	ConditionReasonSynced    string = "ResourceSynced"
	ConditionReasonSyncAbort string = "ResourceSyncAbort"
)

func NewCondition(observedGeneration int64, status bool, message string) metav1.Condition {
	condition := metav1.ConditionTrue
	reason := ConditionReasonSynced
	if !status {
		condition = metav1.ConditionFalse
		reason = ConditionReasonSyncAbort
	}
	return metav1.Condition{
		Type:               ConditionTypeAvailable,
		Reason:             reason,
		Status:             condition,
		Message:            message,
		ObservedGeneration: observedGeneration,
	}
}

func VerifyConditions(conditions *[]metav1.Condition, newCondition metav1.Condition) bool {
	existingCondition := meta.FindStatusCondition(*conditions, newCondition.Type)
	if existingCondition == nil {
		return true
	}

	if existingCondition.ObservedGeneration > newCondition.ObservedGeneration {
		return false
	}
	if *existingCondition == newCondition {
		return false
	}
	return true
}

func NewPolicyCondition(observedGeneration int64, status bool, message string) metav1.Condition {
	conditionStatus := metav1.ConditionTrue
	reason := string(gatewayv1alpha2.PolicyReasonAccepted)
	if !status {
		conditionStatus = metav1.ConditionFalse
		reason = string(gatewayv1alpha2.PolicyReasonInvalid)
	}

	return metav1.Condition{
		Type:               string(gatewayv1alpha2.PolicyConditionAccepted),
		Reason:             reason,
		Status:             conditionStatus,
		Message:            message,
		ObservedGeneration: observedGeneration,
		LastTransitionTime: metav1.Now(),
	}
}

func NewPolicyConflictCondition(observedGeneration int64, message string) metav1.Condition {
	return metav1.Condition{
		Type:               string(gatewayv1alpha2.PolicyConditionAccepted),
		Reason:             string(gatewayv1alpha2.PolicyReasonConflicted),
		Status:             metav1.ConditionFalse,
		Message:            message,
		ObservedGeneration: observedGeneration,
		LastTransitionTime: metav1.Now(),
	}
}

func UpdateStatus(
	updater status.Updater,
	log logr.Logger,
	tctx *provider.TranslateContext,
) {
	for _, update := range tctx.StatusUpdaters {
		updater.Update(update)
	}
}
