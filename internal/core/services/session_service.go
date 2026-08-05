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

package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/maybewaityou/lazytmux/internal/core/domain"
	"github.com/maybewaityou/lazytmux/internal/core/ports"
)

type service struct {
	repo        ports.SessionRepository
	meta        ports.MetadataStore
	snapshotter ports.SessionSnapshotter
	terminator  ports.SessionTerminator
	suspend     ports.SuspendFunc
}

// NewSessionService builds the business-logic service. suspend is wired
// positionally for tests; main.go uses SetSuspend after constructing the TUI.
func NewSessionService(
	repo ports.SessionRepository,
	meta ports.MetadataStore,
	snapshotter ports.SessionSnapshotter,
	terminator ports.SessionTerminator,
	suspend ports.SuspendFunc,
) ports.SessionService {
	return &service{
		repo:        repo,
		meta:        meta,
		snapshotter: snapshotter,
		terminator:  terminator,
		suspend:     suspend,
	}
}

// SetSuspend wires the TUI's suspend function after construction.
func (s *service) SetSuspend(fn ports.SuspendFunc) { s.suspend = fn }

func (s *service) ListSessions() ([]domain.Session, error) {
	sessions, err := s.repo.ListSessions()
	if err != nil {
		return nil, err
	}
	for i := range sessions {
		sessions[i].Pinned = s.meta.IsPinned(sessions[i].Name)
		sessions[i].Tags = s.meta.Tags(sessions[i].Name)
		sessions[i].LastAttached, _ = s.meta.LastAttached(sessions[i].Name)
		sessions[i].Note = s.meta.Note(sessions[i].Name)
	}
	return sessions, nil
}

func (s *service) LoadWindows(sess *domain.Session) error {
	ws, err := s.repo.ListWindows(sess.Name)
	if err != nil {
		return err
	}
	sess.Windows = ws
	return nil
}

func (s *service) CreateSession(name string) error {
	if err := s.repo.CreateSession(name); err != nil {
		return err
	}
	if s.snapshotter != nil {
		s.snapshotter.SaveSession(name)
	}
	return nil
}

func (s *service) KillSession(name string) error {
	if s.terminator == nil {
		return fmt.Errorf("session terminator is unavailable")
	}
	return s.terminator.KillSession(name)
}

func (s *service) DetachSession(name string) error { return s.repo.DetachSession(name) }
func (s *service) RenameSession(o, n string) error {
	if err := s.repo.RenameSession(o, n); err != nil {
		return err
	}
	// tmux rename succeeded — migrate UI metadata so pin/tags/lastAttached
	// follow the session instead of dangling under the old name. Best-effort:
	// a write failure is ignored to mirror EnterSession's handling of
	// SetLastAttached (the primary tmux state change already succeeded).
	_ = s.meta.Rename(o, n)
	return nil
}

func (s *service) TogglePin(name string) error {
	return s.meta.SetPinned(name, !s.meta.IsPinned(name))
}

func (s *service) SaveTags(name string, tags []string) error {
	return s.meta.SetTags(name, tags)
}

func (s *service) SaveNote(name string, note string) error {
	return s.meta.SetNote(name, note)
}

func (s *service) LastAttached(name string) (time.Time, bool) {
	return s.meta.LastAttached(name)
}

func (s *service) CurrentSession() (string, bool) { return s.repo.CurrentSession() }

func (s *service) EnterSession(name string) error {
	err := s.repo.SwitchOrAttach(name)
	if err == nil {
		return s.meta.SetLastAttached(name)
	}
	if errors.Is(err, ports.ErrSuspendRequired) {
		if s.suspend == nil {
			return err
		}
		var attachErr error
		susErr := s.suspend(func() error {
			attachErr = s.repo.AttachInteractive(name)
			return attachErr
		})
		if susErr != nil {
			return susErr
		}
		if attachErr == nil {
			_ = s.meta.SetLastAttached(name)
		}
		return attachErr
	}
	return err
}

var _ ports.SessionService = (*service)(nil)
