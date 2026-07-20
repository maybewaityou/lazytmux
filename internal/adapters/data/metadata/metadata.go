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
	s.mu.Lock()
	defer s.mu.Unlock()
	if pinned {
		s.data.Pins[name] = true
	} else {
		delete(s.data.Pins, name)
	}
	return s.save()
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
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(tags) == 0 {
		delete(s.data.Tags, name)
	} else {
		s.data.Tags[name] = tags
	}
	return s.save()
}

func (s *Store) SetLastAttached(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.LastAttached[name] = time.Now().Unix()
	return s.save()
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
