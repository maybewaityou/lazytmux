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
	"errors"
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

// Write methods implemented in Task 5.
func (r *repository) CreateSession(name string) error { return errNotImplemented }
func (r *repository) KillSession(name string) error   { return errNotImplemented }
func (r *repository) RenameSession(_, _ string) error { return errNotImplemented }
func (r *repository) SwitchOrAttach(_ string) error   { return errNotImplemented }

var errNotImplemented = errors.New("not implemented yet")

// ensure os import stays referenced (used by Task 5's env detection).
var _ = os.Getenv
