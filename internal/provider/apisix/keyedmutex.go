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

import "sync"

// keyedMutex is a registry of per-key locks. It exists so that reading a GatewayProxy's
// current resource snapshot and pushing it can be one atomic step per cacheKey: whichever
// caller is granted a key's lock decides what to push only once it actually holds the lock,
// so nothing it sends can already be stale relative to whatever the other caller committed
// to the store before losing the race for the same key.
//
// The registry only grows -- entries are never evicted. Harmless in practice: cacheKey
// tracks a small, effectively fixed set of GatewayProxies for the life of the process. This
// mirrors ADC server's own per-cacheKey sync_lock.
type keyedMutex struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

func newKeyedMutex() *keyedMutex {
	return &keyedMutex{locks: make(map[string]*sync.Mutex)}
}

// Lock blocks until key's lock is held, and returns the func that releases it.
func (k *keyedMutex) Lock(key string) func() {
	k.mu.Lock()
	l, ok := k.locks[key]
	if !ok {
		l = &sync.Mutex{}
		k.locks[key] = l
	}
	k.mu.Unlock()

	l.Lock()
	return l.Unlock
}
