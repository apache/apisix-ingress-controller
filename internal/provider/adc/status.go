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

package adc

import (
	"fmt"
	"strings"

	"github.com/api7/gopkg/pkg/log"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	cutils "github.com/apache/apisix-ingress-controller/internal/controller/utils"
	"github.com/apache/apisix-ingress-controller/internal/types"
)

// handleStatusUpdate updates resource conditions based on the latest sync results.
//
// It maintains a history of failed resources in d.statusUpdateMap.
//
// For resources in the current failure map (statusUpdateMap), it marks them as failed.
// For resources that exist only in the previous failure history (i.e. not in this sync's failures),
// it marks them as accepted (success).
func (d *adcClient) handleStatusUpdate(statusUpdateMap map[types.NamespacedNameKind][]string) {
	// Mark all resources in the current failure set as failed.
	for nnk, msgs := range statusUpdateMap {
		d.updateStatus(nnk, cutils.NewConditionTypeAccepted(
			apiv2.ConditionReasonSyncFailed,
			false,
			0,
			strings.Join(msgs, "; "),
		))
	}

	// Mark resources that exist only in the previous failure history as successful.
	for nnk := range d.statusUpdateMap {
		if _, ok := statusUpdateMap[nnk]; !ok {
			d.updateStatus(nnk, cutils.NewConditionTypeAccepted(
				apiv2.ConditionReasonAccepted,
				true,
				0,
				"",
			))
		}
	}
	// Update the failure history with the current failure set.
	d.statusUpdateMap = statusUpdateMap
}

func (d *adcClient) updateStatus(nnk types.NamespacedNameKind, condition metav1.Condition) {
	switch nnk.Kind {
	case types.KindApisixRoute:
		d.updater.Update(status.Update{
			NamespacedName: nnk.NamespacedName(),
			Resource:       &apiv2.ApisixRoute{},
			Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
				cp := obj.(*apiv2.ApisixRoute).DeepCopy()
				cutils.SetApisixCRDConditionWithGeneration(&cp.Status, cp.GetGeneration(), condition)
				return cp
			}),
		})
	case types.KindApisixGlobalRule:
		d.updater.Update(status.Update{
			NamespacedName: nnk.NamespacedName(),
			Resource:       &apiv2.ApisixGlobalRule{},
			Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
				cp := obj.(*apiv2.ApisixGlobalRule).DeepCopy()
				cutils.SetApisixCRDConditionWithGeneration(&cp.Status, cp.GetGeneration(), condition)
				return cp
			}),
		})
	case types.KindApisixTls:
		d.updater.Update(status.Update{
			NamespacedName: nnk.NamespacedName(),
			Resource:       &apiv2.ApisixTls{},
			Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
				cp := obj.(*apiv2.ApisixTls).DeepCopy()
				cutils.SetApisixCRDConditionWithGeneration(&cp.Status, cp.GetGeneration(), condition)
				return cp
			}),
		})
	case types.KindApisixConsumer:
		d.updater.Update(status.Update{
			NamespacedName: nnk.NamespacedName(),
			Resource:       &apiv2.ApisixConsumer{},
			Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
				cp := obj.(*apiv2.ApisixConsumer).DeepCopy()
				cutils.SetApisixCRDConditionWithGeneration(&cp.Status, cp.GetGeneration(), condition)
				return cp
			}),
		})
	case types.KindHTTPRoute:
		parentRefs := d.getParentRefs(nnk)
		gatewayRefs := map[types.NamespacedNameKind]struct{}{}
		for _, parentRef := range parentRefs {
			if parentRef.Kind == types.KindGateway {
				gatewayRefs[parentRef] = struct{}{}
			}
		}
		d.updater.Update(status.Update{
			NamespacedName: nnk.NamespacedName(),
			Resource:       &gatewayv1.HTTPRoute{},
			Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
				cp := obj.(*gatewayv1.HTTPRoute).DeepCopy()
				gatewayNs := cp.GetNamespace()
				for i, ref := range cp.Status.Parents {
					ns := gatewayNs
					if ref.ParentRef.Namespace != nil {
						ns = string(*ref.ParentRef.Namespace)
					}
					if ref.ParentRef.Kind == nil || *ref.ParentRef.Kind == types.KindGateway {
						nnk := types.NamespacedNameKind{
							Name:      string(ref.ParentRef.Name),
							Namespace: ns,
							Kind:      types.KindGateway,
						}
						if _, ok := gatewayRefs[nnk]; ok {
							ref.Conditions = cutils.MergeCondition(ref.Conditions, condition)
							cp.Status.Parents[i] = ref
						}
					}
				}
				return cp
			}),
		})
	}
}

func (d *adcClient) resolveADCExecutionErrors(
	statusesMap map[string]types.ADCExecutionErrors,
) map[types.NamespacedNameKind][]string {
	statusUpdateMap := map[types.NamespacedNameKind][]string{}
	for configName, execErrors := range statusesMap {
		for _, execErr := range execErrors.Errors {
			for _, failedStatus := range execErr.FailedErrors {
				if len(failedStatus.FailedStatuses) == 0 {
					d.handleEmptyFailedStatuses(configName, failedStatus, statusUpdateMap)
				} else {
					d.handleDetailedFailedStatuses(configName, failedStatus, statusUpdateMap)
				}
			}
		}
	}

	return statusUpdateMap
}

func (d *adcClient) handleEmptyFailedStatuses(
	configName string,
	failedStatus types.ADCExecutionServerAddrError,
	statusUpdateMap map[types.NamespacedNameKind][]string,
) {
	resource, err := d.store.GetResources(configName)
	if err != nil {
		log.Errorw("failed to get resources from store", zap.String("configName", configName), zap.Error(err))
		return
	}

	for _, obj := range resource.Services {
		d.addResourceToStatusUpdateMap(obj.GetLabels(), failedStatus.Error(), statusUpdateMap)
	}

	for _, obj := range resource.Consumers {
		d.addResourceToStatusUpdateMap(obj.GetLabels(), failedStatus.Error(), statusUpdateMap)
	}

	for _, obj := range resource.SSLs {
		d.addResourceToStatusUpdateMap(obj.GetLabels(), failedStatus.Error(), statusUpdateMap)
	}

	globalRules, err := d.store.ListGlobalRules(configName)
	if err != nil {
		log.Errorw("failed to list global rules", zap.String("configName", configName), zap.Error(err))
		return
	}
	for _, rule := range globalRules {
		d.addResourceToStatusUpdateMap(rule.GetLabels(), failedStatus.Error(), statusUpdateMap)
	}
}

func (d *adcClient) handleDetailedFailedStatuses(
	configName string,
	failedStatus types.ADCExecutionServerAddrError,
	statusUpdateMap map[types.NamespacedNameKind][]string,
) {
	for _, status := range failedStatus.FailedStatuses {
		id := status.Event.ResourceID
		labels, err := d.store.GetResourceLabel(configName, status.Event.ResourceType, id)
		if err != nil {
			log.Errorw("failed to get resource label",
				zap.String("configName", configName),
				zap.String("resourceType", status.Event.ResourceType),
				zap.String("id", id),
				zap.Error(err),
			)
			continue
		}
		d.addResourceToStatusUpdateMap(
			labels,
			fmt.Sprintf("ServerAddr: %s, Error: %s", failedStatus.ServerAddr, status.Reason),
			statusUpdateMap,
		)
	}
}

func (d *adcClient) addResourceToStatusUpdateMap(
	labels map[string]string,
	msg string,
	statusUpdateMap map[types.NamespacedNameKind][]string,
) {
	statusKey := types.NamespacedNameKind{
		Name:      labels[label.LabelName],
		Namespace: labels[label.LabelNamespace],
		Kind:      labels[label.LabelKind],
	}
	statusUpdateMap[statusKey] = append(statusUpdateMap[statusKey], msg)
}
