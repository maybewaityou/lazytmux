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

	"github.com/rivo/tview"
)

// TestHelpColumnsRendersStatusAndGroups verifies the help content shows the
// current sort + filter status line plus every key/action/group from the
// single-source table, in the scrollable two-column body.
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

// TestRenderHelpColumnsReturnsStatusAndBody verifies the layout now produces two
// regions — a status line and the scrollable body — rather than separate left
// and right column strings. The body is the single text block both columns are
// pre-rendered into, which is what lets it scroll as one.
func TestRenderHelpColumnsReturnsStatusAndBody(t *testing.T) {
	cols := renderHelpColumns("Name ↑", "")
	if len(cols) != 2 {
		t.Fatalf("renderHelpColumns returned %d regions, want 2 (status, body)", len(cols))
	}
	if cols[0] == "" {
		t.Errorf("status line should not be empty")
	}
	if cols[1] == "" {
		t.Errorf("body should not be empty")
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

// TestBuildHelpBodyNavigateRowHasEmptyRightColumn pins the layout rule that the
// first row (Navigate) is solo: its lines never reach the right column.
// Concretely, every Navigate-row line's screen width stays within the shared
// left-column width, so nothing is drawn in the right column for that row — the
// requested "Navigate second group empty".
func TestBuildHelpBodyNavigateRowHasEmptyRightColumn(t *testing.T) {
	if len(keyBindings) == 0 {
		t.Fatal("keyBindings is empty")
	}
	groups := collectHelpGroups()
	w := leftColumnWidth(groups)
	body := buildHelpBody(groups, pairHelpGroups(len(groups)))

	for _, line := range strings.Split(body, "\n") {
		if line == "" {
			break // blank line ends the Navigate row
		}
		if got := tview.TaggedStringWidth(line); got > w {
			t.Errorf("Navigate row line reaches the right column (width %d > %d): %q", got, w, line)
		}
	}
}

// TestBuildHelpBodyPairsTwoColumnsPerRow pins that the paired rows really do
// carry both a left and a right group: at least one line is wide enough to span
// past the gutter into the right column (left-column width + gutter), proving
// the two columns are laid out side by side in the one text block.
func TestBuildHelpBodyPairsTwoColumnsPerRow(t *testing.T) {
	groups := collectHelpGroups()
	w := leftColumnWidth(groups)
	body := buildHelpBody(groups, pairHelpGroups(len(groups)))

	for _, line := range strings.Split(body, "\n") {
		if tview.TaggedStringWidth(line) > w+helpGutter {
			return // found a line that spans into the right column
		}
	}
	t.Errorf("no line spans into the right column; the two-column layout is missing")
}
