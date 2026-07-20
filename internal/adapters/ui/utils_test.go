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
	"time"

	"github.com/maybewaityou/lazytmux/internal/core/domain"
)

// The status icon encodes two dimensions in three emoji, with the attached
// fact checked first so it always wins over the 60s activity heuristic:
//
//	⚡ attached          (you are in this session — never kill)
//	⏳ detached + live  (you left, but a task is still running — don't kill)
//	💤 detached + idle  (no recent activity — safe to clean up)
//
// "live" means session_activity is within activeThreshold of now.

func TestFormatSessionLineAttached(t *testing.T) {
	s := domain.Session{
		Name: "main", Attached: true, WindowsCount: 3, Pinned: true,
		LastActivity: time.Now(),
	}
	line := formatSessionLine(s)
	assertContains(t, line, "main")
	assertContains(t, line, "3 win")
	assertContains(t, line, "📌")
	assertContains(t, line, "⚡")
	assertNotContains(t, line, "⏳")
	assertNotContains(t, line, "💤")
	if !strings.Contains(line, "Last Attached: never") {
		t.Errorf("missing last attached time for zero value: %q", line)
	}
}

func TestFormatSessionLineDetachedLive(t *testing.T) {
	s := domain.Session{Name: "build", Attached: false, WindowsCount: 1, LastActivity: time.Now()}
	line := formatSessionLine(s)
	assertContains(t, line, "⏳")
	assertNotContains(t, line, "⚡")
	assertNotContains(t, line, "💤")
	assertNotContains(t, line, "📌")
}

// An attached session shows ⚡ even when idle — the attached fact wins over
// the activity heuristic, so a session you are watching never reads as 💤.
func TestFormatSessionLineAttachedStaysActiveWhenIdle(t *testing.T) {
	stale := time.Now().Add(-10 * time.Minute)
	s := domain.Session{Name: "reading", Attached: true, WindowsCount: 1, LastActivity: stale}
	line := formatSessionLine(s)
	assertContains(t, line, "⚡")
	assertNotContains(t, line, "⏳")
	assertNotContains(t, line, "💤")
}

func TestFormatSessionLineDetachedIdle(t *testing.T) {
	stale := time.Now().Add(-10 * time.Minute)
	s := domain.Session{Name: "old", Attached: false, WindowsCount: 1, LastActivity: stale}
	line := formatSessionLine(s)
	assertContains(t, line, "💤")
	assertNotContains(t, line, "⚡")
	assertNotContains(t, line, "⏳")
}

// A zero LastActivity (e.g. freshly constructed, never populated) on a
// detached session is treated as "never active" and therefore idle, mirroring
// humanizeDuration's "never". An attached session with zero activity still
// shows ⚡, since the attached fact takes priority.
func TestFormatSessionLineDetachedZeroActivity(t *testing.T) {
	s := domain.Session{Name: "blank", Attached: false, WindowsCount: 1}
	line := formatSessionLine(s)
	assertContains(t, line, "💤")
	assertNotContains(t, line, "⚡")
	assertNotContains(t, line, "⏳")
}

// TestFormatSessionLineAlignment verifies that the Name and Last Attached
// columns line up across rows of different name lengths: the fixed-width
// padding must place "Last Attached:" at the same byte offset in every line.
func TestFormatSessionLineAlignment(t *testing.T) {
	short := formatSessionLine(domain.Session{Name: "api", WindowsCount: 1})
	long := formatSessionLine(domain.Session{Name: "a-much-longer-name", WindowsCount: 99})
	iShort := strings.Index(short, "Last Attached:")
	iLong := strings.Index(long, "Last Attached:")
	if iShort < 0 || iLong < 0 {
		t.Fatalf("missing Last Attached column:\n short=%q\n long=%q", short, long)
	}
	if iShort != iLong {
		t.Errorf("Last Attached column not aligned: short@%d long@%d\n short=%q\n long=%q", iShort, iLong, short, long)
	}
}

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("expected output to contain %q\n got: %q", substr, s)
	}
}

func assertNotContains(t *testing.T, s, substr string) {
	t.Helper()
	if strings.Contains(s, substr) {
		t.Errorf("expected output NOT to contain %q\n got: %q", substr, s)
	}
}
