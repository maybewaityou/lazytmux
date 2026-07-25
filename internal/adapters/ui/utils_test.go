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
	line := formatSessionLine(s, "")
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
	line := formatSessionLine(s, "")
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
	line := formatSessionLine(s, "")
	assertContains(t, line, "⚡")
	assertNotContains(t, line, "⏳")
	assertNotContains(t, line, "💤")
}

func TestFormatSessionLineDetachedIdle(t *testing.T) {
	stale := time.Now().Add(-10 * time.Minute)
	s := domain.Session{Name: "old", Attached: false, WindowsCount: 1, LastActivity: stale}
	line := formatSessionLine(s, "")
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
	line := formatSessionLine(s, "")
	assertContains(t, line, "💤")
	assertNotContains(t, line, "⚡")
	assertNotContains(t, line, "⏳")
}

// TestFormatSessionLineAlignment verifies that the Name and Last Attached
// columns line up across rows of different name lengths: the fixed-width
// padding must place "Last Attached:" at the same byte offset in every line.
func TestFormatSessionLineAlignment(t *testing.T) {
	short := formatSessionLine(domain.Session{Name: "api", WindowsCount: 1}, "")
	long := formatSessionLine(domain.Session{Name: "a-much-longer-name", WindowsCount: 99}, "")
	iShort := strings.Index(short, "Last Attached:")
	iLong := strings.Index(long, "Last Attached:")
	if iShort < 0 || iLong < 0 {
		t.Fatalf("missing Last Attached column:\n short=%q\n long=%q", short, long)
	}
	if iShort != iLong {
		t.Errorf("Last Attached column not aligned: short@%d long@%d\n short=%q\n long=%q", iShort, iLong, short, long)
	}
}

// TestFormatSessionLineHighlightsCurrentName verifies the current session's
// name is rendered with the accent color (colorAccent::b) so it stands out
// beyond the ▶ marker, while every other session keeps the default primary +
// bold (colorPrimary::b).
func TestFormatSessionLineHighlightsCurrentName(t *testing.T) {
	s := domain.Session{Name: "main", WindowsCount: 1, LastActivity: time.Now()}
	if line := formatSessionLine(s, "main"); !strings.Contains(line, colorAccent+"::b]") {
		t.Errorf("current session name must use accent color, got: %q", line)
	}
	if line := formatSessionLine(s, "main"); strings.Contains(line, colorPrimary+"::b]") {
		t.Errorf("current session name must not keep primary color, got: %q", line)
	}
	if line := formatSessionLine(s, "other"); !strings.Contains(line, colorPrimary+"::b]") {
		t.Errorf("non-current name must use primary color, got: %q", line)
	}
}

// TestFormatSessionLineMarksCurrent verifies that the current session line is
// prefixed with ▶ and that non-current or empty-current lines are not.
func TestFormatSessionLineMarksCurrent(t *testing.T) {
	s := domain.Session{Name: "main", WindowsCount: 1, LastActivity: time.Now()}
	if line := formatSessionLine(s, "main"); !strings.Contains(line, "▶") {
		t.Errorf("current session line must contain ▶, got: %q", line)
	}
	if line := formatSessionLine(s, "other"); strings.Contains(line, "▶") {
		t.Errorf("non-current line must not contain ▶, got: %q", line)
	}
	if line := formatSessionLine(s, ""); strings.Contains(line, "▶") {
		t.Errorf("empty-current line must not contain ▶, got: %q", line)
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

// Tag chips render as black text on a colorAccent (#7aa2f7) pill, space-padded
// inside the brackets so adjacent chips don't touch. The literal hex values
// are intentional — the chip color is a visual contract, not an incidental
// constant, so the test locks it down rather than aliasing colorAccent.
func TestRenderTagChips(t *testing.T) {
	if got := renderTagChips(nil); got != "" {
		t.Errorf("nil tags: want empty, got %q", got)
	}
	if got := renderTagChips([]string{}); got != "" {
		t.Errorf("empty tags: want empty, got %q", got)
	}
	one := renderTagChips([]string{"work"})
	wantOne := "[black:#7aa2f7] work [-:-:-]"
	if one != wantOne {
		t.Errorf("single chip: want %q, got %q", wantOne, one)
	}
	three := renderTagChips([]string{"a", "b", "c"})
	wantThree := "[black:#7aa2f7] a [-:-:-] [black:#7aa2f7] b [-:-:-] [black:#7aa2f7] c [-:-:-]"
	if three != wantThree {
		t.Errorf("three chips (no truncation in details): want %q, got %q", wantThree, three)
	}
}

func TestRenderTagBadgesForList(t *testing.T) {
	if got := renderTagBadgesForList(nil); got != "" {
		t.Errorf("nil tags: want empty, got %q", got)
	}
	two := renderTagBadgesForList([]string{"a", "b"})
	wantTwo := "[black:#7aa2f7] a [-:-:-] [black:#7aa2f7] b [-:-:-]"
	if two != wantTwo {
		t.Errorf("two badges (no overflow): want %q, got %q", wantTwo, two)
	}
	three := renderTagBadgesForList([]string{"a", "b", "c"})
	wantThree := "[black:#7aa2f7] a [-:-:-] [black:#7aa2f7] b [-:-:-] [#414868]+1[-]"
	if three != wantThree {
		t.Errorf("three badges (truncate to 2 + +1): want %q, got %q", wantThree, three)
	}
	four := renderTagBadgesForList([]string{"a", "b", "c", "d"})
	wantFour := "[black:#7aa2f7] a [-:-:-] [black:#7aa2f7] b [-:-:-] [#414868]+2[-]"
	if four != wantFour {
		t.Errorf("four badges (truncate to 2 + +2): want %q, got %q", wantFour, four)
	}
}

func TestFormatSessionLineTagsAppended(t *testing.T) {
	s := domain.Session{
		Name: "api", WindowsCount: 1, LastActivity: time.Now(),
		Tags: []string{"work", "urgent"},
	}
	line := formatSessionLine(s, "")
	if !strings.Contains(line, "[black:"+colorAccent+"] work [-:-:-]") {
		t.Errorf("tag chip missing in list line: %q", line)
	}
	if !strings.Contains(line, "[black:"+colorAccent+"] urgent [-:-:-]") {
		t.Errorf("second tag chip missing: %q", line)
	}
	idxLA := strings.Index(line, "Last Attached:")
	idxTag := strings.Index(line, "[black:")
	if idxLA < 0 || idxTag < 0 || idxTag < idxLA {
		t.Errorf("tags must follow Last Attached: la=%d tag=%d line=%q", idxLA, idxTag, line)
	}
}

func TestFormatSessionLineNoTagTailWhenEmpty(t *testing.T) {
	s := domain.Session{Name: "api", WindowsCount: 1, LastActivity: time.Now()}
	line := formatSessionLine(s, "")
	if strings.Contains(line, "[black:") {
		t.Errorf("no tag chips expected when Tags empty: %q", line)
	}
}

func TestFormatSessionLineTagsTruncated(t *testing.T) {
	s := domain.Session{
		Name: "api", WindowsCount: 1, LastActivity: time.Now(),
		Tags: []string{"a", "b", "c", "d"},
	}
	line := formatSessionLine(s, "")
	if !strings.Contains(line, "["+colorDim+"]+2[-]") {
		t.Errorf("expected +2 overflow marker: %q", line)
	}
}
