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

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/apisix-ingress-controller/pkg/kube"
	configv2alpha1 "github.com/apache/apisix-ingress-controller/pkg/kube/apisix/apis/config/v2alpha1"
)

const (
	_conditionType        = "ResourcesAvailable"
	_commonSuccessMessage = "Sync Successfully"
)

// recordRouteStatus record ApisixRoute v2alpha1 status
func recordRouteStatus(ar *configv2alpha1.ApisixRoute, reason, message string, status v1.ConditionStatus) {
	// build condition
	condition := metav1.Condition{
		Type:    _conditionType,
		Reason:  reason,
		Status:  status,
		Message: message,
	}

	// set to status
	if ar.Status.Conditions == nil {
		conditions := make([]metav1.Condition, 0)
		ar.Status.Conditions = &conditions
	}
	meta.SetStatusCondition(ar.Status.Conditions, condition)
	_, _ = kube.GetApisixClient().ApisixV2alpha1().ApisixRoutes(ar.Namespace).
		UpdateStatus(context.TODO(), ar, metav1.UpdateOptions{})
}
