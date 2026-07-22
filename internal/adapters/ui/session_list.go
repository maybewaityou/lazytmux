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

	"github.com/maybewaityou/lazytmux/internal/core/domain"
)

// SessionList wraps tview.List and holds the current sessions for index lookup.
type SessionList struct {
	*tview.List
	sessions          []domain.Session
	current           string
	sortLabel         string // current sort mode label, e.g. "Name ↑"
	filterLabel       string // current filter label, "" when no filter
	onSelectionChange func(domain.Session)
	onReturnToSearch  func()
}

func NewSessionList() *SessionList {
	sl := &SessionList{List: tview.NewList()}
	sl.build()
	return sl
}

func (sl *SessionList) build() {
	sl.List.ShowSecondaryText(false)
	sl.List.SetBorder(true).
		SetTitle(" Sessions ").
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.GetColor(colorBorder)).
		SetTitleColor(tcell.GetColor(colorTitle))
	sl.List.
		SetSelectedBackgroundColor(tcell.GetColor(colorSelected)).
		SetSelectedTextColor(tcell.GetColor(colorPrimary)).
		SetHighlightFullLine(true)

	sl.List.SetChangedFunc(func(index int, _, _ string, _ rune) {
		if index >= 0 && index < len(sl.sessions) && sl.onSelectionChange != nil {
			sl.onSelectionChange(sl.sessions[index])
		}
	})

	sl.List.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		switch e.Key() {
		case tcell.KeyLeft, tcell.KeyRight, tcell.KeyBackspace, tcell.KeyBackspace2, tcell.KeyESC:
			if sl.onReturnToSearch != nil {
				sl.onReturnToSearch()
			}
			return nil
		}
		return e
	})
}

func (sl *SessionList) UpdateSessions(sessions []domain.Session) {
	sl.sessions = sessions
	sl.List.Clear()
	for i := range sessions {
		sl.List.AddItem(formatSessionLine(sessions[i], sl.current), "", 0, nil)
	}
	if sl.List.GetItemCount() > 0 {
		sl.List.SetCurrentItem(0)
	}
}

func (sl *SessionList) GetSelected() (domain.Session, bool) {
	idx := sl.List.GetCurrentItem()
	if idx >= 0 && idx < len(sl.sessions) {
		return sl.sessions[idx], true
	}
	return domain.Session{}, false
}

// SelectByName moves the cursor to the first session with the given name, if
// any. UpdateSessions always resets the cursor to the first item, so after a
// refresh we call this to keep the user's current selection rather than snapping
// back to the top of the list. If the name is gone — e.g. the session was killed
// out-of-band — the cursor is left wherever UpdateSessions last placed it.
func (sl *SessionList) SelectByName(name string) {
	for i, s := range sl.sessions {
		if s.Name == name {
			sl.List.SetCurrentItem(i)
			return
		}
	}
}

func (sl *SessionList) OnSelectionChange(fn func(domain.Session)) *SessionList {
	sl.onSelectionChange = fn
	return sl
}

func (sl *SessionList) OnReturnToSearch(fn func()) *SessionList {
	sl.onReturnToSearch = fn
	return sl
}

// SetCurrent records the name of the tmux session the user is currently inside
// (empty string = not inside tmux). The next UpdateSessions/refresh renders a
// ▶ marker at the start of the matching row. Call once before the first render.
func (sl *SessionList) SetCurrent(name string) *SessionList { sl.current = name; return sl }

// SetSortTitle records the sort label and refreshes the composed border title.
func (sl *SessionList) SetSortTitle(mode string) {
	sl.sortLabel = mode
	sl.refreshTitle()
}

// SetFilter records the active filter label ("" = no filter) and refreshes the
// composed border title. The filter is shown alongside the sort mode so both
// pieces of view state share one surface — mirroring how sort is already shown.
func (sl *SessionList) SetFilter(filter string) {
	sl.filterLabel = filter
	sl.refreshTitle()
}

// refreshTitle composes the list border title: always "Sort: <mode>", plus
// "— Filter: <tags>" when a filter is active.
func (sl *SessionList) refreshTitle() {
	title := " Sessions — Sort: " + sl.sortLabel + " "
	if sl.filterLabel != "" {
		title = " Sessions — Sort: " + sl.sortLabel + " — Filter: " + sl.filterLabel + " "
	}
	sl.List.SetTitle(title)
}
