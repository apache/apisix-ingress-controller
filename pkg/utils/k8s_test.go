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

package utils

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
)

// fastBackoff keeps the tests quick while preserving the shape of the
// production backoffs (a growing interval over a fixed number of steps).
func fastBackoff(steps int) wait.Backoff {
	return wait.Backoff{Duration: time.Millisecond, Factor: backoffFactor, Jitter: backoffJitter, Steps: steps}
}

// declaredBudget is how long a backoff sleeps in total before jitter, i.e. the
// sum of the intervals between its attempts.
func declaredBudget(backoff wait.Backoff) time.Duration {
	var total, interval time.Duration = 0, backoff.Duration
	for i := 0; i < backoff.Steps-1; i++ {
		total += interval
		interval = time.Duration(float64(interval) * backoff.Factor)
	}
	return total
}

// countingBackoff reports how many times a backoff actually runs its condition.
func countAttempts(backoff wait.Backoff) int {
	var attempts int
	_ = wait.ExponentialBackoff(backoff, func() (bool, error) {
		attempts++
		return false, nil
	})
	return attempts
}

// TestBackoffsUseAllTheirSteps guards against a subtle wait.Backoff behavior:
// setting Cap zeroes the remaining steps as soon as the capped interval is
// reached, silently cutting the retries short. Both backoffs must run every
// step they declare.
func TestBackoffsUseAllTheirSteps(t *testing.T) {
	for _, tt := range []struct {
		name    string
		backoff wait.Backoff
		steps   int
		budget  time.Duration // as documented next to the constants
	}{
		{"discovery", discoveryBackoff(), discoveryRetrySteps, 7500 * time.Millisecond},
		{"apiServerWait", apiServerWaitBackoff(), apiServerWaitSteps, 31 * time.Second},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Zero(t, tt.backoff.Cap, "Cap truncates the retries, see wait.delay()")
			assert.Equal(t, tt.budget, declaredBudget(tt.backoff), "the documented budget must match the backoff")

			b := tt.backoff
			b.Duration = time.Microsecond // keep the test fast, preserve the shape
			assert.Equal(t, tt.steps, countAttempts(b))
		})
	}
}

func TestIsResourceAbsent(t *testing.T) {
	gr := schema.GroupResource{Group: "gateway.networking.k8s.io", Resource: "tcproutes"}
	tests := []struct {
		name   string
		err    error
		absent bool
	}{
		// Definitive: the group/version is genuinely not served.
		{"not found", apierrors.NewNotFound(gr, "tcproutes"), true},
		// Definitive: RBAC will not self-heal by waiting.
		{"forbidden", apierrors.NewForbidden(gr, "tcproutes", errors.New("nope")), true},
		// Indeterminate: the API server is temporarily unreachable or overloaded.
		{"connection refused", &net.OpError{Op: "dial", Err: errors.New("connection refused")}, false},
		{"service unavailable", apierrors.NewServiceUnavailable("apiserver is starting up"), false},
		{"server timeout", apierrors.NewServerTimeout(gr, "get", 1), false},
		{"too many requests", apierrors.NewTooManyRequestsError("slow down"), false},
		{"internal error", apierrors.NewInternalError(errors.New("boom")), false},
		{"generic transport error", errors.New("EOF"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.absent, isResourceAbsent(tt.err))
		})
	}
}

// TestRetryUntilDefinitive_ReportsIndeterminateFailure is the core regression
// test for #2734: a discovery failure that gives no definitive answer must
// surface as an error so the caller aborts startup, instead of collapsing into
// "resource absent" and permanently skipping the field index and its controller.
func TestRetryUntilDefinitive_ReportsIndeterminateFailure(t *testing.T) {
	var calls int
	probe := func() error {
		calls++
		return apierrors.NewServiceUnavailable("apiserver not ready")
	}

	err := retryUntilDefinitive(fastBackoff(3), logr.Discard(), probe)
	require.Error(t, err, "an unreachable API server must not be reported as a definitive answer")
	assert.False(t, isResourceAbsent(err))
	assert.Equal(t, 3, calls, "the whole retry budget must be used")
}

func TestRetryUntilDefinitive_TransientThenSuccess(t *testing.T) {
	var calls int
	probe := func() error {
		calls++
		if calls < 3 {
			return apierrors.NewServiceUnavailable("apiserver not ready")
		}
		return nil
	}

	require.NoError(t, retryUntilDefinitive(fastBackoff(5), logr.Discard(), probe))
	assert.Equal(t, 3, calls, "an indeterminate failure must be retried, not given up on")
}

// TestRetryUntilDefinitive_DefinitiveAbsent ensures a genuinely absent CRD is
// answered immediately without retrying, preserving optional-CRD support.
func TestRetryUntilDefinitive_DefinitiveAbsent(t *testing.T) {
	gr := schema.GroupResource{Group: "gateway.networking.k8s.io", Resource: "tcproutes"}
	var calls int
	probe := func() error {
		calls++
		return apierrors.NewNotFound(gr, "tcproutes")
	}

	err := retryUntilDefinitive(fastBackoff(5), logr.Discard(), probe)
	assert.True(t, isResourceAbsent(err))
	assert.Equal(t, 1, calls, "a definitive NotFound must not be retried")
}

// TestRetryUntilDefinitive_NeverProbed guards the caller's assumption that a
// nil error means "the probe succeeded": a backoff with no steps left never runs
// the condition, and reporting nil there would be read as "resource present".
func TestRetryUntilDefinitive_NeverProbed(t *testing.T) {
	var calls int
	probe := func() error {
		calls++
		return nil
	}

	err := retryUntilDefinitive(wait.Backoff{Steps: 0}, logr.Discard(), probe)
	require.Error(t, err)
	assert.Zero(t, calls)
	assert.False(t, isResourceAbsent(err), "an unattempted probe must not resolve to absent either")
}

func TestWaitForAPIServer_RetriesUntilReachable(t *testing.T) {
	var calls int
	probe := func() error {
		calls++
		if calls < 3 {
			return &net.OpError{Op: "dial", Err: errors.New("connection refused")}
		}
		return nil
	}

	require.NoError(t, waitForAPIServer(context.Background(), probe, fastBackoff(6), logr.Discard()))
	assert.Equal(t, 3, calls)
}

// TestWaitForAPIServer_AuthRejectionIsReachable covers a hardened cluster that
// does not grant the controller access to the probed endpoint: the API server
// answered, so startup must proceed instead of stalling until the budget runs
// out and then failing on a perfectly healthy cluster.
func TestWaitForAPIServer_AuthRejectionIsReachable(t *testing.T) {
	for name, probeErr := range map[string]error{
		"unauthorized": apierrors.NewUnauthorized("no token"),
		"forbidden":    apierrors.NewForbidden(schema.GroupResource{}, "", errors.New("nope")),
	} {
		t.Run(name, func(t *testing.T) {
			var calls int
			probe := func() error {
				calls++
				return probeErr
			}
			require.NoError(t, waitForAPIServer(context.Background(), probe, fastBackoff(5), logr.Discard()))
			assert.Equal(t, 1, calls, "an answer, even a rejection, must not be retried")
		})
	}
}

func TestWaitForAPIServer_GivesUpWithTheLastError(t *testing.T) {
	probe := func() error { return apierrors.NewServiceUnavailable("apiserver not ready") }

	err := waitForAPIServer(context.Background(), probe, fastBackoff(3), logr.Discard())
	require.Error(t, err)
	assert.True(t, apierrors.IsServiceUnavailable(errors.Unwrap(err)), "the last probe error must be reported, got %v", err)
}

// TestWaitForAPIServer_HonorsContext ensures the wait is interrupted mid-flight,
// so a shutdown signal is not ignored while the API server is unreachable.
func TestWaitForAPIServer_HonorsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var calls int
	probe := func() error {
		if calls++; calls == 2 {
			cancel()
		}
		return &net.OpError{Op: "dial", Err: errors.New("connection refused")}
	}

	// A budget far larger than the number of attempts the test expects: the wait
	// must end because ctx was cancelled, not because it ran out of steps.
	err := waitForAPIServer(ctx, probe, fastBackoff(100), logr.Discard())
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 2, calls, "the probe must stop being called once ctx is cancelled")
}

// newDiscoveryClient serves discovery responses from handler.
func newDiscoveryClient(t *testing.T, handler http.HandlerFunc) discovery.DiscoveryInterface {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c, err := discovery.NewDiscoveryClientForConfig(&rest.Config{Host: srv.URL})
	require.NoError(t, err)
	return c
}

// TestHasAPIResource covers the contract every caller depends on, against a real
// discovery client rather than a fake: only a definitive answer may resolve to
// "absent", everything else must be an error.
func TestHasAPIResource(t *testing.T) {
	gvk := schema.GroupVersionKind{Group: "gateway.networking.k8s.io", Version: "v1alpha2", Kind: "TCPRoute"}

	tests := []struct {
		name      string
		handler   http.HandlerFunc
		wantFound bool
		wantErr   bool
	}{
		{
			name: "kind served",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"kind":"APIResourceList","groupVersion":"gateway.networking.k8s.io/v1alpha2",` +
					`"resources":[{"name":"tcproutes","kind":"TCPRoute"}]}`))
			},
			wantFound: true,
		},
		{
			name: "group/version served but kind missing",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"kind":"APIResourceList","groupVersion":"gateway.networking.k8s.io/v1alpha2",` +
					`"resources":[{"name":"udproutes","kind":"UDPRoute"}]}`))
			},
		},
		{
			// What a real API server returns for a group/version that is not
			// installed: a plain-text 404, not a Status object.
			name: "group/version not installed",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				http.NotFound(w, nil)
			},
		},
		{
			// The #2734 scenario: the API server cannot answer. This must not be
			// reported as "the CRD is not installed".
			name: "api server unavailable",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
			},
			wantErr: true,
		},
		{
			name: "discovery forbidden",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusForbidden)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found, err := hasAPIResource(newDiscoveryClient(t, tt.handler), gvk, fastBackoff(2), logr.Discard())
			if tt.wantErr {
				require.Error(t, err)
				assert.False(t, found)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantFound, found)
		})
	}
}
