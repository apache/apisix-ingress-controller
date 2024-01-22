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

package utils

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
)

const (
	ConditionType        = "ResourcesAvailable"
	CommonSuccessMessage = "Sync Successfully"

	// Component is used for event component
	Component = "ApisixIngress"
	// ResourceSynced is used when a resource is synced successfully
	ResourceSynced = "ResourcesSynced"
	// MessageResourceSynced is used to specify controller
	MessageResourceSynced = "%s synced successfully"
	// ResourceSyncAborted is used when a resource synced failed
	ResourceSyncAborted = "ResourceSyncAborted"
	// MessageResourceFailed is used to report error
	MessageResourceFailed = "%s synced failed, with error: %s"
)

// RecorderEvent recorder events for resources
func RecorderEvent(recorder record.EventRecorder, object runtime.Object, eventtype, reason string, err error) {
	if err != nil {
		message := fmt.Sprintf(MessageResourceFailed, Component, err.Error())
		recorder.Event(object, eventtype, reason, message)
	} else {
		message := fmt.Sprintf(MessageResourceSynced, Component)
		recorder.Event(object, eventtype, reason, message)
	}
}

// RecorderEventS recorder events for resources
func RecorderEventS(recorder record.EventRecorder, object runtime.Object, eventtype, reason string, msg string) {
	recorder.Event(object, eventtype, reason, msg)
}

// VerifyGeneration verify generation to decide whether to update status
func VerifyGeneration(conditions *[]metav1.Condition, newCondition metav1.Condition) bool {
	existingCondition := meta.FindStatusCondition(*conditions, newCondition.Type)
	if existingCondition != nil && existingCondition.ObservedGeneration > newCondition.ObservedGeneration {
		return false
	}
	return true
}

// VerifyConditions verify conditions to decide whether to update status
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
