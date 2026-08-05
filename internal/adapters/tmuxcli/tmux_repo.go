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
	"strings"

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
		// tmux has no long-running daemon: its server starts with the first
		// session and exits with the last, so the "0 sessions" state surfaces
		// here as a "no server running" failure rather than empty stdout. That
		// is a normal empty result, not a fault — swallow it so the UI renders
		// its empty state instead of a red error line.
		if isNoServerError(err) {
			return nil, nil
		}
		return nil, err
	}
	return ParseSessions(out)
}

// isNoServerError reports whether err is tmux reporting that no server is
// running — i.e. the normal "0 sessions" state. tmux's wording varies across
// versions and platforms ("no server running on ..." on Linux, "error
// connecting to ... (No such file or directory)" on macOS), so either is
// matched. A missing server is semantically equivalent to having no sessions.
func isNoServerError(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no server running") ||
		strings.Contains(msg, "error connecting to")
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

// DetachSession disconnects the client attached to the named session via
// `tmux detach-client -s <name>`. The session itself keeps running in the
// background. detach-client acts on the tmux server, so unlike attach there is
// no $TMUX branch and no interactive suspend — it works the same inside or
// outside tmux. A session with no client surfaces as a tmux error, which the
// caller surfaces as a short "detach failed" toast.
func (r *repository) DetachSession(name string) error {
	_, err := r.runner.RunOutput("detach-client", "-s", name)
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

// CurrentSession resolves the tmux session the lazytmux process is attached to,
// via `tmux display-message -p '#S'`. Outside tmux ($TMUX unset) there is no such
// session. The lookup is best-effort: any tmux failure degrades to ("", false) so
// this decorative feature never disturbs list rendering.
func (r *repository) CurrentSession() (string, bool) {
	if os.Getenv("TMUX") == "" {
		return "", false
	}
	out, err := r.runner.RunOutput("display-message", "-p", "#S")
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(out)), true
}
