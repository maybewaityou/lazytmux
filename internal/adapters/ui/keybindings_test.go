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
	"testing"
)

// TestKeyBindingsComplete locks the set of keys the TUI advertises, so adding a
// new binding without registering it here (and thus in the help modal / README)
// is caught. Every Key and Action must be non-empty, and keys must be unique.
func TestKeyBindingsComplete(t *testing.T) {
	if len(keyBindings) == 0 {
		t.Fatal("keyBindings must not be empty")
	}
	seen := make(map[string]bool, len(keyBindings))
	for _, kb := range keyBindings {
		if kb.Group == "" || kb.Key == "" || kb.Action == "" {
			t.Errorf("incomplete binding: %+v", kb)
		}
		if seen[kb.Key] {
			t.Errorf("duplicate key %q", kb.Key)
		}
		seen[kb.Key] = true
	}
	// The keys these features add must be present.
	for _, want := range []string{"f", "?", "n"} {
		if !seen[want] {
			t.Errorf("keyBindings missing key %q", want)
		}
	}
}

// TestKeyBindingsGroupsContiguous verifies bindings are grouped contiguously:
// once a group's run ends and a new group begins, the old group must never
// reappear later. This keeps the help modal's section rendering sane.
func TestKeyBindingsGroupsContiguous(t *testing.T) {
	seen := make(map[string]bool)
	prev := ""
	for _, kb := range keyBindings {
		if kb.Group != prev {
			if seen[kb.Group] {
				t.Errorf("group %q is not contiguous: reappears after %q", kb.Group, prev)
			}
			seen[kb.Group] = true
			prev = kb.Group
		}
	}
}
