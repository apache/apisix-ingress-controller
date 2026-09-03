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
	"testing"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	apiv2 "github.com/apache/apisix-ingress-controller/api/v2"
	"github.com/apache/apisix-ingress-controller/internal/types"
)

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

func TestMarkGatewayProxyDataPlaneUnavailableTargetsTheConfigNameItself(t *testing.T) {
	d := &apisixProvider{log: logr.Discard()}
	statusUpdateMap := map[types.NamespacedNameKind][]string{}

	d.markGatewayProxyDataPlaneUnavailable("GatewayProxy/ns/gp", "boom", statusUpdateMap)

	want := types.NamespacedNameKind{Kind: types.KindGatewayProxy, Namespace: "ns", Name: "gp"}
	if got := statusUpdateMap[want]; len(got) != 1 || got[0] != "boom" {
		t.Errorf("statusUpdateMap[%v] = %v, want [boom]", want, got)
	}
	if len(statusUpdateMap) != 1 {
		t.Errorf("expected only the GatewayProxy itself to be marked, got %v", statusUpdateMap)
	}
}

func TestMarkGatewayProxyDataPlaneUnavailableIgnoresAnUnparseableConfigName(t *testing.T) {
	d := &apisixProvider{log: logr.Discard()}
	statusUpdateMap := map[types.NamespacedNameKind][]string{}

	d.markGatewayProxyDataPlaneUnavailable("not-a-valid-key", "boom", statusUpdateMap)

	if len(statusUpdateMap) != 0 {
		t.Errorf("expected nothing marked for an unparseable configName, got %v", statusUpdateMap)
	}
}
