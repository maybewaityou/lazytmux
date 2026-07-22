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

// NewHelpModal builds the help panel: a top status line (current sort, and the
// active filter when one is set) followed by the key bindings grouped by
// section. Content comes entirely from the keyBindings single source, so this
// view can never drift from README.
func NewHelpModal(sortMode, filter string) *tview.TextView {
	tv := tview.NewTextView()
	tv.SetDynamicColors(true)
	tv.SetTextAlign(tview.AlignLeft)

	var b strings.Builder
	// Status line.
	b.WriteString("[" + colorSecondary + "]Sort: " + sortMode + "[-]")
	if filter != "" {
		b.WriteString("    [" + colorSecondary + "]Filter: " + filter + "[-]")
	}
	b.WriteString("\n\n")

	// Grouped bindings.
	group := ""
	for _, kb := range keyBindings {
		if kb.Group != group {
			group = kb.Group
			b.WriteString("[" + colorAccent + "::b]" + group + "[-]\n")
		}
		fmt.Fprintf(&b, "  ["+colorCyan+"]%-6s[-]  %s\n", kb.Key, kb.Action)
	}
	tv.SetText(b.String())
	return tv
}
