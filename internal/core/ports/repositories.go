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

package ports

import "github.com/maybewaityou/lazytmux/internal/core/domain"

// SessionRepository is the data-source port (implemented by the tmux CLI adapter).
type SessionRepository interface {
	ListSessions() ([]domain.Session, error)
	ListWindows(sessionName string) ([]domain.Window, error)
	CreateSession(name string) error
	// DetachSession disconnects the client attached to the named session via
	// `tmux detach-client -s <name>`. The session itself keeps running. Unlike
	// SwitchOrAttach, this targets the tmux server directly, so it works whether
	// or not lazytmux runs inside tmux.
	DetachSession(name string) error
	RenameSession(oldName, newName string) error
	// SwitchOrAttach runs the appropriate "enter" command. It returns a sentinel
	// error ErrSuspendRequired when the TUI must be suspended for an interactive
	// attach (i.e. not running inside tmux).
	SwitchOrAttach(name string) error
	// AttachInteractive runs `tmux attach -t <name>` against the parent's stdio.
	// Called by the service only after it has suspended the TUI (ErrSuspendRequired path).
	AttachInteractive(name string) error
	// CurrentSession returns the tmux session the lazytmux process is attached to.
	// ok is false when the process is not inside tmux (or detection failed); the UI
	// then omits the "current session" markers.
	CurrentSession() (name string, ok bool)
}

// SessionSnapshotter best-effort persists tmux state after a session is created.
// Implementations diagnose their own failures so snapshotting cannot turn a
// successful tmux state change into a misleading CreateSession error.
type SessionSnapshotter interface {
	SaveSession(name string)
}

// SessionTerminator performs the primary tmux kill and best-effort persistence
// reconciliation. Its error reports only whether the primary kill succeeded.
type SessionTerminator interface {
	KillSession(name string) error
}

// ErrSuspendRequired signals that the caller must suspend the TUI and run an
// interactive attach out-of-band (used when $TMUX is unset).
var ErrSuspendRequired = errSentinel("suspend required for interactive attach")

type errSentinel string

func (e errSentinel) Error() string { return string(e) }
