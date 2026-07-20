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
}

func NewSessionDetails() *SessionDetails {
	d := &SessionDetails{TextView: tview.NewTextView().SetDynamicColors(true).SetWrap(true)}
	d.SetBorder(true).
		SetTitle(" Details ").
		SetTitleColor(tcell.GetColor(colorTitle)).
		SetBorderColor(tcell.GetColor(colorBorder))
	d.SetText("[" + colorSecondary + "]select a session[-]")
	return d
}

// Render fills the pane. If s.Windows is populated, lists them (active marked *).
func (d *SessionDetails) Render(s domain.Session) {
	var b strings.Builder
	b.WriteString("[" + colorAccent + "::b]" + s.Name + "[-]\n\n")
	b.WriteString(fmt.Sprintf("[%s]created:   [%s]%s[-]\n", colorSecondary, colorPrimary, s.Created.Format("2006-01-02 15:04")))
	b.WriteString(fmt.Sprintf("[%s]activity:  [%s]%s[-]\n", colorSecondary, colorPrimary, s.LastActivity.Format("2006-01-02 15:04")))
	b.WriteString(fmt.Sprintf("[%s]attached:  [%s]%d client(s)[-]\n", colorSecondary, colorPrimary, s.AttachedCount))
	b.WriteString(fmt.Sprintf("[%s]windows:   [%s]%d[-]\n", colorSecondary, colorPrimary, s.WindowsCount))
	if s.Path != "" {
		b.WriteString(fmt.Sprintf("[%s]path:      [%s]%s[-]\n", colorSecondary, colorPrimary, s.Path))
	}
	if len(s.Tags) > 0 {
		b.WriteString(fmt.Sprintf("[%s]tags:      [%s]%s[-]\n", colorSecondary, colorGreen, strings.Join(s.Tags, ", ")))
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
			b.WriteString(fmt.Sprintf("[%s]%s [%s]%d [%s]%s [%s](%s)[-]\n",
				colorSecondary, marker, colorSecondary, w.Index, clr, w.Name, colorSecondary, w.Command))
		}
	}
	d.SetText(b.String())
}
