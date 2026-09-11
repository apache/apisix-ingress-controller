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
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
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

// updateStatusFromSyncResults updates every resource and GatewayProxy status this
// round's sync results call for. results holds one entry per config sync() actually
// reached pushConfig for this round, success (a zero-value types.ADCExecutionErrors) or
// failure; a config sync() could not even build a SyncInput for is absent here entirely
// and its status is left untouched this round, see sync().
//
// GatewayProxy's DataPlaneAvailable condition is recomputed and written fresh every
// round directly from this round's result, never compared against any remembered
// history: a config with no error this round is written True, one with any error is
// written False. That is what makes a GatewayProxy that has always been healthy actually
// get a True the first time, and what keeps a restart from leaving a stale False stuck
// forever: the write only ever depends on this round's actual outcome.
//
// Resource status can't afford the same full recompute: a config's resource set can be
// large, and rewriting every one of them every round even when nothing changed would be
// wasteful. So resources keep a small persisted delta in d.resourceFailures instead:
// newly (or still) failing resources are written SyncFailed, and any resource that was
// failing last round but isn't failing this one gets its error explicitly cleared with
// an Accepted write.
func (d *apisixProvider) updateStatusFromSyncResults(ctx context.Context, results map[string]types.ADCExecutionErrors) {
	resourceFailures := map[types.NamespacedNameKind][]string{}

	for configName, execErrs := range results {
		var gatewayProxy types.NamespacedNameKind
		if err := gatewayProxy.FromString(configName); err != nil {
			d.log.Error(err, "failed to parse config name as a GatewayProxy key", "configName", configName)
			continue
		}

		gatewayProxyMsgs, failedEndpoints := d.classifySyncResult(configName, execErrs, resourceFailures)
		if len(gatewayProxyMsgs) > 0 {
			d.updateStatus(gatewayProxy, failureCondition(gatewayProxy, strings.Join(gatewayProxyMsgs, "; ")))
			d.recordFailedEndpointEvents(ctx, gatewayProxy, failedEndpoints)
		} else {
			d.updateStatus(gatewayProxy, successCondition(gatewayProxy))
		}
	}

	d.applyResourceFailures(resourceFailures)
	d.log.V(1).Info("updated status from sync results", "results", results, "resource_failures", resourceFailures)
}

// classifySyncResult splits one config's this-round result into what belongs on the
// GatewayProxy (returned) and what belongs on specific Kubernetes resources (added into
// resourceFailures). A FailedStatuses entry that resolves to a resource via its Event
// goes there; everything else, no FailedStatuses at all, or a FailedStatuses entry
// with no Event to resolve, which is what apisix-standalone's own endpoint-driven
// rejections look like, is a GatewayProxy-level signal instead, using
// EndpointStatuses for the message when there is one and the raw error otherwise.
func (d *apisixProvider) classifySyncResult(
	configName string,
	execErrs types.ADCExecutionErrors,
	resourceFailures map[types.NamespacedNameKind][]string,
) (gatewayProxyMsgs []string, failedEndpoints []adctypes.EndpointStatus) {
	unattributed := func(addrErr types.ADCExecutionServerAddrError) {
		msg := unavailableEndpointsMessage(addrErr.EndpointStatuses)
		if msg == "" {
			msg = addrErr.Error()
		}
		gatewayProxyMsgs = append(gatewayProxyMsgs, msg)
		failedEndpoints = append(failedEndpoints, addrErr.EndpointStatuses...)
	}

	for _, execErr := range execErrs.Errors {
		for _, addrErr := range execErr.FailedErrors {
			if len(addrErr.FailedStatuses) == 0 {
				unattributed(addrErr)
				continue
			}

			attributedAll := true
			for _, syncStatus := range addrErr.FailedStatuses {
				if syncStatus.Event.ResourceType == "" {
					// standalone: this whole addrErr carries no per-resource attribution.
					attributedAll = false
					break
				}
				labels, err := d.store.GetResourceLabel(configName, syncStatus.Event.ResourceType, syncStatus.Event.ResourceID)
				if err != nil {
					d.log.Error(err, "failed to get resource label",
						"configName", configName, "resourceType", syncStatus.Event.ResourceType, "id", syncStatus.Event.ResourceID)
					continue
				}
				resourceKey := types.NamespacedNameKind{
					Name:      labels[label.LabelName],
					Namespace: labels[label.LabelNamespace],
					Kind:      labels[label.LabelKind],
				}
				msg := fmt.Sprintf("ServerAddr: %s, Error: %s", addrErr.ServerAddr, syncStatus.Reason)
				resourceFailures[resourceKey] = append(resourceFailures[resourceKey], msg)
			}
			if !attributedAll {
				unattributed(addrErr)
			}
		}
	}
	return gatewayProxyMsgs, failedEndpoints
}

// applyResourceFailures writes this round's newly (or still) failing resources, and
// clears the recorded error from any resource that was failing last round but isn't in
// newFailures now. See updateStatusFromSyncResults for why resources use this delta
// instead of GatewayProxy's full recompute.
func (d *apisixProvider) applyResourceFailures(newFailures map[types.NamespacedNameKind][]string) {
	for resourceKey, msgs := range newFailures {
		d.updateStatus(resourceKey, failureCondition(resourceKey, strings.Join(msgs, "; ")))
	}
	for resourceKey := range d.resourceFailures {
		if _, stillFailing := newFailures[resourceKey]; !stillFailing {
			d.updateStatus(resourceKey, successCondition(resourceKey))
		}
	}
	d.resourceFailures = newFailures
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

// recordFailedEndpointEvents fires one Warning event per failed EndpointStatus entry,
// so each instance's own failure history (when it started, how often) is visible on
// its own, not folded into everyone else's. The GatewayProxy is fetched fresh from the
// API server first so the Event's involvedObject carries a real UID: a hand-built stub
// with only Name/Namespace leaves that UID empty, and kubectl describe resolves events
// by matching it, so an event against such a stub never shows up there.
func (d *apisixProvider) recordFailedEndpointEvents(ctx context.Context, nnk types.NamespacedNameKind, endpoints []adctypes.EndpointStatus) {
	if d.EventRecorder == nil {
		return
	}
	hasFailure := false
	for _, ep := range endpoints {
		if !ep.Success {
			hasFailure = true
			break
		}
	}
	if !hasFailure {
		return
	}

	gatewayProxy := &apiv1alpha1.GatewayProxy{}
	if err := d.K8sClient.Get(ctx, nnk.NamespacedName(), gatewayProxy); err != nil {
		d.log.Error(err, "failed to get GatewayProxy to record failed endpoint events", "name", nnk.Name, "namespace", nnk.Namespace)
		return
	}

	for _, ep := range endpoints {
		if ep.Success {
			continue
		}
		d.EventRecorder.Event(gatewayProxy, corev1.EventTypeWarning, GatewayProxyReasonDataPlaneInstanceUnavailable,
			fmt.Sprintf("%s: %s", ep.Server, ep.Reason))
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
		parentRefs := d.configManager.GetConfigRefsByResourceKey(nnk)
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
		parentRefs := d.configManager.GetConfigRefsByResourceKey(nnk)
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
		parentRefs := d.configManager.GetConfigRefsByResourceKey(nnk)
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
		parentRefs := d.configManager.GetConfigRefsByResourceKey(nnk)
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

// unavailableEndpointsMessage summarizes every EndpointStatus entry that didn't
// succeed, in the order given. Empty means none did (or there were none to check).
func unavailableEndpointsMessage(endpoints []adctypes.EndpointStatus) string {
	failed := make([]string, 0, len(endpoints))
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
