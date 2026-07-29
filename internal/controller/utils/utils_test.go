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
	"strings"
	"testing"
	"unicode/utf8"
)

func TestTruncateConditionMessage(t *testing.T) {
	t.Run("message under limit is unchanged", func(t *testing.T) {
		msg := strings.Repeat("a", maxConditionMessageBytes)
		if got := TruncateConditionMessage(msg); got != msg {
			t.Fatalf("expected message to be unchanged, got len=%d", len(got))
		}
	})

	t.Run("ascii message over limit is capped and marked", func(t *testing.T) {
		msg := strings.Repeat("a", maxConditionMessageBytes+100)
		got := TruncateConditionMessage(msg)
		if len(got) > maxConditionMessageBytes {
			t.Fatalf("truncated message len=%d exceeds limit %d", len(got), maxConditionMessageBytes)
		}
		if !strings.HasSuffix(got, conditionMessageTruncationMarker) {
			t.Fatalf("truncated message does not end with marker: %q", got[len(got)-len(conditionMessageTruncationMarker):])
		}
		if !utf8.ValidString(got) {
			t.Fatalf("truncated message is not valid UTF-8")
		}
	})

	// Build a message from 3-byte runes so the byte budget lands in the middle
	// of a rune, proving the cut backs off to a rune boundary.
	t.Run("multi-byte rune straddling the boundary is rune-safe", func(t *testing.T) {
		const rune3 = "中"                                      // 3 bytes
		msg := strings.Repeat(rune3, maxConditionMessageBytes) // ~3x over the limit
		got := TruncateConditionMessage(msg)

		if len(got) > maxConditionMessageBytes {
			t.Fatalf("truncated message len=%d exceeds limit %d", len(got), maxConditionMessageBytes)
		}
		if !utf8.ValidString(got) {
			t.Fatalf("truncated message is not valid UTF-8 (a rune was split)")
		}
		if !strings.HasSuffix(got, conditionMessageTruncationMarker) {
			t.Fatalf("truncated message does not end with marker")
		}
		// The content before the marker must consist only of whole 3-byte runes.
		content := strings.TrimSuffix(got, conditionMessageTruncationMarker)
		if strings.Trim(content, rune3) != "" {
			t.Fatalf("truncated content contains a partial rune")
		}
	})
}

// TestNewConditionTypeAcceptedTruncates ensures the constructor routes its
// Message through the cap so no status update can exceed the Kubernetes limit.
func TestNewConditionTypeAcceptedTruncates(t *testing.T) {
	huge := strings.Repeat("x", maxConditionMessageBytes*2)
	cond := NewConditionTypeAccepted("SyncFailed", false, 1, huge)
	if len(cond.Message) > maxConditionMessageBytes {
		t.Fatalf("condition message len=%d exceeds limit %d", len(cond.Message), maxConditionMessageBytes)
	}
}
