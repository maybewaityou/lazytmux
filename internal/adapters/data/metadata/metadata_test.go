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

package metadata

import (
	"path/filepath"
	"sort"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "metadata.json")
	s, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return s
}

func TestPin(t *testing.T) {
	s := newTestStore(t)
	if s.IsPinned("main") {
		t.Fatal("default should be unpinned")
	}
	if err := s.SetPinned("main", true); err != nil {
		t.Fatalf("SetPinned: %v", err)
	}
	if !s.IsPinned("main") {
		t.Fatal("should be pinned after set")
	}
	if err := s.SetPinned("main", false); err != nil {
		t.Fatalf("SetPinned: %v", err)
	}
	if s.IsPinned("main") {
		t.Fatal("should be unpinned after unset")
	}
}

func TestTags(t *testing.T) {
	s := newTestStore(t)
	if got := s.Tags("main"); len(got) != 0 {
		t.Fatalf("default tags: %v", got)
	}
	if err := s.SetTags("main", []string{"prod", "work"}); err != nil {
		t.Fatalf("SetTags: %v", err)
	}
	got := s.Tags("main")
	sort.Strings(got)
	if len(got) != 2 || got[0] != "prod" || got[1] != "work" {
		t.Fatalf("tags: %v", got)
	}
}

func TestLastAttached(t *testing.T) {
	s := newTestStore(t)
	if _, ok := s.LastAttached("main"); ok {
		t.Fatal("should be absent initially")
	}
	before := time.Now().Unix()
	if err := s.SetLastAttached("main"); err != nil {
		t.Fatalf("SetLastAttached: %v", err)
	}
	la, ok := s.LastAttached("main")
	if !ok {
		t.Fatal("should be present after set")
	}
	if la.Unix() < before {
		t.Fatal("lastAttached timestamp too old")
	}
}

func TestPersistenceAcrossInstances(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metadata.json")
	s1, _ := NewStore(path)
	_ = s1.SetPinned("dev", true)

	s2, err := NewStore(path) // reload from disk
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if !s2.IsPinned("dev") {
		t.Fatal("pin did not persist to disk")
	}
}

func TestRename(t *testing.T) {
	s := newTestStore(t)
	_ = s.SetPinned("foo", true)
	_ = s.SetTags("foo", []string{"work"})
	_ = s.SetLastAttached("foo")
	before, _ := s.LastAttached("foo")

	if err := s.Rename("foo", "bar"); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	// old name fully cleared
	if s.IsPinned("foo") || len(s.Tags("foo")) != 0 {
		t.Errorf("old name should be cleared, pinned=%v tags=%v", s.IsPinned("foo"), s.Tags("foo"))
	}
	// pin migrated
	if !s.IsPinned("bar") {
		t.Error("pin did not migrate to bar")
	}
	// tags migrated
	got := s.Tags("bar")
	sort.Strings(got)
	if len(got) != 1 || got[0] != "work" {
		t.Errorf("tags did not migrate to bar: %v", got)
	}
	// lastAttached migrated AND timestamp preserved
	la, ok := s.LastAttached("bar")
	if !ok {
		t.Fatal("lastAttached did not migrate to bar")
	}
	if !la.Equal(before) {
		t.Errorf("lastAttached timestamp not preserved: got %v want %v", la, before)
	}
}

func TestRenameOverwritesTarget(t *testing.T) {
	s := newTestStore(t)
	_ = s.SetTags("foo", []string{"work"})
	_ = s.SetTags("bar", []string{"stale-orphan"}) // pre-existing entry at target

	if err := s.Rename("foo", "bar"); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	got := s.Tags("bar")
	sort.Strings(got)
	// mv semantics: bar becomes an exact copy of foo, overwriting the orphan.
	if len(got) != 1 || got[0] != "work" {
		t.Errorf("target should be overwritten with source tags, got %v", got)
	}
	if len(s.Tags("foo")) != 0 {
		t.Errorf("source should be removed, got %v", s.Tags("foo"))
	}
}

func TestSetTagsEmptyDeletes(t *testing.T) {
	s := newTestStore(t)
	_ = s.SetTags("foo", []string{"work"})
	if err := s.SetTags("foo", nil); err != nil {
		t.Fatalf("SetTags(nil): %v", err)
	}
	if got := s.Tags("foo"); len(got) != 0 {
		t.Errorf("expected empty after SetTags(nil), got %v", got)
	}
	// key must be gone entirely, not left as a zero-length slice
	if _, ok := s.data.Tags["foo"]; ok {
		t.Error("expected tags key to be deleted, not left empty")
	}
}
