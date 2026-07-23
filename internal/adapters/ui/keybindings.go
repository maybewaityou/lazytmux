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

// KeyBinding is one row of the help panel (and the source the README Key
// Bindings table is derived from). Keeping every binding here means the help
// modal and the docs can never drift apart — add a key once, in this slice.
type KeyBinding struct {
	Group  string
	Key    string
	Action string
}

// keyBindings is the single source of truth for advertised keys, grouped for
// display. Group strings double as the help modal's section headers.
var keyBindings = []KeyBinding{
	{"Navigate", "/", "Search"},
	{"Navigate", "↑↓", "Move"},
	{"Navigate", "Enter", "Enter session"},
	{"Navigate", "q", "Quit"},
	{"Session", "a", "New"},
	{"Session", "e", "Edit"},
	{"Session", "k", "Kill"},
	{"Session", "d", "Detach"},
	{"Session", "r", "Refresh"},
	{"Search/Sort", "s/S", "Cycle sort field"},
	{"Search/Sort", "f", "Filter by tag"},
	{"Metadata", "p", "Pin / unpin"},
	{"Metadata", "t", "Edit tags"},
	{"Metadata", "n", "Edit note"},
	{"Metadata", "c", "Copy tmux attach"},
	{"Other", "?", "Help"},
}
