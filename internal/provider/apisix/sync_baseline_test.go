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
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	adcclient "github.com/apache/apisix-ingress-controller/internal/adc/client"
)

// The ADC diff baseline lives on apisixProvider now: it decides when ADC's cached view of
// a data plane cannot be trusted (BypassCache), retries a stale-conf_version rejection
// once against a rebuilt baseline, and records which cacheKeys this leadership term has
// already rebuilt. The adc client only sends one request and parses the reply. These
// exercise that decision through the real HTTP path.

type adcResp struct {
	status int
	body   any
}

func respOK() adcResp {
	return adcResp{status: http.StatusOK, body: adctypes.SyncResult{Status: adctypes.StatusSuccess}}
}

func respRejected(reason string) adcResp {
	return adcResp{
		status: http.StatusUnprocessableEntity,
		body: adctypes.SyncResult{
			Status: "all_failed",
			Failed: []adctypes.SyncStatus{{Reason: reason}},
		},
	}
}

func respConfVersionRejected() adcResp {
	return respRejected("upstreams_conf_version must be greater than or equal to (1779434128737)")
}

// scriptedADC stands up a mock ADC server that answers each request with the next
// response in the script (repeating the last once the script runs out), and returns a
// snapshot func for the requests it received.
func scriptedADC(t *testing.T, script ...adcResp) func() []adcclient.ADCServerRequest {
	t.Helper()
	var mu sync.Mutex
	var got []adcclient.ADCServerRequest
	withMockADCServer(t, func(w http.ResponseWriter, r *http.Request) {
		var req adcclient.ADCServerRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		mu.Lock()
		i := len(got)
		got = append(got, req)
		mu.Unlock()
		resp := script[min(i, len(script)-1)]
		w.WriteHeader(resp.status)
		if resp.body != nil {
			_ = json.NewEncoder(w).Encode(resp.body)
		}
	})
	return func() []adcclient.ADCServerRequest {
		mu.Lock()
		defer mu.Unlock()
		return append([]adcclient.ADCServerRequest(nil), got...)
	}
}

func standaloneInput() adcclient.SyncInput {
	return adcclient.SyncInput{
		Name: "proxy",
		Config: adctypes.Config{
			Name:        "proxy",
			BackendType: adcclient.BackendAPISIXStandalone,
			ServerAddrs: []string{"http://apisix:9180"},
		},
		Resources: &adctypes.Resources{},
	}
}

func bypassSeq(reqs []adcclient.ADCServerRequest) []bool {
	seq := make([]bool, len(reqs))
	for i, req := range reqs {
		seq[i] = req.Task.Opts.BypassCache
	}
	return seq
}

func TestPushRebuildsBaselineOncePerTermThenReusesIt(t *testing.T) {
	reqs := scriptedADC(t, respOK())
	d := newTestProvider(t)
	in := standaloneInput()

	require.Empty(t, d.pushConfig(context.Background(), in).Errors)
	require.Empty(t, d.pushConfig(context.Background(), in).Errors)
	d.standaloneSyncer.InvalidateBaselines()
	require.Empty(t, d.pushConfig(context.Background(), in).Errors)

	assert.Equal(t, []bool{true, false, true}, bypassSeq(reqs()),
		"the first push of a term rebuilds the baseline, later ones reuse it, a new term rebuilds again")
}

func TestPushRebuildsAgainWhenTheRebuildWasNotAccepted(t *testing.T) {
	// Nothing proves the baseline current except ADC accepting the push that rebuilt it.
	reqs := scriptedADC(t, respRejected("connection refused"), respOK())
	d := newTestProvider(t)
	in := standaloneInput()

	require.NotEmpty(t, d.pushConfig(context.Background(), in).Errors)
	require.Empty(t, d.pushConfig(context.Background(), in).Errors)

	assert.Equal(t, []bool{true, true}, bypassSeq(reqs()))
}

func TestPushRebuildsBaselineWhenTheDataPlaneRejectsAStaleConfVersion(t *testing.T) {
	reqs := scriptedADC(t, respOK(), respConfVersionRejected(), respOK())
	d := newTestProvider(t)
	in := standaloneInput()

	require.Empty(t, d.pushConfig(context.Background(), in).Errors) // settles the baseline
	require.Empty(t, d.pushConfig(context.Background(), in).Errors) // rejected, then retried with a rebuild

	assert.Equal(t, []bool{true, false, true}, bypassSeq(reqs()))
	assert.False(t, in.Config.BypassCache,
		"the rebuild must not write BypassCache back into the caller's input")
}

func TestPushDoesNotRebuildOnUnrelatedFailures(t *testing.T) {
	// Re-deriving the baseline answers a stale conf_version and nothing else. An
	// unreachable data plane, or one refusing the configuration on its merits, is not a
	// question a rebuild can answer.
	for name, reason := range map[string]string{
		"unreachable":     "connection refused",
		"invalid plugins": `failed to check the configuration of plugin limit-count: value should match only one schema`,
	} {
		t.Run(name, func(t *testing.T) {
			reqs := scriptedADC(t, respOK(), respRejected(reason))
			d := newTestProvider(t)
			in := standaloneInput()

			require.Empty(t, d.pushConfig(context.Background(), in).Errors)
			require.NotEmpty(t, d.pushConfig(context.Background(), in).Errors)

			assert.Equal(t, []bool{true, false}, bypassSeq(reqs()))
		})
	}
}

func TestPushDoesNotRebuildOutsideStandalone(t *testing.T) {
	// conf_version, and the whole notion of a version the data plane can refuse, only
	// exists in standalone mode.
	reqs := scriptedADC(t, respConfVersionRejected())
	d := newTestProvider(t)
	in := standaloneInput()
	in.Config.BackendType = "apisix"

	require.NotEmpty(t, d.pushConfig(context.Background(), in).Errors)

	assert.Equal(t, []bool{false}, bypassSeq(reqs()))
}

func TestPushSurfacesBothReasonsWhenTheRebuildAlsoFails(t *testing.T) {
	// An ADC server older than 0.27.0 answers the rebuild with a schema error, which on
	// its own points nowhere near the cause. The rejection that triggered it must stay.
	reqs := scriptedADC(t, respOK(), respConfVersionRejected(), respRejected(`unrecognized key "bypassCache"`))
	d := newTestProvider(t)
	in := standaloneInput()

	require.Empty(t, d.pushConfig(context.Background(), in).Errors)
	execErrs := d.pushConfig(context.Background(), in)

	require.NotEmpty(t, execErrs.Errors)
	msg := execErrs.Error()
	assert.Contains(t, msg, "conf_version must be greater than or equal to")
	assert.Contains(t, msg, `unrecognized key "bypassCache"`)
	assert.Len(t, reqs(), 3, "the rebuild is attempted once, not in a loop")
}

func TestPushDoesNotReportTheSameRejectionTwice(t *testing.T) {
	// Someone else keeps writing to this data plane, so the rebuilt baseline is stale
	// again by the time it is pushed. Reporting that one rejection twice only pads status.
	scriptedADC(t, respOK(), respConfVersionRejected(), respConfVersionRejected())
	d := newTestProvider(t)
	in := standaloneInput()

	require.Empty(t, d.pushConfig(context.Background(), in).Errors)
	execErrs := d.pushConfig(context.Background(), in)

	assert.Len(t, execErrs.Errors, 1)
}

func TestPushRebuildsHoweverTheRejectionIsWorded(t *testing.T) {
	// The rejection is recognised by the field it names, not the sentence around it.
	reqs := scriptedADC(t, respOK(), respRejected("upstreams_conf_version has moved backwards"), respOK())
	d := newTestProvider(t)
	in := standaloneInput()

	require.Empty(t, d.pushConfig(context.Background(), in).Errors)
	require.Empty(t, d.pushConfig(context.Background(), in).Errors)

	assert.Equal(t, []bool{true, false, true}, bypassSeq(reqs()))
}
