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
	"sync"

	"github.com/go-logr/logr"

	pkgmetrics "github.com/apache/apisix-ingress-controller/pkg/metrics"
)

// StandaloneSyncer drives apisix-standalone's diff-baseline recovery on top of the
// one-shot Client. It is only for apisix-standalone: no other backend type keeps a
// conf_version, so no other backend type needs any of this.
//
// APISIX standalone keeps a monotonic conf_version per resource type and refuses a whole
// push whose version is behind the data plane's. ADC diffs against a cached baseline to
// build that push, and the baseline can go stale two ways:
//
//   - Across a leadership change. The ADC server is a sidecar that outlives the manager
//     container, so what it holds for a cacheKey can be the snapshot this pod left behind
//     in an earlier term, while the leader in between moved the data plane's conf_version
//     past it. InvalidateBaselines on leader acquisition forces the first push of every
//     cacheKey this term to re-derive its baseline from the data plane.
//   - Within a term, from a desync no leadership change explains, e.g. another writer on
//     the same data plane. A conf_version the data plane refuses is the only way that
//     shows itself; Sync answers it with one rebuild-and-retry.
type StandaloneSyncer struct {
	client *Client
	log    logr.Logger

	mu      sync.Mutex
	rebuilt map[string]struct{}
}

func NewStandaloneSyncer(c *Client, log logr.Logger) *StandaloneSyncer {
	return &StandaloneSyncer{
		client:  c,
		log:     log.WithName("standalone-syncer"),
		rebuilt: make(map[string]struct{}),
	}
}

// InvalidateBaselines forgets every rebuilt-this-term record, so the next push of each
// cacheKey re-derives its baseline. Call on leader acquisition.
func (s *StandaloneSyncer) InvalidateBaselines() {
	s.mu.Lock()
	defer s.mu.Unlock()
	clear(s.rebuilt)
}

func (s *StandaloneSyncer) isCurrent(cacheKey string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.rebuilt[cacheKey]
	return ok
}

func (s *StandaloneSyncer) markCurrent(cacheKey string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rebuilt[cacheKey] = struct{}{}
}

// Sync pushes in, rebuilding ADC's baseline first if this term has not yet pushed this
// cacheKey, and once more if the data plane rejects the push over a stale conf_version.
//
// It returns every error worth reporting, so 0, 1, or 2 of them: the final failure, plus
// the conf_version rejection that triggered a rebuild which then failed for a different
// reason (on its own that failure points nowhere near its cause, e.g. an ADC server too
// old to know bypassCache answers with a schema error). BypassCache is scoped to the call
// that recovers from a rejection and never written back into in.
func (s *StandaloneSyncer) Sync(ctx context.Context, in SyncInput) []error {
	in.Config.BypassCache = !s.isCurrent(in.Name)
	err := s.client.Sync(ctx, in)

	var report []error
	if !in.Config.BypassCache && IsConfVersionRejection(err) {
		s.log.Info("data plane rejected a stale conf_version, rebuilding the ADC baseline",
			"config", in.Name, "error", err.Error())
		// The rebuild is not rate limited, so a rejection on every push (someone else
		// writing to this data plane) turns every push into a full fetch and diff, and
		// this counter is what says so.
		pkgmetrics.RecordExecutionError(in.Name, "conf_version_conflict")

		rejection := err
		in.Config.BypassCache = true
		err = s.client.Sync(ctx, in)

		if err != nil && err.Error() != rejection.Error() {
			report = append(report, rejection)
		}
	}

	// Only a push ADC accepted proves its baseline is now derived from the data plane.
	if err == nil && in.Config.BypassCache {
		s.markCurrent(in.Name)
	}
	if err != nil {
		report = append(report, err)
	}
	return report
}
