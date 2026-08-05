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
	"slices"
	"testing"
	"time"

	"github.com/maybewaityou/lazytmux/internal/core/domain"
	"github.com/maybewaityou/lazytmux/internal/core/ports"
)

type fakeRepo struct {
	sessions  []domain.Session
	windows   []domain.Window
	createErr error
	enterErr  error
	renameErr error
	calls     []string
	current   string
	currentOk bool
}

func (f *fakeRepo) ListSessions() ([]domain.Session, error) {
	f.calls = append(f.calls, "list-sessions")
	return f.sessions, nil
}
func (f *fakeRepo) ListWindows(string) ([]domain.Window, error) { return f.windows, nil }
func (f *fakeRepo) CreateSession(name string) error {
	f.calls = append(f.calls, "create:"+name)
	return f.createErr
}

func (f *fakeRepo) DetachSession(name string) error {
	f.calls = append(f.calls, "detach:"+name)
	return nil
}

func (f *fakeRepo) RenameSession(o, n string) error {
	f.calls = append(f.calls, "rename:"+o+"->"+n)
	return f.renameErr
}

func (f *fakeRepo) AttachInteractive(name string) error {
	f.calls = append(f.calls, "attach:"+name)
	return nil
}

func (f *fakeRepo) SwitchOrAttach(name string) error {
	f.calls = append(f.calls, "enter:"+name)
	return f.enterErr
}

func (f *fakeRepo) CurrentSession() (string, bool) {
	f.calls = append(f.calls, "current")
	return f.current, f.currentOk
}

type fakeSnapshotter struct {
	calls *[]string
}

func (f fakeSnapshotter) SaveSession(name string) {
	*f.calls = append(*f.calls, "snapshot:"+name)
}

type fakeTerminator struct {
	calls *[]string
	err   error
}

func (f fakeTerminator) KillSession(name string) error {
	*f.calls = append(*f.calls, "terminate:"+name)
	return f.err
}

type fakeMeta struct {
	pins              map[string]bool
	tags              map[string][]string
	notes             map[string]string
	lastAttachedCalls int
	renameCalls       [][2]string
}

func newFakeMeta() *fakeMeta {
	return &fakeMeta{pins: map[string]bool{}, tags: map[string][]string{}, notes: map[string]string{}}
}

func (m *fakeMeta) IsPinned(n string) bool { return m.pins[n] }
func (m *fakeMeta) SetPinned(n string, p bool) error {
	if p {
		m.pins[n] = true
	} else {
		delete(m.pins, n)
	}
	return nil
}
func (m *fakeMeta) Tags(n string) []string             { return m.tags[n] }
func (m *fakeMeta) SetTags(n string, t []string) error { m.tags[n] = t; return nil }
func (m *fakeMeta) Note(n string) string               { return m.notes[n] }
func (m *fakeMeta) SetNote(n, note string) error {
	if note == "" {
		delete(m.notes, n)
	} else {
		m.notes[n] = note
	}
	return nil
}
func (m *fakeMeta) SetLastAttached(n string) error          { m.lastAttachedCalls++; return nil }
func (m *fakeMeta) LastAttached(n string) (time.Time, bool) { return time.Time{}, false }
func (m *fakeMeta) Rename(oldName, newName string) error {
	m.renameCalls = append(m.renameCalls, [2]string{oldName, newName})
	if p, ok := m.pins[oldName]; ok {
		m.pins[newName] = p
		delete(m.pins, oldName)
	}
	if tg, ok := m.tags[oldName]; ok {
		m.tags[newName] = tg
		delete(m.tags, oldName)
	}
	if nt, ok := m.notes[oldName]; ok {
		m.notes[newName] = nt
		delete(m.notes, oldName)
	}
	return nil
}

func TestCreateSessionSavesSnapshotAfterCreating(t *testing.T) {
	repo := &fakeRepo{}
	snapshotter := fakeSnapshotter{calls: &repo.calls}
	svc := NewSessionService(repo, newFakeMeta(), snapshotter, nil, nil)

	if err := svc.CreateSession("work"); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	want := []string{"create:work", "snapshot:work"}
	if !slices.Equal(repo.calls, want) {
		t.Fatalf("CreateSession calls = %v, want %v", repo.calls, want)
	}
}

func TestCreateSessionSkipsSnapshotWhenCreateFails(t *testing.T) {
	createErr := errors.New("create failed")
	repo := &fakeRepo{createErr: createErr}
	svc := NewSessionService(repo, newFakeMeta(), fakeSnapshotter{calls: &repo.calls}, nil, nil)

	if err := svc.CreateSession("work"); !errors.Is(err, createErr) {
		t.Fatalf("CreateSession error = %v, want %v", err, createErr)
	}
	want := []string{"create:work"}
	if !slices.Equal(repo.calls, want) {
		t.Fatalf("CreateSession calls = %v, want %v", repo.calls, want)
	}
}

func TestCreateSessionAllowsNilSnapshotter(t *testing.T) {
	svc := NewSessionService(&fakeRepo{}, newFakeMeta(), nil, nil, nil)

	if err := svc.CreateSession("work"); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
}

func TestKillSessionUsesTerminator(t *testing.T) {
	calls := []string{}
	terminator := fakeTerminator{calls: &calls}
	svc := NewSessionService(&fakeRepo{}, newFakeMeta(), nil, terminator, nil)

	if err := svc.KillSession("work"); err != nil {
		t.Fatalf("KillSession: %v", err)
	}
	want := []string{"terminate:work"}
	if !slices.Equal(calls, want) {
		t.Fatalf("KillSession calls = %v, want %v", calls, want)
	}
}

func TestKillSessionReturnsTerminatorError(t *testing.T) {
	killErr := errors.New("kill failed")
	calls := []string{}
	terminator := fakeTerminator{calls: &calls, err: killErr}
	svc := NewSessionService(&fakeRepo{}, newFakeMeta(), nil, terminator, nil)

	if err := svc.KillSession("work"); !errors.Is(err, killErr) {
		t.Fatalf("KillSession error = %v, want %v", err, killErr)
	}
}

func TestListSessionsInjectsMetadata(t *testing.T) {
	repo := &fakeRepo{sessions: []domain.Session{{Name: "main"}, {Name: "dev"}}}
	meta := newFakeMeta()
	_ = meta.SetPinned("main", true)
	_ = meta.SetTags("dev", []string{"work"})
	_ = meta.SetNote("dev", "dev box")

	svc := NewSessionService(repo, meta, nil, nil, nil)
	got, err := svc.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if !got[0].Pinned || len(got[1].Tags) != 1 || got[1].Tags[0] != "work" {
		t.Fatalf("metadata not injected: %+v", got)
	}
	if got[1].Note != "dev box" {
		t.Fatalf("note not injected: got %q, want %q", got[1].Note, "dev box")
	}
	if got[0].Note != "" {
		t.Fatalf("absent note should default to empty, got %q", got[0].Note)
	}
}

func TestSaveNoteDelegates(t *testing.T) {
	meta := newFakeMeta()
	svc := NewSessionService(&fakeRepo{}, meta, nil, nil, nil)

	if err := svc.SaveNote("api", "main service"); err != nil {
		t.Fatalf("SaveNote: %v", err)
	}
	if got := meta.Note("api"); got != "main service" {
		t.Errorf("after SaveNote, meta.Note = %q, want %q", got, "main service")
	}
	// Saving an empty note clears it (store deletes the key).
	if err := svc.SaveNote("api", ""); err != nil {
		t.Fatalf("SaveNote(empty): %v", err)
	}
	if got := meta.Note("api"); got != "" {
		t.Errorf("empty SaveNote should clear note, got %q", got)
	}
}

func TestEnterSessionSwitchesDirectly(t *testing.T) {
	repo := &fakeRepo{enterErr: nil}
	svc := NewSessionService(repo, newFakeMeta(), nil, nil, nil)
	if err := svc.EnterSession("main"); err != nil {
		t.Fatalf("EnterSession: %v", err)
	}
	if repo.calls[len(repo.calls)-1] != "enter:main" {
		t.Errorf("expected direct enter, calls=%v", repo.calls)
	}
}

func TestEnterSessionSuspendsOnSignal(t *testing.T) {
	repo := &fakeRepo{enterErr: ports.ErrSuspendRequired}
	attached := false
	suspend := func(fn func() error) error {
		attached = true
		return fn()
	}
	svc := NewSessionService(repo, newFakeMeta(), nil, nil, suspend)
	if err := svc.EnterSession("main"); err != nil {
		t.Fatalf("EnterSession: %v", err)
	}
	if !attached {
		t.Fatal("expected suspend to be invoked")
	}
	if repo.calls[len(repo.calls)-1] != "attach:main" {
		t.Errorf("expected AttachInteractive after suspend, calls=%v", repo.calls)
	}
}

func TestCurrentSessionDelegates(t *testing.T) {
	repo := &fakeRepo{current: "work", currentOk: true}
	svc := NewSessionService(repo, newFakeMeta(), nil, nil, nil)

	name, ok := svc.CurrentSession()
	if !ok || name != "work" {
		t.Fatalf("CurrentSession = (%q, %v), want (work, true)", name, ok)
	}
	if repo.calls[len(repo.calls)-1] != "current" {
		t.Errorf("expected repo.CurrentSession to be called, calls=%v", repo.calls)
	}
}

// TestDetachSessionDelegates verifies detach forwards to the repo and — unlike
// EnterSession — does NOT refresh LastAttached: that records attach time, and
// detach is the inverse direction.
func TestDetachSessionDelegates(t *testing.T) {
	repo := &fakeRepo{}
	meta := newFakeMeta()
	svc := NewSessionService(repo, meta, nil, nil, nil)

	if err := svc.DetachSession("work"); err != nil {
		t.Fatalf("DetachSession: %v", err)
	}
	if repo.calls[len(repo.calls)-1] != "detach:work" {
		t.Errorf("expected repo detach, calls=%v", repo.calls)
	}
	if meta.lastAttachedCalls != 0 {
		t.Errorf("detach must not update LastAttached, got %d call(s)", meta.lastAttachedCalls)
	}
}

func TestRenameSessionMigratesMetadata(t *testing.T) {
	repo := &fakeRepo{}
	meta := newFakeMeta()
	svc := NewSessionService(repo, meta, nil, nil, nil)

	if err := svc.RenameSession("foo", "bar"); err != nil {
		t.Fatalf("RenameSession: %v", err)
	}
	if repo.calls[len(repo.calls)-1] != "rename:foo->bar" {
		t.Errorf("expected repo rename, calls=%v", repo.calls)
	}
	if len(meta.renameCalls) != 1 || meta.renameCalls[0] != [2]string{"foo", "bar"} {
		t.Errorf("expected meta.Rename(foo,bar), calls=%v", meta.renameCalls)
	}
}

func TestRenameSessionSkipsMetaOnRepoError(t *testing.T) {
	repo := &fakeRepo{renameErr: errors.New("boom")}
	meta := newFakeMeta()
	svc := NewSessionService(repo, meta, nil, nil, nil)

	if err := svc.RenameSession("foo", "bar"); err == nil {
		t.Fatal("expected error from repo")
	}
	if len(meta.renameCalls) != 0 {
		t.Errorf("meta.Rename must not be called when repo fails, calls=%v", meta.renameCalls)
	}
}
