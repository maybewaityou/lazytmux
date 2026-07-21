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

import (
	"time"

	"github.com/maybewaityou/lazytmux/internal/core/domain"
)

// MetadataStore is the persistence port for UI-only metadata (pins/tags/etc.).
type MetadataStore interface {
	IsPinned(name string) bool
	SetPinned(name string, pinned bool) error
	Tags(name string) []string
	SetTags(name string, tags []string) error
	SetLastAttached(name string) error
	LastAttached(name string) (time.Time, bool)
}

// SuspendFunc suspends the TUI, runs the callback interactively, then resumes.
// Implemented by the UI adapter (tview app.Suspend).
type SuspendFunc func(fn func() error) error

// SessionService is the business-logic port.
type SessionService interface {
	ListSessions() ([]domain.Session, error)
	LoadWindows(s *domain.Session) error
	CreateSession(name string) error
	KillSession(name string) error
	DetachSession(name string) error
	RenameSession(oldName, newName string) error
	EnterSession(name string) error
	TogglePin(name string) error
	SaveTags(name string, tags []string) error
	LastAttached(name string) (time.Time, bool)
	CurrentSession() (name string, ok bool)
}
