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

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/maybewaityou/lazytmux/internal/core/domain"
)

// SessionDetails is the right-hand pane showing session metadata + windows.
type SessionDetails struct {
	*tview.TextView
	current string
}

func NewSessionDetails() *SessionDetails {
	d := &SessionDetails{TextView: tview.NewTextView().SetDynamicColors(true).SetWrap(true)}
	d.SetBorder(true).
		SetTitle(" Details ").
		SetTitleColor(tcell.GetColor(colorTitle)).
		SetBorderColor(tcell.GetColor(colorBorder))
	// Placeholder states (initial + empty) read better centered; Render flips
	// back to left alignment for multi-line session content.
	d.SetTextAlign(tview.AlignCenter)
	d.SetText("[" + colorSecondary + "]select a session[-]")
	return d
}

// SetCurrent records the tmux session the user is currently attached to. When
// Render is called for that session, its name is suffixed with (current).
func (d *SessionDetails) SetCurrent(name string) *SessionDetails { d.current = name; return d }

// Render fills the pane. If s.Windows is populated, lists them (active marked *).
func (d *SessionDetails) Render(s domain.Session) {
	// Multi-line content is left-aligned; restore it in case the previous
	// state was a centered placeholder.
	d.SetTextAlign(tview.AlignLeft)
	var b strings.Builder
	name := s.Name
	if d.current != "" && s.Name == d.current {
		name = s.Name + " [" + colorGreen + "](current)[-]"
	}
	b.WriteString("[" + colorAccent + "::b]" + name + "[-]\n\n")
	b.WriteString("[" + colorTitle + "::b]Basic Info[-]\n")
	b.WriteString(detailField("created", s.Created.Format("2006-01-02 15:04")))
	b.WriteString(detailField("activity", s.LastActivity.Format("2006-01-02 15:04")))
	b.WriteString(detailField("attached", fmt.Sprintf("%d client(s)", s.AttachedCount)))
	b.WriteString(detailField("windows", fmt.Sprintf("%d", s.WindowsCount)))
	if s.Path != "" {
		b.WriteString(detailField("path", s.Path))
	}
	b.WriteString(detailField("pinned", pinnedStr(s.Pinned)))
	if len(s.Tags) > 0 {
		// Tags carry their own [black:accent]...[-:-:-] chips, so they bypass
		// detailField's colorPrimary wrap to avoid nested conflicting tags.
		b.WriteString(padTagged("  ["+colorSecondary+"]tags:[-]", detailLabelWidth) + renderTagChips(s.Tags) + "\n")
	}
	if s.Note != "" {
		b.WriteString(detailField("note", s.Note))
	}

	if len(s.Windows) > 0 {
		b.WriteString("\n[" + colorTitle + "::b]Windows[-]\n")
		for _, w := range s.Windows {
			marker := " "
			clr := colorPrimary
			if w.Active {
				marker = "*"
				clr = colorGreen
			}
			b.WriteString(fmt.Sprintf("  [%s]%s [%s]%d [%s]%s [%s](%s)[-]\n",
				colorSecondary, marker, colorSecondary, w.Index, clr, w.Name, colorSecondary, w.Command))
		}
	}
	d.SetText(b.String())
}

// RenderEmpty replaces the pane with a centered placeholder when nothing is
// selected (e.g. the session list is empty after a kill).
func (d *SessionDetails) RenderEmpty(msg string) {
	d.SetTextAlign(tview.AlignCenter)
	d.SetText("[" + colorSecondary + "]" + msg + "[-]")
}

// detailLabelWidth is the visible width every detail label is padded to (the
// 2-space indent plus label and colon), sized to the longest label
// "  last attached:" so all values start at the same column. tview color tags
// occupy bytes but no visible width, so padding must use tview.TaggedStringWidth,
// not fmt's %-N.
const detailLabelWidth = 16

// detailField renders one "  label: value" line with the label in
// colorSecondary and the value in colorPrimary, label padded to
// detailLabelWidth. Callers rendering values that carry their own color tags
// (e.g. tag chips) must NOT use this helper — see Render's tags line.
func detailField(label, value string) string {
	return padTagged("  ["+colorSecondary+"]"+label+":[-]", detailLabelWidth) + "[" + colorPrimary + "]" + value + "[-]\n"
}

// pinnedStr renders the pinned bool as the literal "true"/"false" shown in the
// details pane (matching lazyssh).
func pinnedStr(p bool) string {
	if p {
		return "true"
	}
	return "false"
}
