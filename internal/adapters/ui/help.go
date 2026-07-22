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
	"fmt"
	"strings"

	"github.com/rivo/tview"
)

// helpBinding is one rendered key-binding entry within a column.
type helpBinding struct {
	key    string
	action string
}

// helpGroup is a named section of consecutive bindings (e.g. "Navigate").
type helpGroup struct {
	name     string
	bindings []helpBinding
}

// collectHelpGroups walks the keyBindings single source in order and groups
// consecutive same-Group entries. Preserves order and never merges non-adjacent
// groups (the contiguity test forbids those).
func collectHelpGroups() []helpGroup {
	var groups []helpGroup
	for _, kb := range keyBindings {
		if len(groups) == 0 || groups[len(groups)-1].name != kb.Group {
			groups = append(groups, helpGroup{name: kb.Group})
		}
		groups[len(groups)-1].bindings = append(groups[len(groups)-1].bindings, helpBinding{key: kb.Key, action: kb.Action})
	}
	return groups
}

// helpGroupHeight is the rendered line count of a group: one header plus one
// line per binding.
func helpGroupHeight(g helpGroup) int { return 1 + len(g.bindings) }

// statusLine builds the top status string: the current sort, plus the active
// filter when one is set.
func statusLine(sortMode, filter string) string {
	s := "[" + colorSecondary + "]Sort: " + sortMode + "[-]"
	if filter != "" {
		s += "    [" + colorSecondary + "]Filter: " + filter + "[-]"
	}
	return s
}

// renderHelpColumn renders a sequence of groups as one text block: a colored
// group header followed by each binding indented under it.
func renderHelpColumn(groups []helpGroup) string {
	var b strings.Builder
	for _, g := range groups {
		b.WriteString("[" + colorAccent + "::b]" + g.name + "[-]\n")
		for _, bd := range g.bindings {
			fmt.Fprintf(&b, "  ["+colorCyan+"]%-6s[-]  %s\n", bd.key, bd.action)
		}
	}
	return b.String()
}

// pairHelpGroups lays groups out as rows for a two-column grid. The first group
// takes the top row alone (right side empty) so the primary "Navigate" section
// stands out; the remaining groups pair up two per row in order. A trailing odd
// group takes a row alone. Each returned pair is [leftIndex, rightIndex], with
// rightIndex = -1 meaning that row's right side is empty.
func pairHelpGroups(n int) [][2]int {
	var rows [][2]int
	if n == 0 {
		return rows
	}
	rows = append(rows, [2]int{0, -1})
	for i := 1; i < n; i += 2 {
		right := -1
		if i+1 < n {
			right = i + 1
		}
		rows = append(rows, [2]int{i, right})
	}
	return rows
}

// buildHelpColumns turns the row layout into left/right column text. Each row
// is rendered at a height of max(leftGroup, rightGroup) lines, with both sides
// padded with blank lines to that height — this is what keeps group headers
// aligned across the two columns even when one side's group is taller.
func buildHelpColumns(groups []helpGroup, rows [][2]int) (left, right string) {
	var leftB, rightB strings.Builder
	for _, row := range rows {
		hasLeft := row[0] >= 0 && row[0] < len(groups)
		hasRight := row[1] >= 0 && row[1] < len(groups)

		hL, hR := 0, 0
		if hasLeft {
			hL = helpGroupHeight(groups[row[0]])
		}
		if hasRight {
			hR = helpGroupHeight(groups[row[1]])
		}
		rowH := hL
		if hR > rowH {
			rowH = hR
		}

		// Left column: the group text (hL lines) then blank lines up to rowH.
		if hasLeft {
			leftB.WriteString(renderHelpColumn([]helpGroup{groups[row[0]]}))
		}
		for i := hL; i < rowH; i++ {
			leftB.WriteString("\n")
		}
		// Right column: the group text (hR lines) then blank lines up to rowH,
		// or fully blank when this row has no right group.
		if hasRight {
			rightB.WriteString(renderHelpColumn([]helpGroup{groups[row[1]]}))
		}
		for i := hR; i < rowH; i++ {
			rightB.WriteString("\n")
		}
	}
	// Keep trailing newlines on both sides: each row pads to the same height on
	// the left and the right, so the newline counts stay equal and group headers
	// align. tview renders a trailing newline as no extra visible row.
	left = leftB.String()
	right = rightB.String()
	return
}

// renderHelpColumns builds the help panel content as one string per region:
// [0] = status line, [1] = left column, [2] = right column. Pure function so
// the two-column layout is unit-testable without tview.
func renderHelpColumns(sortMode, filter string) []string {
	groups := collectHelpGroups()
	rows := pairHelpGroups(len(groups))
	left, right := buildHelpColumns(groups, rows)
	return []string{statusLine(sortMode, filter), left, right}
}

// helpTextView is a non-wrapping, dynamic-color text pane used for each help
// region. Wrap is disabled so a column never breaks a line across rows.
func helpTextView(text string) *tview.TextView {
	tv := tview.NewTextView()
	tv.SetDynamicColors(true)
	tv.SetTextAlign(tview.AlignLeft)
	tv.SetWrap(false)
	tv.SetText(text)
	return tv
}

// HelpModal is the two-column help panel: a status line on top, then the key
// bindings laid out below with the first group prominent on the top row and the
// rest paired two-per-row. Content comes entirely from the keyBindings single
// source via renderHelpColumns.
type HelpModal struct {
	*tview.Flex
	// focus is the left column; pointing the app focus here lets the modal's
	// InputCapture (?/Esc/q to close) receive keys reliably.
	focus tview.Primitive
}

// NewHelpModal builds the panel. The returned HelpModal embeds the layout Flex
// (to be placed as the modal body) and exposes the focus target via the struct.
func NewHelpModal(sortMode, filter string) *HelpModal {
	cols := renderHelpColumns(sortMode, filter)
	statusTv := helpTextView(cols[0])
	leftTv := helpTextView(cols[1])
	rightTv := helpTextView(cols[2])

	body := tview.NewFlex().
		AddItem(leftTv, 0, 1, false).
		AddItem(tview.NewTextView(), 2, 0, false). // gutter between the two columns
		AddItem(rightTv, 0, 1, false)

	root := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(statusTv, 1, 0, false).
		AddItem(nil, 1, 0, false). // blank line under the status line
		AddItem(body, 0, 1, true)

	return &HelpModal{Flex: root, focus: leftTv}
}
