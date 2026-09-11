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
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	apiv1alpha1 "github.com/apache/apisix-ingress-controller/api/v1alpha1"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/adc/cache"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
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

func TestFailureConditionMarksGatewayProxyDataPlaneAvailableFalse(t *testing.T) {
	nnk := types.NamespacedNameKind{Kind: types.KindGatewayProxy, Namespace: "ns", Name: "gp"}
	c := failureCondition(nnk, "boom")

	if c.Type != GatewayProxyConditionDataPlaneAvailable {
		t.Errorf("Type = %q, want %q", c.Type, GatewayProxyConditionDataPlaneAvailable)
	}
	if c.Status != metav1.ConditionFalse {
		t.Errorf("Status = %v, want False", c.Status)
	}
	if c.Reason != GatewayProxyReasonDataPlaneInstanceUnavailable {
		t.Errorf("Reason = %q, want %q", c.Reason, GatewayProxyReasonDataPlaneInstanceUnavailable)
	}
	if c.Message != "boom" {
		t.Errorf("Message = %q, want %q", c.Message, "boom")
	}
}

func TestSuccessConditionMarksGatewayProxyDataPlaneAvailableTrue(t *testing.T) {
	nnk := types.NamespacedNameKind{Kind: types.KindGatewayProxy, Namespace: "ns", Name: "gp"}
	c := successCondition(nnk)

	if c.Type != GatewayProxyConditionDataPlaneAvailable {
		t.Errorf("Type = %q, want %q", c.Type, GatewayProxyConditionDataPlaneAvailable)
	}
	if c.Status != metav1.ConditionTrue {
		t.Errorf("Status = %v, want True", c.Status)
	}
	if c.Reason != GatewayProxyReasonDataPlaneAvailable {
		t.Errorf("Reason = %q, want %q", c.Reason, GatewayProxyReasonDataPlaneAvailable)
	}
}

func TestFailureAndSuccessConditionKeepUsingAcceptedForNonGatewayProxyKinds(t *testing.T) {
	nnk := types.NamespacedNameKind{Kind: types.KindApisixRoute, Namespace: "ns", Name: "route"}

	failed := failureCondition(nnk, "boom")
	if failed.Type != string(apiv2.ConditionTypeAccepted) {
		t.Errorf("failure Type = %q, want %q", failed.Type, apiv2.ConditionTypeAccepted)
	}
	if failed.Reason != string(apiv2.ConditionReasonSyncFailed) {
		t.Errorf("failure Reason = %q, want %q", failed.Reason, apiv2.ConditionReasonSyncFailed)
	}

	ok := successCondition(nnk)
	if ok.Type != string(apiv2.ConditionTypeAccepted) {
		t.Errorf("success Type = %q, want %q", ok.Type, apiv2.ConditionTypeAccepted)
	}
	if ok.Reason != string(apiv2.ConditionReasonAccepted) {
		t.Errorf("success Reason = %q, want %q", ok.Reason, apiv2.ConditionReasonAccepted)
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

func TestClassifySyncResultHardErrorGoesToGatewayProxy(t *testing.T) {
	d := &apisixProvider{log: logr.Discard()}
	execErrs := types.ADCExecutionErrors{Errors: []types.ADCExecutionError{{
		Name: "GatewayProxy/ns/gp",
		FailedErrors: []types.ADCExecutionServerAddrError{{
			ServerAddr: "http://apisix:9180",
			Err:        "HTTP 500: boom",
		}},
	}}}

	resourceFailures := map[types.NamespacedNameKind][]string{}
	gatewayProxyMsgs, failedEndpoints := d.classifySyncResult("GatewayProxy/ns/gp", execErrs, resourceFailures)

	if len(resourceFailures) != 0 {
		t.Errorf("expected no resource attributed, got %v", resourceFailures)
	}
	if len(failedEndpoints) != 0 {
		t.Errorf("expected no endpoints, got %v", failedEndpoints)
	}
	if len(gatewayProxyMsgs) != 1 || !strings.Contains(gatewayProxyMsgs[0], "HTTP 500: boom") {
		t.Errorf("gatewayProxyMsgs = %v, want the raw error", gatewayProxyMsgs)
	}
}

func TestClassifySyncResultEndpointFailuresGoToGatewayProxy(t *testing.T) {
	d := &apisixProvider{log: logr.Discard()}
	endpoints := []adctypes.EndpointStatus{
		{Server: "http://apisix-1:9180", Success: true},
		{Server: "http://apisix-2:9180", Success: false, Reason: "connection refused"},
	}
	execErrs := types.ADCExecutionErrors{Errors: []types.ADCExecutionError{{
		Name: "GatewayProxy/ns/gp",
		FailedErrors: []types.ADCExecutionServerAddrError{{
			EndpointStatuses: endpoints,
		}},
	}}}

	resourceFailures := map[types.NamespacedNameKind][]string{}
	gatewayProxyMsgs, failedEndpoints := d.classifySyncResult("GatewayProxy/ns/gp", execErrs, resourceFailures)

	if len(resourceFailures) != 0 {
		t.Errorf("expected no resource attributed, got %v", resourceFailures)
	}
	if len(gatewayProxyMsgs) != 1 || !strings.Contains(gatewayProxyMsgs[0], "http://apisix-2:9180: connection refused") {
		t.Errorf("gatewayProxyMsgs = %v, want the endpoint summary", gatewayProxyMsgs)
	}
	if len(failedEndpoints) != 2 {
		t.Errorf("failedEndpoints = %v, want every EndpointStatus entry passed through for event firing", failedEndpoints)
	}
}

func TestClassifySyncResultAttributesFailedStatusesToTheirResource(t *testing.T) {
	d := &apisixProvider{log: logr.Discard(), store: cache.NewStore(logr.Discard())}
	const configName = "GatewayProxy/ns/gp"
	if err := d.store.Insert(configName, []string{adctypes.TypeService}, &adctypes.Resources{
		Services: []*adctypes.Service{{
			Metadata: adctypes.Metadata{
				ID: "svc1",
				Labels: map[string]string{
					label.LabelKind:      "ApisixRoute",
					label.LabelName:      "route1",
					label.LabelNamespace: "ns1",
				},
			},
		}},
	}, nil); err != nil {
		t.Fatalf("seeding the store: %v", err)
	}

	execErrs := types.ADCExecutionErrors{Errors: []types.ADCExecutionError{{
		Name: configName,
		FailedErrors: []types.ADCExecutionServerAddrError{{
			ServerAddr: "http://apisix:9180",
			FailedStatuses: []adctypes.SyncStatus{{
				Reason: "unknown plugin foo",
				Event:  adctypes.StatusEvent{ResourceType: adctypes.TypeService, ResourceID: "svc1"},
			}},
		}},
	}}}

	resourceFailures := map[types.NamespacedNameKind][]string{}
	gatewayProxyMsgs, _ := d.classifySyncResult(configName, execErrs, resourceFailures)

	if len(gatewayProxyMsgs) != 0 {
		t.Errorf("expected nothing attributed to the GatewayProxy, got %v", gatewayProxyMsgs)
	}
	want := types.NamespacedNameKind{Kind: "ApisixRoute", Namespace: "ns1", Name: "route1"}
	if got := resourceFailures[want]; len(got) != 1 || !strings.Contains(got[0], "unknown plugin foo") {
		t.Errorf("resourceFailures[%v] = %v, want the failure reason", want, got)
	}
}

func TestClassifySyncResultReportsEndpointStatusesEvenOnAFullyAttributedAddrErr(t *testing.T) {
	// apisix-standalone's own re-validate path attaches EndpointStatuses to every addrErr
	// regardless of whether FailedStatuses also named specific resources: an all-rejected
	// write leaves every endpoint success:false, and one of those endpoints may be
	// failing for a reason that has nothing to do with the resource FailedStatuses names
	// (e.g. genuinely unreachable, not just rejecting this content). The two signals are
	// independent: attributing the resource failure must never suppress reporting the
	// endpoint failure too.
	d := &apisixProvider{log: logr.Discard(), store: cache.NewStore(logr.Discard())}
	const configName = "GatewayProxy/ns/gp"
	if err := d.store.Insert(configName, []string{adctypes.TypeService}, &adctypes.Resources{
		Services: []*adctypes.Service{{
			Metadata: adctypes.Metadata{
				ID: "svc1",
				Labels: map[string]string{
					label.LabelKind:      "ApisixRoute",
					label.LabelName:      "route1",
					label.LabelNamespace: "ns1",
				},
			},
		}},
	}, nil); err != nil {
		t.Fatalf("seeding the store: %v", err)
	}

	execErrs := types.ADCExecutionErrors{Errors: []types.ADCExecutionError{{
		Name: configName,
		FailedErrors: []types.ADCExecutionServerAddrError{{
			ServerAddr: "http://apisix:9180",
			FailedStatuses: []adctypes.SyncStatus{{
				Reason: "unknown plugin foo",
				Event:  adctypes.StatusEvent{ResourceType: adctypes.TypeService, ResourceID: "svc1"},
			}},
			EndpointStatuses: []adctypes.EndpointStatus{
				{Server: "http://apisix-1:9180", Success: false, Reason: "content rejected"},
				{Server: "http://apisix-2:9180", Success: false, Reason: "connection refused"},
			},
		}},
	}}}

	resourceFailures := map[types.NamespacedNameKind][]string{}
	gatewayProxyMsgs, failedEndpoints := d.classifySyncResult(configName, execErrs, resourceFailures)

	if len(gatewayProxyMsgs) != 1 || !strings.Contains(gatewayProxyMsgs[0], "http://apisix-2:9180: connection refused") {
		t.Errorf("gatewayProxyMsgs = %v, want the endpoint summary", gatewayProxyMsgs)
	}
	if len(failedEndpoints) != 2 {
		t.Errorf("failedEndpoints = %v, want every EndpointStatus entry passed through for event firing", failedEndpoints)
	}
	want := types.NamespacedNameKind{Kind: "ApisixRoute", Namespace: "ns1", Name: "route1"}
	if got := resourceFailures[want]; len(got) != 1 || !strings.Contains(got[0], "unknown plugin foo") {
		t.Errorf("resourceFailures[%v] = %v, want the failure reason", want, got)
	}
}

func TestClassifySyncResultFallsBackToGatewayProxyWhenAFailedStatusHasNoResourceAttribution(t *testing.T) {
	// apisix-standalone: FailedStatuses can be non-empty yet carry no Event to resolve a
	// resource from at all, the whole addrErr is then a GatewayProxy-level signal.
	d := &apisixProvider{log: logr.Discard()}
	execErrs := types.ADCExecutionErrors{Errors: []types.ADCExecutionError{{
		Name: "GatewayProxy/ns/gp",
		FailedErrors: []types.ADCExecutionServerAddrError{{
			Err:            "all_failed",
			FailedStatuses: []adctypes.SyncStatus{{Reason: "schema error"}},
		}},
	}}}

	resourceFailures := map[types.NamespacedNameKind][]string{}
	gatewayProxyMsgs, _ := d.classifySyncResult("GatewayProxy/ns/gp", execErrs, resourceFailures)

	if len(resourceFailures) != 0 {
		t.Errorf("expected no resource attributed, got %v", resourceFailures)
	}
	if len(gatewayProxyMsgs) != 1 {
		t.Errorf("gatewayProxyMsgs = %v, want exactly one fallback message", gatewayProxyMsgs)
	}
}

func TestApplyResourceFailuresWritesNewFailuresAndClearsResolvedOnes(t *testing.T) {
	updater := &fakeUpdater{}
	d := &apisixProvider{
		log:     logr.Discard(),
		updater: updater,
		resourceFailures: map[types.NamespacedNameKind][]string{
			{Kind: types.KindApisixRoute, Namespace: "ns", Name: "resolved"}:  {"used to fail"},
			{Kind: types.KindApisixRoute, Namespace: "ns", Name: "still-bad"}: {"still failing"},
		},
	}

	newFailures := map[types.NamespacedNameKind][]string{
		{Kind: types.KindApisixRoute, Namespace: "ns", Name: "still-bad"}: {"still failing"},
		{Kind: types.KindApisixRoute, Namespace: "ns", Name: "newly-bad"}: {"new failure"},
	}

	d.applyResourceFailures(newFailures)

	byName := map[string]bool{} // name -> whether the mutator it was given marks success
	for _, u := range updater.updates {
		cp := u.Mutator.Mutate(&apiv2.ApisixRoute{})
		route := cp.(*apiv2.ApisixRoute)
		accepted := false
		for _, c := range route.Status.Conditions {
			if c.Type == string(apiv2.ConditionTypeAccepted) {
				accepted = c.Status == metav1.ConditionTrue
			}
		}
		byName[u.NamespacedName.Name] = accepted
	}

	if accepted, ok := byName["resolved"]; !ok || !accepted {
		t.Errorf("expected \"resolved\" to be written Accepted=true, got present=%v accepted=%v", ok, accepted)
	}
	if accepted, ok := byName["still-bad"]; !ok || accepted {
		t.Errorf("expected \"still-bad\" to be written Accepted=false, got present=%v accepted=%v", ok, accepted)
	}
	if accepted, ok := byName["newly-bad"]; !ok || accepted {
		t.Errorf("expected \"newly-bad\" to be written Accepted=false, got present=%v accepted=%v", ok, accepted)
	}
	if _, ok := byName["resolved"]; len(byName) != 3 || !ok {
		t.Errorf("expected exactly 3 writes (resolved, still-bad, newly-bad), got %v", byName)
	}

	if len(d.resourceFailures) != 2 {
		t.Errorf("d.resourceFailures should be replaced with newFailures, got %v", d.resourceFailures)
	}
}
