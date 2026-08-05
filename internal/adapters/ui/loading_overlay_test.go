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
)

// TestSpinnerFrame verifies the spinner index wraps around the glyph cycle
// rather than going out of bounds, so the animation step can keep incrementing.
func TestSpinnerFrame(t *testing.T) {
	n := len(spinnerFrames)
	if n == 0 {
		t.Fatal("spinnerFrames must not be empty")
	}
	// Step 0 is the first glyph.
	if spinnerFrame(0) != spinnerFrames[0] {
		t.Errorf("spinnerFrame(0) = %q, want %q", spinnerFrame(0), spinnerFrames[0])
	}
	// Index equal to the cycle length wraps back to the first glyph.
	if spinnerFrame(n) != spinnerFrames[0] {
		t.Errorf("spinnerFrame(%d) = %q, want wrap to %q", n, spinnerFrame(n), spinnerFrames[0])
	}
	// An interior index maps to the obvious glyph.
	if spinnerFrame(3) != spinnerFrames[3] {
		t.Errorf("spinnerFrame(3) = %q, want %q", spinnerFrame(3), spinnerFrames[3])
	}
}

// TestLoadingText verifies the loading line carries the message, the current
// spinner glyph, and the Tokyo Night color tags (cyan glyph + primary text).
func TestLoadingText(t *testing.T) {
	got := loadingText("Creating session…", 0)
	if !strings.Contains(got, "Creating session…") {
		t.Errorf("loadingText missing message, got %q", got)
	}
	if !strings.Contains(got, string(spinnerFrames[0])) {
		t.Errorf("loadingText missing spinner glyph, got %q", got)
	}
	if !strings.Contains(got, colorCyan) {
		t.Errorf("loadingText missing colorCyan tag, got %q", got)
	}
	if !strings.Contains(got, colorPrimary) {
		t.Errorf("loadingText missing colorPrimary tag, got %q", got)
	}
	// Advancing the frame swaps the glyph in the composed text.
	next := loadingText("Creating session…", 1)
	if strings.Contains(next, string(spinnerFrames[0])) {
		t.Errorf("frame 1 should drop glyph 0, got %q", next)
	}
	if !strings.Contains(next, string(spinnerFrames[1])) {
		t.Errorf("frame 1 should carry glyph 1, got %q", next)
	}
}

// TestNewLoadingOverlaySeedsMessage verifies the overlay starts at frame 0 with
// the requested message and a non-nil modal.
func TestNewLoadingOverlaySeedsMessage(t *testing.T) {
	o := NewLoadingOverlay("Killing session…")
	if o.message != "Killing session…" {
		t.Errorf("message = %q, want %q", o.message, "Killing session…")
	}
	if o.frame != 0 {
		t.Errorf("initial frame = %d, want 0", o.frame)
	}
	if o.text == nil {
		t.Error("text widget must be set")
	}
	if o.root == nil {
		t.Error("root primitive must be set")
	}
}

// TestLoadingOverlayAdvanceStepsFrame verifies advance increments the frame
// counter on every tick and crosses the cycle boundary without panicking — the
// glyph wraps via modulo inside loadingText, while the counter keeps climbing.
func TestLoadingOverlayAdvanceStepsFrame(t *testing.T) {
	o := NewLoadingOverlay("Creating session…")
	for i := 1; i <= len(spinnerFrames)+2; i++ {
		o.advance() // must not panic crossing the cycle boundary
		if o.frame != i {
			t.Errorf("after %d advances frame = %d, want %d", i, o.frame, i)
		}
	}
}
