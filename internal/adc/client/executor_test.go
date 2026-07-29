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
			&adctypes.Resources{}, http.MethodPut, path)
		require.NoError(t, err)
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		var parsed ADCServerRequest
		require.NoError(t, json.Unmarshal(body, &parsed))
		return parsed.Task.Opts, string(body)
	}

	config := adctypes.Config{Name: "GatewayProxy/ns/name"}

	// A sync that is not recovering from a rejection must stay byte for byte the request an
	// ADC server older than 0.27.0 -- which rejects unknown fields -- already accepts.
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

// fakeExecutor answers each Execute call with the next error in errs, and records the
// BypassCache flag it was called with.
type fakeExecutor struct {
	errs      []error
	bypassSeq []bool
}

func (f *fakeExecutor) Execute(_ context.Context, config adctypes.Config, _ []string) error {
	f.bypassSeq = append(f.bypassSeq, config.BypassCache)
	if len(f.errs) == 0 {
		return nil
	}
	err := f.errs[0]
	f.errs = f.errs[1:]
	return err
}

func (f *fakeExecutor) Validate(context.Context, adctypes.Config, []string) error { return nil }

// newTestClient starts out as a controller that has just been elected: no ADC baseline is
// known to be current, so the first sync of a cacheKey rebuilds it.
func newTestClient(exec ADCExecutor) *Client {
	return &Client{
		executor:         exec,
		rebuiltBaselines: make(map[string]struct{}),
		log:              logr.Discard(),
	}
}

// afterFirstSync is the state a controller settles into once the first sync of its term
// has landed: the ADC baseline for this cacheKey is known to be derived from the data
// plane, so nothing rebuilds it again unless the data plane says otherwise.
func afterFirstSync(exec ADCExecutor) *Client {
	c := newTestClient(exec)
	c.markBaselineCurrent(syncTaskCacheKey)
	return c
}

const syncTaskCacheKey = "GatewayProxy/ns/name"

func newSyncTask() Task {
	return Task{
		Name: "GatewayProxy/ns/name-sync",
		Configs: map[types.NamespacedNameKind]adctypes.Config{
			{}: {Name: "GatewayProxy/ns/name", BackendType: "apisix-standalone"},
		},
		Resources: &adctypes.Resources{},
	}
}

func TestClientSyncRebuildsOnceAfterElectionThenReusesTheADCCache(t *testing.T) {
	exec := &fakeExecutor{}
	c := newTestClient(exec)

	// The sidecar may still hold a baseline from an earlier term, so the first sync of a
	// cacheKey re-derives it from the data plane. Once ADC has accepted that sync, its
	// baseline is current and later syncs diff against it.
	require.NoError(t, c.sync(context.Background(), newSyncTask()))
	require.NoError(t, c.sync(context.Background(), newSyncTask()))
	assert.Equal(t, []bool{true, false}, exec.bypassSeq)

	// Winning the election again puts every baseline back in doubt.
	c.InvalidateADCCache()
	require.NoError(t, c.sync(context.Background(), newSyncTask()))
	assert.Equal(t, []bool{true, false, true}, exec.bypassSeq)
}

func TestClientSyncRebuildsAgainWhenTheRebuildWasNotAccepted(t *testing.T) {
	// Nothing proves the baseline is current except ADC accepting the sync that rebuilt it.
	exec := &fakeExecutor{errs: []error{types.ADCExecutionError{
		Name:         "GatewayProxy/ns/name",
		FailedErrors: []types.ADCExecutionServerAddrError{{Err: "connection refused"}},
	}}}
	c := newTestClient(exec)

	require.Error(t, c.sync(context.Background(), newSyncTask()))
	require.NoError(t, c.sync(context.Background(), newSyncTask()))

	assert.Equal(t, []bool{true, true}, exec.bypassSeq)
}

func TestClientSyncRebuildsADCBaselineWhenTheDataPlaneRejectsThePush(t *testing.T) {
	exec := &fakeExecutor{errs: []error{confVersionError()}}
	c := afterFirstSync(exec)

	// The data plane holds a conf_version newer than the one the ADC baseline carries, so
	// the push is rejected. The retry rebuilds that baseline from the data plane.
	task := newSyncTask()
	require.NoError(t, c.sync(context.Background(), task))

	assert.Equal(t, []bool{false, true}, exec.bypassSeq)

	// BypassCache is scoped to the request that recovers from the rejection. Were it to
	// survive in the task, it would reach the config the ConfigManager holds and turn a
	// one-off rebuild into a data plane fetch on every later sync.
	assert.False(t, task.Configs[types.NamespacedNameKind{}].BypassCache,
		"the rebuild must not write BypassCache back into the task config")
}

func TestClientSyncDoesNotRebuildOnUnrelatedFailures(t *testing.T) {
	// Re-deriving the baseline answers a stale conf_version and nothing else. A data plane
	// that cannot be reached, or one that refuses the configuration on its merits, is not a
	// question the baseline can answer, and a rebuild would only cost a fetch.
	for name, err := range map[string]error{
		"unreachable":     rejection("connection refused"),
		"invalid plugins": rejection(`failed to check the configuration of plugin limit-count: value should match only one schema`),
	} {
		t.Run(name, func(t *testing.T) {
			exec := &fakeExecutor{errs: []error{err}}
			c := afterFirstSync(exec)

			require.Error(t, c.sync(context.Background(), newSyncTask()))

			assert.Equal(t, []bool{false}, exec.bypassSeq)
		})
	}
}

func TestClientSyncRebuildsHoweverTheRejectionIsWorded(t *testing.T) {
	// The rejection is recognised by the field it names, not by the sentence around it:
	// conf_version is part of the standalone admin API, the wording is APISIX's to change.
	exec := &fakeExecutor{errs: []error{rejection("upstreams_conf_version has moved backwards")}}
	c := afterFirstSync(exec)

	require.NoError(t, c.sync(context.Background(), newSyncTask()))

	assert.Equal(t, []bool{false, true}, exec.bypassSeq)
}

func TestClientSyncDoesNotRebuildOutsideStandalone(t *testing.T) {
	// conf_version, and the whole notion of a version the data plane can refuse, only
	// exists in standalone mode.
	exec := &fakeExecutor{errs: []error{confVersionError()}}
	c := afterFirstSync(exec)

	task := newSyncTask()
	task.Configs[types.NamespacedNameKind{}] = adctypes.Config{Name: "GatewayProxy/ns/name", BackendType: "apisix"}
	require.Error(t, c.sync(context.Background(), task))

	assert.Equal(t, []bool{false}, exec.bypassSeq)
}

func TestClientSyncSurfacesErrorWhenRebuildFails(t *testing.T) {
	// An ADC server older than 0.27.0 answers the rebuild with a schema error, which on
	// its own points nowhere near the cause.
	exec := &fakeExecutor{errs: []error{confVersionError(), rejection(`unrecognized key "bypassCache"`)}}
	c := afterFirstSync(exec)

	err := c.sync(context.Background(), newSyncTask())

	require.Error(t, err, "a rebuild that still fails must not be swallowed")
	assert.Equal(t, []bool{false, true}, exec.bypassSeq, "the rebuild is attempted once, not in a loop")
	assert.Contains(t, err.Error(), "conf_version must be greater than or equal to",
		"the rejection that triggered the rebuild must stay in the reported error")
	assert.Contains(t, err.Error(), `unrecognized key "bypassCache"`,
		"so must the reason the rebuild itself failed")
}

func TestClientSyncDoesNotReportTheSameRejectionTwice(t *testing.T) {
	// Someone else keeps writing to this data plane, so the rebuilt baseline is stale again
	// by the time it is pushed. Reporting that one rejection twice only pads the status.
	exec := &fakeExecutor{errs: []error{confVersionError(), confVersionError()}}
	c := afterFirstSync(exec)

	err := c.sync(context.Background(), newSyncTask())

	var execErrs types.ADCExecutionErrors
	require.ErrorAs(t, err, &execErrs)
	assert.Len(t, execErrs.Errors, 1)
}

func TestIsConfVersionRejection(t *testing.T) {
	assert.False(t, isConfVersionRejection(nil))
	assert.False(t, isConfVersionRejection(errors.New("context deadline exceeded")))
	assert.False(t, isConfVersionRejection(rejection("connection refused")))
	assert.True(t, isConfVersionRejection(confVersionError()))
	assert.True(t, isConfVersionRejection(rejection("routes_conf_version has moved backwards")),
		"the field is what names the rejection, not the sentence")
}
