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
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/version"
)

// testBackoff keeps the API server wait tests fast.
var testBackoff = wait.Backoff{Duration: time.Millisecond, Factor: 1, Steps: 10}

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

// TestDiscoverWithRetry_ReportsIndeterminateFailure is the core regression test
// for #2734: a discovery failure that gives no definitive answer must surface as
// an error so the caller aborts startup, instead of collapsing into "resource
// absent" and permanently skipping the field index and its controller.
func TestDiscoverWithRetry_ReportsIndeterminateFailure(t *testing.T) {
	var calls int
	probe := func() error {
		calls++
		return apierrors.NewServiceUnavailable("apiserver not ready")
	}

	var slept int
	err := discoverWithRetry(probe, logr.Discard(), func(time.Duration) { slept++ })
	require.Error(t, err, "an unreachable API server must not be reported as a definitive answer")
	assert.Equal(t, discoveryRetrySteps+1, calls)
	assert.Equal(t, discoveryRetrySteps, slept)
}

func TestDiscoverWithRetry_TransientThenSuccess(t *testing.T) {
	var calls int
	probe := func() error {
		calls++
		if calls < 3 {
			return apierrors.NewServiceUnavailable("apiserver not ready")
		}
		return nil
	}

	var slept int
	require.NoError(t, discoverWithRetry(probe, logr.Discard(), func(time.Duration) { slept++ }))
	assert.Equal(t, 3, calls, "an indeterminate failure must be retried, not given up on")
	assert.Equal(t, 2, slept, "backoff must be applied between retries")
}

// TestDiscoverWithRetry_DefinitiveAbsent ensures a genuinely absent CRD is
// answered immediately without retrying, preserving optional-CRD support.
func TestDiscoverWithRetry_DefinitiveAbsent(t *testing.T) {
	gr := schema.GroupResource{Group: "gateway.networking.k8s.io", Resource: "tcproutes"}
	var calls int
	probe := func() error {
		calls++
		return apierrors.NewNotFound(gr, "tcproutes")
	}

	var slept int
	err := discoverWithRetry(probe, logr.Discard(), func(time.Duration) { slept++ })
	assert.True(t, isResourceAbsent(err))
	assert.Equal(t, 1, calls, "a definitive NotFound must not be retried")
	assert.Equal(t, 0, slept)
}

func TestWaitForAPIServer_RetriesUntilReachable(t *testing.T) {
	var calls int
	probe := func() (*version.Info, error) {
		calls++
		if calls < 3 {
			return nil, &net.OpError{Op: "dial", Err: errors.New("connection refused")}
		}
		return &version.Info{GitVersion: "v1.31.0"}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	require.NoError(t, waitForAPIServer(ctx, probe, testBackoff, logr.Discard()))
	assert.Equal(t, 3, calls)
}

// TestWaitForAPIServer_HonorsContext ensures the wait is interruptible, so a
// shutdown signal is not ignored while the API server is unreachable.
func TestWaitForAPIServer_HonorsContext(t *testing.T) {
	probe := func() (*version.Info, error) {
		return nil, &net.OpError{Op: "dial", Err: errors.New("connection refused")}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := waitForAPIServer(ctx, probe, testBackoff, logr.Discard())
	require.Error(t, err)
}
