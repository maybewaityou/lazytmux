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

package domain

import "time"

// Session is a tmux session. Windows is populated lazily by the details view.
type Session struct {
	ID            string    // session_id, e.g. "$1"
	Name          string    // session_name
	Attached      bool      // session_attached (0/1)
	AttachedCount int       // number of attached clients
	Created       time.Time // session_created (unix ts)
	LastActivity  time.Time // session_activity (unix ts)
	WindowsCount  int       // session_windows
	Path          string    // session_path

	// Pinned/Tags/LastAttached are UI metadata; injected from the metadata
	// store, not tmux.
	Pinned       bool
	Tags         []string
	LastAttached time.Time

	// Windows is only filled when details are requested.
	Windows []Window
}

// Window is a single window within a session.
type Window struct {
	Index   int
	Name    string
	Active  bool
	Command string // pane_current_command
}
