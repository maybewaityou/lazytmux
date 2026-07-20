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

import "github.com/rivo/tview"

// StatusBar shows the keybinding hint line. SetStatus updates it (e.g. errors).
type StatusBar struct {
	*tview.TextView
}

func NewStatusBar() *StatusBar {
	sb := &StatusBar{TextView: tview.NewTextView()}
	sb.SetDynamicColors(true)
	sb.SetText(defaultHints())
	return sb
}

// SetStatus replaces the hint line with a (possibly error) message.
func (s *StatusBar) SetStatus(msg string) { s.SetText(msg) }

// ResetHints restores the default keybinding hints.
func (s *StatusBar) ResetHints() { s.SetText(defaultHints()) }

func defaultHints() string {
	k := colorPrimary // key color (descriptions use literal #565f89)
	return "[#565f89]/[-][" + k + "]search[-]  " +
		"[#565f89]↑↓/jk[-][" + k + "]nav[-]  " +
		"[" + k + "]Enter[-][#565f89]enter[-]  " +
		"[" + k + "]a[-][#565f89]new[-]  " +
		"[" + k + "]e[-][#565f89]rename[-]  " +
		"[" + k + "]d[-][#565f89]kill[-]  " +
		"[" + k + "]p[-][#565f89]pin[-]  " +
		"[" + k + "]t[-][#565f89]tags[-]  " +
		"[" + k + "]s[-][#565f89]sort[-]  " +
		"[" + k + "]r[-][#565f89]refresh[-]  " +
		"[" + k + "]c[-][#565f89]copy[-]  " +
		"[" + k + "]q[-][#565f89]quit[-]"
}
