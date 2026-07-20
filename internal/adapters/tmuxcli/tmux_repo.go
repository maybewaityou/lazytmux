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

package tmuxcli

import (
	"os"

	"github.com/maybewaityou/lazytmux/internal/core/domain"
	"github.com/maybewaityou/lazytmux/internal/core/ports"
)

// format strings kept as constants so tests can assert exact tmux invocation.
const (
	sessionsFormat = "#{session_id}|#{session_name}|#{session_attached}|" +
		"#{session_created}|#{session_activity}|#{session_windows}|#{session_path}"
	windowsFormat = "#{window_index}|#{window_name}|#{window_active}|#{pane_current_command}"
)

type repository struct {
	runner CommandRunner
}

// NewRepository builds a SessionRepository backed by the tmux CLI.
func NewRepository(runner CommandRunner) ports.SessionRepository {
	return &repository{runner: runner}
}

func (r *repository) ListSessions() ([]domain.Session, error) {
	out, err := r.runner.RunOutput("list-sessions", "-F", sessionsFormat)
	if err != nil {
		return nil, err
	}
	return ParseSessions(out)
}

func (r *repository) ListWindows(sessionName string) ([]domain.Window, error) {
	out, err := r.runner.RunOutput("list-windows", "-t", sessionName, "-F", windowsFormat)
	if err != nil {
		return nil, err
	}
	return ParseWindows(out)
}

func (r *repository) CreateSession(name string) error {
	_, err := r.runner.RunOutput("new-session", "-d", "-s", name)
	return err
}

func (r *repository) KillSession(name string) error {
	_, err := r.runner.RunOutput("kill-session", "-t", name)
	return err
}

func (r *repository) RenameSession(oldName, newName string) error {
	_, err := r.runner.RunOutput("rename-session", "-t", oldName, newName)
	return err
}

// SwitchOrAttach either switches the tmux client (when already inside tmux) or
// signals the caller to suspend the TUI for an interactive attach.
func (r *repository) SwitchOrAttach(name string) error {
	if os.Getenv("TMUX") != "" {
		_, err := r.runner.RunOutput("switch-client", "-t", name)
		return err
	}
	return ports.ErrSuspendRequired
}

// AttachInteractive performs the out-of-tmux interactive attach. Called by the
// service after it suspends the TUI.
func (r *repository) AttachInteractive(name string) error {
	return r.runner.RunInteractive("attach", "-t", name)
}
