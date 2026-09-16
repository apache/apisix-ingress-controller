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
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	apiv1alpha1 "github.com/apache/apisix-ingress-controller/api/v1alpha1"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/adc/cache"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	"github.com/apache/apisix-ingress-controller/internal/provider/common"
	"github.com/apache/apisix-ingress-controller/internal/types"
)

// fakeK8sClient builds a controller-runtime fake client seeded with objects, for the
// GatewayProxy lookup recordFailedEndpointEvents does before firing an Event.
func fakeK8sClient(t *testing.T, objects ...runtime.Object) client.Client {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, apiv1alpha1.AddToScheme(scheme))
	return fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(objects...).Build()
}

func TestUnavailableEndpointsMessageEmptyWhenEverythingSucceeded(t *testing.T) {
	msg := unavailableEndpointsMessage([]adctypes.EndpointStatus{
		{Server: "http://apisix-1:9180", Success: true},
		{Server: "http://apisix-2:9180", Success: true},
	})
	if msg != "" {
		t.Fatalf("expected no message, got %q", msg)
	}
}

func TestUnavailableEndpointsMessageEmptyWhenThereAreNoEndpoints(t *testing.T) {
	if msg := unavailableEndpointsMessage(nil); msg != "" {
		t.Fatalf("expected no message, got %q", msg)
	}
}

func TestUnavailableEndpointsMessageSummarizesOnlyTheFailedOnes(t *testing.T) {
	msg := unavailableEndpointsMessage([]adctypes.EndpointStatus{
		{Server: "http://apisix-1:9180", Success: true},
		{Server: "http://apisix-2:9180", Success: false, Reason: "connection refused"},
	})
	want := "1/2 gateway instance(s) failed to apply the last sync: http://apisix-2:9180: connection refused"
	if msg != want {
		t.Fatalf("got %q, want %q", msg, want)
	}
}

func TestRecordFailedEndpointEventsFiresOneWarningPerFailedEndpoint(t *testing.T) {
	recorder := record.NewFakeRecorder(2)
	d := &apisixProvider{log: logr.Discard()}
	d.EventRecorder = recorder
	d.K8sClient = fakeK8sClient(t, &apiv1alpha1.GatewayProxy{
		ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "gp", UID: "gp-uid"},
	})
	nnk := types.NamespacedNameKind{Kind: types.KindGatewayProxy, Namespace: "ns", Name: "gp"}

	d.recordFailedEndpointEvents(context.Background(), nnk, []adctypes.EndpointStatus{
		{Server: "http://apisix-1:9180", Success: true},
		{Server: "http://apisix-2:9180", Success: false, Reason: "connection refused"},
		{Server: "http://apisix-3:9180", Success: false, Reason: "TLS handshake failed"},
	})

	first := <-recorder.Events
	if !strings.Contains(first, "Warning") || !strings.Contains(first, "DataPlaneInstanceUnavailable") ||
		!strings.Contains(first, "http://apisix-2:9180: connection refused") {
		t.Errorf("first event = %q, want the apisix-2 failure", first)
	}

	second := <-recorder.Events
	if !strings.Contains(second, "Warning") || !strings.Contains(second, "DataPlaneInstanceUnavailable") ||
		!strings.Contains(second, "http://apisix-3:9180: TLS handshake failed") {
		t.Errorf("second event = %q, want the apisix-3 failure", second)
	}

	select {
	case e := <-recorder.Events:
		t.Errorf("unexpected third event %q, the successful endpoint should not fire one", e)
	default:
	}
}

func TestRecordFailedEndpointEventsNoopsWithoutARecorder(t *testing.T) {
	d := &apisixProvider{log: logr.Discard()}
	nnk := types.NamespacedNameKind{Kind: types.KindGatewayProxy, Namespace: "ns", Name: "gp"}

	// Must not panic when no EventRecorder was configured. No K8sClient either: a nil
	// dereference here would mean this didn't actually return before reaching it.
	d.recordFailedEndpointEvents(context.Background(), nnk, []adctypes.EndpointStatus{{Server: "http://apisix-1:9180", Success: false}})
}

func TestRecordFailedEndpointEventsNoopsWhenEveryEndpointSucceeded(t *testing.T) {
	recorder := record.NewFakeRecorder(1)
	d := &apisixProvider{log: logr.Discard()}
	d.EventRecorder = recorder
	nnk := types.NamespacedNameKind{Kind: types.KindGatewayProxy, Namespace: "ns", Name: "gp"}

	// No K8sClient configured either: with nothing failed, the GatewayProxy lookup this
	// needs to attribute a failure must never even be attempted.
	d.recordFailedEndpointEvents(context.Background(), nnk, []adctypes.EndpointStatus{{Server: "http://apisix-1:9180", Success: true}})

	select {
	case e := <-recorder.Events:
		t.Errorf("unexpected event %q, nothing failed", e)
	default:
	}
}

func TestRecordFailedEndpointEventsSkipsWhenTheGatewayProxyCannotBeFetched(t *testing.T) {
	recorder := record.NewFakeRecorder(1)
	d := &apisixProvider{log: logr.Discard()}
	d.EventRecorder = recorder
	d.K8sClient = fakeK8sClient(t) // no GatewayProxy seeded, so Get returns NotFound
	nnk := types.NamespacedNameKind{Kind: types.KindGatewayProxy, Namespace: "ns", Name: "gp"}

	// Must not panic, and must not fire an event against a zero-value stand-in.
	d.recordFailedEndpointEvents(context.Background(), nnk, []adctypes.EndpointStatus{{Server: "http://apisix-1:9180", Success: false}})

	select {
	case e := <-recorder.Events:
		t.Errorf("unexpected event %q, the GatewayProxy could not be fetched", e)
	default:
	}
}

// fakeUpdater records every status.Update handed to it, and applies each Mutator against
// a caller-supplied base object so a test can inspect the condition it actually wrote.
type fakeUpdater struct {
	updates []status.Update
}

func (f *fakeUpdater) Update(u status.Update) {
	f.updates = append(f.updates, u)
}

const testConfigName = "GatewayProxy/ns/gp"

var testGatewayProxy = types.NamespacedNameKind{Kind: types.KindGatewayProxy, Namespace: "ns", Name: "gp"}

func labelsOf(owner types.NamespacedNameKind) map[string]string {
	return map[string]string{
		label.LabelKind:      owner.Kind,
		label.LabelNamespace: owner.Namespace,
		label.LabelName:      owner.Name,
	}
}

func newStatusTestProvider() (*apisixProvider, *fakeUpdater) {
	updater := &fakeUpdater{}
	return &apisixProvider{
		log:           logr.Discard(),
		store:         cache.NewStore(logr.Discard()),
		configManager: common.NewConfigManager[types.NamespacedNameKind, adctypes.Config](),
		skipped:       newSkipTable(),
		updater:       updater,
		gatewayEvents: make(chan event.GenericEvent, 10),
	}, updater
}

// rejected builds a sync result in which the data plane rejected every given event.
func rejected(events ...adctypes.StatusEvent) types.ADCExecutionErrors {
	statuses := make([]adctypes.SyncStatus, 0, len(events))
	for _, ev := range events {
		statuses = append(statuses, adctypes.SyncStatus{Reason: "rejected " + ev.ResourceID, Event: ev})
	}
	return types.ADCExecutionErrors{Errors: []types.ADCExecutionError{{
		Name: testConfigName,
		FailedErrors: []types.ADCExecutionServerAddrError{{
			ServerAddr:     "http://apisix:9180",
			FailedStatuses: statuses,
		}},
	}}}
}

func TestClassifySyncResultHardErrorGoesToGatewayProxy(t *testing.T) {
	d, _ := newStatusTestProvider()
	execErrs := types.ADCExecutionErrors{Errors: []types.ADCExecutionError{{
		Name: testConfigName,
		FailedErrors: []types.ADCExecutionServerAddrError{{
			ServerAddr: "http://apisix:9180",
			Err:        "HTTP 500: boom",
		}},
	}}}

	dropped := map[wireKey]exclusion{}
	gatewayProxyMsgs, failedEndpoints := d.classifySyncResult(testConfigName, execErrs, d.store.Revision(), dropped)

	assert.Empty(t, dropped)
	assert.Empty(t, failedEndpoints)
	require.Len(t, gatewayProxyMsgs, 1)
	assert.Contains(t, gatewayProxyMsgs[0], "HTTP 500: boom")
}

func TestClassifySyncResultEndpointFailuresGoToGatewayProxy(t *testing.T) {
	d, _ := newStatusTestProvider()
	endpoints := []adctypes.EndpointStatus{
		{Server: "http://apisix-1:9180", Success: true},
		{Server: "http://apisix-2:9180", Success: false, Reason: "connection refused"},
	}
	execErrs := types.ADCExecutionErrors{Errors: []types.ADCExecutionError{{
		Name:         testConfigName,
		FailedErrors: []types.ADCExecutionServerAddrError{{EndpointStatuses: endpoints}},
	}}}

	dropped := map[wireKey]exclusion{}
	gatewayProxyMsgs, failedEndpoints := d.classifySyncResult(testConfigName, execErrs, d.store.Revision(), dropped)

	assert.Empty(t, dropped)
	require.Len(t, gatewayProxyMsgs, 1)
	assert.Contains(t, gatewayProxyMsgs[0], "http://apisix-2:9180: connection refused")
	assert.Len(t, failedEndpoints, 2, "every EndpointStatus entry is passed through for event firing")
}

func TestClassifySyncResultReportsEndpointStatusesEvenOnAFullyAttributedAddrErr(t *testing.T) {
	// apisix-standalone attaches EndpointStatuses to every addrErr, whether or not
	// FailedStatuses also names resources: attributing the resource must never suppress
	// reporting an endpoint that failed for its own reasons.
	d, _ := newStatusTestProvider()
	route := types.NamespacedNameKind{Kind: types.KindApisixRoute, Namespace: "ns1", Name: "route1"}
	require.NoError(t, d.store.Insert(testConfigName, []string{adctypes.TypeService}, &adctypes.Resources{
		Services: []*adctypes.Service{{Metadata: adctypes.Metadata{ID: "svc1"}}},
	}, labelsOf(route)))

	execErrs := rejected(adctypes.StatusEvent{ResourceType: adctypes.TypeService, ResourceID: "svc1"})
	execErrs.Errors[0].FailedErrors[0].EndpointStatuses = []adctypes.EndpointStatus{
		{Server: "http://apisix-1:9180", Success: false, Reason: "content rejected"},
		{Server: "http://apisix-2:9180", Success: false, Reason: "connection refused"},
	}

	dropped := map[wireKey]exclusion{}
	gatewayProxyMsgs, failedEndpoints := d.classifySyncResult(testConfigName, execErrs, d.store.Revision(), dropped)

	require.Len(t, gatewayProxyMsgs, 1)
	assert.Contains(t, gatewayProxyMsgs[0], "http://apisix-2:9180: connection refused")
	assert.Len(t, failedEndpoints, 2)
	assert.Equal(t, route, dropped[wireKey{resourceType: adctypes.TypeService, id: "svc1"}].owner)
}

func TestClassifySyncResultFallsBackToGatewayProxyWhenAFailedStatusHasNoResourceAttribution(t *testing.T) {
	// apisix-standalone: FailedStatuses can be non-empty yet carry no Event at all.
	d, _ := newStatusTestProvider()
	execErrs := types.ADCExecutionErrors{Errors: []types.ADCExecutionError{{
		Name: testConfigName,
		FailedErrors: []types.ADCExecutionServerAddrError{{
			Err:            "all_failed",
			FailedStatuses: []adctypes.SyncStatus{{Reason: "schema error"}},
		}},
	}}}

	dropped := map[wireKey]exclusion{}
	gatewayProxyMsgs, _ := d.classifySyncResult(testConfigName, execErrs, d.store.Revision(), dropped)

	assert.Empty(t, dropped)
	assert.Len(t, gatewayProxyMsgs, 1)
}

func TestClassifySyncResultFallsBackToGatewayProxyForANestedEventWithoutItsParent(t *testing.T) {
	d, _ := newStatusTestProvider()
	dropped := map[wireKey]exclusion{}
	gatewayProxyMsgs, _ := d.classifySyncResult(testConfigName,
		rejected(adctypes.StatusEvent{ResourceType: adctypes.TypeRoute, ResourceID: "r1"}), d.store.Revision(), dropped)

	assert.Empty(t, dropped)
	assert.Len(t, gatewayProxyMsgs, 1)
}

// TestClassifySyncResultDropUnits covers, for every resource type ADC can report, which
// unit gets dropped and who it is attributed to.
func TestClassifySyncResultDropUnits(t *testing.T) {
	httpRoute := types.NamespacedNameKind{Kind: types.KindHTTPRoute, Namespace: "ns", Name: "http"}
	apisixRoute := types.NamespacedNameKind{Kind: types.KindApisixRoute, Namespace: "ns", Name: "apisix"}
	tls := types.NamespacedNameKind{Kind: types.KindApisixTls, Namespace: "ns", Name: "tls"}
	consumer := types.NamespacedNameKind{Kind: types.KindConsumer, Namespace: "ns", Name: "consumer"}
	globalRule := types.NamespacedNameKind{Kind: types.KindApisixGlobalRule, Namespace: "ns", Name: "global"}

	d, _ := newStatusTestProvider()
	require.NoError(t, d.store.Insert(testConfigName, []string{adctypes.TypeService}, &adctypes.Resources{
		Services: []*adctypes.Service{{Metadata: adctypes.Metadata{ID: "http-svc", Name: "ns_http_0"}}},
	}, labelsOf(httpRoute)))
	require.NoError(t, d.store.Insert(testConfigName, []string{adctypes.TypeService}, &adctypes.Resources{
		Services: []*adctypes.Service{{Metadata: adctypes.Metadata{ID: "apisix-svc", Name: "ns_apisix_0"}}},
	}, labelsOf(apisixRoute)))
	require.NoError(t, d.store.Insert(testConfigName, []string{adctypes.TypeSSL}, &adctypes.Resources{
		SSLs: []*adctypes.SSL{{Metadata: adctypes.Metadata{ID: "ssl1"}}},
	}, labelsOf(tls)))
	require.NoError(t, d.store.Insert(testConfigName, []string{adctypes.TypeConsumer}, &adctypes.Resources{
		Consumers: []*adctypes.Consumer{{Username: "alice"}},
	}, labelsOf(consumer)))
	require.NoError(t, d.store.SetGlobalRules(testConfigName, globalRule, adctypes.GlobalRule{"prometheus": map[string]any{}}))
	require.NoError(t, d.store.SetPluginMetadata(testConfigName, adctypes.PluginMetadata{"http-logger": map[string]any{}}))

	cases := []struct {
		name  string
		event adctypes.StatusEvent
		key   wireKey
		owner types.NamespacedNameKind
	}{
		{"service", adctypes.StatusEvent{ResourceType: adctypes.TypeService, ResourceID: "apisix-svc"},
			wireKey{resourceType: adctypes.TypeService, id: "apisix-svc"}, apisixRoute},
		{"named upstream takes its service", adctypes.StatusEvent{ResourceType: adctypes.TypeUpstream, ResourceID: "ups", ParentID: "apisix-svc"},
			wireKey{resourceType: adctypes.TypeService, id: "apisix-svc"}, apisixRoute},
		{"Gateway API route takes its rule", adctypes.StatusEvent{ResourceType: adctypes.TypeRoute, ResourceID: "r", ParentID: "http-svc"},
			wireKey{resourceType: adctypes.TypeService, id: "http-svc"}, httpRoute},
		{"Gateway API stream route takes its rule", adctypes.StatusEvent{ResourceType: adctypes.TypeStreamRoute, ResourceID: "sr", ParentID: "http-svc"},
			wireKey{resourceType: adctypes.TypeService, id: "http-svc"}, httpRoute},
		{"other route is dropped alone", adctypes.StatusEvent{ResourceType: adctypes.TypeRoute, ResourceID: "r", ParentID: "apisix-svc"},
			wireKey{resourceType: adctypes.TypeRoute, parentID: "apisix-svc", id: "r"}, apisixRoute},
		{"other stream route is dropped alone", adctypes.StatusEvent{ResourceType: adctypes.TypeStreamRoute, ResourceID: "sr", ParentID: "apisix-svc"},
			wireKey{resourceType: adctypes.TypeStreamRoute, parentID: "apisix-svc", id: "sr"}, apisixRoute},
		{"ssl", adctypes.StatusEvent{ResourceType: adctypes.TypeSSL, ResourceID: "ssl1"},
			wireKey{resourceType: adctypes.TypeSSL, id: "ssl1"}, tls},
		{"consumer", adctypes.StatusEvent{ResourceType: adctypes.TypeConsumer, ResourceID: "alice"},
			wireKey{resourceType: adctypes.TypeConsumer, id: "alice"}, consumer},
		{"credential is dropped alone", adctypes.StatusEvent{ResourceType: adctypes.TypeConsumerCredential, ResourceID: "cred", ParentID: "alice"},
			wireKey{resourceType: adctypes.TypeConsumerCredential, parentID: "alice", id: "cred"}, consumer},
		{"global rule", adctypes.StatusEvent{ResourceType: adctypes.TypeGlobalRule, ResourceID: "prometheus"},
			wireKey{resourceType: adctypes.TypeGlobalRule, id: "prometheus"}, globalRule},
		{"plugin metadata belongs to the GatewayProxy", adctypes.StatusEvent{ResourceType: adctypes.TypePluginMetadata, ResourceID: "http-logger"},
			wireKey{resourceType: adctypes.TypePluginMetadata, id: "http-logger"}, testGatewayProxy},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dropped := map[wireKey]exclusion{}
			gatewayProxyMsgs, _ := d.classifySyncResult(testConfigName, rejected(tc.event), d.store.Revision(), dropped)

			assert.Empty(t, gatewayProxyMsgs)
			require.Len(t, dropped, 1)
			require.Contains(t, dropped, tc.key)
			assert.Equal(t, tc.owner, dropped[tc.key].owner)
			assert.Contains(t, dropped[tc.key].reason, "rejected "+tc.event.ResourceID)
		})
	}
}

func TestClassifySyncResultTellsApartUpstreamsOfTheSameIDInDifferentServices(t *testing.T) {
	a := types.NamespacedNameKind{Kind: types.KindApisixRoute, Namespace: "ns", Name: "a"}
	b := types.NamespacedNameKind{Kind: types.KindApisixRoute, Namespace: "ns", Name: "b"}
	d, _ := newStatusTestProvider()
	for owner, svc := range map[types.NamespacedNameKind]string{a: "svc-a", b: "svc-b"} {
		require.NoError(t, d.store.Insert(testConfigName, []string{adctypes.TypeService}, &adctypes.Resources{
			Services: []*adctypes.Service{{
				Metadata:  adctypes.Metadata{ID: svc},
				Upstreams: []*adctypes.Upstream{{Metadata: adctypes.Metadata{ID: "shared"}}},
			}},
		}, labelsOf(owner)))
	}

	dropped := map[wireKey]exclusion{}
	d.classifySyncResult(testConfigName, rejected(adctypes.StatusEvent{ResourceType: adctypes.TypeUpstream, ResourceID: "shared", ParentID: "svc-b"}), d.store.Revision(), dropped)

	require.Len(t, dropped, 1)
	assert.Equal(t, b, dropped[wireKey{resourceType: adctypes.TypeService, id: "svc-b"}].owner)
}

// TestClassifySyncResultIgnoresFailuresAgainstReplacedContent covers the race between a
// sync and a reconcile: a failure reported for content the store has since replaced must
// not exclude the replacement.
func TestClassifySyncResultIgnoresFailuresAgainstReplacedContent(t *testing.T) {
	route := types.NamespacedNameKind{Kind: types.KindApisixRoute, Namespace: "ns", Name: "route"}
	d, _ := newStatusTestProvider()
	write := func(plugins adctypes.Plugins) {
		require.NoError(t, d.store.Insert(testConfigName, []string{adctypes.TypeService}, &adctypes.Resources{
			Services: []*adctypes.Service{{Metadata: adctypes.Metadata{ID: "svc", Labels: labelsOf(route)}, Plugins: plugins}},
		}, labelsOf(route)))
	}
	write(adctypes.Plugins{"bad": map[string]any{}})
	_, built, err := d.store.GetResources(testConfigName)
	require.NoError(t, err)

	execErrs := rejected(adctypes.StatusEvent{ResourceType: adctypes.TypeService, ResourceID: "svc"})

	write(adctypes.Plugins{"bad": map[string]any{}})
	dropped := map[wireKey]exclusion{}
	d.classifySyncResult(testConfigName, execErrs, built, dropped)
	assert.Len(t, dropped, 1, "rewriting identical content does not replace what the failure was about")

	write(adctypes.Plugins{"fixed": map[string]any{}})
	dropped = map[wireKey]exclusion{}
	gatewayProxyMsgs, _ := d.classifySyncResult(testConfigName, execErrs, built, dropped)
	assert.Empty(t, dropped, "the failure was about content that has since changed")
	assert.Empty(t, gatewayProxyMsgs, "a stale failure is not a GatewayProxy error either")
}

func conditionsOf(t *testing.T, updater *fakeUpdater, name string, obj client.Object) []metav1.Condition {
	t.Helper()
	for _, u := range updater.updates {
		if u.NamespacedName.Name != name {
			continue
		}
		obj = u.Mutator.Mutate(obj)
	}
	switch o := obj.(type) {
	case *apiv2.ApisixRoute:
		return o.Status.Conditions
	case *apiv1alpha1.GatewayProxy:
		return o.Status.Conditions
	case *gatewayv1.HTTPRoute:
		return o.Status.Parents[0].Conditions
	}
	t.Fatalf("unsupported object %T", obj)
	return nil
}

func findCondition(conditions []metav1.Condition, conditionType string) *metav1.Condition {
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return &conditions[i]
		}
	}
	return nil
}

func TestUpdateStatusFromSyncResultsReportsPartialFullAndRecoveredDrops(t *testing.T) {
	route := types.NamespacedNameKind{Kind: types.KindApisixRoute, Namespace: "ns", Name: "route"}
	d, updater := newStatusTestProvider()
	require.NoError(t, d.store.Insert(testConfigName, []string{adctypes.TypeService}, &adctypes.Resources{
		Services: []*adctypes.Service{
			{Metadata: adctypes.Metadata{ID: "svc1", Name: "ns_route_0"}},
			{Metadata: adctypes.Metadata{ID: "svc2", Name: "ns_route_1"}},
		},
	}, labelsOf(route)))
	sync := func(execErrs types.ADCExecutionErrors) []metav1.Condition {
		updater.updates = nil
		d.updateStatusFromSyncResults(context.Background(),
			map[string]types.ADCExecutionErrors{testConfigName: execErrs},
			map[string]uint64{testConfigName: d.store.Revision()})
		return conditionsOf(t, updater, "route", &apiv2.ApisixRoute{})
	}

	conditions := sync(rejected(adctypes.StatusEvent{ResourceType: adctypes.TypeService, ResourceID: "svc1"}))
	require.NotNil(t, findCondition(conditions, string(apiv2.ConditionTypeAccepted)))
	assert.Equal(t, metav1.ConditionTrue, findCondition(conditions, string(apiv2.ConditionTypeAccepted)).Status)
	partiallyInvalid := findCondition(conditions, conditionTypePartiallyInvalid)
	require.NotNil(t, partiallyInvalid, "part of the route is still served")
	assert.Equal(t, metav1.ConditionTrue, partiallyInvalid.Status)
	assert.Contains(t, partiallyInvalid.Message, "ns_route_0")

	// svc1 is excluded now, so ADC only reports svc2; the route is dropped entirely.
	conditions = sync(rejected(adctypes.StatusEvent{ResourceType: adctypes.TypeService, ResourceID: "svc2"}))
	assert.Equal(t, metav1.ConditionFalse, findCondition(conditions, string(apiv2.ConditionTypeAccepted)).Status)
	assert.Nil(t, findCondition(conditions, conditionTypePartiallyInvalid))

	d.skipped.ClearOwner(route)
	conditions = sync(types.ADCExecutionErrors{})
	assert.Equal(t, metav1.ConditionTrue, findCondition(conditions, string(apiv2.ConditionTypeAccepted)).Status)
	assert.Nil(t, findCondition(conditions, conditionTypePartiallyInvalid))
}

func TestUpdateStatusFromSyncResultsReportsADroppedRuleOnTheRouteParent(t *testing.T) {
	route := types.NamespacedNameKind{Kind: types.KindHTTPRoute, Namespace: "ns", Name: "route"}
	gateway := types.NamespacedNameKind{Kind: types.KindGateway, Namespace: "ns", Name: "gw"}
	d, updater := newStatusTestProvider()
	d.configManager.Update(route, map[types.NamespacedNameKind]adctypes.Config{testGatewayProxy: {Name: testConfigName}})
	d.configManager.SetConfigRefs(testGatewayProxy, []types.NamespacedNameKind{gateway})
	require.NoError(t, d.store.Insert(testConfigName, []string{adctypes.TypeService}, &adctypes.Resources{
		Services: []*adctypes.Service{
			{Metadata: adctypes.Metadata{ID: "rule0", Name: "ns_route_0"}},
			{Metadata: adctypes.Metadata{ID: "rule1", Name: "ns_route_1"}},
		},
	}, labelsOf(route)))

	d.updateStatusFromSyncResults(context.Background(),
		map[string]types.ADCExecutionErrors{testConfigName: rejected(adctypes.StatusEvent{ResourceType: adctypes.TypeRoute, ResourceID: "r", ParentID: "rule1"})},
		map[string]uint64{testConfigName: d.store.Revision()})

	httpRoute := &gatewayv1.HTTPRoute{ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "route"}}
	httpRoute.Status.Parents = []gatewayv1.RouteParentStatus{{ParentRef: gatewayv1.ParentReference{Name: "gw"}}}
	conditions := conditionsOf(t, updater, "route", httpRoute)
	partiallyInvalid := findCondition(conditions, conditionTypePartiallyInvalid)
	require.NotNil(t, partiallyInvalid)
	assert.True(t, strings.HasPrefix(partiallyInvalid.Message, "Dropped Rule"), partiallyInvalid.Message)
	assert.Contains(t, partiallyInvalid.Message, "ns_route_1")
}

func TestUpdateStatusFromSyncResultsReportsDroppedGatewayProxyPlugins(t *testing.T) {
	d, updater := newStatusTestProvider()
	require.NoError(t, d.store.SetGlobalRules(testConfigName, testGatewayProxy, adctypes.GlobalRule{"prometheus": map[string]any{}}))
	sync := func(execErrs types.ADCExecutionErrors) *metav1.Condition {
		updater.updates = nil
		d.updateStatusFromSyncResults(context.Background(),
			map[string]types.ADCExecutionErrors{testConfigName: execErrs},
			map[string]uint64{testConfigName: d.store.Revision()})
		return findCondition(conditionsOf(t, updater, "gp", &apiv1alpha1.GatewayProxy{}), GatewayProxyConditionPluginsProgrammed)
	}

	condition := sync(rejected(adctypes.StatusEvent{ResourceType: adctypes.TypeGlobalRule, ResourceID: "prometheus"}))
	require.NotNil(t, condition)
	assert.Equal(t, metav1.ConditionFalse, condition.Status)
	assert.Contains(t, condition.Message, "prometheus")
	dataPlane := findCondition(conditionsOf(t, updater, "gp", &apiv1alpha1.GatewayProxy{}), GatewayProxyConditionDataPlaneAvailable)
	require.NotNil(t, dataPlane)
	assert.Equal(t, metav1.ConditionTrue, dataPlane.Status, "a rejected plugin says nothing about the instances")

	d.skipped.ClearOwner(testGatewayProxy)
	condition = sync(types.ADCExecutionErrors{})
	assert.Equal(t, metav1.ConditionTrue, condition.Status)
}

func TestUpdateStatusFromSyncResultsHandsRejectedListenerCertificatesToTheGatewayController(t *testing.T) {
	gateway := types.NamespacedNameKind{Kind: types.KindGateway, Namespace: "ns", Name: "gw"}
	d, _ := newStatusTestProvider()
	require.NoError(t, d.store.Insert(testConfigName, []string{adctypes.TypeSSL}, &adctypes.Resources{
		SSLs: []*adctypes.SSL{{Metadata: adctypes.Metadata{ID: "ssl1"}}},
	}, labelsOf(gateway)))
	sync := func(execErrs types.ADCExecutionErrors) {
		d.updateStatusFromSyncResults(context.Background(),
			map[string]types.ADCExecutionErrors{testConfigName: execErrs},
			map[string]uint64{testConfigName: d.store.Revision()})
	}

	sync(rejected(adctypes.StatusEvent{ResourceType: adctypes.TypeSSL, ResourceID: "ssl1"}))
	assert.Contains(t, d.RejectedCertificates(gateway.NamespacedName())["ssl1"], "rejected ssl1")
	require.Len(t, d.gatewayEvents, 1)
	ev := <-d.gatewayEvents
	assert.Equal(t, "gw", ev.Object.GetName())

	sync(types.ADCExecutionErrors{})
	assert.Empty(t, d.gatewayEvents, "nothing changed for the Gateway")

	d.skipped.ClearOwner(gateway)
	sync(types.ADCExecutionErrors{})
	assert.Empty(t, d.RejectedCertificates(gateway.NamespacedName()))
	assert.Len(t, d.gatewayEvents, 1, "the Gateway is notified that its certificate is no longer rejected")
}

func TestUpdateStatusFromSyncResultsFiresAnEventForADroppedIngress(t *testing.T) {
	ingress := types.NamespacedNameKind{Kind: types.KindIngress, Namespace: "ns", Name: "ing"}
	recorder := record.NewFakeRecorder(1)
	d, _ := newStatusTestProvider()
	d.EventRecorder = recorder
	d.K8sClient = fakeK8sClient(t, &networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "ing"}})
	require.NoError(t, d.store.Insert(testConfigName, []string{adctypes.TypeService}, &adctypes.Resources{
		Services: []*adctypes.Service{{Metadata: adctypes.Metadata{ID: "svc", Name: "ns_ing_0"}}},
	}, labelsOf(ingress)))

	d.updateStatusFromSyncResults(context.Background(),
		map[string]types.ADCExecutionErrors{testConfigName: rejected(adctypes.StatusEvent{ResourceType: adctypes.TypeService, ResourceID: "svc"})},
		map[string]uint64{testConfigName: d.store.Revision()})

	require.Len(t, recorder.Events, 1)
	e := <-recorder.Events
	assert.Contains(t, e, "Warning")
	assert.Contains(t, e, "rejected svc")
}

// TestUpdateStatusFromSyncResultsReportsAServiceWithoutRoutesLeftAsFullyDropped covers a
// resource whose routes are all dropped: the service itself was never rejected, but it
// serves nothing, so the resource is not partially served.
func TestUpdateStatusFromSyncResultsReportsAServiceWithoutRoutesLeftAsFullyDropped(t *testing.T) {
	route := types.NamespacedNameKind{Kind: types.KindApisixRoute, Namespace: "ns", Name: "route"}
	d, updater := newStatusTestProvider()
	require.NoError(t, d.store.Insert(testConfigName, []string{adctypes.TypeService}, &adctypes.Resources{
		Services: []*adctypes.Service{{
			Metadata: adctypes.Metadata{ID: "svc", Name: "ns_route_0"},
			Routes: []*adctypes.Route{
				{Metadata: adctypes.Metadata{ID: "r1", Name: "ns_route_0-0"}},
				{Metadata: adctypes.Metadata{ID: "r2", Name: "ns_route_0-1"}},
			},
		}},
	}, labelsOf(route)))
	sync := func(events ...adctypes.StatusEvent) []metav1.Condition {
		updater.updates = nil
		d.updateStatusFromSyncResults(context.Background(),
			map[string]types.ADCExecutionErrors{testConfigName: rejected(events...)},
			map[string]uint64{testConfigName: d.store.Revision()})
		return conditionsOf(t, updater, "route", &apiv2.ApisixRoute{})
	}

	conditions := sync(adctypes.StatusEvent{ResourceType: adctypes.TypeRoute, ResourceID: "r1", ParentID: "svc"})
	assert.Equal(t, metav1.ConditionTrue, findCondition(conditions, string(apiv2.ConditionTypeAccepted)).Status)
	require.NotNil(t, findCondition(conditions, conditionTypePartiallyInvalid), "the other route is still served")

	conditions = sync(adctypes.StatusEvent{ResourceType: adctypes.TypeRoute, ResourceID: "r2", ParentID: "svc"})
	assert.Equal(t, metav1.ConditionFalse, findCondition(conditions, string(apiv2.ConditionTypeAccepted)).Status)
	assert.Equal(t, string(apiv2.ConditionReasonSyncFailed), findCondition(conditions, string(apiv2.ConditionTypeAccepted)).Reason)
	assert.Nil(t, findCondition(conditions, conditionTypePartiallyInvalid))
	assert.NotContains(t, findCondition(conditions, string(apiv2.ConditionTypeAccepted)).Message, "ns_route_0-1: ",
		"a resource with nothing left reports the data plane's reasons on their own")
}
