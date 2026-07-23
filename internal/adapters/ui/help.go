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

// statusLine builds the top status string: the current sort, plus the active
// filter when one is set.
func statusLine(sortMode, filter string) string {
	s := "[" + colorSecondary + "]Sort: " + sortMode + "[-]"
	if filter != "" {
		s += "    [" + colorSecondary + "]Filter: " + filter + "[-]"
	}
	return s
}

// renderGroupLines renders one group as a slice of screen lines: a colored
// header, then each binding indented under it. Returning lines (not a joined
// string) lets the two-column body merge a left group and a right group
// line-by-line, so their headers land on the same row.
func renderGroupLines(g helpGroup) []string {
	lines := []string{"[" + colorAccent + "::b]" + g.name + "[-]"}
	for _, bd := range g.bindings {
		lines = append(lines, fmt.Sprintf("  ["+colorCyan+"]%-6s[-]  %s", bd.key, bd.action))
	}
	return lines
}

// helpGutter is the blank gutter between the two columns, in screen columns.
const helpGutter = 2

// leftColumnWidth is the width every left-column line is right-padded to. Every
// row shares it, so the right column (after the gutter) starts at the same
// screen column on every row — this is what keeps group headers aligned across
// the two columns, including the Navigate row whose right side is empty.
func leftColumnWidth(groups []helpGroup) int {
	w := 0
	for _, g := range groups {
		for _, line := range renderGroupLines(g) {
			if tw := tview.TaggedStringWidth(line); tw > w {
				w = tw
			}
		}
	}
	return w
}

// padTagged right-pads s with spaces to screen width w, preserving tview style
// tags (they don't count toward width, so colors survive the padding).
func padTagged(s string, w int) string {
	if pad := w - tview.TaggedStringWidth(s); pad > 0 {
		return s + strings.Repeat(" ", pad)
	}
	return s
}

// renderHelpRow merges one paired row into screen lines: the left group and the
// right group line-by-line, the left line padded to leftColumnWidth, then the
// gutter, then the right line. A row whose right group is absent (rightIndex
// -1 — the Navigate row) prints only the left line, so its right side stays
// empty (the requested "Navigate second group empty"). Shorter side is padded
// with blank lines so the two headers still align when one group is taller.
func renderHelpRow(groups []helpGroup, row [2]int, w int) []string {
	hasRight := row[1] >= 0 && row[1] < len(groups)
	var left, right []string
	if row[0] >= 0 && row[0] < len(groups) {
		left = renderGroupLines(groups[row[0]])
	}
	if hasRight {
		right = renderGroupLines(groups[row[1]])
	}
	h := len(left)
	if len(right) > h {
		h = len(right)
	}
	out := make([]string, 0, h)
	for k := 0; k < h; k++ {
		var l, r string
		if k < len(left) {
			l = left[k]
		}
		if k < len(right) {
			r = right[k]
		}
		line := padTagged(l, w)
		if hasRight {
			line += strings.Repeat(" ", helpGutter) + r
		}
		out = append(out, strings.TrimRight(line, " "))
	}
	return out
}

// buildHelpBody lays the paired groups out as one scrollable text block: each
// row's merged lines, with a blank line between rows. Because both columns live
// in one text block, scrolling moves them together and the headers never drift
// out of alignment — the reason the body is one TextView, not two.
func buildHelpBody(groups []helpGroup, rows [][2]int) string {
	w := leftColumnWidth(groups)
	var b strings.Builder
	for ri, row := range rows {
		for _, line := range renderHelpRow(groups, row, w) {
			b.WriteString(line)
			b.WriteByte('\n')
		}
		if ri < len(rows)-1 {
			b.WriteByte('\n') // blank line between rows
		}
	}
	return b.String()
}

// pairHelpGroups lays groups out as rows for the two-column body. The first
// group takes the top row alone (right side empty) so the primary "Navigate"
// section stands out; the remaining groups pair up two per row in order. A
// trailing odd group takes a row alone. Each returned pair is [leftIndex,
// rightIndex], with rightIndex = -1 meaning that row's right side is empty.
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

// renderHelpColumns builds the help panel content as one string per region:
// [0] = status line, [1] = the scrollable body (both columns pre-rendered into
// one aligned text block). Pure function so the layout is unit-testable
// without tview.
func renderHelpColumns(sortMode, filter string) []string {
	groups := collectHelpGroups()
	rows := pairHelpGroups(len(groups))
	return []string{statusLine(sortMode, filter), buildHelpBody(groups, rows)}
}

// helpTextView is a non-wrapping, dynamic-color text pane. Wrap is disabled so
// a column never breaks a line across rows; the body uses it with scrolling
// enabled so the two-column layout scrolls as one.
func helpTextView(text string) *tview.TextView {
	tv := tview.NewTextView()
	tv.SetDynamicColors(true)
	tv.SetTextAlign(tview.AlignLeft)
	tv.SetWrap(false)
	tv.SetText(text)
	return tv
}

// HelpModal is the help panel: a status line on top, then the key bindings as a
// scrollable two-column body below (first group prominent on the top row, the
// rest paired two-per-row). Content comes entirely from the keyBindings single
// source via renderHelpColumns.
type HelpModal struct {
	*tview.Flex
	// focus is the scrollable body; pointing the app focus here lets the modal's
	// InputCapture (?/Esc/q to close) receive keys, and lets ↑↓/PgUp·PgDn/wheel
	// scroll the body.
	focus tview.Primitive
}

// NewHelpModal builds the panel. The returned HelpModal embeds the layout Flex
// (to be placed as the modal body) and exposes the scrollable body as the focus
// target.
func NewHelpModal(sortMode, filter string) *HelpModal {
	cols := renderHelpColumns(sortMode, filter)
	statusTv := helpTextView(cols[0])
	bodyTv := helpTextView(cols[1])
	bodyTv.SetScrollable(true) // ↑↓/PgUp·PgDn/wheel scroll both columns together

	root := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(statusTv, 1, 0, false).
		AddItem(nil, 1, 0, false). // blank line under the status line
		AddItem(bodyTv, 0, 1, true)

	return &HelpModal{Flex: root, focus: bodyTv}
}
