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

// TestHelpModalRendersStatusAndGroups verifies the help panel shows the current
// sort + filter status line, every key from the single-source table, and the
// group section headers.
func TestHelpModalRendersStatusAndGroups(t *testing.T) {
	tv := NewHelpModal("Activity ↓", "work, personal")
	got := tv.GetText(true)

	if !strings.Contains(got, "Sort: Activity ↓") {
		t.Errorf("help missing sort status: %q", got)
	}
	if !strings.Contains(got, "Filter: work, personal") {
		t.Errorf("help missing filter status: %q", got)
	}
	for _, kb := range keyBindings {
		if !strings.Contains(got, kb.Key) {
			t.Errorf("help missing key %q: %q", kb.Key, got)
		}
		if !strings.Contains(got, kb.Action) {
			t.Errorf("help missing action %q: %q", kb.Action, got)
		}
		if !strings.Contains(got, kb.Group) {
			t.Errorf("help missing group header %q: %q", kb.Group, got)
		}
	}
}

// TestHelpModalOmitsFilterWhenEmpty verifies the status line drops the Filter
// segment when no filter is active (mirrors the list-title behavior).
func TestHelpModalOmitsFilterWhenEmpty(t *testing.T) {
	tv := NewHelpModal("Name ↑", "")
	got := tv.GetText(true)
	if strings.Contains(got, "Filter:") {
		t.Errorf("help should omit Filter when empty: %q", got)
	}
	if !strings.Contains(got, "Sort: Name ↑") {
		t.Errorf("help should still show sort: %q", got)
	}
}
