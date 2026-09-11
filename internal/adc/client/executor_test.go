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

package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/internal/types"
)

func TestHTTPADCExecutorBuildHTTPRequestBypassCache(t *testing.T) {
	e := &HTTPADCExecutor{
		serverURL: "http://127.0.0.1:3000",
		log:       logr.Discard(),
	}

	build := func(config adctypes.Config, path string) (ADCServerOpts, string) {
		req, err := e.buildHTTPRequest(context.Background(), "http://apisix:9180", config, nil, nil,
			&adctypes.Resources{}, path)
		require.NoError(t, err)
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		var parsed ADCServerRequest
		require.NoError(t, json.Unmarshal(body, &parsed))
		return parsed.Task.Opts, string(body)
	}

	config := adctypes.Config{Name: "GatewayProxy/ns/name"}

	// A sync that is not recovering from a rejection must stay byte for byte the request an
	// ADC server older than 0.27.0 (which rejects unknown fields) already accepts.
	opts, raw := build(config, pathSync)
	assert.Equal(t, "GatewayProxy/ns/name", opts.CacheKey)
	assert.NotContains(t, raw, "bypassCache")

	config.BypassCache = true
	opts, raw = build(config, pathSync)
	assert.Equal(t, "GatewayProxy/ns/name", opts.CacheKey, "the cacheKey itself must stay stable")
	assert.True(t, opts.BypassCache)
	assert.Contains(t, raw, "bypassCache")

	// The ADC validate task schema rejects unknown fields too, so bypassCache must never
	// reach it, even when the config carries the flag.
	_, raw = build(config, pathValidate)
	assert.NotContains(t, raw, "bypassCache")
}

func TestHTTPADCExecutorBuildHTTPRequestCaCert(t *testing.T) {
	e := &HTTPADCExecutor{
		serverURL: "http://127.0.0.1:3000",
		log:       logr.Discard(),
	}

	build := func(config adctypes.Config) (ADCServerOpts, string) {
		req, err := e.buildHTTPRequest(context.Background(), "https://apisix:9180", config, nil, nil,
			&adctypes.Resources{}, pathSync)
		require.NoError(t, err)
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		var parsed ADCServerRequest
		require.NoError(t, json.Unmarshal(body, &parsed))
		return parsed.Task.Opts, string(body)
	}

	// Without a CA bundle the request stays what an ADC server that predates caCert
	// already accepts.
	opts, raw := build(adctypes.Config{Name: "GatewayProxy/ns/name", TlsVerify: true})
	assert.Empty(t, opts.CaCert)
	assert.NotContains(t, raw, "caCert")

	const caCert = "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----"
	opts, raw = build(adctypes.Config{Name: "GatewayProxy/ns/name", TlsVerify: true, CaCert: caCert})
	assert.Equal(t, caCert, opts.CaCert)
	assert.Contains(t, raw, "caCert")
	// verification stays on, otherwise the bundle would be pointless
	assert.Equal(t, false, *opts.TlsSkipVerify)
}

// confVersionError is what a push carrying a conf_version older than the data plane's
// comes back as, once the ADC server has relayed the rejection to us.
func confVersionError() error {
	return rejection("upstreams_conf_version must be greater than or equal to (1779434128737)")
}

func rejection(reason string) error {
	return types.ADCExecutionError{
		Name: "GatewayProxy/ns/name",
		FailedErrors: []types.ADCExecutionServerAddrError{{
			ServerAddr: "http://apisix:9180",
			Err:        reason,
		}},
	}
}

func httpResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))}
}

func TestHandleHTTPResponseParsesStructuredReasonOn422(t *testing.T) {
	// apisix-standalone's /sync answers an all-rejected document with a 422 carrying the same
	// structured `failed` shape a 2xx partial failure uses. That detail must survive, not
	// collapse into a generic "HTTP 422: <raw body>" indistinguishable from a 5xx.
	e := &HTTPADCExecutor{log: logr.Discard()}
	body, err := json.Marshal(adctypes.SyncResult{
		Status:      "all_failed",
		FailedCount: 1,
		Failed:      []adctypes.SyncStatus{{Reason: "unknown plugin foo"}},
	})
	require.NoError(t, err)

	err = e.handleHTTPResponse(httpResponse(http.StatusUnprocessableEntity, string(body)), "http://apisix:9180")

	var addrErr types.ADCExecutionServerAddrError
	require.ErrorAs(t, err, &addrErr)
	assert.Equal(t, "unknown plugin foo", addrErr.Err)
	require.Len(t, addrErr.FailedStatuses, 1)
	assert.Equal(t, "unknown plugin foo", addrErr.FailedStatuses[0].Reason)
}

func TestHandleHTTPResponseErrorsOnAnUnparseableBodyForAStatusThatShouldCarryOne(t *testing.T) {
	// Nothing actually sends a 2xx/422 body that isn't a SyncResult, but the parse must
	// still fail loudly rather than silently reading as success.
	e := &HTTPADCExecutor{log: logr.Discard()}

	err := e.handleHTTPResponse(httpResponse(http.StatusUnprocessableEntity, "not json"), "http://apisix:9180")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal response body")
}

func TestHandleHTTPResponse400NeverAttemptsToParseTheBody(t *testing.T) {
	// 400 never carries a SyncResult body (that's 422's job); must stay a raw error even
	// when the body happens to look like one.
	e := &HTTPADCExecutor{log: logr.Discard()}
	body, err := json.Marshal(adctypes.SyncResult{
		Status:      "all_failed",
		FailedCount: 1,
		Failed:      []adctypes.SyncStatus{{Reason: "unknown plugin foo"}},
	})
	require.NoError(t, err)

	err = e.handleHTTPResponse(httpResponse(http.StatusBadRequest, string(body)), "http://apisix:9180")

	var addrErr types.ADCExecutionServerAddrError
	require.ErrorAs(t, err, &addrErr)
	assert.Contains(t, addrErr.Err, "HTTP 400")
	assert.Nil(t, addrErr.FailedStatuses)
}

func TestHandleHTTPResponseTreats5xxAsGenericTransportFailure(t *testing.T) {
	// A 5xx never carries the /sync response shape at all. Unlike 400, it must not be
	// parsed as one, and stays the plain "HTTP <code>: <body>" it always was.
	e := &HTTPADCExecutor{log: logr.Discard()}

	err := e.handleHTTPResponse(httpResponse(http.StatusBadGateway, "upstream connect error"), "http://apisix:9180")

	var addrErr types.ADCExecutionServerAddrError
	require.ErrorAs(t, err, &addrErr)
	assert.Contains(t, addrErr.Err, "HTTP 502")
	assert.Nil(t, addrErr.FailedStatuses)
}

func TestHandleHTTPResponseNeverParses500AsASyncResult(t *testing.T) {
	// A 500's body is {"message": ...}. Most SyncResult fields are optional, so this would
	// unmarshal without error into an empty, unremarkable "success" if it were ever tried.
	e := &HTTPADCExecutor{log: logr.Discard()}

	err := e.handleHTTPResponse(httpResponse(http.StatusInternalServerError, `{"message": "boom"}`), "http://apisix:9180")

	var addrErr types.ADCExecutionServerAddrError
	require.ErrorAs(t, err, &addrErr)
	assert.Contains(t, addrErr.Err, "HTTP 500")
	assert.Nil(t, addrErr.FailedStatuses)
}

func TestHandleHTTPResponse413HasNoBodyToParse(t *testing.T) {
	e := &HTTPADCExecutor{log: logr.Discard()}

	err := e.handleHTTPResponse(httpResponse(http.StatusRequestEntityTooLarge, ""), "http://apisix:9180")

	var addrErr types.ADCExecutionServerAddrError
	require.ErrorAs(t, err, &addrErr)
	assert.Contains(t, addrErr.Err, "HTTP 413")
}

func TestHandleHTTPResponse422WithNeitherFailedNorEndpointStatusIsTreatedAsSuccess(t *testing.T) {
	// This combination shouldn't happen; a genuine rejection always shows up in one of
	// them. So it isn't specially guarded against, it just falls through as success.
	e := &HTTPADCExecutor{log: logr.Discard()}
	body, err := json.Marshal(adctypes.SyncResult{Status: "all_failed"})
	require.NoError(t, err)

	err = e.handleHTTPResponse(httpResponse(http.StatusUnprocessableEntity, string(body)), "http://apisix:9180")

	assert.NoError(t, err)
}

func TestHandleHTTPResponse422FallsBackToEndpointStatusWhenNothingWasAttributed(t *testing.T) {
	// An old gateway with no /validate endpoint (or a conf_version race) leaves Failed
	// empty even on a genuine all-server rejection. EndpointStatus is then the only place
	// a reason shows up, and it must not be ignored just because the status is 422.
	e := &HTTPADCExecutor{log: logr.Discard()}
	body, err := json.Marshal(adctypes.SyncResult{
		Status: "all_failed",
		EndpointStatus: []adctypes.EndpointStatus{
			{Server: "http://apisix-1:9180", Success: false, Reason: "unknown plugin foo"},
			{Server: "http://apisix-2:9180", Success: false, Reason: "unknown plugin foo"},
		},
	})
	require.NoError(t, err)

	err = e.handleHTTPResponse(httpResponse(http.StatusUnprocessableEntity, string(body)), "http://apisix:9180")

	var addrErr types.ADCExecutionServerAddrError
	require.ErrorAs(t, err, &addrErr)
	assert.Contains(t, addrErr.Err, "unknown plugin foo")
	assert.NotContains(t, addrErr.Err, "HTTP 422", "should report the real per-server reason, not the generic fallback")
	assert.Nil(t, addrErr.FailedStatuses, "nothing was attributed to a resource")
	require.Len(t, addrErr.EndpointStatuses, 2)
}

func TestHandleHTTPResponseReportsAPerServerFailureOnA2xxWithNoFailedCount(t *testing.T) {
	// apisix-standalone: one server rejecting the write while others accept it doesn't fail
	// the document (every event still lands in Success, FailedCount stays 0). This must
	// still surface as an error, not silently succeed because FailedCount says nothing failed.
	e := &HTTPADCExecutor{log: logr.Discard()}
	body, err := json.Marshal(adctypes.SyncResult{
		Status:      "partial_failure",
		FailedCount: 0,
		EndpointStatus: []adctypes.EndpointStatus{
			{Server: "http://apisix-1:9180", Success: true, Confirmation: adctypes.ConfirmationApplied},
			{Server: "http://apisix-2:9180", Success: false, Reason: "connection refused"},
		},
	})
	require.NoError(t, err)

	err = e.handleHTTPResponse(httpResponse(http.StatusAccepted, string(body)), "http://apisix:9180")

	var addrErr types.ADCExecutionServerAddrError
	require.ErrorAs(t, err, &addrErr)
	assert.Contains(t, addrErr.Err, "http://apisix-2:9180: connection refused")
	require.Len(t, addrErr.EndpointStatuses, 2)
}

func TestHandleHTTPResponse422CarriesBothFailedAndEndpointStatusesButErrPicksFailed(t *testing.T) {
	// A real 422 always has EndpointStatus populated too, restating the same rejection per
	// server. Err (the human-readable text) picks Failed's specific reason rather than
	// repeating it, but both structured fields still travel: Failed feeds bad-resource
	// exclusion, EndpointStatuses feeds per-instance Gateway/CRD availability, and neither
	// consumer should have to reconstruct its input from the other's.
	e := &HTTPADCExecutor{log: logr.Discard()}
	body, err := json.Marshal(adctypes.SyncResult{
		Status:      "all_failed",
		FailedCount: 1,
		Failed:      []adctypes.SyncStatus{{Reason: "unknown plugin foo"}},
		EndpointStatus: []adctypes.EndpointStatus{
			{Server: "http://apisix-1:9180", Success: false, Reason: "unknown plugin foo"},
			{Server: "http://apisix-2:9180", Success: false, Reason: "unknown plugin foo"},
		},
	})
	require.NoError(t, err)

	err = e.handleHTTPResponse(httpResponse(http.StatusUnprocessableEntity, string(body)), "http://apisix:9180")

	var addrErr types.ADCExecutionServerAddrError
	require.ErrorAs(t, err, &addrErr)
	assert.Equal(t, "unknown plugin foo", addrErr.Err)
	require.Len(t, addrErr.FailedStatuses, 1)
	require.Len(t, addrErr.EndpointStatuses, 2)
}

func TestRunHTTPSyncFailsForAStandaloneConfigWithNoAddress(t *testing.T) {
	// A GatewayProxy resolving to no data plane address (scaled to zero, say) must sync
	// fail for apisix-standalone, which pushes the whole config to one destination, not
	// pass as a no-op.
	e := &HTTPADCExecutor{log: logr.Discard()}

	statusCode, err := e.runHTTPSync(context.Background(),
		adctypes.Config{Name: "gw", BackendType: BackendAPISIXStandalone}, &adctypes.Resources{}, nil, nil)

	var addrErr types.ADCExecutionServerAddrError
	require.ErrorAs(t, err, &addrErr)
	assert.Contains(t, addrErr.Err, "no data plane address")
	assert.Zero(t, statusCode, "no HTTP response was ever involved")
}

func TestRunHTTPSyncNoOpsForANonStandaloneConfigWithNoAddress(t *testing.T) {
	// Every other backend type pushes per address, so no address is nothing to push.
	e := &HTTPADCExecutor{log: logr.Discard()}

	statusCode, err := e.runHTTPSync(context.Background(),
		adctypes.Config{Name: "gw", BackendType: "apisix"}, &adctypes.Resources{}, nil, nil)

	assert.NoError(t, err)
	assert.Zero(t, statusCode)
}

func TestDistinctReasonsJoinsWithoutDuplicates(t *testing.T) {
	reasons := distinctReasons([]adctypes.SyncStatus{
		{Reason: "unknown plugin foo"},
		{Reason: "connection refused"},
		{Reason: "unknown plugin foo"},
		{Reason: ""},
	})
	assert.Equal(t, "unknown plugin foo; connection refused", reasons)
}

func TestHandleHTTPResponseAcceptedIsNotAFailure(t *testing.T) {
	// "accepted" means apisix-standalone gave up waiting for the data plane to confirm the
	// write, not that the write failed. Turning that into a Gateway/CRD condition is a
	// separate concern from whether this call itself succeeded.
	e := &HTTPADCExecutor{log: logr.Discard()}
	body, err := json.Marshal(adctypes.SyncResult{
		Status: "success",
		EndpointStatus: []adctypes.EndpointStatus{
			{Server: "http://apisix:9180", Success: true, Confirmation: adctypes.ConfirmationAccepted},
		},
	})
	require.NoError(t, err)

	err = e.handleHTTPResponse(httpResponse(http.StatusAccepted, string(body)), "http://apisix:9180")

	assert.NoError(t, err)
}

func TestHandleHTTPResponseAppliedSucceeds(t *testing.T) {
	e := &HTTPADCExecutor{log: logr.Discard()}
	body, err := json.Marshal(adctypes.SyncResult{
		Status: "success",
		EndpointStatus: []adctypes.EndpointStatus{
			{Server: "http://apisix:9180", Success: true, Confirmation: adctypes.ConfirmationApplied},
		},
	})
	require.NoError(t, err)

	err = e.handleHTTPResponse(httpResponse(http.StatusOK, string(body)), "http://apisix:9180")

	assert.NoError(t, err)
}

func TestIsConfVersionRejection(t *testing.T) {
	assert.False(t, IsConfVersionRejection(nil))
	assert.False(t, IsConfVersionRejection(errors.New("context deadline exceeded")))
	assert.False(t, IsConfVersionRejection(rejection("connection refused")))
	assert.True(t, IsConfVersionRejection(confVersionError()))
	assert.True(t, IsConfVersionRejection(rejection("routes_conf_version has moved backwards")),
		"the field is what names the rejection, not the sentence")
}
