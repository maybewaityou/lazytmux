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

// defaultHints follows the same ordering rule as lazyssh:
// Navigate → primary action → feature keys → Pin → Search → Quit.
func defaultHints() string {
	k := colorCyan // key color (matches lazyssh); descriptions use the default foreground
	return "[" + k + "]↑↓[-] Navigate  • " +
		"[" + k + "]Enter[-] Enter  • " +
		"[" + k + "]c[-] Copy  • " +
		"[" + k + "]a[-] New  • " +
		"[" + k + "]e[-] Rename  • " +
		"[" + k + "]d[-] Kill  • " +
		"[" + k + "]t[-] Tags  • " +
		"[" + k + "]s[-] Sort  • " +
		"[" + k + "]r[-] Refresh  • " +
		"[" + k + "]p[-] Pin/Unpin  • " +
		"[" + k + "]/[-] Search  • " +
		"[" + k + "]q[-] Quit"
}
