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

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"

	"github.com/maybewaityou/lazytmux/internal/core/domain"
)

const colorRed = "#f7768e"

func (t *tui) handleGlobalKeys(e *tcell.EventKey) *tcell.EventKey {
	// When the search bar is focused, let it handle all keys (typing).
	if t.app.GetFocus() == t.searchBar {
		return e
	}
	switch e.Rune() {
	case '/':
		t.app.SetFocus(t.searchBar)
		return nil
	case 'q':
		t.app.Stop()
		return nil
	case 'r':
		t.refresh()
		t.statusBar.SetStatus("[" + colorGreen + "]refreshed[-]")
		return nil
	case 's':
		t.sortMode = t.sortMode.Next()
		t.applySortAndRender()
		return nil
	case 'S':
		t.sortMode = t.sortMode.Next().Next()
		t.applySortAndRender()
		return nil
	case 'p':
		t.actOnSelected(func(s domain.Session) {
			if err := t.serve.TogglePin(s.Name); err == nil {
				t.refresh()
			} else {
				t.statusBar.SetStatus("[" + colorRed + "]pin failed[-]")
			}
		})
		return nil
	case 'a':
		t.openForm("New session", "session name", func(name string) {
			if err := t.serve.CreateSession(name); err == nil {
				t.refresh()
			} else {
				t.statusBar.SetStatus("[" + colorRed + "]create failed[-]")
			}
		})
		return nil
	case 'e':
		t.actOnSelected(func(s domain.Session) {
			t.openForm("Rename", "new name", func(newName string) {
				if err := t.serve.RenameSession(s.Name, newName); err == nil {
					t.refresh()
				} else {
					t.statusBar.SetStatus("[" + colorRed + "]rename failed[-]")
				}
			})
		})
		return nil
	case 'd':
		t.actOnSelected(func(s domain.Session) {
			if err := t.serve.KillSession(s.Name); err == nil {
				t.refresh()
			} else {
				t.statusBar.SetStatus("[" + colorRed + "]kill failed[-]")
			}
		})
		return nil
	case 'c':
		t.actOnSelected(func(s domain.Session) {
			_ = clipboard.WriteAll("tmux attach -t " + s.Name)
			t.statusBar.SetStatus("[" + colorGreen + "]copied: tmux attach -t " + s.Name + "[-]")
		})
		return nil
	}
	switch e.Key() {
	case tcell.KeyEnter:
		t.actOnSelected(func(s domain.Session) {
			if err := t.serve.EnterSession(s.Name); err != nil {
				t.statusBar.SetStatus("[" + colorRed + "]enter failed: " + err.Error() + "[-]")
				return
			}
			t.app.Stop()
		})
		return nil
	}
	return e
}

func (t *tui) handleSearchInput(text string) {
	if text == "" {
		t.sessionList.UpdateSessions(t.allCache)
		return
	}
	filtered := make([]domain.Session, 0, len(t.allCache))
	for _, s := range t.allCache {
		if fuzzyMatch(text, s.Name) {
			filtered = append(filtered, s)
		}
	}
	t.sessionList.UpdateSessions(filtered)
}

func (t *tui) handleSelectionChange(s domain.Session) {
	t.details.Render(s)
	// Lazy-load windows in the background, then re-render.
	go func(cp domain.Session) {
		_ = t.serve.LoadWindows(&cp)
		t.app.QueueUpdateDraw(func() {
			t.details.Render(cp)
		})
	}(s)
}

func (t *tui) blurSearchBar() {
	t.searchBar.SetText("")
	t.handleSearchInput("")
	t.app.SetFocus(t.sessionList)
}

func (t *tui) refresh() {
	sessions, err := t.serve.ListSessions()
	if err != nil {
		t.statusBar.SetStatus("[" + colorRed + "]tmux error: " + err.Error() + "[-]")
		t.allCache = nil
		t.sessionList.UpdateSessions(nil)
		return
	}
	t.allCache = sessions
	t.applySortAndRender()
}

func (t *tui) applySortAndRender() {
	sortSessionsForUI(t.allCache, t.sortMode, t.serve.LastAttached)
	t.sessionList.SetSortTitle(t.sortMode.String())
	t.sessionList.UpdateSessions(t.allCache)
}

func (t *tui) actOnSelected(fn func(domain.Session)) {
	s, ok := t.sessionList.GetSelected()
	if !ok {
		return
	}
	fn(s)
}

func (t *tui) openForm(title, placeholder string, onSubmit func(string)) {
	form := NewSessionForm(title, placeholder).
		OnSubmit(func(name string) {
			name = strings.TrimSpace(name)
			if name != "" {
				onSubmit(name)
			}
			t.closeForm()
		}).
		OnCancel(t.closeForm)
	t.app.SetRoot(form.Primitive(), true)
	t.app.SetFocus(form.Input())
}

func (t *tui) closeForm() {
	t.app.SetRoot(t.root, true)
	t.app.SetFocus(t.sessionList)
}
