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

package apisix

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	apiv1alpha1 "github.com/apache/apisix-ingress-controller/api/v1alpha1"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/adc/cache"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	cutils "github.com/apache/apisix-ingress-controller/internal/controller/utils"
	"github.com/apache/apisix-ingress-controller/internal/types"
)

const (
	// GatewayProxyConditionDataPlaneAvailable reports whether every APISIX instance a
	// GatewayProxy addresses took the last sync. It lands on the GatewayProxy rather
	// than a Gateway, an IngressClass, or a CRD, since GatewayProxy is what every path
	// shares.
	GatewayProxyConditionDataPlaneAvailable = "DataPlaneAvailable"

	GatewayProxyReasonDataPlaneAvailable           = "DataPlaneAvailable"
	GatewayProxyReasonDataPlaneInstanceUnavailable = "DataPlaneInstanceUnavailable"

	// GatewayProxyConditionPluginsProgrammed reports whether every plugin and plugin
	// metadata a GatewayProxy declares reached the data plane.
	GatewayProxyConditionPluginsProgrammed = "PluginsProgrammed"

	GatewayProxyReasonPluginsProgrammed = "Programmed"
	GatewayProxyReasonPluginsInvalid    = "Invalid"

	// conditionTypePartiallyInvalid is set on a resource that is served with some of its
	// parts dropped, following gatewayv1.RouteConditionPartiallyInvalid.
	conditionTypePartiallyInvalid = string(gatewayv1.RouteConditionPartiallyInvalid)
)

// resourceDrop is what a sync round dropped from one Kubernetes resource.
type resourceDrop struct {
	// partial means some of the resource is still served.
	partial bool
	msgs    []string
}

// updateStatusFromSyncResults records this round's rejected resources in the skip table
// and updates every status that follows from it. results holds one entry per config
// sync() reached pushConfig for, and revisions the store revision each was built at; a
// config sync() could not build is absent from both and its GatewayProxy status is left
// untouched this round.
//
// GatewayProxy conditions are recomputed from this round's result alone. Resource status
// follows the whole skip table instead, since an excluded resource stays dropped until
// its owner is written again, and resources keep a delta in d.resourceDrops so a resource
// no longer dropped gets its status cleared.
//
// It reports whether anything was newly excluded, which is what makes the next push
// different from the one that just failed: see sync().
func (d *apisixProvider) updateStatusFromSyncResults(ctx context.Context, results map[string]types.ADCExecutionErrors, revisions map[string]uint64) bool {
	newlyExcluded := 0
	for configName, execErrs := range results {
		fresh := map[wireKey]exclusion{}
		gatewayProxyMsgs, failedEndpoints := d.classifySyncResult(configName, execErrs, revisions[configName], fresh)
		newlyExcluded += d.skipped.MarkFailing(configName, fresh)

		var gatewayProxy types.NamespacedNameKind
		if err := gatewayProxy.FromString(configName); err != nil {
			d.log.Error(err, "failed to parse config name as a GatewayProxy key", "configName", configName)
			continue
		}
		if len(gatewayProxyMsgs) > 0 {
			d.updateStatus(gatewayProxy, setConditions(newGatewayProxyCondition(GatewayProxyConditionDataPlaneAvailable, false, GatewayProxyReasonDataPlaneInstanceUnavailable, strings.Join(gatewayProxyMsgs, "; "))))
			d.recordFailedEndpointEvents(ctx, gatewayProxy, failedEndpoints)
		} else {
			d.updateStatus(gatewayProxy, setConditions(newGatewayProxyCondition(GatewayProxyConditionDataPlaneAvailable, true, GatewayProxyReasonDataPlaneAvailable, "")))
		}
	}

	resourceDrops := map[types.NamespacedNameKind]resourceDrop{}
	rejectedCertificates := map[k8stypes.NamespacedName]map[string]string{}
	for configName, entries := range d.skipped.Snapshot() {
		for owner, keys := range groupByOwner(entries) {
			switch owner.Kind {
			case types.KindGatewayProxy:
				// Written below, only for configs this round reached.
			case types.KindGateway:
				certificates := rejectedCertificates[owner.NamespacedName()]
				if certificates == nil {
					certificates = map[string]string{}
					rejectedCertificates[owner.NamespacedName()] = certificates
				}
				for _, key := range keys {
					certificates[key.id] = entries[key].reason
				}
			default:
				drop := d.resourceDropIn(configName, owner, keys, entries)
				if previous, ok := resourceDrops[owner]; ok {
					drop.partial = drop.partial || previous.partial
					drop.msgs = append(previous.msgs, drop.msgs...)
				}
				resourceDrops[owner] = drop
			}
		}
	}

	for configName := range results {
		var gatewayProxy types.NamespacedNameKind
		if err := gatewayProxy.FromString(configName); err != nil {
			continue
		}
		var msgs []string
		for _, ex := range d.skipped.Excluded(configName) {
			if ex.owner == gatewayProxy {
				msgs = append(msgs, fmt.Sprintf("%s: %s", ex.name, ex.reason))
			}
		}
		slices.Sort(msgs)
		switch {
		case len(msgs) > 0:
			d.updateStatus(gatewayProxy, setConditions(newGatewayProxyCondition(GatewayProxyConditionPluginsProgrammed, false, GatewayProxyReasonPluginsInvalid, strings.Join(msgs, "; "))))
		case len(d.store.OwnedEntities(configName, gatewayProxy)) > 0:
			d.updateStatus(gatewayProxy, setConditions(newGatewayProxyCondition(GatewayProxyConditionPluginsProgrammed, true, GatewayProxyReasonPluginsProgrammed, "")))
		default:
			// A GatewayProxy that declares no plugins has nothing for this condition to report on.
			d.updateStatus(gatewayProxy, conditionChange{remove: []string{GatewayProxyConditionPluginsProgrammed}})
		}
	}

	d.applyResourceDrops(ctx, resourceDrops)
	d.setRejectedGatewayCertificates(rejectedCertificates)
	d.log.V(1).Info("updated status from sync results", "results", results, "resource_drops", resourceDrops)
	return newlyExcluded > 0
}

func groupByOwner(entries map[wireKey]exclusion) map[types.NamespacedNameKind][]wireKey {
	byOwner := map[types.NamespacedNameKind][]wireKey{}
	for key, ex := range entries {
		byOwner[ex.owner] = append(byOwner[ex.owner], key)
	}
	return byOwner
}

// resourceDropIn describes what the given skip table entries drop from owner in the
// cacheKey configName. All of it is dropped when every top-level entity owner produced
// there is gone: excluded itself, or, for a service, left with none of its routes.
func (d *apisixProvider) resourceDropIn(configName string, owner types.NamespacedNameKind, keys []wireKey, entries map[wireKey]exclusion) resourceDrop {
	dropped := make(map[wireKey]struct{}, len(keys))
	reasons := make([]string, 0, len(keys))
	located := make([]string, 0, len(keys))
	for _, key := range keys {
		dropped[key] = struct{}{}
		reasons = append(reasons, entries[key].reason)
		located = append(located, fmt.Sprintf("%s: %s", entries[key].name, entries[key].reason))
	}
	slices.Sort(reasons)
	slices.Sort(located)

	isDropped := func(entity cache.Entity) bool {
		if _, ok := dropped[wireKey{resourceType: entity.Type, id: entity.ID}]; ok {
			return true
		}
		if len(entity.Children) == 0 {
			return false
		}
		for _, child := range entity.Children {
			if _, ok := dropped[wireKey{resourceType: child.Type, parentID: entity.ID, id: child.ID}]; !ok {
				return false
			}
		}
		return true
	}

	owned := d.store.OwnedEntities(configName, owner)
	partial := len(owned) == 0
	for _, entity := range owned {
		if !isDropped(entity) {
			partial = true
			break
		}
	}
	// A resource with nothing left reports the data plane's reasons on their own, the
	// way it did before any of it could be dropped piecemeal.
	if !partial {
		return resourceDrop{msgs: reasons}
	}
	return resourceDrop{partial: true, msgs: located}
}

// classifySyncResult splits one config's result into what belongs on the GatewayProxy
// (returned) and the wire resources to drop (added into dropped). The two are
// independent: an addrErr can carry both a resource-attributed FailedStatuses entry and
// a failed EndpointStatuses entry, so reporting one never suppresses the other. A
// FailedStatuses entry this function can't attribute to a resource is a GatewayProxy
// signal instead, unless EndpointStatuses already explained the addrErr.
//
// A failure whose owner's content changed after builtRevision is ignored: it was reported
// against content that has since been replaced, and recording it would keep the
// replacement excluded with nothing left to clear it.
func (d *apisixProvider) classifySyncResult(
	configName string,
	execErrs types.ADCExecutionErrors,
	builtRevision uint64,
	dropped map[wireKey]exclusion,
) (gatewayProxyMsgs []string, failedEndpoints []adctypes.EndpointStatus) {
	for _, execErr := range execErrs.Errors {
		for _, addrErr := range execErr.FailedErrors {
			endpointMsg := unavailableEndpointsMessage(addrErr.EndpointStatuses)
			if endpointMsg != "" {
				gatewayProxyMsgs = append(gatewayProxyMsgs, endpointMsg)
				failedEndpoints = append(failedEndpoints, addrErr.EndpointStatuses...)
			}

			if len(addrErr.FailedStatuses) == 0 {
				if endpointMsg == "" {
					gatewayProxyMsgs = append(gatewayProxyMsgs, addrErr.Error())
				}
				continue
			}

			anyUnattributed := false
			for _, syncStatus := range addrErr.FailedStatuses {
				key, ex, ok := d.dropUnit(configName, syncStatus.Event)
				if !ok {
					d.log.Error(nil, "failed to attribute a rejected resource",
						"configName", configName, "event", syncStatus.Event)
					anyUnattributed = true
					continue
				}
				if d.store.ChangedSince(configName, ex.owner, builtRevision) {
					continue
				}
				reason := fmt.Sprintf("ServerAddr: %s, Error: %s", addrErr.ServerAddr, syncStatus.Reason)
				if previous, ok := dropped[key]; ok {
					reason = previous.reason + "; " + reason
				}
				ex.reason = reason
				dropped[key] = ex
			}
			if anyUnattributed && endpointMsg == "" {
				gatewayProxyMsgs = append(gatewayProxyMsgs, addrErr.Error())
			}
		}
	}
	return gatewayProxyMsgs, failedEndpoints
}

// dropUnit maps a rejected resource to what has to be dropped for the rest to apply:
//
//   - a service, ssl, consumer, global_rule or plugin_metadata is dropped by itself;
//   - a named upstream takes its whole service with it, since the service's
//     traffic-split refers to it by id and would otherwise be left dangling;
//   - a route or stream route of a Gateway API route takes its whole service, which is
//     that route's rule, so the rule is what gets reported as dropped;
//   - any other route, stream route or credential is dropped by itself.
//
// Nested resources are found through their parent, which also tells them apart when
// several services embed upstreams of the same id.
func (d *apisixProvider) dropUnit(configName string, ev adctypes.StatusEvent) (wireKey, exclusion, bool) {
	switch ev.ResourceType {
	case adctypes.TypeService, adctypes.TypeSSL, adctypes.TypeConsumer, adctypes.TypeGlobalRule, adctypes.TypePluginMetadata:
		entity, ok := d.store.Lookup(configName, ev.ResourceType, ev.ResourceID)
		if !ok {
			return wireKey{}, exclusion{}, false
		}
		return wireKey{resourceType: ev.ResourceType, id: ev.ResourceID}, exclusion{owner: entity.Owner, name: entity.Name}, true
	case adctypes.TypeUpstream, adctypes.TypeRoute, adctypes.TypeStreamRoute:
		service, ok := d.store.Lookup(configName, adctypes.TypeService, ev.ParentID)
		if !ok {
			return wireKey{}, exclusion{}, false
		}
		if ev.ResourceType == adctypes.TypeUpstream || isGatewayAPIRoute(service.Owner.Kind) {
			return wireKey{resourceType: adctypes.TypeService, id: service.ID}, exclusion{owner: service.Owner, name: service.Name}, true
		}
		return wireKey{resourceType: ev.ResourceType, parentID: ev.ParentID, id: ev.ResourceID},
			exclusion{owner: service.Owner, name: cmp.Or(ev.ResourceName, ev.ResourceID)}, true
	case adctypes.TypeConsumerCredential:
		consumer, ok := d.store.Lookup(configName, adctypes.TypeConsumer, ev.ParentID)
		if !ok {
			return wireKey{}, exclusion{}, false
		}
		return wireKey{resourceType: ev.ResourceType, parentID: ev.ParentID, id: ev.ResourceID},
			exclusion{owner: consumer.Owner, name: consumer.Name + "/" + cmp.Or(ev.ResourceName, ev.ResourceID)}, true
	}
	return wireKey{}, exclusion{}, false
}

func isGatewayAPIRoute(kind string) bool {
	switch kind {
	case types.KindHTTPRoute, types.KindGRPCRoute, types.KindTCPRoute, types.KindUDPRoute, types.KindTLSRoute:
		return true
	}
	return false
}

// applyResourceDrops writes the status of every resource in drops, and clears it for any
// resource that was dropped last round but isn't anymore.
func (d *apisixProvider) applyResourceDrops(ctx context.Context, drops map[types.NamespacedNameKind]resourceDrop) {
	for owner, drop := range drops {
		d.updateStatus(owner, droppedConditions(owner, drop))
		if owner.Kind == types.KindIngress {
			d.recordIngressDropEvent(ctx, owner, drop)
		}
	}
	for owner := range d.resourceDrops {
		if _, ok := drops[owner]; !ok {
			d.updateStatus(owner, conditionChange{
				set:    []metav1.Condition{cutils.NewConditionTypeAccepted(apiv2.ConditionReasonAccepted, true, 0, "")},
				remove: []string{conditionTypePartiallyInvalid},
			})
		}
	}
	d.resourceDrops = drops
}

// droppedConditions reports a resource served without some of its parts as Accepted with
// PartiallyInvalid, and one with nothing left as not Accepted.
func droppedConditions(owner types.NamespacedNameKind, drop resourceDrop) conditionChange {
	msg := strings.Join(drop.msgs, "; ")
	if !drop.partial {
		return conditionChange{
			set:    []metav1.Condition{cutils.NewConditionTypeAccepted(apiv2.ConditionReasonSyncFailed, false, 0, msg)},
			remove: []string{conditionTypePartiallyInvalid},
		}
	}
	prefix := "Dropped: "
	if isGatewayAPIRoute(owner.Kind) {
		prefix = "Dropped Rule: "
	}
	return conditionChange{set: []metav1.Condition{
		cutils.NewConditionTypeAccepted(apiv2.ConditionReasonAccepted, true, 0, ""),
		{
			Type:               conditionTypePartiallyInvalid,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Now(),
			Reason:             string(apiv2.ConditionReasonSyncFailed),
			Message:            cutils.TruncateConditionMessage(prefix + msg),
		},
	}}
}

// recordIngressDropEvent fires a Warning event for a dropped Ingress, whose status has no
// conditions to carry it. It is fired every round the Ingress stays dropped, since events
// expire.
func (d *apisixProvider) recordIngressDropEvent(ctx context.Context, nnk types.NamespacedNameKind, drop resourceDrop) {
	if d.EventRecorder == nil || d.K8sClient == nil {
		return
	}
	ingress := &networkingv1.Ingress{}
	if err := d.K8sClient.Get(ctx, nnk.NamespacedName(), ingress); err != nil {
		d.log.Error(err, "failed to get Ingress to record a drop event", "name", nnk.Name, "namespace", nnk.Namespace)
		return
	}
	msg := strings.Join(drop.msgs, "; ")
	if drop.partial {
		msg = "Dropped: " + msg
	}
	d.EventRecorder.Event(ingress, corev1.EventTypeWarning, string(apiv2.ConditionReasonSyncFailed), msg)
}

// setRejectedGatewayCertificates replaces which Gateway listener certificates the data
// plane rejected, and notifies the Gateway controller of every Gateway whose set changed.
func (d *apisixProvider) setRejectedGatewayCertificates(next map[k8stypes.NamespacedName]map[string]string) {
	d.rejectedCertificatesMu.Lock()
	previous := d.rejectedCertificates
	d.rejectedCertificates = next
	d.rejectedCertificatesMu.Unlock()

	changed := map[k8stypes.NamespacedName]struct{}{}
	for gateway, certificates := range next {
		if !maps.Equal(previous[gateway], certificates) {
			changed[gateway] = struct{}{}
		}
	}
	for gateway := range previous {
		if _, ok := next[gateway]; !ok {
			changed[gateway] = struct{}{}
		}
	}
	for gateway := range changed {
		ev := event.GenericEvent{Object: &gatewayv1.Gateway{ObjectMeta: metav1.ObjectMeta{Namespace: gateway.Namespace, Name: gateway.Name}}}
		select {
		case d.gatewayEvents <- ev:
		default:
			d.log.Info("dropped a Gateway status notification, the Gateway controller is not keeping up", "gateway", gateway)
		}
	}
}

// RejectedCertificates returns, per SSL id, why the data plane rejected a certificate of
// gateway's listeners.
func (d *apisixProvider) RejectedCertificates(gateway k8stypes.NamespacedName) map[string]string {
	d.rejectedCertificatesMu.Lock()
	defer d.rejectedCertificatesMu.Unlock()
	return maps.Clone(d.rejectedCertificates[gateway])
}

// GatewayEvents fires for a Gateway whenever RejectedCertificates changes for it.
func (d *apisixProvider) GatewayEvents() <-chan event.GenericEvent {
	return d.gatewayEvents
}

// conditionChange sets and removes conditions by type.
type conditionChange struct {
	set    []metav1.Condition
	remove []string
}

func setConditions(conditions ...metav1.Condition) conditionChange {
	return conditionChange{set: conditions}
}

func (c conditionChange) apply(conditions []metav1.Condition, generation int64) []metav1.Condition {
	conditions = slices.DeleteFunc(slices.Clone(conditions), func(condition metav1.Condition) bool {
		return slices.Contains(c.remove, condition.Type)
	})
	for _, condition := range c.set {
		condition.ObservedGeneration = generation
		conditions = cutils.MergeCondition(conditions, condition)
	}
	return conditions
}

func (c conditionChange) applyToApisixStatus(s *apiv2.ApisixStatus, generation int64) {
	s.Conditions = slices.DeleteFunc(s.Conditions, func(condition metav1.Condition) bool {
		return slices.Contains(c.remove, condition.Type)
	})
	for _, condition := range c.set {
		cutils.SetApisixCRDConditionWithGeneration(s, generation, condition)
	}
}

func newGatewayProxyCondition(conditionType string, ok bool, reason, msg string) metav1.Condition {
	conditionStatus := metav1.ConditionFalse
	if ok {
		conditionStatus = metav1.ConditionTrue
	}
	return metav1.Condition{
		Type:               conditionType,
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

func (d *apisixProvider) updateStatus(nnk types.NamespacedNameKind, change conditionChange) {
	switch nnk.Kind {
	case types.KindGatewayProxy:
		d.updater.Update(status.Update{
			NamespacedName: nnk.NamespacedName(),
			Resource:       &apiv1alpha1.GatewayProxy{},
			Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
				cp := obj.(*apiv1alpha1.GatewayProxy).DeepCopy()
				cp.Status.Conditions = change.apply(cp.Status.Conditions, cp.GetGeneration())
				return cp
			}),
		})
	case types.KindConsumer:
		d.updater.Update(status.Update{
			NamespacedName: nnk.NamespacedName(),
			Resource:       &apiv1alpha1.Consumer{},
			Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
				cp := obj.(*apiv1alpha1.Consumer).DeepCopy()
				cp.Status.Conditions = change.apply(cp.Status.Conditions, cp.GetGeneration())
				return cp
			}),
		})
	case types.KindApisixRoute, types.KindApisixGlobalRule, types.KindApisixTls, types.KindApisixConsumer:
		resource := newApisixObject(nnk.Kind)
		d.updater.Update(status.Update{
			NamespacedName: nnk.NamespacedName(),
			Resource:       resource,
			Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
				cp, apisixStatus := apisixStatusOf(obj)
				change.applyToApisixStatus(apisixStatus, cp.GetGeneration())
				return cp
			}),
		})
	case types.KindHTTPRoute, types.KindGRPCRoute, types.KindTCPRoute, types.KindUDPRoute, types.KindTLSRoute:
		gatewayRefs := map[types.NamespacedNameKind]struct{}{}
		for _, parentRef := range d.configManager.GetConfigRefsByResourceKey(nnk) {
			if parentRef.Kind == types.KindGateway {
				gatewayRefs[parentRef] = struct{}{}
			}
		}
		d.updater.Update(status.Update{
			NamespacedName: nnk.NamespacedName(),
			Resource:       newRouteObject(nnk.Kind),
			Mutator: status.MutatorFunc(func(obj client.Object) client.Object {
				cp, routeStatus := routeStatusOf(obj)
				for i, ref := range routeStatus.Parents {
					if ref.ParentRef.Kind != nil && *ref.ParentRef.Kind != types.KindGateway {
						continue
					}
					parent := types.NamespacedNameKind{
						Name:      string(ref.ParentRef.Name),
						Namespace: string(cmp.Or(ptrValue(ref.ParentRef.Namespace), gatewayv1.Namespace(cp.GetNamespace()))),
						Kind:      types.KindGateway,
					}
					if _, ok := gatewayRefs[parent]; ok {
						routeStatus.Parents[i].Conditions = change.apply(ref.Conditions, cp.GetGeneration())
					}
				}
				return cp
			}),
		})
	}
}

func ptrValue[T any](p *T) T {
	var zero T
	if p == nil {
		return zero
	}
	return *p
}

func newApisixObject(kind string) client.Object {
	switch kind {
	case types.KindApisixRoute:
		return &apiv2.ApisixRoute{}
	case types.KindApisixGlobalRule:
		return &apiv2.ApisixGlobalRule{}
	case types.KindApisixTls:
		return &apiv2.ApisixTls{}
	default:
		return &apiv2.ApisixConsumer{}
	}
}

func apisixStatusOf(obj client.Object) (client.Object, *apiv2.ApisixStatus) {
	switch o := obj.(type) {
	case *apiv2.ApisixRoute:
		cp := o.DeepCopy()
		return cp, &cp.Status
	case *apiv2.ApisixGlobalRule:
		cp := o.DeepCopy()
		return cp, &cp.Status
	case *apiv2.ApisixTls:
		cp := o.DeepCopy()
		return cp, &cp.Status
	case *apiv2.ApisixConsumer:
		cp := o.DeepCopy()
		return cp, &cp.Status
	}
	panic(fmt.Sprintf("unsupported object type %T", obj))
}

func newRouteObject(kind string) client.Object {
	switch kind {
	case types.KindHTTPRoute:
		return &gatewayv1.HTTPRoute{}
	case types.KindGRPCRoute:
		return &gatewayv1.GRPCRoute{}
	case types.KindTCPRoute:
		return &gatewayv1.TCPRoute{}
	case types.KindUDPRoute:
		return &gatewayv1.UDPRoute{}
	default:
		return &gatewayv1.TLSRoute{}
	}
}

func routeStatusOf(obj client.Object) (client.Object, *gatewayv1.RouteStatus) {
	switch o := obj.(type) {
	case *gatewayv1.HTTPRoute:
		cp := o.DeepCopy()
		return cp, &cp.Status.RouteStatus
	case *gatewayv1.GRPCRoute:
		cp := o.DeepCopy()
		return cp, &cp.Status.RouteStatus
	case *gatewayv1.TCPRoute:
		cp := o.DeepCopy()
		return cp, &cp.Status.RouteStatus
	case *gatewayv1.UDPRoute:
		cp := o.DeepCopy()
		return cp, &cp.Status.RouteStatus
	case *gatewayv1.TLSRoute:
		cp := o.DeepCopy()
		return cp, &cp.Status.RouteStatus
	}
	panic(fmt.Sprintf("unsupported object type %T", obj))
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
