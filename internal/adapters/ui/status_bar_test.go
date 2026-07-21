// Copyright 2026.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ui

import (
	"strings"
	"testing"
)

// TestEmptyHintsAnchorsUsefulKeys verifies the no-sessions footer surfaces the
// handful of keys that still work on an empty list.
func TestEmptyHintsAnchorsUsefulKeys(t *testing.T) {
	got := emptyHints()
	for _, want := range []string{"No sessions", "a", "r", "q"} {
		if !strings.Contains(got, "["+colorCyan+"]"+want+"[-]") {
			t.Errorf("emptyHints missing key %q in: %q", want, got)
		}
	}
}

// TestEmptyHintsOmitsNoOpKeys verifies the empty-state footer drops the keys that
// are meaningless with nothing selected (Enter/copy/rename/kill/detach/tags/pin/sort).
func TestEmptyHintsOmitsNoOpKeys(t *testing.T) {
	got := emptyHints()
	for _, dead := range []string{"Kill", "Detach", "Copy", "Rename", "Pin", "Tags", "Sort", "Enter"} {
		if strings.Contains(got, dead) {
			t.Errorf("emptyHints should not advertise no-op %q: %q", dead, got)
		}
	}
}

// TestDefaultHintsOmitsSortForWidth verifies the populated-state footer hides the
// Sort hint to stay on a single line. Unlike the empty-state omissions (no-ops),
// Sort stays fully functional here — the 's' key still cycles sort mode — so this
// is a display-only trade-off, locked against being silently re-added.
func TestDefaultHintsOmitsSortForWidth(t *testing.T) {
	got := defaultHints()
	if strings.Contains(got, "Sort") {
		t.Errorf("defaultHints should omit Sort to save width, but found it: %q", got)
	}
	// Core keys must still be advertised.
	for _, want := range []string{"Navigate", "Enter", "Kill", "Detach", "Tags", "Refresh", "Pin/Unpin", "Search", "Quit"} {
		if !strings.Contains(got, want) {
			t.Errorf("defaultHints missing expected key %q: %q", want, got)
		}
	}
}
