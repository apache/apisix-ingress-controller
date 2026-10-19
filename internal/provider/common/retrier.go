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

package common

import (
	"sync"
	"time"
)

type Backoff interface {
	Next() time.Duration
	Reset()
}

type ExponentialBackoff struct {
	base, max, current time.Duration
}

func NewExponentialBackoff(base, max time.Duration) *ExponentialBackoff {
	return &ExponentialBackoff{base: base, max: max, current: base}
}

func (b *ExponentialBackoff) Next() time.Duration {
	delay := b.current
	b.current *= 2
	if b.current > b.max {
		b.current = b.max
	}
	return delay
}

func (b *ExponentialBackoff) Reset() {
	b.current = b.base
}

type Retrier struct {
	mu      sync.Mutex
	ch      chan struct{}
	timer   *time.Timer
	backoff Backoff
}

func NewRetrier(b Backoff) *Retrier {
	return &Retrier{
		ch:      make(chan struct{}, 1),
		backoff: b,
	}
}

func (r *Retrier) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.timer != nil {
		r.timer.Stop()
		r.timer = nil
	}
	select {
	case <-r.ch:
	default:
	}
	r.backoff.Reset()
}

func (r *Retrier) Next() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Event-triggered failures must not postpone an already scheduled retry.
	if r.timer != nil || len(r.ch) > 0 {
		return
	}

	delay := r.backoff.Next()
	var timer *time.Timer
	timer = time.AfterFunc(delay, func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		// Reset may have canceled this callback while it waited for the lock.
		if r.timer != timer {
			return
		}
		r.timer = nil
		select {
		case r.ch <- struct{}{}:
		default:
		}
	})
	r.timer = timer
}

func (r *Retrier) C() <-chan struct{} {
	return r.ch
}
