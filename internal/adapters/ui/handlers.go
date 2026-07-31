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
	"github.com/maybewaityou/lazytmux/internal/core/ports"
)

const colorRed = "#f7768e"

// statusToastTimeout is how long a transient footer message (e.g. "Refreshed")
// stays visible before reverting to the default keybinding hints.
const statusToastTimeout = 3 * time.Second

func (t *tui) handleGlobalKeys(e *tcell.EventKey) *tcell.EventKey {
	// When the search bar is focused, let it handle all keys (typing).
	if t.searchBarHasFocus() {
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
		t.openMultiFieldSessionForm("New session", "", "", "", t.createAndEnter)
		return nil
	case 'e':
		t.actOnSelected(func(s domain.Session) {
			t.openMultiFieldSessionForm("Edit session", s.Name, strings.Join(s.Tags, ", "), s.Note, func(name, tags, note string) {
				// Rename only when the name actually changed: a no-op rename
				// (same name) can error on some tmux versions. A successful
				// rename migrates pin/tags/note/lastAttached to the new name,
				// so the SaveTags/SaveNote below overwrite at the right key.
				if name != s.Name {
					if err := t.serve.RenameSession(s.Name, name); err != nil {
						t.setStatusTemporary("[" + colorRed + "]Rename failed[-]")
						t.closeForm()
						return
					}
				}
				_ = t.serve.SaveTags(name, parseTags(tags))
				_ = t.serve.SaveNote(name, strings.TrimSpace(note))
				t.refresh()
				t.closeForm()
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
			t.openForm("Tags", "comma-separated tags", strings.Join(s.Tags, ", "), true, func(input string) {
				tags := parseTags(input)
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
	case 'n':
		t.actOnSelected(t.editNote)
		return nil
	case 'f':
		t.openTagFilter()
		return nil
	case '?':
		t.openHelp()
		return nil
	}
	switch e.Key() {
	case tcell.KeyRight:
		// List → Details: arrow into the right-hand pane. The list's own capture
		// no longer swallows Right (it used to return to the search bar), so the
		// key bubbles up to this global handler.
		if t.listHasFocus() {
			t.focusDetails()
			return nil
		}
	case tcell.KeyLeft:
		// Details → List: arrow back to the session list.
		if t.detailsHasFocus() {
			t.focusList()
			return nil
		}
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

func (t *tui) handleSearchInput(_ string) {
	// The query is read fresh from the search bar inside visibleSessions, so the
	// tag filter and name search always compose through one pipeline. Routed
	// through renderVisibleList so the details pane re-syncs to the filtered
	// list — see renderVisibleList for why the sync cannot rely on SetChangedFunc.
	t.renderVisibleList()
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
	t.focusList()
}

// focusDetails moves focus to the right-hand details pane. The pane keeps its
// normal border styling on focus — it intentionally matches how the session
// list looks when focused, rather than highlighting, so the two panes read as
// one consistent surface.
func (t *tui) focusDetails() {
	t.app.SetFocus(t.details)
}

// focusList returns focus to the session list. Centralizing the list as the
// "home" focus target keeps every return-to-list path — arrow-back from
// details, ESC from the search bar, closing a form or modal — in one place.
func (t *tui) focusList() {
	t.app.SetFocus(t.sessionList)
}

// listHasFocus reports whether the session list currently holds focus.
//
// Focus is queried via the widget's own HasFocus() rather than by comparing the
// *tview.Application's focused-primitive pointer. Pointer comparison is fragile
// across tview's focus layers: a keyboard SetFocus lands on the *SessionList
// wrapper, a mouse click lands on the embedded *tview.List, and — as of tview
// v0.42.0 — some widgets delegate mouse focus to an inner, unexported widget
// (InputField does this via its *TextArea). Enumerating every layer's pointer is
// brittle and, for the unexported ones, impossible; HasFocus recurses through
// them uniformly and works headless (no live *tview.Application).
func (t *tui) listHasFocus() bool {
	return t.sessionList.HasFocus()
}

// detailsHasFocus reports whether the details pane currently holds focus. Same
// HasFocus-based reasoning as listHasFocus.
func (t *tui) detailsHasFocus() bool {
	return t.details.HasFocus()
}

// searchBarHasFocus reports whether the search bar currently holds focus. This
// is the fix for the bug where clicking the search bar left it reporting
// "unfocused": tview v0.42.0's InputField delegates mouse focus to an internal
// *TextArea, so the focused primitive was neither the *SearchBar wrapper nor the
// embedded *tview.InputField — the old pointer comparison matched neither, and
// the TextArea field is unexported so it could not be named anyway. HasFocus
// recurses through that delegation. See listHasFocus for the full reasoning.
func (t *tui) searchBarHasFocus() bool {
	return t.searchBar.HasFocus()
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

// renderVisibleList re-renders the session list from the filtered pipeline and
// re-syncs the details pane to the resulting selection. It is the single entry
// point that every "visible set changed" path — search input, sort cycle, tag
// filter — must go through.
//
// The sync cannot be left to tview: UpdateSessions clears the list and resets the
// cursor to item 0, but List.Clear()+SetCurrentItem(0) never fires SetChangedFunc,
// so handleSelectionChange is never invoked and the details pane stays pinned to
// whatever was selected before the filter changed. When the filter matches
// nothing the list empties and the pane would keep showing the last match
// indefinitely (the reported bug). syncDetails clears (or re-points) the pane to
// match the new list state.
//
// refresh() runs this via applySortAndRender and then re-syncs once more after
// SelectByName restores the saved cursor; the two passes are intentionally
// redundant — syncDetails is idempotent — and each one leaves the pane
// self-consistent at its point in the flow.
func (t *tui) renderVisibleList() {
	t.sessionList.UpdateSessions(t.visibleSessions())
	t.syncDetails()
}

func (t *tui) applySortAndRender() {
	sortSessionsForUI(t.allCache, t.sortMode, t.serve.LastAttached)
	t.sessionList.SetSortTitle(t.sortMode.String())
	// Title renders the active filter tags as chips (black on accent) so they
	// stand out from the surrounding title text. The help modal keeps the plain
	// filterDescription form — title wants a glanceable badge, help wants text.
	t.sessionList.SetFilter(renderTagChips(t.tagFilter))
	t.renderVisibleList()
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
	t.focusList()
}

// openMultiFieldSessionForm opens the shared Name / Tags / Note modal. New
// passes empty initial values and creates the session on submit; Edit passes the
// session's current values and (re)saves them on submit (renaming first if the
// name changed). The form guarantees a non-empty Name (Save is a no-op on an
// empty Name), so onSubmit never receives "". Metadata writes after Create /
// Rename are best-effort, mirroring RenameSession's tolerance of failures after
// the primary tmux action already succeeded.
func (t *tui) openMultiFieldSessionForm(title, initName, initTags, initNote string, onSubmit func(name, tags, note string)) {
	form := NewMultiFieldSessionForm(title).
		InitialValues(initName, initTags, initNote).
		OnSubmit(onSubmit).
		OnCancel(t.closeForm)
	t.app.SetRoot(form.Primitive(), true)
	t.app.SetFocus(form.Form())
}

// createAndEnterAction is the outcome of the 'a'-key create-then-enter flow.
// planCreateAndEnter decides it purely from service results so the branching is
// unit-testable headlessly; the handler only translates it into tview effects.
type createAndEnterAction int

const (
	actCreateFailed createAndEnterAction = iota // CreateSession errored.
	actEnterFailed                              // EnterSession errored, session was created.
	actEntered                                  // success: entered the new session.
)

// planCreateAndEnter runs the service side of the 'a'-key flow: create the
// session, best-effort persist tags/note, then enter it. It holds no tview
// state and returns the outcome plus the failing error (nil on success) so the
// call order and every branch are unit-testable without a live event loop.
func planCreateAndEnter(svc ports.SessionService, name, tags, note string) (createAndEnterAction, error) {
	if err := svc.CreateSession(name); err != nil {
		return actCreateFailed, err
	}
	if parsed := parseTags(tags); len(parsed) > 0 {
		_ = svc.SaveTags(name, parsed)
	}
	if note = strings.TrimSpace(note); note != "" {
		_ = svc.SaveNote(name, note)
	}
	if err := svc.EnterSession(name); err != nil {
		return actEnterFailed, err
	}
	return actEntered, nil
}

// createAndEnter is the 'a'-key submit flow: create the session, persist its
// tags/note, then enter it, quitting the TUI on success (we are now inside the
// new session). A create failure reuses the existing wording; an enter failure
// (rare for a just-created session) refreshes the list so the new session is
// visible and surfaces the error, leaving the user in the TUI to retry.
func (t *tui) createAndEnter(name, tags, note string) {
	act, err := planCreateAndEnter(t.serve, name, tags, note)
	switch act {
	case actCreateFailed:
		t.setStatusTemporary("[" + colorRed + "]Create failed[-]")
		t.closeForm()
	case actEnterFailed:
		t.refresh()
		t.setStatusTemporary("[" + colorRed + "]Enter failed: " + err.Error() + "[-]")
		t.closeForm()
	case actEntered:
		t.app.Stop()
	}
}

// editNote opens a multi-line text-area modal to edit the selected session's
// note — the same *tview.TextArea the Note field of the New/Edit form uses, so
// editing feels identical standalone and inside the form. Enter saves;
// Shift+Enter inserts a newline; Esc cancels. An empty submit clears the note
// (no confirm modal): clearing freeform text is trivially reversible, so the
// confirm gate that tags get would be ceremony, not safety. This is why it
// does not reuse openForm — openForm's allowEmpty path assumes the empty case
// takes over the root to show a confirm modal (tags-only), which note never
// does. NoteForm calls onSubmit on every plain Enter, giving the
// always-submit-and-close semantics note needs.
func (t *tui) editNote(s domain.Session) {
	form := NewNoteForm("Note", "freeform note").
		InitialValue(s.Note).
		OnSubmit(func(input string) {
			note := strings.TrimSpace(input)
			if err := t.serve.SaveNote(s.Name, note); err == nil {
				t.refresh()
			} else {
				t.setStatusTemporary("[" + colorRed + "]Note failed[-]")
			}
			t.closeForm()
		}).
		OnCancel(t.closeForm)
	t.app.SetRoot(form.Primitive(), true)
	t.app.SetFocus(form.Area())
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

// openTagFilter opens the multi-select tag filter modal. Candidates are the
// union of every loaded session's tags; the currently active filter is
// pre-selected. Applying writes the selection to tagFilter (in-memory only)
// and re-runs the render pipeline so the list and its title update together.
// When there are no tags at all, we surface a transient hint instead of an
// empty modal.
func (t *tui) openTagFilter() {
	candidates := collectTags(t.allCache)
	if len(candidates) == 0 {
		t.setStatusTemporary("[" + colorYellow + "]No tags yet — tag a session with t[-]")
		return
	}
	form := NewTagFilterForm(candidates, t.tagFilter).
		OnApply(func(tags []string) {
			t.tagFilter = tags
			t.closeModal()
			t.applySortAndRender()
		}).
		OnCancel(t.closeModal)
	t.app.SetRoot(form.Primitive(), true)
	t.app.SetFocus(form)
}

// openHelp shows the key-binding reference. The content is derived from the
// keyBindings single source, topped with the current sort + filter status.
// ?, ESC, or q dismiss it.
func (t *tui) openHelp() {
	help := NewHelpModal(t.sortMode.String(), filterDescription(t.tagFilter))
	help.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		switch {
		case e.Key() == tcell.KeyESC, e.Rune() == '?', e.Rune() == 'q':
			t.closeModal()
			return nil
		}
		return e
	})
	flex := tview.NewFlex().AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 1, 0, false).
			AddItem(help, 0, 1, true).
			AddItem(nil, 1, 0, false), 64, 0, true).
		AddItem(nil, 0, 1, false)
	t.app.SetRoot(flex, true)
	t.app.SetFocus(help.focus)
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
	t.focusList()
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

// parseTags splits a tag input string on commas — accepting both the ASCII ","
// and the fullwidth Chinese "，" — trims surrounding whitespace from each token,
// and drops empty tokens. Unlike strings.Fields, spaces are NOT separators: a
// tag may itself contain spaces (e.g. "release v2"). Extracted as a pure
// function so the parsing can be unit-tested like clearedTagsMessage.
func parseTags(input string) []string {
	raw := strings.FieldsFunc(input, func(r rune) bool {
		return r == ',' || r == '，'
	})
	tags := make([]string, 0, len(raw))
	for _, tag := range raw {
		if tag = strings.TrimSpace(tag); tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags
}
