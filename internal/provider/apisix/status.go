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

package apisix

import (
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	apiv1alpha1 "github.com/apache/apisix-ingress-controller/api/v1alpha1"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	cutils "github.com/apache/apisix-ingress-controller/internal/controller/utils"
	"github.com/apache/apisix-ingress-controller/internal/types"
)

// GatewayProxyConditionDataPlaneAvailable reports whether every APISIX instance a
// GatewayProxy addresses took the last sync. It lands on the GatewayProxy rather than
// a Gateway, an IngressClass, or a CRD, since GatewayProxy is what every path shares.
const (
	GatewayProxyConditionDataPlaneAvailable = "DataPlaneAvailable"

	GatewayProxyReasonDataPlaneAvailable           = "DataPlaneAvailable"
	GatewayProxyReasonDataPlaneInstanceUnavailable = "DataPlaneInstanceUnavailable"
)

// handleStatusUpdate updates resource conditions based on the latest sync results.
//
// It maintains a history of failed resources in d.statusUpdateMap.
//
// For resources in the current failure map (statusUpdateMap), it marks them as failed.
// For resources that exist only in the previous failure history (i.e. not in this sync's failures),
// it marks them as accepted (success).
func (d *apisixProvider) handleStatusUpdate(statusUpdateMap map[types.NamespacedNameKind][]string) {
	// Mark all resources in the current failure set as failed.
	for nnk, msgs := range statusUpdateMap {
		d.updateStatus(nnk, failureCondition(nnk, strings.Join(msgs, "; ")))
	}

	// Mark resources that exist only in the previous failure history as successful.
	for nnk := range d.statusUpdateMap {
		if _, ok := statusUpdateMap[nnk]; !ok {
			d.updateStatus(nnk, successCondition(nnk))
		}
	}
	// Update the failure history with the current failure set.
	d.statusUpdateMap = statusUpdateMap
}

// failureCondition and successCondition pick which condition a NamespacedNameKind
// gets: GatewayProxyConditionDataPlaneAvailable for a GatewayProxy, the existing
// Accepted/SyncFailed condition for everything else.
func failureCondition(nnk types.NamespacedNameKind, msg string) metav1.Condition {
	if nnk.Kind == types.KindGatewayProxy {
		return newGatewayProxyDataPlaneAvailableCondition(false, GatewayProxyReasonDataPlaneInstanceUnavailable, msg)
	}
	return cutils.NewConditionTypeAccepted(apiv2.ConditionReasonSyncFailed, false, 0, msg)
}

func successCondition(nnk types.NamespacedNameKind) metav1.Condition {
	if nnk.Kind == types.KindGatewayProxy {
		return newGatewayProxyDataPlaneAvailableCondition(true, GatewayProxyReasonDataPlaneAvailable, "")
	}
	return cutils.NewConditionTypeAccepted(apiv2.ConditionReasonAccepted, true, 0, "")
}

func newGatewayProxyDataPlaneAvailableCondition(available bool, reason, msg string) metav1.Condition {
	conditionStatus := metav1.ConditionFalse
	if available {
		conditionStatus = metav1.ConditionTrue
	}
	return metav1.Condition{
		Type:               GatewayProxyConditionDataPlaneAvailable,
		Status:             conditionStatus,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            cutils.TruncateConditionMessage(msg),
	}
}

//nolint:gocyclo
func (d *apisixProvider) updateStatus(nnk types.NamespacedNameKind, condition metav1.Condition) {
	switch nnk.Kind {
	case types.KindGatewayProxy:
		// Unlike the route kinds below, the condition lands on the GatewayProxy's own
		// top-level Status.Conditions, not on a per-parent entry.
		d.updater.Update(status.Update{
			NamespacedName: nnk.NamespacedName(),
			Resource:       &apiv1alpha1.GatewayProxy{},
			Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
				cp := obj.(*apiv1alpha1.GatewayProxy).DeepCopy()
				condition.ObservedGeneration = cp.GetGeneration()
				cp.Status.Conditions = cutils.MergeCondition(cp.Status.Conditions, condition)
				return cp
			}),
		})
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
		parentRefs := d.client.ConfigManager.GetConfigRefsByResourceKey(nnk)
		d.log.V(1).Info("updating HTTPRoute status", "parentRefs", parentRefs)
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
				condition.ObservedGeneration = cp.GetGeneration()
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
	case types.KindUDPRoute:
		parentRefs := d.client.ConfigManager.GetConfigRefsByResourceKey(nnk)
		d.log.V(1).Info("updating UDPRoute status", "parentRefs", parentRefs)
		gatewayRefs := map[types.NamespacedNameKind]struct{}{}
		for _, parentRef := range parentRefs {
			if parentRef.Kind == types.KindGateway {
				gatewayRefs[parentRef] = struct{}{}
			}
		}
		d.updater.Update(status.Update{
			NamespacedName: nnk.NamespacedName(),
			Resource:       &gatewayv1.UDPRoute{},
			Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
				cp := obj.(*gatewayv1.UDPRoute).DeepCopy()
				condition.ObservedGeneration = cp.GetGeneration()
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
	case types.KindTCPRoute:
		parentRefs := d.client.ConfigManager.GetConfigRefsByResourceKey(nnk)
		d.log.V(1).Info("updating TCPRoute status", "parentRefs", parentRefs)
		gatewayRefs := map[types.NamespacedNameKind]struct{}{}
		for _, parentRef := range parentRefs {
			if parentRef.Kind == types.KindGateway {
				gatewayRefs[parentRef] = struct{}{}
			}
		}
		d.updater.Update(status.Update{
			NamespacedName: nnk.NamespacedName(),
			Resource:       &gatewayv1.TCPRoute{},
			Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
				cp := obj.(*gatewayv1.TCPRoute).DeepCopy()
				condition.ObservedGeneration = cp.GetGeneration()
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
	case types.KindGRPCRoute:
		parentRefs := d.client.ConfigManager.GetConfigRefsByResourceKey(nnk)
		d.log.V(1).Info("updating GRPCRoute status", "parentRefs", parentRefs)
		gatewayRefs := map[types.NamespacedNameKind]struct{}{}
		for _, parentRef := range parentRefs {
			if parentRef.Kind == types.KindGateway {
				gatewayRefs[parentRef] = struct{}{}
			}
		}
		d.updater.Update(status.Update{
			NamespacedName: nnk.NamespacedName(),
			Resource:       &gatewayv1.GRPCRoute{},
			Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
				cp := obj.(*gatewayv1.GRPCRoute).DeepCopy()
				condition.ObservedGeneration = cp.GetGeneration()
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

func (d *apisixProvider) resolveADCExecutionErrors(
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

// handleEmptyFailedStatuses runs when nothing could be attributed to a specific
// resource. A failed EndpointStatus entry means an unreachable or rejecting instance,
// so it's marked on the GatewayProxy instead of smearing every resource; only a plain
// transport error with no structured response at all falls back to smearing.
func (d *apisixProvider) handleEmptyFailedStatuses(
	configName string,
	failedStatus types.ADCExecutionServerAddrError,
	statusUpdateMap map[types.NamespacedNameKind][]string,
) {
	if msg := unavailableEndpointsMessage(failedStatus.EndpointStatuses); msg != "" {
		d.markGatewayProxyDataPlaneUnavailable(configName, msg, statusUpdateMap)
		return
	}
	d.smearAllResources(configName, failedStatus.Error(), statusUpdateMap)
}

// unavailableEndpointsMessage summarizes every EndpointStatus entry that didn't
// succeed, in the order given. Empty means none did (or there were none to check).
func unavailableEndpointsMessage(endpoints []adctypes.EndpointStatus) string {
	var failed []string
	for _, ep := range endpoints {
		if ep.Success {
			continue
		}
		failed = append(failed, fmt.Sprintf("%s: %s", ep.Server, ep.Reason))
	}
	if len(failed) == 0 {
		return ""
	}
	return fmt.Sprintf("%d/%d gateway instance(s) failed to apply the last sync: %s",
		len(failed), len(endpoints), strings.Join(failed, "; "))
}

// markGatewayProxyDataPlaneUnavailable marks the GatewayProxy configName names.
func (d *apisixProvider) markGatewayProxyDataPlaneUnavailable(
	configName string,
	msg string,
	statusUpdateMap map[types.NamespacedNameKind][]string,
) {
	var gatewayProxy types.NamespacedNameKind
	if err := gatewayProxy.FromString(configName); err != nil {
		d.log.Error(err, "failed to parse config name as a GatewayProxy key", "configName", configName)
		return
	}
	statusUpdateMap[gatewayProxy] = append(statusUpdateMap[gatewayProxy], msg)
}

// smearAllResources is the last resort when nothing can be attributed: it marks every
// resource under this config.
func (d *apisixProvider) smearAllResources(
	configName string,
	msg string,
	statusUpdateMap map[types.NamespacedNameKind][]string,
) {
	resource, err := d.client.GetResources(configName)
	if err != nil {
		d.log.Error(err, "failed to get resources from store", "configName", configName)
		return
	}

	for _, obj := range resource.Services {
		d.addResourceToStatusUpdateMap(obj.GetLabels(), msg, statusUpdateMap)
	}

	for _, obj := range resource.Consumers {
		d.addResourceToStatusUpdateMap(obj.GetLabels(), msg, statusUpdateMap)
	}

	for _, obj := range resource.SSLs {
		d.addResourceToStatusUpdateMap(obj.GetLabels(), msg, statusUpdateMap)
	}

	globalRules, err := d.client.ListGlobalRules(configName)
	if err != nil {
		d.log.Error(err, "failed to list global rules", "configName", configName)
		return
	}
	for _, rule := range globalRules {
		d.addResourceToStatusUpdateMap(rule.GetLabels(), msg, statusUpdateMap)
	}
}

func (d *apisixProvider) handleDetailedFailedStatuses(
	configName string,
	failedStatus types.ADCExecutionServerAddrError,
	statusUpdateMap map[types.NamespacedNameKind][]string,
) {
	for _, status := range failedStatus.FailedStatuses {
		// in the APISIX standalone mode, the related values in the sync failure event are empty.
		if status.Event.ResourceType == "" {
			d.handleEmptyFailedStatuses(configName, failedStatus, statusUpdateMap)
			return
		}
		id := status.Event.ResourceID
		labels, err := d.client.GetResourceLabel(configName, status.Event.ResourceType, id)
		if err != nil {
			d.log.Error(err, "failed to get resource label",
				"configName", configName,
				"resourceType", status.Event.ResourceType,
				"id", id,
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

func (d *apisixProvider) addResourceToStatusUpdateMap(
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
