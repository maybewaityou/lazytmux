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
// few keys that still work on an empty list, plus the help pointer.
func TestEmptyHintsAnchorsUsefulKeys(t *testing.T) {
	got := emptyHints()
	for _, want := range []string{"No sessions", "a", "?", "q"} {
		if !strings.Contains(got, "["+colorCyan+"]"+want+"[-]") {
			t.Errorf("emptyHints missing key %q in: %q", want, got)
		}
	}
}

// TestEmptyHintsOmitsNoOpKeys verifies the empty-state footer drops the keys
// that are meaningless with nothing selected.
func TestEmptyHintsOmitsNoOpKeys(t *testing.T) {
	got := emptyHints()
	for _, dead := range []string{"Kill", "Detach", "Copy", "Edit", "Pin", "Tags", "Sort", "Enter", "Filter"} {
		if strings.Contains(got, dead) {
			t.Errorf("emptyHints should not advertise no-op %q: %q", dead, got)
		}
	}
}

// TestDefaultHintsCompactAndPointsToHelp verifies the populated-state footer is
// intentionally short: it keeps only the most-used keys plus a ? Help pointer,
// and moves the less-frequent actions (and Sort) behind the help panel. The
// active sort and filter are already surfaced in the list title, so they do not
// need a footer slot.
func TestDefaultHintsCoreKeysAndPointsToHelp(t *testing.T) {
	got := defaultHints()
	// Core keys that stay in the footer.
	for _, want := range []string{"Navigate", "Enter", "New", "Edit", "Detach", "Kill", "Refresh", "Search", "Help", "Quit"} {
		if !strings.Contains(got, want) {
			t.Errorf("defaultHints missing expected key %q: %q", want, got)
		}
	}
	// Keys moved behind ? to keep the footer short.
	for _, gone := range []string{"Sort", "Copy", "Tags", "Pin", "Filter"} {
		if strings.Contains(got, gone) {
			t.Errorf("defaultHints should omit %q (now behind ?): %q", gone, got)
		}
	}
}
