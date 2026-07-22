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

// TestHelpColumnsRendersStatusAndGroups verifies the help content shows the
// current sort + filter status line plus every key/action/group from the
// single-source table, spread across the two columns.
func TestHelpColumnsRendersStatusAndGroups(t *testing.T) {
	cols := renderHelpColumns("Activity ↓", "work, personal")
	got := strings.Join(cols, "\n")

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

// TestHelpColumnsOmitFilterWhenEmpty verifies the status line drops the Filter
// segment when no filter is active.
func TestHelpColumnsOmitFilterWhenEmpty(t *testing.T) {
	cols := renderHelpColumns("Name ↑", "")
	got := strings.Join(cols, "\n")
	if strings.Contains(got, "Filter:") {
		t.Errorf("help should omit Filter when empty: %q", got)
	}
	if !strings.Contains(got, "Sort: Name ↑") {
		t.Errorf("help should still show sort: %q", got)
	}
}

// TestHelpColumnsSplitsAcrossTwoColumns verifies both columns carry content.
func TestHelpColumnsSplitsAcrossTwoColumns(t *testing.T) {
	cols := renderHelpColumns("Name ↑", "")
	if len(cols) != 3 {
		t.Fatalf("renderHelpColumns returned %d regions, want 3 (status, left, right)", len(cols))
	}
	if cols[1] == "" {
		t.Errorf("left column should not be empty")
	}
	if cols[2] == "" {
		t.Errorf("right column should not be empty")
	}
}

// TestHelpColumnsFirstGroupSoloOnTopRow verifies the layout rule: the first
// group (Navigate) takes the top row alone — it appears in the left column and
// NOT in the right column, whose top is padded blank for that row.
func TestHelpColumnsFirstGroupSoloOnTopRow(t *testing.T) {
	if len(keyBindings) == 0 {
		t.Fatal("keyBindings is empty")
	}
	cols := renderHelpColumns("Name ↑", "")
	first := keyBindings[0].Group
	if !strings.Contains(cols[1], first) {
		t.Errorf("left column should contain first group %q: %q", first, cols[1])
	}
	if strings.Contains(cols[2], first) {
		t.Errorf("right column should not contain first group %q (top row is solo): %q", first, cols[2])
	}
}

// TestPairHelpGroups verifies the row-pairing rule: first group solo, then
// pairs, with a trailing solo if the count is even (odd after the first).
func TestPairHelpGroups(t *testing.T) {
	// 5 groups (the real set) -> [(0,-1),(1,2),(3,4)]
	got := pairHelpGroups(5)
	want := [][2]int{{0, -1}, {1, 2}, {3, 4}}
	if len(got) != len(want) {
		t.Fatalf("pairHelpGroups(5) = %v, want %v rows", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("pairHelpGroups(5)[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

// TestBuildHelpColumnsAligned verifies both columns end up with the same number
// of rendered lines. Together with per-row padding to max(left, right) height,
// that is what keeps group headers aligned across the two columns.
func TestBuildHelpColumnsAligned(t *testing.T) {
	groups := collectHelpGroups()
	rows := pairHelpGroups(len(groups))
	left, right := buildHelpColumns(groups, rows)
	leftLines := strings.Count(left, "\n") + 1
	rightLines := strings.Count(right, "\n") + 1
	if leftLines != rightLines {
		t.Errorf("columns misaligned: left %d lines, right %d lines", leftLines, rightLines)
	}
}
