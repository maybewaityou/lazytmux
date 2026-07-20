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
	"time"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/maybewaityou/lazytmux/internal/core/domain"
)

const colorRed = "#f7768e"

// statusToastTimeout is how long a transient footer message (e.g. "refreshed")
// stays visible before reverting to the default keybinding hints.
const statusToastTimeout = 3 * time.Second

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
		t.setStatusTemporary("[" + colorGreen + "]Refreshed[-]")
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
				t.setStatusTemporary("[" + colorRed + "]pin failed[-]")
			}
		})
		return nil
	case 'a':
		t.openForm("New session", "session name", func(name string) {
			if err := t.serve.CreateSession(name); err == nil {
				t.refresh()
			} else {
				t.setStatusTemporary("[" + colorRed + "]create failed[-]")
			}
		})
		return nil
	case 'e':
		t.actOnSelected(func(s domain.Session) {
			t.openForm("Rename", "new name", func(newName string) {
				if err := t.serve.RenameSession(s.Name, newName); err == nil {
					t.refresh()
				} else {
					t.setStatusTemporary("[" + colorRed + "]rename failed[-]")
				}
			})
		})
		return nil
	case 'd':
		t.actOnSelected(t.showKillConfirmModal)
		return nil
	case 'c':
		t.actOnSelected(func(s domain.Session) {
			_ = clipboard.WriteAll("tmux attach -t " + s.Name)
			t.setStatusTemporary("[" + colorGreen + "]copied: tmux attach -t " + s.Name + "[-]")
		})
		return nil
	}
	switch e.Key() {
	case tcell.KeyEnter:
		t.actOnSelected(func(s domain.Session) {
			if err := t.serve.EnterSession(s.Name); err != nil {
				t.setStatusTemporary("[" + colorRed + "]enter failed: " + err.Error() + "[-]")
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

// blurSearchBar returns focus to the session list without clearing the query.
// Mirrors lazyssh: ESC only blurs the input, so the list (and its cursor)
// stays intact. Clear the filter by editing the field instead.
func (t *tui) blurSearchBar() {
	t.app.SetFocus(t.sessionList)
}

func (t *tui) refresh() {
	sessions, err := t.serve.ListSessions()
	if err != nil {
		// Only genuine faults reach here — the normal "0 sessions" no-server
		// state is translated to an empty list by the repository. Log the full
		// error so a real failure is diagnosable even though the footer only
		// shows a short summary.
		t.logger.Warnw("list sessions failed", "error", err)
		t.statusBar.SetStatus("[" + colorRed + "]tmux error: " + err.Error() + "[-]")
		t.allCache = nil
		t.sessionList.UpdateSessions(nil)
		t.details.RenderEmpty("No sessions")
		return
	}
	t.allCache = sessions
	t.applySortAndRender()
	t.syncDetails()
	t.refreshStatusBarHints()
}

// syncDetails mirrors the details pane to the list's current selection.
//
// tview's List.SetCurrentItem only fires SetChangedFunc when the index actually
// changes, and UpdateSessions Clear()s the list first (which resets the cursor
// to 0) before calling SetCurrentItem(0) — so programmatic reloads never trigger
// handleSelectionChange. After a kill/create/rename/refresh the right pane would
// otherwise keep showing stale data, so we re-sync explicitly here.
func (t *tui) syncDetails() {
	if s, ok := t.sessionList.GetSelected(); ok {
		t.handleSelectionChange(s)
		return
	}
	t.details.RenderEmpty("No sessions")
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

// setStatusTemporary shows a transient footer message, then reverts to the
// default keybinding hints after statusToastTimeout. Any pending toast is
// cancelled first so rapid presses reuse a single timer instead of stacking.
// The reset runs via QueueUpdateDraw because time.AfterFunc fires on its own
// goroutine and tview widgets are not concurrency-safe.
func (t *tui) setStatusTemporary(msg string) {
	t.statusBar.SetStatus(msg)
	if t.statusTimer != nil {
		t.statusTimer.Stop()
	}
	t.statusTimer = time.AfterFunc(statusToastTimeout, func() {
		t.app.QueueUpdateDraw(t.refreshStatusBarHints)
	})
}

// refreshStatusBarHints restores the footer line appropriate for the current
// list state. After a transient toast ("Refreshed", "copied: ...") the timer
// fires on its own goroutine, so we route through QueueUpdateDraw and pick the
// empty-state or full hint set based on whether any session is loaded.
func (t *tui) refreshStatusBarHints() {
	if len(t.allCache) == 0 {
		t.statusBar.ShowEmpty()
		return
	}
	t.statusBar.ResetHints()
}

// showKillConfirmModal asks the user to confirm before killing a session,
// mirroring lazyssh's delete-confirm modal. Cancel is the safe default:
// the focused button and ESC both land on Cancel, so a stray Enter can't
// destroy a session. Confirm with k/K or by focusing Kill + Enter.
func (t *tui) showKillConfirmModal(s domain.Session) {
	msg := fmt.Sprintf("Kill session %s?\n\nThis action cannot be undone.", s.Name)
	modal := tview.NewModal().
		SetText(msg).
		AddButtons([]string{
			"[" + colorAccent + "]C[-]ancel",
			"[" + colorRed + "]K[-]ill",
		}).
		SetDoneFunc(func(buttonIndex int, _ string) {
			if buttonIndex == 1 {
				t.killSession(s)
			}
			t.closeModal()
		})
	// Letter shortcuts mirror the buttons. ESC falls through to SetDoneFunc
	// with buttonIndex -1, which matches no branch and therefore cancels.
	modal.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		switch e.Rune() {
		case 'c', 'C':
			t.closeModal()
			return nil
		case 'k', 'K':
			t.killSession(s)
			t.closeModal()
			return nil
		}
		return e
	})
	t.app.SetRoot(modal, true)
}

// killSession runs `tmux kill-session` and reports the outcome on the footer.
func (t *tui) killSession(s domain.Session) {
	if err := t.serve.KillSession(s.Name); err == nil {
		t.refresh()
	} else {
		t.setStatusTemporary("[" + colorRed + "]kill failed[-]")
	}
}

// closeModal restores the main layout after a modal dialog.
func (t *tui) closeModal() {
	t.app.SetRoot(t.root, true)
	t.app.SetFocus(t.sessionList)
}
