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
	"strings"
	"testing"

	"github.com/maybewaityou/lazytmux/internal/core/domain"
)

// TestSelectByName verifies the cursor lands on the matching session and that a
// missing name leaves the cursor untouched (so a vanished session degrades to
// wherever UpdateSessions last placed it).
func TestSelectByName(t *testing.T) {
	sl := NewSessionList()
	sl.UpdateSessions([]domain.Session{
		{Name: "alpha"},
		{Name: "beta"},
		{Name: "gamma"},
	})

	if got := sl.GetCurrentItem(); got != 0 {
		t.Fatalf("initial cursor = %d, want 0", got)
	}

	sl.SelectByName("beta")
	if got := sl.GetCurrentItem(); got != 1 {
		t.Errorf("after SelectByName(beta) cursor = %d, want 1", got)
	}

	// A missing name must not move the cursor.
	sl.SelectByName("does-not-exist")
	if got := sl.GetCurrentItem(); got != 1 {
		t.Errorf("after SelectByName(missing) cursor = %d, want 1 (unchanged)", got)
	}

	// GetSelected should reflect the restored cursor.
	if s, ok := sl.GetSelected(); !ok || s.Name != "beta" {
		t.Errorf("GetSelected = (%+v, %v), want (beta, true)", s, ok)
	}
}

// TestSelectByNameRestoresAfterReload mirrors the refresh() flow: UpdateSessions
// snaps the cursor back to the first item on every reload, and SelectByName is
// what brings the user's previous selection back.
func TestSelectByNameRestoresAfterReload(t *testing.T) {
	sl := NewSessionList()
	sessions := []domain.Session{{Name: "alpha"}, {Name: "beta"}, {Name: "gamma"}}
	sl.UpdateSessions(sessions)
	sl.SelectByName("beta")
	if got := sl.GetCurrentItem(); got != 1 {
		t.Fatalf("cursor before reload = %d, want 1", got)
	}

	// Reload (as refresh() does) — UpdateSessions resets the cursor to 0.
	sl.UpdateSessions(sessions)
	if got := sl.GetCurrentItem(); got != 0 {
		t.Fatalf("cursor after reload = %d, want 0", got)
	}

	// SelectByName restores the previously selected session.
	sl.SelectByName("beta")
	if got := sl.GetCurrentItem(); got != 1 {
		t.Errorf("cursor after restore = %d, want 1", got)
	}
}

// TestSetCurrentMarksRenderedLine verifies that after SetCurrent, UpdateSessions
// renders only the current session's row with ▶ and leaves other rows unmarked.
// This proves the component holds the current-session state and flows it into
// rendering.
func TestSetCurrentMarksRenderedLine(t *testing.T) {
	sl := NewSessionList()
	sl.SetCurrent("beta")
	sl.UpdateSessions([]domain.Session{
		{Name: "alpha", WindowsCount: 1},
		{Name: "beta", WindowsCount: 1},
		{Name: "gamma", WindowsCount: 1},
	})

	alpha, _ := sl.GetItemText(0)
	beta, _ := sl.GetItemText(1)
	gamma, _ := sl.GetItemText(2)

	if strings.Contains(alpha, "▶") || strings.Contains(gamma, "▶") {
		t.Errorf("non-current rows must not carry ▶:\n alpha=%q\n gamma=%q", alpha, gamma)
	}
	if !strings.Contains(beta, "▶") {
		t.Errorf("current row must carry ▶, got: %q", beta)
	}
}
