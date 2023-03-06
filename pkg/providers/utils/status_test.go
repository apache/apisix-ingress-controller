// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestVerifyConditions(t *testing.T) {
	// Different status
	conditions := []metav1.Condition{
		{
			Type:               ConditionType,
			Status:             metav1.ConditionTrue,
			ObservedGeneration: 1,
		},
	}
	newCondition := metav1.Condition{
		Type:               ConditionType,
		Status:             metav1.ConditionFalse,
		ObservedGeneration: 1,
	}
	assert.Equal(t, true, VerifyConditions(&conditions, newCondition))

	// same condition
	conditions = []metav1.Condition{
		{
			Type:               ConditionType,
			Status:             metav1.ConditionTrue,
			ObservedGeneration: 1,
		},
	}
	newCondition = metav1.Condition{
		Type:               ConditionType,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: 1,
	}
	assert.Equal(t, false, VerifyConditions(&conditions, newCondition))

	// Different ObservedGeneration
	conditions = []metav1.Condition{
		{
			Type:               ConditionType,
			Status:             metav1.ConditionTrue,
			ObservedGeneration: 1,
		},
	}
	newCondition = metav1.Condition{
		Type:               ConditionType,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: 2,
	}
	assert.Equal(t, true, VerifyConditions(&conditions, newCondition))

	conditions = []metav1.Condition{
		{
			Type:               ConditionType,
			Status:             metav1.ConditionTrue,
			ObservedGeneration: 2,
		},
	}
	newCondition = metav1.Condition{
		Type:               ConditionType,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: 1,
	}
	assert.Equal(t, false, VerifyConditions(&conditions, newCondition))

	// Different message
	conditions = []metav1.Condition{
		{
			Type:               ConditionType,
			Status:             metav1.ConditionFalse,
			Message:            "port does not exist",
			ObservedGeneration: 1,
		},
	}
	newCondition = metav1.Condition{
		Type:               ConditionType,
		Status:             metav1.ConditionFalse,
		Message:            "service does not exist",
		ObservedGeneration: 1,
	}
	assert.Equal(t, true, VerifyConditions(&conditions, newCondition))
}
