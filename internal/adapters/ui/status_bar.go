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
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// StatusBar shows the keybinding hint line. SetStatus updates it (e.g. errors).
type StatusBar struct {
	*tview.TextView
}

func NewStatusBar() *StatusBar {
	sb := &StatusBar{TextView: tview.NewTextView()}
	sb.SetDynamicColors(true)
	sb.SetTextAlign(tview.AlignCenter)
	sb.SetBackgroundColor(tcell.ColorDefault)
	sb.SetText(defaultHints())
	return sb
}

// SetStatus replaces the hint line with a (possibly error) message.
func (s *StatusBar) SetStatus(msg string) { s.SetText(msg) }

// ResetHints restores the default keybinding hints.
func (s *StatusBar) ResetHints() { s.SetText(defaultHints()) }

// ShowEmpty swaps in the minimal empty-state hint. With no sessions loaded, most
// keys are no-ops, so we surface only the actions that still work plus a pointer
// to the full reference.
func (s *StatusBar) ShowEmpty() { s.SetText(emptyHints()) }

// defaultHints shows the primary navigation and session-action keys plus a
// pointer to the full reference (?). Less-frequent actions (copy/tags/pin/sort/
// filter) live behind ?; the active sort and tag filter are surfaced in the list
// title (SetSortTitle / SetFilter) so they don't need a footer slot.
func defaultHints() string {
	k := colorCyan
	return "[" + k + "]↑↓[-] Navigate  • " +
		"[" + k + "]Enter[-] Enter  • " +
		"[" + k + "]a[-] New  • " +
		"[" + k + "]e[-] Rename  • " +
		"[" + k + "]d[-] Detach  • " +
		"[" + k + "]k[-] Kill  • " +
		"[" + k + "]r[-] Refresh  • " +
		"[" + k + "]/[-] Search  • " +
		"[" + k + "]?[-] Help  • " +
		"[" + k + "]q[-] Quit"
}

// emptyHints is the footer for the no-sessions state — a lead-in label plus the
// few keys that remain meaningful when the list is empty.
func emptyHints() string {
	k := colorCyan
	return "[" + k + "]No sessions[-]  •  " +
		"[" + k + "]a[-] New  •  " +
		"[" + k + "]?[-] Help  •  " +
		"[" + k + "]q[-] Quit"
}
