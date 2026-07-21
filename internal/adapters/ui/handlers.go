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

// statusToastTimeout is how long a transient footer message (e.g. "Refreshed")
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
		count := t.refresh()
		t.setStatusTemporary(refreshStatusMessage(count))
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
				t.setStatusTemporary("[" + colorRed + "]Pin failed[-]")
			}
		})
		return nil
	case 'a':
		t.openForm("New session", "session name", "", false, func(name string) {
			if err := t.serve.CreateSession(name); err == nil {
				t.refresh()
			} else {
				t.setStatusTemporary("[" + colorRed + "]Create failed[-]")
			}
		})
		return nil
	case 'e':
		t.actOnSelected(func(s domain.Session) {
			t.openForm("Rename", "new name", s.Name, false, func(newName string) {
				if err := t.serve.RenameSession(s.Name, newName); err == nil {
					t.refresh()
				} else {
					t.setStatusTemporary("[" + colorRed + "]Rename failed[-]")
				}
			})
		})
		return nil
	case 'k':
		t.actOnSelected(t.showKillConfirmModal)
		return nil
	case 'd':
		t.actOnSelected(t.showDetachConfirmModal)
		return nil
	case 'c':
		t.actOnSelected(func(s domain.Session) {
			_ = clipboard.WriteAll("tmux attach -t " + s.Name)
			t.setStatusTemporary("[" + colorGreen + "]Copied: tmux attach -t " + s.Name + "[-]")
		})
		return nil
	case 't':
		t.actOnSelected(func(s domain.Session) {
			t.openForm("Tags", "space-separated tags", strings.Join(s.Tags, " "), true, func(input string) {
				tags := strings.Fields(input)
				if len(tags) == 0 {
					t.showClearTagsConfirmModal(s)
					return
				}
				if err := t.serve.SaveTags(s.Name, tags); err == nil {
					t.refresh()
				} else {
					t.setStatusTemporary("[" + colorRed + "]Tags failed[-]")
				}
			})
		})
		return nil
	}
	switch e.Key() {
	case tcell.KeyEnter:
		t.actOnSelected(func(s domain.Session) {
			if err := t.serve.EnterSession(s.Name); err != nil {
				t.setStatusTemporary("[" + colorRed + "]Enter failed: " + err.Error() + "[-]")
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
	t.selectionGen++
	t.details.Render(s)
	// Windows come from a separate tmux call, so load them asynchronously and
	// re-render once they arrive.
	go t.loadWindowsAndRender(s, t.selectionGen)
}

// loadWindowsAndRender fetches windows for s and re-renders the details pane,
// but only if gen still matches the current selection generation. This drops
// stale in-flight loads — e.g. the first session shown at startup, or a session
// navigated past — whose slow LoadWindows completes after the selection has
// already moved on, which would otherwise overwrite the pane with stale data.
// The render is routed through queueDraw because tview widgets are not safe to
// touch from a goroutine.
func (t *tui) loadWindowsAndRender(s domain.Session, gen uint64) {
	cp := s
	_ = t.serve.LoadWindows(&cp)
	t.queueDraw(func() {
		if t.selectionGen != gen {
			return
		}
		t.details.Render(cp)
	})
}

// blurSearchBar returns focus to the session list without clearing the query.
// Mirrors lazyssh: ESC only blurs the input, so the list (and its cursor)
// stays intact. Clear the filter by editing the field instead.
func (t *tui) blurSearchBar() {
	t.app.SetFocus(t.sessionList)
}

// refresh reloads sessions from tmux, re-renders the list, and returns the
// session count so callers (the 'r' key) can surface it in the footer toast.
//
// The current selection is preserved by name across the reload: UpdateSessions
// always resets the cursor to the first item, so without this a refresh would
// snap the user back to the top of the list. If the selected session is gone,
// the cursor stays on the first item.
func (t *tui) refresh() int {
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
		return 0
	}
	prevName := ""
	if s, ok := t.sessionList.GetSelected(); ok {
		prevName = s.Name
	}
	t.allCache = sessions
	t.applySortAndRender()
	if prevName != "" {
		t.sessionList.SelectByName(prevName)
	}
	t.syncDetails()
	t.refreshStatusBarHints()
	return len(sessions)
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

func (t *tui) openForm(title, placeholder, initialValue string, allowEmpty bool, onSubmit func(string)) {
	form := NewSessionForm(title, placeholder).
		InitialValue(initialValue).
		OnSubmit(func(name string) {
			name = strings.TrimSpace(name)
			if name == "" && !allowEmpty {
				t.closeForm()
				return
			}
			onSubmit(name)
			if name == "" {
				// Empty + allowEmpty: onSubmit took over the root (opened a
				// confirm modal). Don't closeForm, or it would clobber the modal.
				return
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

// refreshStatusMessage builds the transient footer toast shown after pressing
// 'r'. A non-empty list reports how many sessions were refreshed; an empty list
// (no tmux server running / every session gone) gets a clearer line instead of
// the awkward "Refreshed 0 sessions".
func refreshStatusMessage(count int) string {
	if count == 0 {
		return "[" + colorCyan + "]No sessions to refresh[-]"
	}
	return fmt.Sprintf("["+colorGreen+"]Refreshed %d sessions[-]", count)
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
// list state. After a transient toast ("Refreshed", "Copied: ...") the timer
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
		t.setStatusTemporary("[" + colorRed + "]Kill failed[-]")
	}
}

// closeModal restores the main layout after a modal dialog.
func (t *tui) closeModal() {
	t.app.SetRoot(t.root, true)
	t.app.SetFocus(t.sessionList)
}

// showDetachConfirmModal asks the user to confirm before detaching a session.
// Detach is non-destructive — the session keeps running in the background — so
// the confirm button is green rather than kill's red. Cancel stays the safe
// default: the focused button and ESC both land on Cancel. Confirm with d/D or
// by focusing Detach + Enter (the trigger key doubles as the confirm key, same
// pattern as k for kill).
func (t *tui) showDetachConfirmModal(s domain.Session) {
	msg := fmt.Sprintf("Detach session %s?\n\nThe session keeps running in the background.", s.Name)
	modal := tview.NewModal().
		SetText(msg).
		AddButtons([]string{
			"[" + colorAccent + "]C[-]ancel",
			"[" + colorGreen + "]D[-]etach",
		}).
		SetDoneFunc(func(buttonIndex int, _ string) {
			if buttonIndex == 1 {
				t.detachSession(s)
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
		case 'd', 'D':
			t.detachSession(s)
			t.closeModal()
			return nil
		}
		return e
	})
	t.app.SetRoot(modal, true)
}

// detachSession runs `tmux detach-client` and reports the outcome on the footer.
// Unlike kill, a success toast is shown: the session does not disappear, so
// without explicit feedback the user could not tell whether anything happened.
func (t *tui) detachSession(s domain.Session) {
	if err := t.serve.DetachSession(s.Name); err == nil {
		t.refresh()
		t.setStatusTemporary("[" + colorGreen + "]Detached: " + s.Name + "[-]")
	} else {
		t.setStatusTemporary("[" + colorRed + "]Detach failed[-]")
	}
}

// showClearTagsConfirmModal asks the user to confirm before clearing all tags,
// mirroring kill/detach's confirm pattern. Cancel is the safe default. Confirm
// with t/T — the tag trigger key doubles as the confirm key, same pattern as k
// for kill and d for detach — or by focusing Clear tags + Enter. ESC falls
// through to SetDoneFunc with buttonIndex -1, which matches no branch and so
// cancels.
func (t *tui) showClearTagsConfirmModal(s domain.Session) {
	msg := fmt.Sprintf("Clear all tags for session %s?\n\nThe session itself is unaffected; only its tags are removed.", s.Name)
	modal := tview.NewModal().
		SetText(msg).
		AddButtons([]string{
			"[" + colorAccent + "]C[-]ancel",
			"[" + colorRed + "]Clear tags",
		}).
		SetDoneFunc(func(buttonIndex int, _ string) {
			if buttonIndex == 1 {
				t.clearTags(s)
			}
			t.closeModal()
		})
	modal.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		switch e.Rune() {
		case 'c', 'C':
			t.closeModal()
			return nil
		case 't', 'T':
			t.clearTags(s)
			t.closeModal()
			return nil
		}
		return e
	})
	t.app.SetRoot(modal, true)
}

// clearTags removes every tag for s and reports the outcome on the footer.
func (t *tui) clearTags(s domain.Session) {
	if err := t.serve.SaveTags(s.Name, nil); err == nil {
		t.refresh()
		t.setStatusTemporary(clearedTagsMessage(s.Name))
	} else {
		t.setStatusTemporary("[" + colorRed + "]Tags failed[-]")
	}
}

// clearedTagsMessage builds the transient footer toast shown after clearing a
// session's tags. Extracted as a pure function so it can be unit-tested like
// refreshStatusMessage.
func clearedTagsMessage(name string) string {
	return "[" + colorGreen + "]Cleared tags: " + name + "[-]"
}
