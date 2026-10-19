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
	"testing"
	"testing/synctest"
	"time"
)

func TestRetrierPreservesPendingDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := NewRetrier(NewExponentialBackoff(10*time.Second, time.Minute))
		defer r.Reset()
		r.Next()
		for range 9 {
			time.Sleep(time.Second)
			r.Next()
		}
		time.Sleep(time.Second)
		synctest.Wait()
		select {
		case <-r.C():
		default:
			t.Fatal("event-triggered failures postponed the pending retry")
		}
		r.Next()
		time.Sleep(20 * time.Second)
		synctest.Wait()
		select {
		case <-r.C():
		default:
			t.Fatal("the next retry must advance backoff only once")
		}
	})
}

func TestRetrierResetClearsPendingRetry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := NewRetrier(NewExponentialBackoff(time.Second, time.Minute))
		defer r.Reset()
		r.Next()
		time.Sleep(time.Second)
		synctest.Wait()
		r.Reset()
		select {
		case <-r.C():
			t.Fatal("reset left a stale retry notification")
		default:
		}
		r.Next()
		time.Sleep(time.Second)
		synctest.Wait()
		select {
		case <-r.C():
		default:
			t.Fatal("reset did not restore the base delay")
		}
	})
}
