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
