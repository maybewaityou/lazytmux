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

package ui

import (
	"sort"
	"time"

	"github.com/maybewaityou/lazytmux/internal/core/domain"
)

// SortMode selects the field used to order the session list.
type SortMode int

const (
	SortByNameAsc SortMode = iota
	SortByCreatedDesc
	SortByActivityDesc
	SortByLastAttachedDesc
)

func (s SortMode) String() string {
	switch s {
	case SortByCreatedDesc:
		return "Created ↓"
	case SortByActivityDesc:
		return "Activity ↓"
	case SortByLastAttachedDesc:
		return "Last Attached ↓"
	default:
		return "Name ↑"
	}
}

// Next cycles to the following sort mode.
func (s SortMode) Next() SortMode { return (s + 1) % 4 }

// sortSessionsForUI sorts in place: pinned sessions always first, then by mode.
func sortSessionsForUI(sessions []domain.Session, mode SortMode, lastAttached func(string) (time.Time, bool)) {
	sort.SliceStable(sessions, func(i, j int) bool {
		if sessions[i].Pinned != sessions[j].Pinned {
			return sessions[i].Pinned
		}
		switch mode {
		case SortByCreatedDesc:
			return sessions[i].Created.After(sessions[j].Created)
		case SortByActivityDesc:
			return sessions[i].LastActivity.After(sessions[j].LastActivity)
		case SortByLastAttachedDesc:
			li, _ := lastAttached(sessions[i].Name)
			lj, _ := lastAttached(sessions[j].Name)
			return li.After(lj)
		default:
			return sessions[i].Name < sessions[j].Name
		}
	})
}
