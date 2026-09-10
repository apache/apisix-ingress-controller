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
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestKeyedMutexSerializesTheSameKey covers what makes syncConfigNow correct: two holders
// of the same key must never be inside their critical section at the same time.
func TestKeyedMutexSerializesTheSameKey(t *testing.T) {
	k := newKeyedMutex()
	var busy atomic.Bool

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			unlock := k.Lock("shared")
			defer unlock()
			if !busy.CompareAndSwap(false, true) {
				t.Error("another holder was already in the critical section")
				return
			}
			time.Sleep(5 * time.Millisecond)
			busy.Store(false)
		}()
	}
	wg.Wait()
}

// TestKeyedMutexDoesNotBlockDifferentKeys covers the other half: a busy key must not stall
// callers working on an unrelated one, or an immediate delete push for one GatewayProxy
// would wait behind a slow periodic sync of a completely different one.
func TestKeyedMutexDoesNotBlockDifferentKeys(t *testing.T) {
	k := newKeyedMutex()

	unlockOne := k.Lock("one")
	defer unlockOne()

	done := make(chan struct{})
	go func() {
		unlockTwo := k.Lock("two")
		defer unlockTwo()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("locking a different key blocked behind an unrelated key's holder")
	}
}

// TestKeyedMutexUnlockReleasesTheKey covers that the returned func actually frees the key
// for the next caller, not just for the same goroutine that locked it.
func TestKeyedMutexUnlockReleasesTheKey(t *testing.T) {
	k := newKeyedMutex()

	unlock := k.Lock("x")
	unlock()

	done := make(chan struct{})
	go func() {
		unlockAgain := k.Lock("x")
		defer unlockAgain()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("unlock did not release the key for the next caller")
	}
}
