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

	// selectionGen tags each selection change so an in-flight async window load
	// can detect that a newer selection has superseded it (see loadWindowsAndRender).
	selectionGen uint64
	// queueDraw schedules a function on the tview main loop; overridable in tests
	// so an async render can be driven synchronously.
	queueDraw func(func())
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
	t.app.SetFocus(t.sessionList)
	t.logger.Infow("starting TUI", "version", t.version, "commit", t.commit)
	if err := t.app.Run(); err != nil {
		t.logger.Errorw("application run error", "error", err)
		return err
	}
	return nil
}

// Suspend implements ports.SuspendFunc for out-of-tmux attach. It adapts
// tview's Suspend(func()) bool to the func()-returns-error shape the service uses.
func (t *tui) Suspend(fn func() error) error {
	var runErr error
	if !t.app.Suspend(func() { runErr = fn() }) {
		return fmt.Errorf("failed to suspend TUI for attach")
	}
	return runErr
}

func (t *tui) buildComponents() *tui {
	t.header = NewAppHeader(t.version, t.commit, RepoURL)
	t.searchBar = NewSearchBar().
		OnSearch(t.handleSearchInput).
		OnEscape(t.blurSearchBar).
		OnNavigate(func() { t.app.SetFocus(t.sessionList) })
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
	t.refresh()
	return t
}
