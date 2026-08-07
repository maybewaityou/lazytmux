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
	"time"

	"github.com/rivo/tview"
	"go.uber.org/zap"

	"github.com/maybewaityou/lazytmux/internal/core/domain"
	"github.com/maybewaityou/lazytmux/internal/core/ports"
)

// App is the runnable TUI.
type App interface {
	Run() error
}

type tui struct {
	logger *zap.SugaredLogger

	version string
	commit  string

	app   *tview.Application
	serve ports.SessionService

	header      *AppHeader
	searchBar   *SearchBar
	sessionList *SessionList
	details     *SessionDetails
	statusBar   *StatusBar

	root    *tview.Flex
	content *tview.Flex

	sortMode    SortMode
	allCache    []domain.Session
	statusTimer *time.Timer

	// loading is the active loading overlay, or nil when none is shown. It is
	// touched only on the tview main loop — showLoading/hideLoading run there,
	// and the spinner advance is routed through queueDraw — so it needs no mutex.
	loading *LoadingOverlay
	// loadingDone is closed to terminate the current overlay's spinner goroutine.
	// nil means no spinner is running. See showLoading for why a done channel is
	// used instead of Ticker.Stop alone (Ticker.Stop leaves the channel open and
	// would leak the goroutine).
	loadingDone chan struct{}

	// selectionGen tags each selection change so an in-flight async window load
	// can detect that a newer selection has superseded it (see loadWindowsAndRender).
	selectionGen uint64
	// queueDraw schedules a function on the tview main loop; overridable in tests
	// so an async render can be driven synchronously.
	queueDraw func(func())
	// currentSession is the tmux session the lazytmux process is currently in
	// (empty = not inside tmux). It is stable after reading once — switching
	// sessions necessarily exits the TUI.
	currentSession string
	// tagFilter holds the currently active tag filter (OR semantics). Empty/nil
	// means no filter. It is pure in-memory view state — never persisted — so a
	// fresh launch always starts unfiltered.
	tagFilter []string
}

// NewTUI constructs the TUI. Sub-components are built in Run() via the
// builder chain.
func NewTUI(logger *zap.SugaredLogger, svc ports.SessionService, version, commit string) App {
	return &tui{logger: logger, serve: svc, version: version, commit: commit}
}

func (t *tui) Run() error {
	defer func() {
		if r := recover(); r != nil {
			t.logger.Errorw("panic recovered", "error", r)
		}
	}()
	t.app = initializeTheme()
	t.app.EnableMouse(true)
	t.queueDraw = func(f func()) { t.app.QueueUpdateDraw(f) }
	t.buildComponents().buildLayout().bindEvents().loadInitialData()
	t.app.SetRoot(t.root, true)
	t.focusList()
	t.logger.Infow("starting TUI", "version", t.version, "commit", t.commit)
	if err := t.app.Run(); err != nil {
		t.logger.Errorw("application run error", "error", err)
		return err
	}
	return nil
}

// Suspend implements ports.SuspendFunc for out-of-tmux attach. It adapts
// tview's Suspend(func()) bool to the func()-returns-error shape the service uses.
//
// Once the `tmux attach` subprocess returns, the user has left the session
// (detached, exited, killed — or, rarely, the session vanished and attach
// failed fast). In every one of those cases lazytmux is done: the Enter handler
// quits the app regardless, so we Stop *now*, from inside the suspend callback,
// unconditionally on return. tview's Suspend then sees the screen was finalized
// during the callback and skips the Resume() it would otherwise perform. That
// Resume is pure waste here (re-enter the alt screen, hide the cursor, enable
// mouse, clear, … only for Stop/Fini to tear it straight back down), and the
// needless engage→disengage burst right at exit was confirmed to leave some
// real terminals with a screen mode half-restored, so the shell looked frozen
// until the user hit Ctrl-C. Stopping from inside the callback makes the attach
// path exit like a plain quit: one disengage, no spurious re-engagement.
//
// We Stop whether the subprocess reported success or failure: a non-zero exit
// is how tmux signals a normal session-end (e.g. exiting the last window) on
// some configs, not necessarily an error, and even a genuine attach failure
// already prints its cause to the terminal via tmux's own stderr. Note this
// only covers the case where `tmux attach` returns at all; if the session
// persists (other windows/panes still alive) the subprocess blocks until the
// user detaches, which is correct — they are still inside tmux.
func (t *tui) Suspend(fn func() error) error {
	var runErr error
	if !t.app.Suspend(func() {
		runErr = fn()
		t.logger.Infow("attach subprocess returned", "err", runErr)
		t.app.Stop()
	}) {
		return fmt.Errorf("failed to suspend TUI for attach")
	}
	return runErr
}

func (t *tui) buildComponents() *tui {
	t.header = NewAppHeader(t.version, t.commit, RepoURL)
	t.searchBar = NewSearchBar().
		OnSearch(t.handleSearchInput).
		OnEscape(t.blurSearchBar).
		OnNavigate(func() { t.focusList() })
	t.sessionList = NewSessionList().
		OnSelectionChange(t.handleSelectionChange).
		OnReturnToSearch(func() { t.app.SetFocus(t.searchBar) })
	t.details = NewSessionDetails()
	t.statusBar = NewStatusBar()
	t.sortMode = SortByNameAsc
	return t
}

func (t *tui) buildLayout() *tui {
	left := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(t.searchBar, 3, 0, true).
		AddItem(t.sessionList, 0, 1, false)
	right := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(t.details, 0, 1, false)

	t.content = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(left, 0, 3, true).
		AddItem(right, 0, 2, false)

	t.root = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(t.header, 2, 0, false).
		AddItem(t.content, 0, 1, true).
		AddItem(t.statusBar, 1, 0, false)
	return t
}

func (t *tui) bindEvents() *tui {
	t.root.SetInputCapture(t.handleGlobalKeys)
	return t
}

func (t *tui) loadInitialData() *tui {
	// The current session must be injected before the first refresh so the
	// first paint already carries the ▶ / (current) markers; cursor
	// positioning must happen after refresh, since UpdateSessions always
	// resets the cursor to the first item.
	t.currentSession, _ = t.serve.CurrentSession()
	t.sessionList.SetCurrent(t.currentSession)
	t.details.SetCurrent(t.currentSession)
	t.refresh()
	if t.currentSession != "" {
		t.sessionList.SelectByName(t.currentSession)
	}
	return t
}

// currentSearchQuery returns the search bar's text, or "" when the search bar
// is not wired (e.g. in unit tests constructing a partial tui).
func (t *tui) currentSearchQuery() string {
	if t.searchBar == nil {
		return ""
	}
	return t.searchBar.GetText()
}

// visibleSessions applies the active tag filter and the current search query to
// the cached session list. It is the single filtering entry point: both the
// sort/render path (applySortAndRender) and the search path (handleSearchInput)
// render its result, so the two filters always compose consistently.
func (t *tui) visibleSessions() []domain.Session {
	return applyFilters(t.allCache, t.tagFilter, t.currentSearchQuery())
}
