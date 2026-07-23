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
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/maybewaityou/lazytmux/internal/core/ports"
)

// fileModel is the on-disk JSON shape.
type fileModel struct {
	Pins         map[string]bool     `json:"pins"`
	Tags         map[string][]string `json:"tags"`
	LastAttached map[string]int64    `json:"lastAttached"`
	Notes        map[string]string   `json:"notes"`
}

// Store implements ports.MetadataStore with thread-safe in-memory state
// backed by a JSON file written atomically.
type Store struct {
	mu   sync.RWMutex
	path string
	data fileModel
}

// NewStore loads (or creates) the metadata store at path.
func NewStore(path string) (*Store, error) {
	s := &Store{
		path: path,
		data: fileModel{
			Pins:         map[string]bool{},
			Tags:         map[string][]string{},
			LastAttached: map[string]int64{},
			Notes:        map[string]string{},
		},
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	b, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(b) == 0 {
		return nil
	}
	return json.Unmarshal(b, &s.data)
}

// reloadLocked refreshes the in-memory snapshot from the latest on-disk state.
// It must be called with the write lock held. A missing or empty file leaves the
// snapshot untouched (the maps are already initialized, and between mutations the
// snapshot already matches the last successful save).
//
// The JSON is decoded into a fresh fileModel rather than &s.data because
// json.Unmarshal *merges* into existing maps — it never drops keys absent from
// the payload — so reusing s.data would leave stale entries behind.
func (s *Store) reloadLocked() error {
	b, err := os.ReadFile(s.path)
	if os.IsNotExist(err) || len(b) == 0 {
		return nil
	}
	if err != nil {
		return err
	}
	fresh := fileModel{
		Pins:         map[string]bool{},
		Tags:         map[string][]string{},
		LastAttached: map[string]int64{},
		Notes:        map[string]string{},
	}
	if err := json.Unmarshal(b, &fresh); err != nil {
		return err
	}
	s.data = fresh
	return nil
}

// mutate re-reads the freshest on-disk state under the write lock, applies fn to
// it, and persists atomically. Re-reading before every write is what stops one
// lazytmux instance from silently clobbering another's concurrent change (common
// inside tmux, where several instances often share the file): each mutation
// becomes a read-modify-write against the latest file instead of a blind
// overwrite of the snapshot captured at startup.
func (s *Store) mutate(fn func(d *fileModel)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.reloadLocked(); err != nil {
		return err
	}
	fn(&s.data)
	return s.save()
}

// save backs up the original file the first time, then writes atomically.
func (s *Store) save() error {
	if err := s.backupOriginalOnce(); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *Store) backupOriginalOnce() error {
	backup := s.path + ".original.backup"
	if _, err := os.Stat(backup); err == nil {
		return nil // already exists, never overwrite
	}
	src, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return os.WriteFile(backup, src, 0o644)
}

func (s *Store) IsPinned(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.Pins[name]
}

func (s *Store) SetPinned(name string, pinned bool) error {
	return s.mutate(func(d *fileModel) {
		if pinned {
			d.Pins[name] = true
		} else {
			delete(d.Pins, name)
		}
	})
}

func (s *Store) Tags(name string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, len(s.data.Tags[name]))
	copy(out, s.data.Tags[name])
	sort.Strings(out)
	return out
}

func (s *Store) SetTags(name string, tags []string) error {
	return s.mutate(func(d *fileModel) {
		if len(tags) == 0 {
			delete(d.Tags, name)
		} else {
			d.Tags[name] = tags
		}
	})
}

func (s *Store) Note(name string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.Notes[name]
}

func (s *Store) SetNote(name string, note string) error {
	return s.mutate(func(d *fileModel) {
		if note == "" {
			delete(d.Notes, name)
		} else {
			d.Notes[name] = note
		}
	})
}

func (s *Store) SetLastAttached(name string) error {
	return s.mutate(func(d *fileModel) {
		d.LastAttached[name] = time.Now().Unix()
	})
}

// Rename moves all metadata (pin/tags/lastAttached) from oldName to newName
// with mv semantics: newName becomes an exact copy of oldName's state
// (overwriting any stale entry at newName), then oldName is removed.
// A single atomic save writes the result.
func (s *Store) Rename(oldName, newName string) error {
	return s.mutate(func(d *fileModel) {
		relocate(d.Pins, oldName, newName)
		relocate(d.Tags, oldName, newName)
		relocate(d.LastAttached, oldName, newName)
		relocate(d.Notes, oldName, newName)
	})
}

// relocate moves a single map entry with mv semantics: the target is cleared
// first (overwrite), then the source value — if present — is moved across,
// then the source key is removed. Generic over the value type so it serves all
// three metadata maps without reflection.
func relocate[V any](m map[string]V, oldName, newName string) {
	delete(m, newName)
	if v, ok := m[oldName]; ok {
		m[newName] = v
	}
	delete(m, oldName)
}

func (s *Store) LastAttached(name string) (time.Time, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ts, ok := s.data.LastAttached[name]
	if !ok {
		return time.Time{}, false
	}
	return time.Unix(ts, 0), true
}

// Compile-time check that Store satisfies the port.
var _ ports.MetadataStore = (*Store)(nil)
